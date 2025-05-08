package localcache

import (
	"fmt"
	"reflect"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/sirupsen/logrus"
)

// ModelCacheAdapter adapts Model to the CacheableResource interface
type ModelCacheAdapter struct {
	*mlv1.Model
	client ctlmlv1.ModelClient
}

// NewModelCacheAdapter creates a new ModelCacheAdapter
func NewModelCacheAdapter(model *mlv1.Model, client ctlmlv1.ModelClient) *ModelCacheAdapter {
	return &ModelCacheAdapter{
		Model:  model,
		client: client,
	}
}

// GetLocalCacheState gets the local cache state
func (m *ModelCacheAdapter) GetLocalCacheState() mlv1.CacheStateType {
	return m.Spec.LocalCache
}

// GetCacheStatus gets the cache status
func (m *ModelCacheAdapter) GetCacheStatus() *mlv1.CacheStatus {
	return m.Status.CacheStatus
}

// SetCacheStatus sets the cache status
// If the client is nil, the function invoker is responsible for updating the model
func (m *ModelCacheAdapter) SetCacheStatus(status *mlv1.CacheStatus) error {
	if reflect.DeepEqual(m.Model.Status.CacheStatus, status) {
		logrus.Infof("Cache status is already up to date for model %s/%s", m.Namespace, m.Name)
		return nil
	}

	m.Status.CacheStatus = status
	if (status.Status == mlv1.CacheStatusCompleted || status.Status == mlv1.CacheStatusFailed) &&
		m.Model.Spec.LocalCache == mlv1.CacheStateActive {
		m.Spec.LocalCache = mlv1.CacheStateInactive
	}
	if m.client == nil {
		return nil
	}

	modelCopy := m.Model.DeepCopy()
	newModel, err := m.client.Update(modelCopy)
	if err != nil {
		return fmt.Errorf("failed to update cache status of model %s/%s: %w", m.Namespace, m.Name, err)
	}
	m.Model = newModel

	return nil
}

// GetResourcePath gets the resource path
func (m *ModelCacheAdapter) GetResourcePath() string {
	return m.Status.RootPath
}

// GetResourceType gets the resource type
func (m *ModelCacheAdapter) GetResourceType() string {
	return mlv1.ModelResourceName
}

// GetResourceVersion gets the resource version
func (m *ModelCacheAdapter) GetResourceVersion() string {
	// For models, we may not have a clear version, so we can use the resource version or other identifier
	return m.ResourceVersion
}

// GetRegistry gets the registry
func (m *ModelCacheAdapter) GetRegistry() string {
	return m.Spec.Registry
}

// Ensure ModelCacheAdapter implements the CacheableResource interface
var _ CacheableResource = &ModelCacheAdapter{}
