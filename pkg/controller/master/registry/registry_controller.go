package registry

import (
	"context"
	"fmt"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	registryOnChangeName = "registry.OnChange"
)

type handler struct {
	ctx context.Context

	registryClient ctlmlv1.RegistryClient
	registryCache  ctlmlv1.RegistryCache
	secretClient   ctlcorev1.SecretClient
	secretCache    ctlcorev1.SecretCache

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()

	h := handler{
		ctx: mgmt.Ctx,

		registryClient: registries,
		registryCache:  registries.Cache(),
		secretClient:   secrets,
		secretCache:    secrets.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache(), registries.Cache())

	registries.OnChange(mgmt.Ctx, registryOnChangeName, h.CheckRegistryAccessibility)
	return nil
}

func (h *handler) CheckRegistryAccessibility(_ string, registry *mlv1.Registry) (*mlv1.Registry, error) {
	if registry == nil || registry.DeletionTimestamp != nil {
		return registry, nil
	}

	logrus.Debugf("Checking registry accessibility of registry %s", registry.Name)

	b, err := h.checkRegistryAccessibility(registry)
	if err != nil {
		registryCopy := registry.DeepCopy()
		registryCopy.Status.StorageAddress = ""
		_, _ = h.updateRegistryAccessibleCondition(registryCopy, false, err.Error())
		return registryCopy, err
	}

	registryCopy := registry.DeepCopy()
	registryCopy.Status.StorageAddress = b.GetObjectURL("")
	return h.updateRegistryAccessibleCondition(registryCopy, true, "S3 bucket is accessible")
}

func (h *handler) checkRegistryAccessibility(registry *mlv1.Registry) (backend.Backend, error) {
	b, err := h.rm.NewBackend(h.ctx, registry)
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
