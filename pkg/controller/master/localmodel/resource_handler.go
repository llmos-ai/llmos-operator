package localmodel

import (
	"context"
	"fmt"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/common/snapshotting"
	"k8s.io/apimachinery/pkg/labels"
)

var _ snapshotting.ResourceHandler = &handler{}

// GetSnapshottingStatus returns the current snapshotting status
// This method implements the ResourceHandler interface
func (h *handler) GetSnapshottingStatus(namespace, name string) (*mlv1.SnapshottingStatus, error) {
	// Get the latest version from cache
	v, err := h.LocalModelVersionCache.Get(namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get local model version %s/%s: %w", namespace, name, err)
	}

	// Return the snapshotting status from the version
	return &v.Status.SnapshottingStatus, nil
}

// UpdateSnapshottingStatus updates the snapshotting status
// This method implements the ResourceHandler interface
func (h *handler) UpdateSnapshottingStatus(namespace, name string, status *mlv1.SnapshottingStatus) error {
	// Get the latest version from cache
	v, err := h.LocalModelVersionCache.Get(namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get local model version %s/%s: %w", namespace, name, err)
	}

	// Update the status
	versionCopy := v.DeepCopy()
	if status != nil {
		versionCopy.Status.SnapshottingStatus = *status
	}

	// Update the overall Ready condition based on snapshotting status
	switch status.Phase {
	case mlv1.SnapshottingPhaseSnapshotReady:
		mlv1.Ready.True(versionCopy)
		mlv1.Ready.Message(versionCopy, "Volume snapshot is ready")
		versionCopy.Status.VolumeSnapshot = status.SnapshotName
	case mlv1.SnapshottingPhaseFailed:
		mlv1.Ready.False(versionCopy)
		mlv1.Ready.Message(versionCopy, status.Message)
	default:
		// For other phases, keep the current Ready status but update the message
		mlv1.Ready.Message(versionCopy, status.Message)
	}

	// Update the status
	_, err = h.LocalModelVersionClient.UpdateStatus(versionCopy)
	if err != nil {
		return fmt.Errorf("failed to update status of local model version %s/%s: %w", namespace, name, err)
	}

	return nil
}

// GetContentSize implements ResourceHandler interface
func (h *handler) GetContentSize(ctx context.Context, namespace, name string) (int64, error) {
	// Get the LocalModelVersion to extract registry and model information
	version, err := h.LocalModelVersionCache.Get(namespace, name)
	if err != nil {
		return -1, fmt.Errorf("failed to get local model version %s/%s: %w", namespace, name, err)
	}

	// Extract registry and model information from labels
	registryName := version.Labels[constant.LabelRegistryName]
	modelNamespace := version.Labels[constant.LabelModelNamespace]
	modelName := version.Labels[constant.LabelModelName]

	b, err := h.rm.NewBackendFromRegistry(ctx, registryName)
	if err != nil {
		return -1, fmt.Errorf("failed to get backend from registry %s: %w", registryName, err)
	}

	model, err := h.ModelCache.Get(modelNamespace, modelName)
	if err != nil {
		return -1, fmt.Errorf("failed to get model %s/%s: %w", modelNamespace, modelName, err)
	}

	return b.GetSize(ctx, model.Status.RootPath)
}

// GetLatestReadySnapshot implements ResourceHandler interface
func (h *handler) GetLatestReadySnapshot(namespace, localModelName string) (string, error) {
	versions, err := h.LocalModelVersionCache.List(namespace, labels.Set{
		constant.LabelLocalModelName: localModelName,
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

// GetResourceType returns the resource type for event handler naming
func (h *handler) GetResourceType() string {
	return mlv1.LocalModelVersionResourceName
}
