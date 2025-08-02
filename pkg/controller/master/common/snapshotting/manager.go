package snapshotting

import (
	"context"
	"fmt"
	"math"

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
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlsnapshotv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/snapshot.storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	// SnapshotManagerLabel is used to identify resources managed by the snapshotting manager
	SnapshotManagerLabel = "llmos.ai/snapshotting-manager"
	SnapshotManagerValue = "true"
	// VolumeSnapshotClassName is the fixed volume snapshot class name
	volumeSnapshotClassName = "llmos-ceph-block-snapshot-class"
	// StorageClassName is the fixed storage class name
	storageClassName = "llmos-ceph-block"
	// ServiceAccountName is the fixed service account name
	serviceAccountName = "llmos-operator-downloader"
	// ClusterRoleBindingName is the fixed cluster role binding name
	clusterRoleBindingName = "llmos-operator"
)

type Manager struct {
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

	ResourceHandler ResourceHandler
}

func NewManager(mgmt *config.Management, resourceHandler ResourceHandler) (*Manager, error) {
	jobs := mgmt.BatchFactory.Batch().V1().Job()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	serviceAccounts := mgmt.CoreFactory.Core().V1().ServiceAccount()
	clusterRoleBindings := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding()
	volumeSnapshots := mgmt.SnapshotFactory.Snapshot().V1().VolumeSnapshot()
	m := &Manager{
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
		ResourceHandler:          resourceHandler,
	}

	resourceType := resourceHandler.GetResourceType()
	jobs.OnChange(mgmt.Ctx, resourceType+"-snapshotting-job", m.OnJobChange)
	volumeSnapshots.OnChange(mgmt.Ctx, resourceType+"-snapshotting-snapshot", m.OnVolumeSnapshotChange)

	return m, nil
}

// DoSnapshot is the main entry point for starting the snapshot process
func (m *Manager) DoSnapshot(ctx context.Context, spec *Spec) error {
	if size, err := m.ResourceHandler.GetContentSize(ctx, spec.Namespace, spec.Name); err != nil {
		return fmt.Errorf("failed to get content size: %w", err)
	} else if size == 0 {
		logrus.Infof("no content to snapshot for %s %s/%s", m.ResourceHandler.GetResourceType(), spec.Namespace, spec.Name)
		return nil
	}

	// Get current status to determine next step
	status, err := m.ResourceHandler.GetSnapshottingStatus(spec.Namespace, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get snapshotting status: %w", err)
	}

	// State machine to handle different phases
	switch status.Phase {
	case "", mlv1.SnapshottingPhasePreparePVC:
		return m.preparePVC(ctx, spec)
	case mlv1.SnapshottingPhasePVCReady:
		return m.handlePVCReady(spec)
	case mlv1.SnapshottingPhaseDownloading:
		// Job is running, nothing to do here
		return nil
	case mlv1.SnapshottingPhaseDownloaded:
		return m.handleDownloaded(spec)
	case mlv1.SnapshottingPhaseSnapshotting:
		// VolumeSnapshot is being created, nothing to do here
		return nil
	case mlv1.SnapshottingPhaseSnapshotReady:
		// Process completed successfully
		return nil
	case mlv1.SnapshottingPhaseFailed:
		// Process failed, nothing to do here
		return nil
	default:
		return fmt.Errorf("unknown snapshotting phase: %s", status.Phase)
	}
}

func (m *Manager) CancelSnapshot(ctx context.Context, spec *Spec) error {
	logrus.Infof("Cancelling snapshot for %s/%s", spec.Namespace, spec.Name)

	// Get current status to check if there's anything to cancel
	status, err := m.ResourceHandler.GetSnapshottingStatus(spec.Namespace, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get snapshotting status: %w", err)
	}

	// If status is nil or empty, nothing to cancel
	if status == nil || status.Phase == "" {
		logrus.Debugf("No active snapshot process found for %s/%s", spec.Namespace, spec.Name)
		return nil
	}

	// Delete VolumeSnapshot if it exists
	if status.SnapshotName != "" {
		err = m.VolumeSnapshotClient.Delete(spec.Namespace, status.SnapshotName, nil)
		if err != nil && !errors.IsNotFound(err) {
			logrus.Warnf("Failed to delete VolumeSnapshot %s/%s: %v", spec.Namespace, status.SnapshotName, err)
		} else if err == nil {
			logrus.Infof("Deleted VolumeSnapshot %s/%s", spec.Namespace, status.SnapshotName)
		}
	}

	// Delete Job if it exists
	if status.JobName != "" {
		err = m.JobClient.Delete(spec.Namespace, status.JobName, nil)
		if err != nil && !errors.IsNotFound(err) {
			logrus.Warnf("Failed to delete Job %s/%s: %v", spec.Namespace, status.JobName, err)
		} else if err == nil {
			logrus.Infof("Deleted Job %s/%s", spec.Namespace, status.JobName)
		}
	}

	// Delete PVC if it exists
	if status.PVCName != "" {
		err = m.PVCClient.Delete(spec.Namespace, status.PVCName, nil)
		if err != nil && !errors.IsNotFound(err) {
			logrus.Warnf("Failed to delete PVC %s/%s: %v", spec.Namespace, status.PVCName, err)
		} else if err == nil {
			logrus.Infof("Deleted PVC %s/%s", spec.Namespace, status.PVCName)
		}
	}

	// Clear the publish status
	err = m.ResourceHandler.UpdateSnapshottingStatus(spec.Namespace, spec.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to clear snapshotting status: %w", err)
	}

	logrus.Infof("Successfully cancelled snapshot for %s/%s", spec.Namespace, spec.Name)
	return nil
}

// handlePreparePVC creates PVC and updates status
func (m *Manager) preparePVC(ctx context.Context, spec *Spec) error {
	logrus.Infof("Starting PVC preparation for %s/%s", spec.Namespace, spec.Name)

	if err := m.createPVC(ctx, spec); err != nil {
		errorMsg := fmt.Sprintf("Failed to create PVC: %v", err)
		return m.updateStatusWithError(spec.Namespace, spec.Name, mlv1.SnapshottingPhasePreparePVC, errorMsg)
	}

	return m.updateStatus(spec.Namespace, spec.Name, &mlv1.SnapshottingStatus{
		Phase:              mlv1.SnapshottingPhasePVCReady,
		LastTransitionTime: metav1.Now(),
		Message:            "PVC created successfully",
		PVCName:            spec.Name,
	})
}

// handlePVCReady creates Job and updates status
func (m *Manager) handlePVCReady(spec *Spec) error {
	logrus.Infof("Starting Job creation for %s/%s", spec.Namespace, spec.Name)

	// Ensure ServiceAccount and ClusterRoleBinding exist before creating Job
	if err := m.ensureServiceAccountAndRoleBinding(spec.Namespace); err != nil {
		errorMsg := fmt.Sprintf("Failed to ensure ServiceAccount and ClusterRoleBinding: %v", err)
		return m.updateStatusWithError(spec.Namespace, spec.Name, mlv1.SnapshottingPhasePVCReady, errorMsg)
	}

	if err := m.createJob(spec); err != nil {
		errorMsg := fmt.Sprintf("Failed to create Job: %v", err)
		return m.updateStatusWithError(spec.Namespace, spec.Name, mlv1.SnapshottingPhasePVCReady, errorMsg)
	}

	return m.updateStatus(spec.Namespace, spec.Name, &mlv1.SnapshottingStatus{
		Phase:              mlv1.SnapshottingPhaseDownloading,
		LastTransitionTime: metav1.Now(),
		Message:            "Job created successfully, downloading in progress",
		PVCName:            spec.Name,
		JobName:            spec.Name,
	})
}

// handleDownloaded creates VolumeSnapshot and updates status
func (m *Manager) handleDownloaded(spec *Spec) error {
	logrus.Infof("Starting VolumeSnapshot creation for %s/%s", spec.Namespace, spec.Name)

	if err := m.createVolumeSnapshot(spec); err != nil {
		errorMsg := fmt.Sprintf("Failed to create VolumeSnapshot: %v", err)
		return m.updateStatusWithError(spec.Namespace, spec.Name, mlv1.SnapshottingPhaseDownloaded, errorMsg)
	}

	return m.updateStatus(spec.Namespace, spec.Name, &mlv1.SnapshottingStatus{
		Phase:              mlv1.SnapshottingPhaseSnapshotting,
		LastTransitionTime: metav1.Now(),
		Message:            "VolumeSnapshot created successfully, snapshotting in progress",
		PVCName:            spec.Name,
		JobName:            spec.Name,
		SnapshotName:       spec.Name,
	})
}

func (m *Manager) updateStatus(namespace, name string, status *mlv1.SnapshottingStatus) error {
	return m.ResourceHandler.UpdateSnapshottingStatus(namespace, name, status)
}

func (m *Manager) updateStatusWithError(namespace, name string, phase mlv1.SnapshottingPhase, message string) error {
	return m.updateStatus(namespace, name, &mlv1.SnapshottingStatus{
		Phase:              phase,
		LastTransitionTime: metav1.Now(),
		Message:            message,
	})
}

// Builder methods for creating resources
// createPVC creates a PVC based on the spec
func (m *Manager) createPVC(ctx context.Context, spec *Spec) error {
	// Check if PVC already exists
	_, err := m.PVCCache.Get(spec.Namespace, spec.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get pvc %s/%s: %w", spec.Namespace, spec.Name, err)
	} else if err == nil {
		// PVC already exists
		return nil
	}

	logrus.Debugf("Creating PVC %s/%s", spec.Namespace, spec.Name)

	// Get content size dynamically
	contentSize, err := m.ResourceHandler.GetContentSize(ctx, spec.Namespace, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get content size: %w", err)
	}
	pvcSize, err := calculatePVCSize(contentSize)
	if err != nil {
		return fmt.Errorf("calculate pvc size: %w", err)
	}

	// Create PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       spec.Namespace,
			Name:            spec.Name,
			Labels:          spec.Labels,
			OwnerReferences: spec.OwnerReferences,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: ptr.To(storageClassName),
			AccessModes:      spec.PVCSpec.AccessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: apiresource.MustParse(pvcSize),
				},
			},
		},
	}

	// If restore from latest snapshot is enabled, get the latest snapshot and set data source
	if spec.PVCSpec.RestoreFromLatestSnapshot {
		latestSnapshot, err := m.ResourceHandler.GetLatestReadySnapshot(
			spec.Namespace, spec.Labels[constant.LabelLocalModelName])
		if err != nil {
			logrus.Warnf("Failed to get latest snapshot: %v", err)
		} else if latestSnapshot != "" {
			vs, err := m.VolumeSnapshotCache.Get(spec.Namespace, latestSnapshot)
			if err == nil {
				// Check if the snapshot is ready to use
				if vs.Status != nil && vs.Status.ReadyToUse != nil && *vs.Status.ReadyToUse {
					pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("snapshot.storage.k8s.io"),
						Kind:     "VolumeSnapshot",
						Name:     latestSnapshot,
					}

					// Adjust PVC size if restore size is larger
					if vs.Status.RestoreSize != nil {
						// Parse current PVC size
						currentSize := apiresource.MustParse(pvcSize)
						if currentSize.Value() < vs.Status.RestoreSize.Value() {
							pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *vs.Status.RestoreSize
						}
					}
				} else {
					logrus.Warnf("Snapshot %s is not ready to use, skip restore from snapshot", latestSnapshot)
				}
			} else {
				logrus.Warnf("Failed to get snapshot %s: %v", latestSnapshot, err)
			}
		}
	}

	_, err = m.PVCClient.Create(pvc)
	if err != nil {
		return fmt.Errorf("failed to create pvc %s/%s: %w", spec.Namespace, spec.Name, err)
	}

	logrus.Debugf("Created PVC %s/%s", spec.Namespace, spec.Name)
	return nil
}

// createJob creates a Job based on the spec
func (m *Manager) createJob(spec *Spec) error {
	// Check if Job already exists
	_, err := m.JobCache.Get(spec.Namespace, spec.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get job %s/%s: %w", spec.Namespace, spec.Name, err)
	} else if err == nil {
		// Job already exists
		return nil
	}

	logrus.Debugf("Creating Job %s/%s", spec.Namespace, spec.Name)

	// Prepare labels with snapshotting manager label and resource type
	labels := make(map[string]string)
	for k, v := range spec.Labels {
		labels[k] = v
	}
	labels[SnapshotManagerLabel] = SnapshotManagerValue
	labels[ResourceTypeLabel] = m.ResourceHandler.GetResourceType()

	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            spec.Name,
			Namespace:       spec.Namespace,
			Labels:          labels,
			OwnerReferences: spec.OwnerReferences,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: spec.JobSpec.BackoffLimit,
			// Set TTL to 0 for successful jobs (immediate deletion)
			// Failed jobs will be handled by the event handler to set 24h TTL
			TTLSecondsAfterFinished: ptr.To(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "downloader",
							Image: spec.JobSpec.Image,
							Args:  spec.JobSpec.Args,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data-volume",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: spec.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = m.JobClient.Create(job)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// createVolumeSnapshot creates a VolumeSnapshot based on the spec
func (m *Manager) createVolumeSnapshot(spec *Spec) error {
	// Check if VolumeSnapshot already exists
	_, err := m.VolumeSnapshotCache.Get(spec.Namespace, spec.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get volume snapshot %s/%s: %w", spec.Namespace, spec.Name, err)
	} else if err == nil {
		// VolumeSnapshot already exists
		return nil
	}

	logrus.Debugf("Creating VolumeSnapshot %s/%s", spec.Namespace, spec.Name)

	// Prepare labels with snapshotting manager label and resource type
	labels := make(map[string]string)
	for k, v := range spec.Labels {
		labels[k] = v
	}
	labels[SnapshotManagerLabel] = SnapshotManagerValue
	labels[ResourceTypeLabel] = m.ResourceHandler.GetResourceType()

	// Create VolumeSnapshot
	snapshot := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       spec.Namespace,
			Name:            spec.Name,
			Labels:          labels,
			OwnerReferences: spec.OwnerReferences,
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: ptr.To(volumeSnapshotClassName),
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: ptr.To(spec.Name),
			},
		},
	}

	_, err = m.VolumeSnapshotClient.Create(snapshot)
	if err != nil {
		return fmt.Errorf("failed to create volume snapshot %s/%s: %w", spec.Namespace, spec.Name, err)
	}

	logrus.Debugf("Created VolumeSnapshot %s/%s", spec.Namespace, spec.Name)
	return nil
}

// ensureServiceAccountAndRoleBinding ensures the service account and cluster role binding exist
func (m *Manager) ensureServiceAccountAndRoleBinding(namespace string) error {
	// Check if ServiceAccount exists
	_, err := m.ServiceAccountCache.Get(namespace, serviceAccountName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get service account %s/%s: %w", namespace, serviceAccountName, err)
	} else if errors.IsNotFound(err) {
		// Create ServiceAccount
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      serviceAccountName,
				Labels: map[string]string{
					SnapshotManagerLabel: SnapshotManagerValue,
				},
			},
		}

		_, err = m.ServiceAccountClient.Create(sa)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create service account %s/%s: %w", namespace, serviceAccountName, err)
		}
		logrus.Debugf("Created ServiceAccount %s/%s", namespace, serviceAccountName)
	}

	// Check if ClusterRoleBinding exists
	_, err = m.ClusterRoleBindingCache.Get(clusterRoleBindingName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get cluster role binding %s: %w", clusterRoleBindingName, err)
	} else if errors.IsNotFound(err) {
		// Create ClusterRoleBinding
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterRoleBindingName,
				Labels: map[string]string{
					SnapshotManagerLabel: SnapshotManagerValue,
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "llmos-operator-registry-reader",
			},
		}

		_, err = m.ClusterRoleBindingClient.Create(crb)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create cluster role binding %s: %w", clusterRoleBindingName, err)
		}
		logrus.Debugf("Created ClusterRoleBinding %s", clusterRoleBindingName)
	}

	return nil
}

func calculatePVCSize(size int64) (string, error) {
	// Convert bytes to GB and apply 110% buffer, with minimum 1GB
	// 1 GB = 1024^3 bytes
	if size <= 0 {
		return "", fmt.Errorf("invalid size: %d", size)
	}

	// Calculate the required size in bytes with 110% buffer.
	// Using integer arithmetic to avoid floating point precision issues.
	// Prevent int64 overflow: ensure size*11 <= MaxInt64.
	const maxSafeSize = math.MaxInt64 / 11
	if size > maxSafeSize {
		return "", fmt.Errorf("requested size too large: %d exceeds maximum allowable", size)
	}

	// Calculate the required size in bytes with 110% buffer
	// Using integer arithmetic to avoid floating point precision issues
	bufferedSize := (size * 11) / 10

	// Convert to GB and round up
	divisor := int64(1024 * 1024 * 1024)
	gbSize := bufferedSize / divisor
	if bufferedSize%divisor > 0 {
		// Round up if there's any remainder
		gbSize++
	}

	// Ensure minimum size of 1GB
	if gbSize < 1 {
		gbSize = 1
	}

	logrus.Debugf("calculated PVC size: %d GB (110%% of %d bytes = %d bytes)", gbSize, size, bufferedSize)

	// Format the size string (e.g., "10Gi")
	return fmt.Sprintf("%dGi", gbSize), nil
}
