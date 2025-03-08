package registry

import (
	"context"

	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	registryOnChangeName       = "registry.OnChange"
	modelOnChangeName          = "model.OnChange"
	modelOnRemoveName          = "model.OnRemove"
	modelVersionOnChangeName   = "modelversion.OnChange"
	modelVersionOnRemoveName   = "modelversion.OnRemove"
	datasetOnChangeName        = "dataset.OnChange"
	datasetOnRemoveName        = "dataset.OnRemove"
	datasetVersionOnChangeName = "datasetversion.OnChange"
	datasetVersionOnRemoveName = "datasetversion.OnRemove"
)

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	models := mgmt.LLMFactory.Ml().V1().Model()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	modelVersions := mgmt.LLMFactory.Ml().V1().ModelVersion()
	datasets := mgmt.LLMFactory.Ml().V1().Dataset()
	datasetVersions := mgmt.LLMFactory.Ml().V1().DatasetVersion()

	h := handler{
		registryClient:       registries,
		registryCache:        registries.Cache(),
		secretClient:         secrets,
		secretCache:          secrets.Cache(),
		modelClient:          models,
		modelCache:           models.Cache(),
		modelVersionClient:   modelVersions,
		modelVersionCache:    modelVersions.Cache(),
		datasetClient:        datasets,
		datasetCache:         datasets.Cache(),
		datasetVersionClient: datasetVersions,
		datasetVersionCache:  datasetVersions.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache(), registries.Cache(), models.Cache(), datasets.Cache())

	registries.OnChange(mgmt.Ctx, registryOnChangeName, h.CheckRegistryAccessibility)
	models.OnChange(mgmt.Ctx, modelOnChangeName, h.OnChangeModel)
	models.OnRemove(mgmt.Ctx, modelOnRemoveName, h.OnRemoveModel)
	modelVersions.OnChange(mgmt.Ctx, modelVersionOnChangeName, h.OnChangeModelVersion)
	modelVersions.OnRemove(mgmt.Ctx, modelVersionOnRemoveName, h.OnRemoveModelVersion)
	datasets.OnChange(mgmt.Ctx, datasetOnChangeName, h.OnChangeDataset)
	datasets.OnRemove(mgmt.Ctx, datasetOnRemoveName, h.OnRemoveDataset)
	datasetVersions.OnChange(mgmt.Ctx, datasetOnChangeName, h.OnChangeDatasetVersion)
	datasetVersions.OnRemove(mgmt.Ctx, datasetOnRemoveName, h.OnRemoveDatasetVersion)
	return nil
}
