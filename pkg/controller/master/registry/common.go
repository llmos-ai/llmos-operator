package registry

import (
	"fmt"
	"path"

	"github.com/sirupsen/logrus"

	ml "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
)

const (
	VersionLabelKey = ml.GroupName + "/version"
)

func (h *handler) createRootDir(registry, resourceName, namespace, name, version string) (string, error) {
	b, err := h.rm.NewBackendFromRegistry(registry)
	if err != nil {
		return "", fmt.Errorf("new backend for %s failed: %w", registry, err)
	}

	dir := path.Join(resourceName, namespace, name, version)
	// Create base model directory
	if err = b.CreateDirectory(dir); err != nil {
		return "", fmt.Errorf("create directory %s failed: %w", dir, err)
	}

	return dir, nil
}

func (h *handler) copyFrom(registry, resourceName, dst string, copyFrom *mlv1.CopyFrom) error {
	if copyFrom == nil {
		return nil
	}

	logrus.Debugf("copy from %s/%s/%s", copyFrom.Namespace, copyFrom.Name, copyFrom.Version)

	var src string
	switch resourceName {
	case mlv1.ModelResourceName:
		model, err := h.modelCache.Get(copyFrom.Namespace, copyFrom.Name)
		if err != nil {
			return fmt.Errorf("get model %s/%s failed: %w", copyFrom.Namespace, copyFrom.Name, err)
		}
		if !mlv1.Ready.IsTrue(model) {
			return fmt.Errorf("model %s/%s is not ready", copyFrom.Namespace, copyFrom.Name)
		}
		if _, exist := versionExists(model.Status.Versions, copyFrom.Version); !exist {
			return fmt.Errorf("version %s of model %s/%s not found", copyFrom.Version, copyFrom.Namespace, copyFrom.Name)
		}
		src = path.Join(model.Status.RootPath, copyFrom.Version)

	case mlv1.DatasetResourceName:
		dataset, err := h.datasetCache.Get(copyFrom.Namespace, copyFrom.Name)
		if err != nil {
			return fmt.Errorf("get dataset %s/%s failed: %w", copyFrom.Namespace, copyFrom.Name, err)
		}
		if !mlv1.Ready.IsTrue(dataset) {
			return fmt.Errorf("dataset %s/%s is not ready", copyFrom.Namespace, copyFrom.Name)
		}
		if _, exist := versionExists(dataset.Status.Versions, copyFrom.Version); !exist {
			return fmt.Errorf("version %s of dataset %s/%s not found", copyFrom.Version, copyFrom.Namespace, copyFrom.Name)
		}
		src = path.Join(dataset.Status.RootPath, copyFrom.Version)

	default:
		return fmt.Errorf("unsupported copy from resource %s", resourceName)
	}

	b, err := h.rm.NewBackendFromRegistry(registry)
	if err != nil {
		return fmt.Errorf("new backend for %s failed: %w", registry, err)
	}

	if err := b.Copy(src, dst); err != nil {
		return fmt.Errorf("copy from %s/%s/%s failed: %w", copyFrom.Namespace, copyFrom.Name, copyFrom.Version, err)
	}

	return nil
}

func (h *handler) deleteRootDir(reg, path string) error {
	b, err := h.rm.NewBackendFromRegistry(reg)
	if err != nil {
		return fmt.Errorf(registry.ErrCreateBackendClient, err)
	}

	if err := b.Delete(path); err != nil {
		return fmt.Errorf(registry.ErrDeleteFile, path, err)
	}
	if err := b.DeleteDirectory(path); err != nil {
		return fmt.Errorf(registry.ErrDeleteFile, path, err)
	}
	return nil
}

func versionExists(versions []mlv1.Version, version string) (int, bool) {
	for i, v := range versions {
		if v.Version == version {
			return i, true
		}
	}
	return -1, false
}
