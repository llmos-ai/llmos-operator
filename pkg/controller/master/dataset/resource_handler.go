package dataset

import (
	"context"
	"fmt"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

// ResourceHandler interface implementation for DatasetVersion
// This allows DatasetVersion to work with the snapshotting manager

// GetSnapshottingStatus returns the current snapshotting status of the DatasetVersion
func (h *handler) GetSnapshottingStatus(namespace, name string) (*mlv1.SnapshottingStatus, error) {
	dv, err := h.datasetVersionCache.Get(namespace, name)
	if err != nil {
		return nil, err
	}

	return &dv.Status.PublishStatus, nil
}

// UpdateSnapshottingStatus updates the snapshotting status of the DatasetVersion
func (h *handler) UpdateSnapshottingStatus(namespace, name string, status *mlv1.SnapshottingStatus) error {
	dv, err := h.datasetVersionCache.Get(namespace, name)
	if err != nil {
		return err
	}

	dvCopy := dv.DeepCopy()
	if status != nil {
		dvCopy.Status.PublishStatus = *status
	} else {
		// Clear the publish status by setting it to empty struct
		dvCopy.Status.PublishStatus = mlv1.SnapshottingStatus{}
	}

	_, err = h.datasetVersionClient.UpdateStatus(dvCopy)
	return err
}

// GetContentSize calculates the total size of all files in the DatasetVersion directory
func (h *handler) GetContentSize(ctx context.Context, namespace, name string) (int64, error) {
	dv, err := h.datasetVersionCache.Get(namespace, name)
	if err != nil {
		return 0, err
	}

	// Get backend and calculate directory size
	b, err := h.rm.NewBackendFromRegistry(ctx, dv.Status.Registry)
	if err != nil {
		return 0, fmt.Errorf("failed to create backend client: %w", err)
	}

	// Use Backend.GetSize method to get the total size
	totalSize, err := b.GetSize(ctx, dv.Status.RootPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	return totalSize, nil
}

// GetLatestReadySnapshot returns the latest ready snapshot name
// For DatasetVersion, we don't track snapshots in the same way as other resources,
// so this returns an empty string
func (h *handler) GetLatestReadySnapshot(namespace, name string) (string, error) {
	// DatasetVersion doesn't track snapshots in the same way
	// The snapshotting manager will handle snapshot tracking
	return "", nil
}

// GetResourceType returns the resource type for event handler naming
func (h *handler) GetResourceType() string {
	return mlv1.DatasetVersionResourceName
}
