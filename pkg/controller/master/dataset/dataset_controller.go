package dataset

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/sirupsen/logrus"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/common/snapshotting"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	datasetOnChangeName        = "dataset.OnChange"
	datasetOnRemoveName        = "dataset.OnRemove"
	datasetVersionOnChangeName = "datasetversion.OnChange"
	datasetVersionOnRemoveName = "datasetversion.OnRemove"

	volumeName      = "dataset-volume"
	volumeMountPath = "/data/"
)

type handler struct {
	ctx context.Context

	datasetClient        ctlmlv1.DatasetClient
	datasetCache         ctlmlv1.DatasetCache
	datasetVersionClient ctlmlv1.DatasetVersionClient
	datasetVersionCache  ctlmlv1.DatasetVersionCache

	rm              *registry.Manager
	snapshotManager *snapshotting.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	datasets := mgmt.LLMFactory.Ml().V1().Dataset()
	datasetVersions := mgmt.LLMFactory.Ml().V1().DatasetVersion()

	h := handler{
		ctx: mgmt.Ctx,

		datasetClient:        datasets,
		datasetCache:         datasets.Cache(),
		datasetVersionClient: datasetVersions,
		datasetVersionCache:  datasetVersions.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	// Initialize snapshotting manager
	snapshotManager, err := snapshotting.NewManager(mgmt, &h)
	if err != nil {
		return fmt.Errorf("failed to create snapshotting manager: %w", err)
	}
	h.snapshotManager = snapshotManager

	datasets.OnChange(mgmt.Ctx, datasetOnChangeName, h.OnChangeDataset)
	datasets.OnRemove(mgmt.Ctx, datasetOnRemoveName, h.OnRemoveDataset)
	datasetVersions.OnChange(mgmt.Ctx, datasetVersionOnChangeName, h.OnChangeDatasetVersion)
	datasetVersions.OnRemove(mgmt.Ctx, datasetVersionOnRemoveName, h.OnRemoveDatasetVersion)
	return nil
}

func (h *handler) OnChangeDataset(_ string, dataset *mlv1.Dataset) (*mlv1.Dataset, error) {
	if dataset == nil || dataset.DeletionTimestamp != nil {
		return dataset, nil
	}

	logrus.Infof("dataset %s/%s changed", dataset.Namespace, dataset.Name)

	datasetCopy := dataset.DeepCopy()

	// create root directory for dataset
	datasetRootDir := path.Join(mlv1.DatasetResourceName, dataset.Namespace, dataset.Name)
	b, err := h.rm.NewBackendFromRegistry(h.ctx, dataset.Spec.Registry)
	if err != nil {
		return h.updateDatasetStatus(datasetCopy, dataset, fmt.Errorf(registry.ErrCreateBackendClient, err))
	}
	if err := b.CreateDirectory(h.ctx, datasetRootDir); err != nil {
		return h.updateDatasetStatus(datasetCopy, dataset, fmt.Errorf(registry.ErrCreateDirectory, datasetRootDir, err))
	}

	datasetCopy.Status.RootPath = datasetRootDir
	return h.updateDatasetStatus(datasetCopy, dataset, nil)
}

func (h *handler) OnRemoveDataset(_ string, dataset *mlv1.Dataset) (*mlv1.Dataset, error) {
	if dataset == nil || dataset.Status.RootPath == "" || dataset.DeletionTimestamp == nil {
		return nil, nil
	}

	logrus.Infof("dataset %s/%s deleted", dataset.Namespace, dataset.Name)

	// delete root directory of the dataset
	b, err := h.rm.NewBackendFromRegistry(h.ctx, dataset.Spec.Registry)
	if err != nil {
		return nil, fmt.Errorf(registry.ErrCreateBackendClient, err)
	}
	if err := b.Delete(h.ctx, dataset.Status.RootPath); err != nil {
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
		return nil, fmt.Errorf("update dataset status failed: %w", updateErr)
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
