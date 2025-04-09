package dataset

import (
	"fmt"
	"path"
	"reflect"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
)

func (h *handler) OnChangeDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.DeletionTimestamp != nil {
		return dv, nil
	}

	logrus.Infof("version %s(%s) of dataset %s/%s changed", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	dvCopy := dv.DeepCopy()

	// check dataset ready
	dataset, err := h.checkDatasetReady(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		return h.updateDatasetVersionStatus(dvCopy, dv, err)
	}
	dvCopy.Status.Registry = dataset.Spec.Registry

	if !mlv1.Ready.IsTrue(dv) {
		// create directory for dataset version
		b, err := h.rm.NewBackendFromRegistry(h.ctx, dataset.Spec.Registry)
		if err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf(registry.ErrCreateBackendClient, err))
		}
		versionDir := path.Join(dataset.Status.RootPath, dv.Spec.Version)
		if err = b.CreateDirectory(h.ctx, versionDir); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, err)
		}
		dvCopy.Status.RootPath = versionDir

		// copy from other dataset version
		if err = h.copyFrom(b, versionDir, dv.Spec.CopyFrom); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("copy failed: %w", err))
		}
	}

	// add version to dataset status
	if _, exist := versionExists(dataset.Status.Versions, dv.Spec.Version); !exist {
		datasetCopy := dataset.DeepCopy()
		datasetCopy.Status.Versions = append(datasetCopy.Status.Versions,
			mlv1.Version{Version: dv.Spec.Version, ObjectName: dv.Name})

		if _, err := h.updateDatasetStatus(datasetCopy, dataset, nil); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("add version %s to dataset %s/%s failed: %w",
				dv.Spec.Version, dv.Namespace, dv.Spec.Dataset, err))
		}
	}

	return h.updateDatasetVersionStatus(dvCopy, dv, nil)
}

func (h *handler) OnRemoveDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.Status.RootPath == "" || dv.DeletionTimestamp == nil {
		return nil, nil
	}

	logrus.Infof("delete dataset version %s(%s) of %s/%s", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	// delete directory of dataset version
	b, err := h.rm.NewBackendFromRegistry(h.ctx, dv.Status.Registry)
	if err != nil {
		return nil, fmt.Errorf(registry.ErrCreateBackendClient, err)
	}
	if err = b.Delete(h.ctx, dv.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete files of dataset version %s/%s/%s failed: %w",
			dv.Namespace, dv.Spec.Dataset, dv.Name, err)
	}

	// remove version from dataset status
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
			return nil, fmt.Errorf("remove version %s from dataset %s/%s failed: %w",
				dv.Spec.Version, dataset.Namespace, dataset.Name, err)
		}
	}

	return dv, nil
}

func (h *handler) copyFrom(b backend.Backend, dst string, copyFrom *mlv1.CopyFrom) error {
	if copyFrom == nil {
		return nil
	}

	logrus.Debugf("copy from %s/%s/%s", copyFrom.Namespace, copyFrom.Dataset, copyFrom.Version)

	// check if the source dataset version is exist
	dataset, err := h.datasetCache.Get(copyFrom.Namespace, copyFrom.Dataset)
	if err != nil {
		return fmt.Errorf("get dataset %s/%s failed: %w", copyFrom.Namespace, copyFrom.Dataset, err)
	}
	if !mlv1.Ready.IsTrue(dataset) {
		return fmt.Errorf("dataset %s/%s is not ready", copyFrom.Namespace, copyFrom.Dataset)
	}
	if _, exist := versionExists(dataset.Status.Versions, copyFrom.Version); !exist {
		return fmt.Errorf("version %s of dataset %s/%s not found", copyFrom.Version, copyFrom.Namespace, copyFrom.Dataset)
	}
	src := path.Join(dataset.Status.RootPath, copyFrom.Version)

	if err := b.Copy(h.ctx, src, dst); err != nil {
		return fmt.Errorf("copy from %s/%s/%s failed: %w", copyFrom.Namespace, copyFrom.Dataset, copyFrom.Version, err)
	}

	return nil
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
		return nil, fmt.Errorf("update dataset version status failed: %w", updateErr)
	}
	return updatedDatasetVersion, err
}

func versionExists(versions []mlv1.Version, version string) (int, bool) {
	for i, v := range versions {
		if v.Version == version {
			return i, true
		}
	}
	return -1, false
}
