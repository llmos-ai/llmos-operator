package model

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
	modelOnChangeName = "model.OnChange"
	modelOnRemoveName = "model.OnRemove"
)

type handler struct {
	ctx context.Context

	modelClient ctlmlv1.ModelClient
	modelCache  ctlmlv1.ModelCache

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	models := mgmt.LLMFactory.Ml().V1().Model()

	h := handler{
		ctx: mgmt.Ctx,

		modelClient: models,
		modelCache:  models.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache(), registries.Cache())

	models.OnChange(mgmt.Ctx, modelOnChangeName, h.OnChange)
	models.OnRemove(mgmt.Ctx, modelOnRemoveName, h.OnRemove)
	return nil
}

func (h *handler) OnChange(_ string, model *mlv1.Model) (*mlv1.Model, error) {
	if model == nil || model.DeletionTimestamp != nil {
		return model, nil
	}

	logrus.Infof("model %s/%s changed", model.Namespace, model.Name)

	modelCopy := model.DeepCopy()

	// create root directory for model
	b, err := h.rm.NewBackendFromRegistry(h.ctx, model.Spec.Registry)
	if err != nil {
		return h.updateModelStatus(modelCopy, model, fmt.Errorf(registry.ErrCreateBackendClient, err))
	}
	modelRootDir := path.Join(mlv1.ModelResourceName, model.Namespace, model.Name)
	if err := b.CreateDirectory(h.ctx, modelRootDir); err != nil {
		return h.updateModelStatus(modelCopy, model, fmt.Errorf(registry.ErrCreateDirectory, modelRootDir, err))
	}

	modelCopy.Status.RootPath = modelRootDir
	return h.updateModelStatus(modelCopy, model, nil)
}

func (h *handler) OnRemove(_ string, model *mlv1.Model) (*mlv1.Model, error) {
	if model == nil || model.Status.RootPath == "" || model.DeletionTimestamp == nil {
		return nil, nil
	}

	logrus.Infof("model %s/%s deleted", model.Namespace, model.Name)

	// delete root directory of the model
	b, err := h.rm.NewBackendFromRegistry(h.ctx, model.Spec.Registry)
	if err != nil {
		return nil, fmt.Errorf(registry.ErrCreateBackendClient, err)
	}
	if err := b.Delete(h.ctx, model.Status.RootPath); err != nil {
		return nil, fmt.Errorf(registry.ErrDeleteFile, model.Status.RootPath, err)
	}

	return model, nil
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
		return nil, fmt.Errorf("update model status failed: %w", updateErr)
	}
	return updatedModel, err
}
