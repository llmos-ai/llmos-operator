package datasetversion

import (
	"fmt"

	cr "github.com/llmos-ai/llmos-operator/pkg/api/common/registry"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type DatasetVersionGetter func(namespace, name string) (*mlv1.DatasetVersion, error)

type Handler struct {
	dvCache ctlmlv1.DatasetVersionCache

	cr.BaseHandler
}

func NewHandler(scaled *config.Scaled) Handler {
	h := Handler{
		dvCache: scaled.Management.LLMFactory.Ml().V1().DatasetVersion().Cache(),
	}

	registryCache := scaled.Management.LLMFactory.Ml().V1().Registry().Cache()
	secretCache := scaled.CoreFactory.Core().V1().Secret().Cache()

	h.BaseHandler = cr.BaseHandler{
		Ctx:                    scaled.Ctx,
		GetRegistryAndRootPath: h.GetRegistryAndRootPath,
		RegistryManager:        registry.NewManager(secretCache.Get, registryCache.Get),
	}

	return h
}

func (h Handler) GetRegistryAndRootPath(namespace, name string) (string, string, error) {
	return GetDatasetVersionRegistryAndRootPath(h.dvCache.Get, namespace, name)
}

func GetDatasetVersionRegistryAndRootPath(dvGetter DatasetVersionGetter, namespace, name string) (string, string, error) {
	var registry, rootPath string
	v, err := dvGetter(namespace, name)
	if err != nil {
		return "", "", fmt.Errorf("get datasetversion %s/%s failed: %w", namespace, name, err)
	}
	if !mlv1.Ready.IsTrue(v) {
		return "", "", fmt.Errorf("datasetversion %s/%s is not ready", namespace, name)
	}
	registry, rootPath = v.Status.Registry, v.Status.RootPath

	return registry, rootPath, nil
}
