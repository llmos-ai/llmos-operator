package localcache

import (
	"fmt"
	"reflect"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
)

// DatasetVersionCacheAdapter adapts DatasetVersion to the CacheableResource interface
type DatasetVersionCacheAdapter struct {
	*mlv1.DatasetVersion
	client ctlmlv1.DatasetVersionClient
}

// NewDatasetVersionCacheAdapter creates a new DatasetVersionCacheAdapter
func NewDatasetVersionCacheAdapter(
	dv *mlv1.DatasetVersion,
	client ctlmlv1.DatasetVersionClient,
) *DatasetVersionCacheAdapter {
	return &DatasetVersionCacheAdapter{
		DatasetVersion: dv,
		client:         client,
	}
}

// GetLocalCacheState gets the local cache state
func (d *DatasetVersionCacheAdapter) GetLocalCacheState() mlv1.CacheStateType {
	return d.Spec.LocalCache
}

// GetCacheStatus gets the cache status
func (d *DatasetVersionCacheAdapter) GetCacheStatus() *mlv1.CacheStatus {
	return d.Status.CacheStatus
}

// SetCacheStatus sets the cache status
func (d *DatasetVersionCacheAdapter) SetCacheStatus(status *mlv1.CacheStatus) error {
	if reflect.DeepEqual(d.DatasetVersion.Status.CacheStatus, status) {
		return nil
	}

	d.Status.CacheStatus = status
	if (status.Status == mlv1.CacheStatusCompleted || status.Status == mlv1.CacheStatusFailed) &&
		d.DatasetVersion.Spec.LocalCache == mlv1.CacheStateActive {
		d.Spec.LocalCache = mlv1.CacheStateInactive
	}
	if d.client == nil {
		return nil
	}

	dvCopy := d.DatasetVersion.DeepCopy()
	newDv, err := d.client.Update(dvCopy)
	if err != nil {
		return fmt.Errorf("failed to update cache status of datasetversion %s/%s: %w", d.Namespace, d.Name, err)
	}
	d.DatasetVersion = newDv

	return nil
}

// GetResourcePath gets the resource path
func (d *DatasetVersionCacheAdapter) GetResourcePath() string {
	return d.Status.RootPath
}

// GetResourceType gets the resource type
func (d *DatasetVersionCacheAdapter) GetResourceType() string {
	return mlv1.DatasetVersionResourceName
}

// GetResourceVersion gets the resource version
func (d *DatasetVersionCacheAdapter) GetResourceVersion() string {
	return d.Spec.Version
}

// GetRegistry gets the registry
func (d *DatasetVersionCacheAdapter) GetRegistry() string {
	return d.Status.Registry
}

// Ensure DatasetVersionCacheAdapter implements the CacheableResource interface
var _ CacheableResource = &DatasetVersionCacheAdapter{}
