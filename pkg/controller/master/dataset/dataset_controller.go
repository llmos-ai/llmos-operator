package dataset

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/sirupsen/logrus"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	datasetOnChangeName        = "dataset.OnChange"
	datasetOnRemoveName        = "dataset.OnRemove"
	datasetVersionOnChangeName = "datasetversion.OnChange"
	datasetVersionOnRemoveName = "datasetversion.OnRemove"
)

type handler struct {
	datasetClient        ctlmlv1.DatasetClient
	datasetCache         ctlmlv1.DatasetCache
	datasetVersionClient ctlmlv1.DatasetVersionClient
	datasetVersionCache  ctlmlv1.DatasetVersionCache

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	datasets := mgmt.LLMFactory.Ml().V1().Dataset()
	datasetVersions := mgmt.LLMFactory.Ml().V1().DatasetVersion()

	h := handler{
		datasetClient:        datasets,
		datasetCache:         datasets.Cache(),
		datasetVersionClient: datasetVersions,
		datasetVersionCache:  datasetVersions.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache(), registries.Cache())

	datasets.OnChange(mgmt.Ctx, datasetOnChangeName, h.OnChangeDataset)
	datasets.OnRemove(mgmt.Ctx, datasetOnRemoveName, h.OnRemoveDataset)
	datasetVersions.OnChange(mgmt.Ctx, datasetOnChangeName, h.OnChangeDatasetVersion)
	datasetVersions.OnRemove(mgmt.Ctx, datasetOnRemoveName, h.OnRemoveDatasetVersion)
	return nil
}

func (h *handler) OnChangeDataset(_ string, dataset *mlv1.Dataset) (*mlv1.Dataset, error) {
	if dataset == nil || dataset.DeletionTimestamp != nil {
		return dataset, nil
	}

	logrus.Debugf("dataset %s/%s changed", dataset.Namespace, dataset.Name)

	datasetCopy := dataset.DeepCopy()

	// create root directory for dataset
	datasetRootDir := path.Join(mlv1.DatasetResourceName, dataset.Namespace, dataset.Name)
	b, err := h.rm.NewBackendFromRegistry(dataset.Spec.Registry)
	if err != nil {
		return h.updateDatasetStatus(datasetCopy, dataset, fmt.Errorf(registry.ErrCreateBackendClient, err))
	}
	if err := b.CreateDirectory(datasetRootDir); err != nil {
		return h.updateDatasetStatus(datasetCopy, dataset, fmt.Errorf(registry.ErrCreateDirectory, datasetRootDir, err))
	}

	datasetCopy.Status.RootPath = datasetRootDir
	return h.updateDatasetStatus(datasetCopy, dataset, nil)
}

func (h *handler) OnRemoveDataset(_ string, dataset *mlv1.Dataset) (*mlv1.Dataset, error) {
	if dataset == nil || dataset.Status.RootPath == "" {
		return nil, nil
	}

	logrus.Debugf("dataset %s/%s deleted", dataset.Namespace, dataset.Name)

	// delete root directory of the dataset
	b, err := h.rm.NewBackendFromRegistry(dataset.Spec.Registry)
	if err != nil {
		return nil, fmt.Errorf(registry.ErrCreateBackendClient, err)
	}
	if err := b.DeleteDirectory(dataset.Status.RootPath); err != nil {
		return nil, fmt.Errorf(registry.ErrDeleteFile, dataset.Status.RootPath, err)
	}

	return dataset, nil
}

func (h *handler) updateDatasetStatus(datasetCopy, dataset *mlv1.Dataset, err error) (*mlv1.Dataset, error) {
	if err == nil {
		mlv1.Ready.True(datasetCopy)
		mlv1.Ready.Message(datasetCopy, "")
	} else {
		mlv1.Ready.False(datasetCopy)
		mlv1.Ready.Message(datasetCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(datasetCopy.Status, dataset.Status) {
		return datasetCopy, err
	}

	updatedDataset, updateErr := h.datasetClient.UpdateStatus(datasetCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update dataset status failed: %w", err)
	}
	return updatedDataset, err
}

func (h *handler) checkDatasetReady(namespace, name string) (*mlv1.Dataset, error) {
	dataset, err := h.datasetCache.Get(namespace, name)
	if err != nil {
		return nil, err
	}

	if !mlv1.Ready.IsTrue(dataset) {
		return nil, fmt.Errorf("dataset %s/%s is not ready", namespace, name)
	}

	return dataset, nil
}
