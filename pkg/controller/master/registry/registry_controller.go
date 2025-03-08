package registry

import (
	"fmt"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
)

type handler struct {
	registryClient       ctlmlv1.RegistryClient
	registryCache        ctlmlv1.RegistryCache
	secretClient         ctlcorev1.SecretClient
	secretCache          ctlcorev1.SecretCache
	modelClient          ctlmlv1.ModelClient
	modelCache           ctlmlv1.ModelCache
	modelVersionCache    ctlmlv1.ModelVersionCache
	modelVersionClient   ctlmlv1.ModelVersionClient
	datasetClient        ctlmlv1.DatasetClient
	datasetCache         ctlmlv1.DatasetCache
	datasetVersionClient ctlmlv1.DatasetVersionClient
	datasetVersionCache  ctlmlv1.DatasetVersionCache

	rm *registry.Manager
}

func (h *handler) CheckRegistryAccessibility(_ string, registry *mlv1.Registry) (*mlv1.Registry, error) {
	if registry == nil || registry.DeletionTimestamp != nil {
		return registry, nil
	}

	logrus.Debugf("Checking registry accessibility of registry %s", registry.Name)

	b, err := h.checkRegistryAccessibility(registry)
	if err != nil {
		return nil, err
	}

	registryCopy := registry.DeepCopy()
	registryCopy.Status.StorageAddress = b.GetObjectURL("")
	return h.updateRegistryAccessibleCondition(registry, true, "S3 bucket is accessible")
}

func (h *handler) checkRegistryAccessibility(registry *mlv1.Registry) (backend.Backend, error) {
	b, err := h.rm.NewBackend(registry)
	if err != nil {
		return nil, fmt.Errorf("check registry accessibility failed: %w", err)
	}

	return b, nil
}

func (h *handler) updateRegistryAccessibleCondition(registry *mlv1.Registry, accessible bool, message string) (*mlv1.Registry, error) {
	toUpdate := registry.DeepCopy()

	if accessible {
		mlv1.Accessible.True(toUpdate)
		mlv1.Accessible.Message(toUpdate, message)
	} else {
		mlv1.Accessible.False(toUpdate)
		mlv1.Accessible.Message(toUpdate, message)
	}

	return h.registryClient.UpdateStatus(toUpdate)
}
