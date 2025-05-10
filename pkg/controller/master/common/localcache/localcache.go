package localcache

import (
	"fmt"
	"path"
	"strconv"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	ctlbatchv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	ctlrbacv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkglabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	ctlsnapshotv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/snapshot.storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

const (
	// Constant definitions
	cacheVolumeName         = "cache-volume"
	cacheVolumeMountPath    = "/cache"
	defaultPVCSize          = "10Gi"
	cacheJobTimeout         = 3600 // 1 hour timeout
	serviceAccountName      = "local-cache"
	storageClassName        = "llmos-ceph-block"
	volumeSnapshotClassName = "llmos-ceph-block-snapshot-class"

	localCacheLabelKey     = "ml.llmos.ai/local-cache"
	modelLabelKey          = "ml.llmos.ai/model-name"
	datasetversionLabelKey = "ml.llmos.ai/dataset-version-name"
)

// Note: Regardless of job, pvc and snapshot, they are in the same namespace and has the same name
// When starting job, we create PVC first with generate name, then create job and volume snapshot
// with the name of the pvc. When stopping job, we get the job name from the cache status, then
// delete the job and pvc.

// CacheableResource interface defines the methods that a cacheable resource must implement
type CacheableResource interface {
	runtime.Object
	metav1.Object
	GetLocalCacheState() mlv1.CacheStateType
	GetCacheStatus() *mlv1.CacheStatus
	SetCacheStatus(status *mlv1.CacheStatus) error
	GetResourcePath() string
	GetResourceType() string
	GetResourceVersion() string
	GetRegistry() string
}

type Handler struct {
	JobClient                ctlbatchv1.JobClient
	JobCache                 ctlbatchv1.JobCache
	PVCClient                ctlcorev1.PersistentVolumeClaimClient
	PVCCache                 ctlcorev1.PersistentVolumeClaimCache
	VolumeSnapshotClient     ctlsnapshotv1.VolumeSnapshotClient
	VolumeSnapshotCache      ctlsnapshotv1.VolumeSnapshotCache
	ServiceAccountClient     ctlcorev1.ServiceAccountClient
	ServiceAccountCache      ctlcorev1.ServiceAccountCache
	ClusterRoleBindingClient ctlrbacv1.ClusterRoleBindingClient
	ClusterRoleBindingCache  ctlrbacv1.ClusterRoleBindingCache
	ModelClient              ctlmlv1.ModelClient
	ModelCache               ctlmlv1.ModelCache
	DatasetVersionClient     ctlmlv1.DatasetVersionClient
	DatasetVersionCache      ctlmlv1.DatasetVersionCache

	downloadImage string
}

// NewHandler creates a new local cache handler
func NewHandler(mgmt *config.Management) *Handler {
	jobs := mgmt.BatchFactory.Batch().V1().Job()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	models := mgmt.LLMFactory.Ml().V1().Model()
	datasetVersions := mgmt.LLMFactory.Ml().V1().DatasetVersion()
	serviceAccounts := mgmt.CoreFactory.Core().V1().ServiceAccount()
	clusterRoleBindings := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding()
	volumeSnapshots := mgmt.SnapshotFactory.Snapshot().V1().VolumeSnapshot()

	return &Handler{
		JobClient:                jobs,
		JobCache:                 jobs.Cache(),
		PVCClient:                pvcs,
		PVCCache:                 pvcs.Cache(),
		ModelClient:              models,
		ModelCache:               models.Cache(),
		DatasetVersionClient:     datasetVersions,
		DatasetVersionCache:      datasetVersions.Cache(),
		VolumeSnapshotClient:     volumeSnapshots,
		VolumeSnapshotCache:      volumeSnapshots.Cache(),
		ServiceAccountClient:     serviceAccounts,
		ServiceAccountCache:      serviceAccounts.Cache(),
		ClusterRoleBindingClient: clusterRoleBindings,
		ClusterRoleBindingCache:  clusterRoleBindings.Cache(),

		downloadImage: settings.LocalCacheDownloaderImage.Get(),
	}
}

// ReconcileCache handles the cache state changes of the resource
func (h *Handler) ReconcileCache(resource CacheableResource, pvcSize int64) error {
	log := logrus.WithFields(logrus.Fields{
		"namespace": resource.GetNamespace(),
		"name":      resource.GetName(),
		"type":      resource.GetResourceType(),
	})

	// Get current cache status
	cacheState := resource.GetLocalCacheState()
	cacheStatus := resource.GetCacheStatus()
	// Process based on cache state and request status
	switch {
	case cacheState == mlv1.CacheStateActive && (cacheStatus == nil || cacheStatus.Status == mlv1.CacheStatusIdle):
		// User requests to activate cache, and there is no cache operation currently
		log.Info("Starting cache operation")
		return h.startCacheJob(resource, pvcSize)

	case cacheState == mlv1.CacheStateActive && cacheStatus != nil &&
		(cacheStatus.Status == mlv1.CacheStatusCompleted || cacheStatus.Status == mlv1.CacheStatusFailed):
		// Cache operation is completed or failed, but user still requests to activate, restart cache
		log.Info("Restarting cache operation")
		return h.startCacheJob(resource, pvcSize)

	case cacheState == mlv1.CacheStateInactive && cacheStatus != nil && cacheStatus.Status == mlv1.CacheStatusDownloading:
		// User requests to stop cache, and cache operation is in progress
		log.Info("Stopping cache operation")
		return h.stopCacheJob(resource)

	default:
		// No action needed for other cases
		return nil
	}
}

// startCacheJob starts the cache job
func (h *Handler) startCacheJob(resource CacheableResource, minPvcSize int64) error {
	// Check if job with same name exists
	labels := makeLabels(resource.GetResourceType(), resource.GetName())
	jobs, err := h.JobCache.List(resource.GetNamespace(), pkglabels.Set(labels).AsSelector())
	if err != nil {
		return fmt.Errorf("failed to list jobs with label selector %v: %w", labels, err)
	}
	for _, job := range jobs {
		if job.Status.Succeeded > 0 || job.Status.Failed > 0 || isJobTimedOut(job) {
			continue
		}
		// Job already exists, skip creation
		return nil
	}

	logrus.Debugf("start to create job for %s/%s", resource.GetNamespace(), resource.GetName())

	// Create service account and cluster role binding
	if err = h.ensureServiceAccountAndRoleBinding(resource); err != nil {
		return err
	}

	// Create PVC
	pvc, err := h.createOrRestorePVC(resource, minPvcSize)
	if err != nil {
		return err
	}

	// Create download job
	job, err := h.createCacheJob(resource, pvc.Name)
	if err != nil {
		return err
	}

	// Update resource status
	if err := resource.SetCacheStatus(&mlv1.CacheStatus{
		Status:         mlv1.CacheStatusDownloading,
		CacheMessage:   "Cache job started",
		JobName:        job.Name,
		VolumeSnapshot: "",
	}); err != nil {
		return fmt.Errorf("failed to update cache status: %w", err)
	}

	return nil
}

// createOrRestorePVC creates a new PVC or restores PVC from snapshot
func (h *Handler) createOrRestorePVC(resource CacheableResource, pvcSize int64) (*corev1.PersistentVolumeClaim, error) {
	// Check if PVC exists
	labels := makeLabels(resource.GetResourceType(), resource.GetName())
	pvcs, err := h.PVCCache.List(resource.GetNamespace(), pkglabels.Set(labels).AsSelector())
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc with label selector %v: %w", labels, err)
	}
	if len(pvcs) == 1 {
		// Job already exists, skip creation
		return pvcs[0], nil
	}
	if len(pvcs) > 1 {
		return nil, fmt.Errorf("found more than one pvc with label selector %v, expected one", labels)
	}

	logrus.Debugf("start to create pvc for %s/%s", resource.GetNamespace(), resource.GetName())

	// Create PVC
	generateName := fmt.Sprintf("%s-%s-", resource.GetResourceType(), resource.GetName())
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
			Namespace:    resource.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: resource.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:       resource.GetObjectKind().GroupVersionKind().Kind,
					Name:       resource.GetName(),
					UID:        resource.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: ptr.To(storageClassName),
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}

	// If snapshot exists, restore from snapshot
	// The size of pvc must no be less than the snapshot restore size
	cacheStatus := resource.GetCacheStatus()
	if cacheStatus != nil && cacheStatus.VolumeSnapshot != "" {
		vs, err := h.VolumeSnapshotCache.Get(resource.GetNamespace(), cacheStatus.VolumeSnapshot)
		if err == nil {
			// Check if the snapshot is ready to use before using it as a data source
			if vs.Status != nil && vs.Status.ReadyToUse != nil && *vs.Status.ReadyToUse {
				pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
					APIGroup: ptr.To("snapshot.storage.k8s.io"),
					Kind:     "VolumeSnapshot",
					Name:     cacheStatus.VolumeSnapshot,
				}
				if vs.Status.RestoreSize != nil && pvcSize < vs.Status.RestoreSize.Value() {
					pvcSize = vs.Status.RestoreSize.Value()
				}
			} else {
				logrus.Warnf("snapshot %s is not ready to use, skip restore from snapshot", cacheStatus.VolumeSnapshot)
			}
		} else {
			logrus.Warnf("failed to get snapshot %s: %v", cacheStatus.VolumeSnapshot, err)
		}
	}
	pvc.Spec.Resources = corev1.VolumeResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: apiresource.MustParse(calculateCacheSize(pvcSize)),
		},
	}

	createdPVC, err := h.PVCClient.Create(pvc)
	if err != nil {
		return nil, err
	}

	logrus.Infof("create pvc %s/%s", createdPVC.Namespace, createdPVC.Name)

	return createdPVC, nil
}

// createCacheJob creates the cache job
func (h *Handler) createCacheJob(resource CacheableResource, name string) (*batchv1.Job, error) {
	labels := makeLabels(resource.GetResourceType(), resource.GetName())
	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resource.GetNamespace(),
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: resource.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:       resource.GetObjectKind().GroupVersionKind().Kind,
					Name:       resource.GetName(),
					UID:        resource.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: batchv1.JobSpec{
			// TODO: set backoff limit and add the PodFailurePolicy to terminate the failed pods to release the PVC
			BackoffLimit:            ptr.To(int32(1)),
			TTLSecondsAfterFinished: ptr.To(int32(86400)), // Preserve job for 24 hours after completion
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:                 corev1.RestartPolicyNever,
					TerminationGracePeriodSeconds: ptr.To(int64(30)),
					ServiceAccountName:            serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "downloader",
							Image: h.downloadImage,
							Args: []string{
								fmt.Sprintf("--type=%s", resource.GetResourceType()),
								fmt.Sprintf("--namespace=%s", resource.GetNamespace()),
								fmt.Sprintf("--name=%s", resource.GetName()),
								fmt.Sprintf("--output-dir=%s", path.Join(cacheVolumeMountPath, resource.GetResourceType())),
								"--debug=true",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      cacheVolumeName,
									MountPath: cacheVolumeMountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: cacheVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
				},
			},
		},
	}

	createdJob, err := h.JobClient.Create(job)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache job %s/%s: %w", job.Namespace, job.Name, err)
	}

	return createdJob, nil
}

// ReconcileJob reconciles the job status
func (h *Handler) ReconcileJob(job *batchv1.Job) error {
	if job == nil || job.DeletionTimestamp != nil {
		return nil
	}

	if len(job.OwnerReferences) == 0 ||
		job.Labels[localCacheLabelKey] != strconv.FormatBool(true) {
		return nil
	}

	resource, err := h.getResourceFromOwnerReference(job.Namespace, job.OwnerReferences[0])
	if err != nil {
		return fmt.Errorf("failed to get resource from owner reference: %w", err)
	}

	if resource.GetLocalCacheState() != mlv1.CacheStateActive {
		return nil
	}

	// Check if job is completed
	if job.Status.Succeeded > 0 {
		// Job completed successfully
		logrus.Infof("job %s/%s succeeded, going to create volume snapshot", job.Namespace, job.Name)
		return h.createVolumeSnapshot(resource, job.Name)
	} else if job.Status.Failed > 0 || isJobTimedOut(job) {
		// Job failed or timed out
		return h.handleJobFailure(resource, job)
	}

	// Job is still running
	return nil
}

// getResourceFromOwnerReference retrieves the CacheableResource from an owner reference
// The CacheableResource got from the owner reference will invoke the client to update the status
func (h *Handler) getResourceFromOwnerReference(
	namespace string,
	ownerReference metav1.OwnerReference,
) (CacheableResource, error) {
	// Check the kind of the owner reference to determine which resource to fetch
	switch ownerReference.Kind {
	case "Model":
		// Fetch the Model resource
		model, err := h.ModelCache.Get(namespace, ownerReference.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Model resource: %w", err)
		}

		// Create a ModelAdapter that implements CacheableResource
		return NewModelCacheAdapter(model, h.ModelClient), nil

	case "DatasetVersion":
		// Fetch the DatasetVersion resource
		datasetVersion, err := h.DatasetVersionCache.Get(namespace, ownerReference.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get DatasetVersion resource: %w", err)
		}

		// Create a DatasetVersionAdapter that implements CacheableResource
		return NewDatasetVersionCacheAdapter(datasetVersion, h.DatasetVersionClient), nil

	default:
		return nil, fmt.Errorf("unsupported owner reference kind: %s", ownerReference.Kind)
	}
}

func (h *Handler) createVolumeSnapshot(resource CacheableResource, name string) error {
	namespace := resource.GetNamespace()
	if _, err := h.VolumeSnapshotCache.Get(namespace, name); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", namespace, name, err)
	}

	snapshot := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    makeLabels(resource.GetResourceType(), resource.GetName()),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: resource.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:       resource.GetObjectKind().GroupVersionKind().Kind,
					Name:       resource.GetName(),
					UID:        resource.GetUID(),
					Controller: ptr.To(true),
				},
			},
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: ptr.To(volumeSnapshotClassName),
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: ptr.To(name),
			},
		},
	}

	// Create the snapshot
	if _, err := h.VolumeSnapshotClient.Create(snapshot); err != nil {
		return fmt.Errorf("failed to create volume snapshot %s/%s: %w", namespace, name, err)
	}

	logrus.Infof("create volumn snapshot %s/%s", snapshot.Namespace, snapshot.Name)

	return nil
}

// handleJobFailure handles the case when the job fails
func (h *Handler) handleJobFailure(resource CacheableResource, job *batchv1.Job) error {
	// Reserve the job
	// Delete PVC
	logrus.Infof("job %s/%s failed, going to delete pvc", job.Namespace, job.Name)
	pvc, err := h.PVCCache.Get(resource.GetNamespace(), job.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get pvc %s/%s: %w", resource.GetNamespace(), job.Name, err)
		}
	} else {
		deletePolicy := metav1.DeletePropagationBackground
		deleteOptions := &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
		if err := h.PVCClient.Delete(pvc.Namespace, pvc.Name, deleteOptions); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	// Update resource status
	if err := resource.SetCacheStatus(&mlv1.CacheStatus{
		Status:       mlv1.CacheStatusFailed,
		CacheMessage: "Cache job failed",
		JobName:      "",
	}); err != nil {
		return fmt.Errorf("failed to update cache status: %w", err)
	}

	return nil
}

// stopCacheJob stops the cache job
func (h *Handler) stopCacheJob(resource CacheableResource) error {
	cacheStatus := resource.GetCacheStatus()
	if cacheStatus == nil || cacheStatus.JobName == "" {
		return nil
	}

	job, err := h.JobCache.Get(resource.GetNamespace(), cacheStatus.JobName)
	if err != nil {
		if errors.IsNotFound(err) {
			// If job not found, it may have been deleted, update status
			if err = resource.SetCacheStatus(&mlv1.CacheStatus{
				Status:       mlv1.CacheStatusIdle,
				CacheMessage: "job not found",
				JobName:      "",
			}); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	deletePolicy := metav1.DeletePropagationBackground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	if err = h.JobClient.Delete(job.Namespace, job.Name, deleteOptions); err != nil && !errors.IsNotFound(err) {
		return err
	}

	// delete PVC
	pvc, err := h.PVCCache.Get(resource.GetNamespace(), job.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if err := h.PVCClient.Delete(pvc.Namespace, pvc.Name, deleteOptions); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	// update status
	if err := resource.SetCacheStatus(&mlv1.CacheStatus{
		Status:       mlv1.CacheStatusIdle,
		CacheMessage: "Cache operation was stopped",
		JobName:      "",
	}); err != nil {
		return fmt.Errorf("failed to update cache status: %w", err)
	}

	return nil
}

// isJobTimedOut checks if the job has timed out
func isJobTimedOut(job *batchv1.Job) bool {
	if job.Status.StartTime == nil {
		return false
	}

	elapsedTime := time.Since(job.Status.StartTime.Time)
	return int(elapsedTime.Seconds()) > cacheJobTimeout
}

func (h *Handler) ReconcileVolumeSnapshot(snapshot *snapshotv1.VolumeSnapshot) error {
	if snapshot == nil || snapshot.DeletionTimestamp != nil {
		return nil
	}
	if snapshot.Labels == nil || snapshot.Labels[localCacheLabelKey] != strconv.FormatBool(true) ||
		len(snapshot.OwnerReferences) == 0 {
		return nil
	}
	if snapshot.Status == nil || snapshot.Status.ReadyToUse == nil || !*snapshot.Status.ReadyToUse {
		logrus.Infof("snapshot %s/%s is not ready yet", snapshot.Namespace, snapshot.Name)
		return nil
	}

	logrus.Infof("snapshot %s/%s is ready to use", snapshot.Namespace, snapshot.Name)

	resource, err := h.getResourceFromOwnerReference(snapshot.Namespace, snapshot.OwnerReferences[0])
	if err != nil {
		return fmt.Errorf("failed to get resource from owner reference %+v: %w", snapshot.OwnerReferences, err)
	}

	if resource.GetLocalCacheState() != mlv1.CacheStateActive {
		return nil
	}

	// Delete old snapshot if it exists
	cacheStatus := resource.GetCacheStatus()
	var oldSnapshotName string
	if cacheStatus != nil && cacheStatus.VolumeSnapshot != "" {
		oldSnapshotName = cacheStatus.VolumeSnapshot
	}

	// Delete PVC and Job
	deletePolicy := metav1.DeletePropagationBackground
	deleteOption := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	if err := h.PVCClient.Delete(snapshot.Namespace, snapshot.Name, deleteOption); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete pvc %s/%s: %w", snapshot.Namespace, snapshot.Name, err)
	}
	if err := h.JobClient.Delete(snapshot.Namespace, snapshot.Name, deleteOption); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete job %s/%s: %w", snapshot.Namespace, snapshot.Name, err)
	}

	// Update resource status
	if err := resource.SetCacheStatus(&mlv1.CacheStatus{
		Status:         mlv1.CacheStatusCompleted,
		VolumeSnapshot: snapshot.Name,
		CacheMessage:   "Cache completed successfully",
		LastCacheTime:  &metav1.Time{Time: time.Now()},
		JobName:        "",
	}); err != nil {
		return fmt.Errorf("failed to update cache status: %w", err)
	}

	if oldSnapshotName != "" && oldSnapshotName != snapshot.Name {
		if err := h.VolumeSnapshotClient.Delete(resource.GetNamespace(), oldSnapshotName,
			&metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			logrus.Errorf("failed to delete old volume snapshot %s/%s: %v", resource.GetNamespace(), oldSnapshotName, err)
		}
	}
	return nil
}

// ensureServiceAccountAndRoleBinding ensures that the service account and cluster role binding exist
func (h *Handler) ensureServiceAccountAndRoleBinding(resource CacheableResource) error {
	if _, err := h.ServiceAccountCache.Get(resource.GetNamespace(), serviceAccountName); err != nil {
		if !errors.IsAlreadyExists(err) && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get service account %s/%s: %w", resource.GetNamespace(), serviceAccountName, err)
		}
		// Create service account
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: resource.GetNamespace(),
			},
		}
		if _, err := h.ServiceAccountClient.Create(sa); err != nil {
			return fmt.Errorf("failed to create service account %s/%s: %w", resource.GetNamespace(), serviceAccountName, err)
		}
	}

	// Create cluster role binding if not exists
	clusterRoleBindingName := fmt.Sprintf("%s-cache", resource.GetNamespace())
	if _, err := h.ClusterRoleBindingCache.Get(clusterRoleBindingName); err != nil {
		if !errors.IsNotFound(err) && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to get cluster role binding %s: %w", clusterRoleBindingName, err)
		}
		// Create cluster role binding
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterRoleBindingName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "llmos-operator-registry-reader",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: resource.GetNamespace(),
				},
			},
		}
		if _, err := h.ClusterRoleBindingClient.Create(crb); err != nil {
			return fmt.Errorf("failed to create cluster role binding %s: %w", clusterRoleBindingName, err)
		}
	}

	return nil
}

func makeLabels(resourceType, name string) map[string]string {
	labels := map[string]string{
		localCacheLabelKey: strconv.FormatBool(true),
	}
	if resourceType == mlv1.ModelResourceName {
		labels[modelLabelKey] = name
	}
	if resourceType == mlv1.DatasetVersionResourceName {
		labels[datasetversionLabelKey] = name
	}

	return labels
}

// Calculate recommended PVC size in GB
func calculateCacheSize(size int64) string {
	// Convert bytes to GB and round up to the nearest GB
	// 1 GB = 1024^3 bytes
	gbSize := int64(1) // Minimum 1GB
	if size > 0 {
		// Calculate GB size and round up if there's a remainder
		divisor := int64(1024 * 1024 * 1024)
		gbSize = size / divisor
		if size%divisor > 0 {
			// Round up if there's any remainder
			gbSize++
		}

		// Ensure minimum size of 1GB
		if gbSize < 1 {
			gbSize = 1
		}
		logrus.Debugf("calculated PVC size: %d GB from %d bytes", gbSize, size)
	}

	// Format the size string (e.g., "10Gi")
	sizeStr := fmt.Sprintf("%dGi", gbSize)
	return sizeStr
}
