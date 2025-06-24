package model

import (
	"fmt"

	cr "github.com/llmos-ai/llmos-operator/pkg/api/common/registry"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type ModelGetter func(namespace, name string) (*mlv1.Model, error)

type Handler struct {
	modelCache ctlmlv1.ModelCache

	cr.BaseHandler
}

func NewHandler(scaled *config.Scaled) Handler {
	h := Handler{
		modelCache: scaled.Management.LLMFactory.Ml().V1().Model().Cache(),
	}

	registryCache := scaled.Management.LLMFactory.Ml().V1().Registry().Cache()
	secretCache := scaled.CoreFactory.Core().V1().Secret().Cache()

	h.BaseHandler = cr.BaseHandler{
		Ctx:                    scaled.Ctx,
		GetRegistryAndRootPath: h.GetRegistryAndRootPath,
		RegistryManager:        registry.NewManager(secretCache.Get, registryCache.Get),
		PostHooks:              make(map[string]cr.PostHook),
	}

	return h
}

func (h Handler) GetRegistryAndRootPath(namespace, name string) (string, string, error) {
	return GetModelRegistryAndRootPath(h.modelCache.Get, namespace, name)
}

func GetModelRegistryAndRootPath(modelGetter ModelGetter, namespace, name string) (string, string, error) {
	var registry, rootPath string
	v, err := modelGetter(namespace, name)
	if err != nil {
		return "", "", fmt.Errorf("get model %s/%s failed: %w", namespace, name, err)
	}
	if !mlv1.Ready.IsTrue(v) {
		return "", "", fmt.Errorf("model %s/%s is not ready", namespace, name)
	}
	registry, rootPath = v.Spec.Registry, v.Status.RootPath

	return registry, rootPath, nil
}
