package registry

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
)

func (h *handler) OnChangeModel(_ string, model *mlv1.Model) (*mlv1.Model, error) {
	if model == nil || model.DeletionTimestamp != nil {
		return model, nil
	}

	logrus.Debugf("model %s/%s changed", model.Namespace, model.Name)

	modelCopy := model.DeepCopy()

	modelRootDir, err := h.createRootDir(model.Spec.Registry, mlv1.ModelResourceName, model.Namespace, model.Name, "")
	if err != nil {
		return h.updateModelStatus(modelCopy, model, fmt.Errorf(registry.ErrCreateDirectory, modelRootDir, err))
	}

	modelCopy.Status.RootPath = modelRootDir
	return h.updateModelStatus(modelCopy, model, nil)
}

func (h *handler) OnRemoveModel(_ string, model *mlv1.Model) (*mlv1.Model, error) {
	if model == nil || model.Status.RootPath == "" {
		return nil, nil
	}

	logrus.Debugf("model %s/%s deleted", model.Namespace, model.Name)

	if err := h.deleteRootDir(model.Spec.Registry, model.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete root path %s failed: %w", model.Status.RootPath, err)
	}

	return model, nil
}

func (h *handler) OnChangeModelVersion(_ string, mv *mlv1.ModelVersion) (*mlv1.ModelVersion, error) {
	if mv == nil || mv.DeletionTimestamp != nil {
		return mv, nil
	}

	logrus.Debugf("version %s(%s) of model %s/%s changed", mv.Spec.Version, mv.Name, mv.Namespace, mv.Spec.Model)

	mvCopy := mv.DeepCopy()

	model, err := h.checkModelReady(mv.Namespace, mv.Spec.Model)
	if err != nil {
		return h.updateModelVersionStatus(mvCopy, mv, err)
	}
	mvCopy.Status.Registry = model.Spec.Registry

	if !mlv1.Ready.IsTrue(mv) {
		versionDir, err := h.createRootDir(model.Spec.Registry, mlv1.ModelResourceName, mv.Namespace, mv.Spec.Model, mv.Spec.Version)
		if err != nil {
			return h.updateModelVersionStatus(mvCopy, mv, err)
		}
		mvCopy.Status.RootPath = versionDir

		if err = h.copyFrom(model.Spec.Registry, mlv1.ModelResourceName, versionDir, mv.Spec.CopyFrom); err != nil {
			return h.updateModelVersionStatus(mvCopy, mv, fmt.Errorf("copy failed: %w", err))
		}
	}

	if _, exist := versionExists(model.Status.Versions, mv.Spec.Version); !exist {
		modelCopy := model.DeepCopy()
		modelCopy.Status.Versions = append(modelCopy.Status.Versions, mlv1.Version{Version: mv.Spec.Version, ObjectName: mv.Name})
		if _, err := h.updateModelStatus(modelCopy, model, nil); err != nil {
			return h.updateModelVersionStatus(mvCopy, mv, fmt.Errorf("add version %s to model %s/%s failed: %w",
				mv.Spec.Version, mv.Namespace, mv.Spec.Model, err))
		}
	}
	return h.updateModelVersionStatus(mvCopy, mv, nil)
}

func (h *handler) OnRemoveModelVersion(_ string, mv *mlv1.ModelVersion) (*mlv1.ModelVersion, error) {
	if mv == nil || mv.Status.RootPath == "" {
		return nil, nil
	}

	logrus.Infof("delete model version %s/%s/%s", mv.Namespace, mv.Spec.Model, mv.Name)

	if err := h.deleteRootDir(mv.Status.Registry, mv.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete root path %s failed: %w", mv.Status.RootPath, err)
	}

	model, err := h.modelCache.Get(mv.Namespace, mv.Spec.Model)
	if err != nil {
		if errors.IsNotFound(err) {
			return mv, nil
		}
		return nil, fmt.Errorf("get model %s/%s failed: %w", mv.Namespace, mv.Spec.Model, err)
	}

	if index, exist := versionExists(model.Status.Versions, mv.Spec.Version); exist {
		modelCopy := model.DeepCopy()
		modelCopy.Status.Versions = append(modelCopy.Status.Versions[:index], modelCopy.Status.Versions[index+1:]...)
		if _, err := h.updateModelStatus(modelCopy, model, nil); err != nil {
			return nil, fmt.Errorf("remove version %s from model %s/%s failed: %w", mv.Name, model.Namespace, model.Name, err)
		}
	}

	return mv, nil
}

func (h *handler) updateModelStatus(modelCopy, model *mlv1.Model, err error) (*mlv1.Model, error) {
	if err == nil {
		mlv1.Ready.True(modelCopy)
		mlv1.Ready.Message(modelCopy, "")
	} else {
		mlv1.Ready.False(modelCopy)
		mlv1.Ready.Message(modelCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(modelCopy.Status, model.Status) {
		return modelCopy, err
	}

	updatedModel, updateErr := h.modelClient.UpdateStatus(modelCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update model status failed: %w", err)
	}
	return updatedModel, err
}

func (h *handler) updateModelVersionStatus(mvCopy, mv *mlv1.ModelVersion, err error) (*mlv1.ModelVersion, error) {
	if err == nil {
		mlv1.Ready.True(mvCopy)
		mlv1.Ready.Message(mvCopy, "")
	} else {
		mlv1.Ready.False(mvCopy)
		mlv1.Ready.Message(mvCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(mvCopy.Status, mv.Status) {
		return mvCopy, err
	}
	updatedModelVersion, updateErr := h.modelVersionClient.UpdateStatus(mvCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update model version status failed: %w", err)
	}
	return updatedModelVersion, err
}

func (h *handler) checkModelReady(namespace, name string) (*mlv1.Model, error) {
	model, err := h.modelCache.Get(namespace, name)
	if err != nil {
		return nil, err
	}

	if !mlv1.Ready.IsTrue(model) {
		return nil, fmt.Errorf("model %s/%s is not ready", namespace, name)
	}

	return model, nil
}
