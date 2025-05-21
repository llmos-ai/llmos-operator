package localmodel

import (
	"context"
	"fmt"
	"sync"

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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	ctlsnapshotv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/snapshot.storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

const (
	localModelOnChangeName        = "localModel.OnChange"
	localModelVersionOnChangeName = "localModelVersion.OnChange"
	localmodelVersionOnRemoveName = "localModelVersion.OnRemove"
	jobOnChangeName               = "modelDownloadJob.OnChange"
	volumeSnapshotOnChangeName    = "volumeSnapshot.OnChange"

	defaultVersionLabel = "ml.llmos.ai/default-local-model-version"
	LocalModelNameLabel = "ml.llmos.ai/local-model-name"
	ModelNamespaceLabel = "ml.llmos.ai/model-namespace"
	ModelNameLabel      = "ml.llmos.ai/model-name"
	RegistryNameLabel   = "ml.llmos.ai/registry-name"

	serviceAccountName = "local-model"
	storageClassName   = "llmos-ceph-block"
	volumeName         = "model-volume"
	// TODO: change the mount path to /cache to huggingface cache path
	volumeMountPath = "/root/.cache/huggingface/hub"
)

type handler struct {
	ctx context.Context

	sync.Mutex

	LocalModelClient         ctlmlv1.LocalModelClient
	LocalModelCache          ctlmlv1.LocalModelCache
	LocalModelVersionClient  ctlmlv1.LocalModelVersionClient
	LocalModelVersionCache   ctlmlv1.LocalModelVersionCache
	RegistryCache            ctlmlv1.RegistryCache
	ModelClient              ctlmlv1.ModelClient
	ModelCache               ctlmlv1.ModelCache
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

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	localModels := mgmt.LLMFactory.Ml().V1().LocalModel()
	localModelVersions := mgmt.LLMFactory.Ml().V1().LocalModelVersion()
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	models := mgmt.LLMFactory.Ml().V1().Model()
	jobs := mgmt.BatchFactory.Batch().V1().Job()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	serviceAccounts := mgmt.CoreFactory.Core().V1().ServiceAccount()
	clusterRoleBindings := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding()
	volumeSnapshots := mgmt.SnapshotFactory.Snapshot().V1().VolumeSnapshot()
	secrets := mgmt.CoreFactory.Core().V1().Secret()

	h := handler{
		ctx: mgmt.Ctx,

		LocalModelClient:         localModels,
		LocalModelCache:          localModels.Cache(),
		LocalModelVersionClient:  localModelVersions,
		LocalModelVersionCache:   localModelVersions.Cache(),
		ModelClient:              models,
		ModelCache:               models.Cache(),
		JobClient:                jobs,
		JobCache:                 jobs.Cache(),
		PVCClient:                pvcs,
		PVCCache:                 pvcs.Cache(),
		VolumeSnapshotClient:     volumeSnapshots,
		VolumeSnapshotCache:      volumeSnapshots.Cache(),
		ServiceAccountClient:     serviceAccounts,
		ServiceAccountCache:      serviceAccounts.Cache(),
		ClusterRoleBindingClient: clusterRoleBindings,
		ClusterRoleBindingCache:  clusterRoleBindings.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	localModels.OnChange(mgmt.Ctx, localModelOnChangeName, h.SetDefaultVersion)
	localModelVersions.OnChange(mgmt.Ctx, localModelVersionOnChangeName, h.OnChangeVersion)
	jobs.OnChange(mgmt.Ctx, jobOnChangeName, h.OnChangeJob)
	volumeSnapshots.OnChange(mgmt.Ctx, volumeSnapshotOnChangeName, h.OnChangeVolumeSnapshot)

	return nil
}

// SetDefaultVersion to add the default version label
func (h *handler) SetDefaultVersion(_ string, localModel *mlv1.LocalModel) (*mlv1.LocalModel, error) {
	if localModel == nil || localModel.DeletionTimestamp != nil {
		return localModel, nil
	}
	ns, name, defaultVersion := localModel.Namespace, localModel.Name, localModel.Spec.DefaultVersion
	if defaultVersion == "" {
		return localModel, nil
	}

	logrus.Debugf("set default version %s/%s to %s", ns, name, defaultVersion)

	v, err := h.LocalModelVersionCache.Get(ns, defaultVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get the default version %s/%s of %s/%s: %w", ns, defaultVersion, ns, name, err)
	}
	if v.Labels[defaultVersion] == "true" {
		return localModel, nil
	}
	// set default version label
	versionCopy := v.DeepCopy()
	if versionCopy.Labels == nil {
		versionCopy.Labels = make(map[string]string)
	}
	versionCopy.Labels[defaultVersionLabel] = "true"
	if _, err = h.LocalModelVersionClient.Update(versionCopy); err != nil {
		return nil, fmt.Errorf("failed to set default version label to %s/%s for %s/%s: %w", ns, defaultVersion, ns, name, err)
	}

	// unset the outdated default version
	versions, err := h.LocalModelVersionCache.List(ns, labels.Set{
		defaultVersionLabel: "true",
		LocalModelNameLabel: name,
	}.AsSelector())
	if err != nil {
		return nil, fmt.Errorf("failed to get default version of %s/%s, error: %w", ns, name, err)
	}
	for _, v := range versions {
		if v.Name != defaultVersion {
			versionCopy := v.DeepCopy()
			delete(versionCopy.Labels, defaultVersionLabel)
			if _, err := h.LocalModelVersionClient.Update(versionCopy); err != nil {
				return nil, fmt.Errorf("failed unset the default version from %s/%s: %w", ns, v.Name, err)
			}
		}
	}

	return localModel, nil
}

func (h *handler) OnChangeVersion(_ string, version *mlv1.LocalModelVersion) (*mlv1.LocalModelVersion, error) {
	if version == nil || version.DeletionTimestamp != nil {
		return version, nil
	}

	logrus.Infof("local model version %s/%s changed", version.Namespace, version.Name)

	v, err := h.assignVersion(version)
	if err != nil {
		return version, fmt.Errorf("failed to assign version to local model version %s/%s: %w", version.Namespace, version.Name, err)
	}

	// create job to download model from registry
	if err := h.ensureJob(version); err != nil {
		return version, fmt.Errorf("failed to ensure job for local model version %s/%s: %w", version.Namespace, version.Name, err)
	}

	return v, nil
}

func (h *handler) assignVersion(v *mlv1.LocalModelVersion) (*mlv1.LocalModelVersion, error) {
	if v.Status.Version != 0 {
		return v, nil
	}

	h.Lock()
	defer h.Unlock()
	// get the latest version of the local model
	var latestVersion int
	list, err := h.LocalModelVersionClient.List(v.Namespace, metav1.ListOptions{
		LabelSelector: labels.Set{LocalModelNameLabel: v.Spec.LocalModel}.AsSelector().String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list local model versions of %s: %w", v.Spec.LocalModel, err)
	}

	for _, item := range list.Items {
		if item.Status.Version > latestVersion {
			latestVersion = item.Status.Version
		}
	}

	// assign version to the local model version
	versionCopy := v.DeepCopy()
	versionCopy.Status.Version = latestVersion + 1
	return h.LocalModelVersionClient.UpdateStatus(versionCopy)
}

// The namespace and name of the job should be the same as the local model version
func (h *handler) ensureJob(version *mlv1.LocalModelVersion) error {
	// check if the job exists and return if it exists
	_, err := h.JobCache.Get(version.Namespace, version.Name)
	if err != nil && !errors.IsNotFound(err) && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to get job %s/%s: %w", version.Namespace, version.Name, err)
	} else if err == nil {
		return nil
	}

	// Create service account and cluster role binding
	if err = h.ensureServiceAccountAndRoleBinding(version.Namespace); err != nil {
		return fmt.Errorf("failed to ensure service account and role binding in namespace %s: %w", version.Namespace, err)
	}

	latestSnapshot, err := h.getLatestReadySnapshot(version.Namespace, version.Spec.LocalModel)
	if err != nil {
		return fmt.Errorf("failed to get default version of %s/%s: %w", version.Namespace, version.Spec.LocalModel, err)
	}
	pvcSize, err := h.getModelSize(version)
	if err != nil {
		return fmt.Errorf("failed to get model content size: %w", err)
	}

	// Create PVC
	if err := h.createOrRestorePVC(version, latestSnapshot, pvcSize); err != nil {
		return err
	}

	// create job
	if err := h.createJob(version); err != nil {
		return fmt.Errorf("failed to create job %s/%s: %w", version.Namespace, version.Name, err)
	}
	return nil
}

func (h *handler) getModelSize(v *mlv1.LocalModelVersion) (int64, error) {
	// webhook should ensure that the registry name and model name labels are set
	registryName, modelNamespace, modelName := v.Labels[RegistryNameLabel], v.Labels[ModelNamespaceLabel], v.Labels[ModelNameLabel]
	b, err := h.rm.NewBackendFromRegistry(h.ctx, registryName)
	if err != nil {
		return -1, fmt.Errorf("failed to get backend from registry %s: %w", registryName, err)
	}

	model, err := h.ModelCache.Get(modelNamespace, modelName)
	if err != nil {
		return -1, fmt.Errorf("failed to get model %s/%s: %w", modelNamespace, modelName, err)
	}

	return b.GetSize(h.ctx, model.Status.RootPath)
}

func (h *handler) createJob(v *mlv1.LocalModelVersion) error {
	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v.Name,
			Namespace: v.Namespace,
			Labels: map[string]string{
				LocalModelNameLabel: v.Spec.LocalModel,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v.APIVersion,
					Kind:       v.Kind,
					Name:       v.Name,
					UID:        v.UID,
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
							Image: settings.ModelDownloaderImage.Get(),
							// TODO: add the args
							Args: []string{
								fmt.Sprintf("--name=%s/%s", v.Labels[ModelNamespaceLabel], v.Labels[ModelNameLabel]),
								fmt.Sprintf("--output-dir=%s", volumeMountPath),
								"--debug=true",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volumeName,
									MountPath: volumeMountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: v.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := h.JobClient.Create(job); err != nil {
		return fmt.Errorf("failed to create job %s/%s: %w", v.Namespace, v.Name, err)
	}

	return nil
}

// ensureServiceAccountAndRoleBinding ensures that the service account and cluster role binding exist
func (h *handler) ensureServiceAccountAndRoleBinding(namespace string) error {
	if _, err := h.ServiceAccountCache.Get(namespace, serviceAccountName); err != nil {
		if !errors.IsAlreadyExists(err) && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get service account %s/%s: %w", namespace, serviceAccountName, err)
		}
		// Create service account
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		}
		if _, err := h.ServiceAccountClient.Create(sa); err != nil {
			return fmt.Errorf("failed to create service account %s/%s: %w", namespace, serviceAccountName, err)
		}
	}

	// Create cluster role binding if not exists
	clusterRoleBindingName := fmt.Sprintf("%s-local-model", namespace)
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
					Namespace: namespace,
				},
			},
		}
		if _, err := h.ClusterRoleBindingClient.Create(crb); err != nil {
			return fmt.Errorf("failed to create cluster role binding %s: %w", clusterRoleBindingName, err)
		}
	}

	return nil
}

// createOrRestorePVC creates a new PVC or restores PVC from snapshot
func (h *handler) createOrRestorePVC(v *mlv1.LocalModelVersion, defaultSnapshot string, pvcSize int64) error {
	// Check if PVC exists
	pvc, err := h.PVCCache.Get(v.Namespace, v.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get pvc %s/%s: %w", v.Namespace, v.Name, err)
	} else if err == nil {
		return nil
	}

	logrus.Debugf("start to create pvc %s/%s", v.Namespace, v.Name)
	// Create PVC
	pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      v.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v.APIVersion,
					Kind:       v.Kind,
					Name:       v.Name,
					UID:        v.UID,
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
	if defaultSnapshot != "" {
		vs, err := h.VolumeSnapshotCache.Get(v.Namespace, defaultSnapshot)
		if err == nil {
			// Check if the snapshot is ready to use before using it as a data source
			if vs.Status != nil && vs.Status.ReadyToUse != nil && *vs.Status.ReadyToUse {
				pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
					APIGroup: ptr.To("snapshot.storage.k8s.io"),
					Kind:     "VolumeSnapshot",
					Name:     defaultSnapshot,
				}
				if vs.Status.RestoreSize != nil && pvcSize < vs.Status.RestoreSize.Value() {
					pvcSize = vs.Status.RestoreSize.Value()
				}
			} else {
				logrus.Warnf("snapshot %s is not ready to use, skip restore from snapshot", defaultSnapshot)
			}
		} else {
			logrus.Warnf("failed to get snapshot %s: %v", defaultSnapshot, err)
		}
	}
	pvc.Spec.Resources = corev1.VolumeResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: apiresource.MustParse(calculatePVCSize(pvcSize)),
		},
	}

	createdPVC, err := h.PVCClient.Create(pvc)
	if err != nil {
		return err
	}
	logrus.Debugf("create pvc %s/%s", createdPVC.Namespace, createdPVC.Name)

	return nil
}

func (h *handler) getLatestReadySnapshot(namespace, localModelName string) (string, error) {
	versions, err := h.LocalModelVersionCache.List(namespace, labels.Set{
		LocalModelNameLabel: localModelName,
	}.AsSelector())
	if err != nil {
		return "", fmt.Errorf("failed to list default local model versions of %s/%s: %w", namespace, localModelName, err)
	}

	snapshot, version := "", 0
	for _, v := range versions {
		if v.Status.VolumeSnapshot != "" && v.Status.Version > version {
			snapshot = v.Status.VolumeSnapshot
			version = v.Status.Version
		}
	}

	return snapshot, nil
}

// Calculate recommended PVC size in GB
func calculatePVCSize(size int64) string {
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
