package registry

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
)

func (h *handler) OnChangeDataset(_ string, dataset *mlv1.Dataset) (*mlv1.Dataset, error) {
	if dataset == nil || dataset.DeletionTimestamp != nil {
		return dataset, nil
	}

	logrus.Debugf("dataset %s/%s changed", dataset.Namespace, dataset.Name)

	datasetCopy := dataset.DeepCopy()

	datasetRootDir, err := h.createRootDir(dataset.Spec.Registry, mlv1.DatasetResourceName, dataset.Namespace, dataset.Name, "")
	if err != nil {
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

	if err := h.deleteRootDir(dataset.Spec.Registry, dataset.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete root path %s failed: %w", dataset.Status.RootPath, err)
	}

	return dataset, nil
}

func (h *handler) OnChangeDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.DeletionTimestamp != nil {
		return dv, nil
	}

	logrus.Debugf("version %s(%s) of dataset %s/%s changed", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	dvCopy := dv.DeepCopy()

	dataset, err := h.checkDatasetReady(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		return h.updateDatasetVersionStatus(dvCopy, dv, err)
	}
	dvCopy.Status.Registry = dataset.Spec.Registry

	if !mlv1.Ready.IsTrue(dv) {
		versionDir, err := h.createRootDir(dataset.Spec.Registry, mlv1.DatasetResourceName, dv.Namespace, dv.Spec.Dataset, dv.Spec.Version)
		if err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, err)
		}
		dvCopy.Status.RootPath = versionDir

		if err = h.copyFrom(dataset.Spec.Registry, mlv1.DatasetResourceName, versionDir, dv.Spec.CopyFrom); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("copy failed: %w", err))
		}
	}

	if _, exist := versionExists(dataset.Status.Versions, dv.Spec.Version); !exist {
		datasetCopy := dataset.DeepCopy()
		datasetCopy.Status.Versions = append(datasetCopy.Status.Versions, mlv1.Version{Version: dv.Spec.Version, ObjectName: dv.Name})
		if _, err := h.updateDatasetStatus(datasetCopy, dataset, nil); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("add version %s to dataset %s/%s failed: %w",
				dv.Spec.Version, dv.Namespace, dv.Spec.Dataset, err))
		}
	}

	return h.updateDatasetVersionStatus(dvCopy, dv, nil)
}

func (h *handler) OnRemoveDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.Status.RootPath == "" {
		return nil, nil
	}

	logrus.Debugf("delete dataset version %s(%s) of %s/%s", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	if err := h.deleteRootDir(dv.Status.Registry, dv.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete files of dataset version %s/%s/%s failed: %w", dv.Namespace, dv.Spec.Dataset, dv.Name, err)
	}

	dataset, err := h.datasetCache.Get(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		if errors.IsNotFound(err) {
			return dv, nil
		}
		return nil, fmt.Errorf("get dataset %s/%s failed: %w", dv.Namespace, dv.Spec.Dataset, err)
	}
	if index, exist := versionExists(dataset.Status.Versions, dv.Spec.Version); exist {
		datasetCopy := dataset.DeepCopy()
		datasetCopy.Status.Versions = append(datasetCopy.Status.Versions[:index], datasetCopy.Status.Versions[index+1:]...)
		if _, err := h.updateDatasetStatus(datasetCopy, dataset, nil); err != nil {
			return nil, fmt.Errorf("remove version %s from dataset %s/%s failed: %w", dv.Spec.Version, dataset.Namespace, dataset.Name, err)
		}
	}

	return dv, nil
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

func (h *handler) updateDatasetVersionStatus(dvCopy, dv *mlv1.DatasetVersion, err error) (*mlv1.DatasetVersion, error) {
	if err == nil {
		mlv1.Ready.True(dvCopy)
		mlv1.Ready.Message(dvCopy, "")
	} else {
		mlv1.Ready.False(dvCopy)
		mlv1.Ready.Message(dvCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(dvCopy.Status, dv.Status) {
		return dvCopy, err
	}
	updatedDatasetVersion, updateErr := h.datasetVersionClient.UpdateStatus(dvCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update dataset version status failed: %w", err)
	}
	return updatedDatasetVersion, err
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
