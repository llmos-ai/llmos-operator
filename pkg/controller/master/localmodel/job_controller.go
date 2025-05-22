package localmodel

import (
	"fmt"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

const (
	volumeSnapshotClassName = "llmos-ceph-block-snapshot-class"
	jobTimeout              = 3600 // 1 hour timeout
)

func (h *handler) OnChangeJob(_ string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil || job.DeletionTimestamp != nil {
		return job, nil
	}

	if job.Labels == nil || job.Labels[LocalModelNameLabel] == "" {
		return job, nil
	}

	ns, name := job.Namespace, job.Name

	if job.Status.Succeeded > 0 {
		if err := h.ensureSnapshot(job); err != nil {
			return nil, fmt.Errorf("failed to ensure snapshot %s/%s: %w", ns, name, err)
		}
	} else if job.Status.Failed > 0 || isJobTimedOut(job) {
		if err := h.handleJobFailure(ns, name); err != nil {
			return nil, fmt.Errorf("failed to handler job failure %s/%s: %w", ns, name, err)
		}
	}

	return job, nil
}

func (h *handler) ensureSnapshot(job *batchv1.Job) error {
	ns, name := job.Namespace, job.Name
	if _, err := h.VolumeSnapshotCache.Get(ns, name); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", ns, name, err)
	}

	snapshot := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels: map[string]string{
				LocalModelNameLabel: job.Labels[LocalModelNameLabel],
			},
			OwnerReferences: job.OwnerReferences,
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: ptr.To(volumeSnapshotClassName),
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: ptr.To(name),
			},
		},
	}

	if _, err := h.VolumeSnapshotClient.Create(snapshot); err != nil {
		return fmt.Errorf("failed to create volume snapshot %s/%s: %w", ns, name, err)
	}
	return nil
}

func (h *handler) handleJobFailure(ns, name string) error {
	version, err := h.LocalModelVersionCache.Get(ns, name)
	if err != nil {
		return fmt.Errorf("failed to get local model version %s/%s: %w", ns, name, err)
	}
	versionCopy := version.DeepCopy()
	mlv1.Ready.False(versionCopy)
	mlv1.Ready.Message(versionCopy, "failed to download model contents")
	if _, err := h.LocalModelVersionClient.UpdateStatus(versionCopy); err != nil {
		return fmt.Errorf("failed to update status of local model version %s/%s: %w", ns, name, err)
	}
	return nil
}

// isJobTimedOut checks if the job has timed out
func isJobTimedOut(job *batchv1.Job) bool {
	if job.Status.StartTime == nil {
		return false
	}

	elapsedTime := time.Since(job.Status.StartTime.Time)
	return int(elapsedTime.Seconds()) > jobTimeout
}
