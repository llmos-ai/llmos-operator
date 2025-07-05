package snapshotting

import (
	"fmt"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

const (
	jobTimeout = 3600 // 1 hour timeout
)

// OnJobChange handles Job status changes
func (m *Manager) OnJobChange(_ string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil || job.DeletionTimestamp != nil {
		return job, nil
	}

	// Only handle jobs that are managed by this snapshotting manager
	// We can identify them by checking if they have the expected labels or owner references
	if !m.isJobManagedByUs(job) {
		return job, nil
	}

	ns, name := job.Namespace, job.Name

	if job.Status.Succeeded > 0 {
		// Job succeeded, update status to Downloaded and set TTL for immediate cleanup
		logrus.Infof("Job %s/%s succeeded, updating status to Downloaded", ns, name)
		if err := m.updateStatus(ns, name, &mlv1.SnapshottingStatus{
			Phase:              mlv1.SnapshottingPhaseDownloaded,
			LastTransitionTime: metav1.Now(),
			Message:            "Job completed successfully",
			PVCName:            name,
			JobName:            name,
		}); err != nil {
			return nil, fmt.Errorf("failed to update status for job %s/%s: %w", ns, name, err)
		}
	} else if job.Status.Failed > 0 || m.isJobTimedOut(job) {
		// Job failed or timed out
		logrus.Errorf("Job %s/%s failed or timed out", ns, name)
		errorMsg := "Job failed or timed out"
		if err := m.updateStatusWithError(ns, name, mlv1.SnapshottingPhaseDownloading, errorMsg); err != nil {
			return nil, fmt.Errorf("failed to update status for failed job %s/%s: %w", ns, name, err)
		}

		// Set TTL to 24 hours for failed jobs
		if err := m.setJobTTLForFailure(job); err != nil {
			logrus.Warnf("Failed to set TTL for failed job %s/%s: %v", ns, name, err)
		}
	}

	return job, nil
}

// OnVolumeSnapshotChange handles VolumeSnapshot status changes
func (m *Manager) OnVolumeSnapshotChange(
	_ string, snapshot *snapshotv1.VolumeSnapshot,
) (*snapshotv1.VolumeSnapshot, error) {
	if snapshot == nil || snapshot.DeletionTimestamp != nil {
		return snapshot, nil
	}

	// Only handle snapshots that are managed by this snapshotting manager
	if !m.isSnapshotManagedByUs(snapshot) {
		return snapshot, nil
	}

	ns, name := snapshot.Namespace, snapshot.Name

	if snapshot.Status != nil {
		if snapshot.Status.Error != nil {
			// Snapshot creation failed
			var message string
			if snapshot.Status.Error.Message != nil {
				message = *snapshot.Status.Error.Message
			}
			logrus.Errorf("VolumeSnapshot %s/%s failed: %s", ns, name, message)
			errorMsg := fmt.Sprintf("VolumeSnapshot failed: %s", message)
			if err := m.updateStatusWithError(ns, name, mlv1.SnapshottingPhaseSnapshotting, errorMsg); err != nil {
				return nil, fmt.Errorf("failed to update status for failed snapshot %s/%s: %w", ns, name, err)
			}
		} else if snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse {
			// Snapshot is ready
			logrus.Infof("VolumeSnapshot %s/%s is ready", ns, name)
			if err := m.updateStatus(ns, name, &mlv1.SnapshottingStatus{
				Phase:              mlv1.SnapshottingPhaseSnapshotReady,
				LastTransitionTime: metav1.Now(),
				Message:            "VolumeSnapshot is ready",
				PVCName:            name,
				JobName:            name,
				SnapshotName:       name,
			}); err != nil {
				return nil, fmt.Errorf("failed to update status for ready snapshot %s/%s: %w", ns, name, err)
			}

			// Clean up PVC after successful snapshot
			if err := m.cleanupPVC(ns, name); err != nil {
				logrus.Warnf("Failed to cleanup PVC %s/%s: %v", ns, name, err)
			}
		}
	}

	return snapshot, nil
}

// Helper methods

// isJobManagedByUs checks if a job is managed by this snapshotting manager
func (m *Manager) isJobManagedByUs(job *batchv1.Job) bool {
	if job.Labels == nil {
		return false
	}
	// Check if it's managed by snapshotting manager
	value, exists := job.Labels[SnapshotManagerLabel]
	if !exists || value != SnapshotManagerValue {
		return false
	}
	// Check if the resource type matches this handler
	resourceType, exists := job.Labels[ResourceTypeLabel]
	return exists && resourceType == m.ResourceHandler.GetResourceType()
}

// isSnapshotManagedByUs checks if a snapshot is managed by this snapshotting manager
func (m *Manager) isSnapshotManagedByUs(snapshot *snapshotv1.VolumeSnapshot) bool {
	if snapshot.Labels == nil {
		return false
	}
	// Check if it's managed by snapshotting manager
	value, exists := snapshot.Labels[SnapshotManagerLabel]
	if !exists || value != SnapshotManagerValue {
		return false
	}
	// Check if the resource type matches this handler
	resourceType, exists := snapshot.Labels[ResourceTypeLabel]
	return exists && resourceType == m.ResourceHandler.GetResourceType()
}

// isJobTimedOut checks if the job has timed out
func (m *Manager) isJobTimedOut(job *batchv1.Job) bool {
	if job.Status.StartTime == nil {
		return false
	}

	elapsedTime := time.Since(job.Status.StartTime.Time)
	return int(elapsedTime.Seconds()) > jobTimeout
}

// setJobTTLForFailure sets TTL to 24 hours for failed jobs
func (m *Manager) setJobTTLForFailure(job *batchv1.Job) error {
	// Check if TTL is already set to avoid unnecessary updates
	if job.Spec.TTLSecondsAfterFinished != nil && *job.Spec.TTLSecondsAfterFinished == 86400 {
		return nil
	}

	// Create a copy of the job and update TTL
	jobCopy := job.DeepCopy()
	jobCopy.Spec.TTLSecondsAfterFinished = ptr.To(int32(86400)) // 24 hours

	_, err := m.JobClient.Update(jobCopy)
	if err != nil {
		return fmt.Errorf("failed to update job TTL: %w", err)
	}

	logrus.Infof("Set TTL to 24 hours for failed job %s/%s", job.Namespace, job.Name)
	return nil
}

// cleanupPVC deletes the PVC after successful snapshot creation
func (m *Manager) cleanupPVC(namespace, name string) error {
	if err := m.PVCClient.Delete(namespace, name, &metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pvc %s/%s: %w", namespace, name, err)
		}
	}
	return nil
}
