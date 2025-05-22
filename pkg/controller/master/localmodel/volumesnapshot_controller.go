package localmodel

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

func (h *handler) OnChangeVolumeSnapshot(
	_ string,
	snapshot *snapshotv1.VolumeSnapshot,
) (*snapshotv1.VolumeSnapshot, error) {
	if snapshot == nil || snapshot.DeletionTimestamp != nil {
		return snapshot, nil
	}

	if snapshot.Labels == nil || snapshot.Labels[LocalModelNameLabel] == "" {
		return snapshot, nil
	}

	ns, name := snapshot.Namespace, snapshot.Name
	if snapshot.Status != nil {
		if snapshot.Status.Error != nil {
			if err := h.setVolumeSnapshot(ns, name, "", snapshot.Status.Error); err != nil {
				return nil, fmt.Errorf("failed to update snapshot status error to local model version %s/%s: %w", ns, name, err)
			}
		}
		if snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse {
			if err := h.setVolumeSnapshot(ns, name, name, nil); err != nil {
				return nil, fmt.Errorf("failed to set snapshot to local model version %s/%s: %w", ns, name, err)
			}
			// delete pvc
			if snapshot.Spec.Source.PersistentVolumeClaimName != nil {
				if err := h.PVCClient.Delete(snapshot.Namespace,
					*snapshot.Spec.Source.PersistentVolumeClaimName, &metav1.DeleteOptions{}); err != nil {
					if !errors.IsNotFound(err) {
						return nil, fmt.Errorf("failed to delete pvc %s/%s: %w", ns, *snapshot.Spec.Source.PersistentVolumeClaimName, err)
					}
				}
			}
		}
	}

	return snapshot, nil
}

func (h *handler) setVolumeSnapshot(ns, name, snapshot string, snapshotErr *snapshotv1.VolumeSnapshotError) error {
	version, err := h.LocalModelVersionCache.Get(ns, name)
	if err != nil {
		return fmt.Errorf("falied to get local model version %s/%s: %w", ns, name, err)
	}

	versionCopy := version.DeepCopy()

	if snapshotErr != nil {
		var message string
		if snapshotErr.Message != nil {
			message = *snapshotErr.Message
		}
		err := fmt.Errorf("failed to create volume snapshot, message: %s", message)
		if mlv1.Ready.MatchesError(version, "", err) {
			return nil
		}
		mlv1.Ready.SetError(versionCopy, "", err)
	} else {
		if mlv1.Ready.IsTrue(version) && version.Status.VolumeSnapshot == snapshot {
			return nil
		}
		mlv1.Ready.True(versionCopy)
		mlv1.Ready.Message(versionCopy, "volume snapshot is ready")
		mlv1.Ready.Reason(versionCopy, "")
		versionCopy.Status.VolumeSnapshot = snapshot
	}

	if _, err := h.LocalModelVersionClient.UpdateStatus(versionCopy); err != nil {
		return err
	}

	return nil
}
