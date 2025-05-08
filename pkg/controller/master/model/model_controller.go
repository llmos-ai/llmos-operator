package model

import (
	"context"
	"fmt"
	"path"
	"reflect"

	snapshotstoragev1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/common/localcache"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	modelOnChangeName          = "model.OnChange"
	modelOnRemoveName          = "model.OnRemove"
	jobOnChangeName            = "job.model.OnChange"
	volumeSnapshotOnChangeName = "volumesnapshot.model.OnChange"
)

type handler struct {
	ctx context.Context

	modelClient ctlmlv1.ModelClient
	modelCache  ctlmlv1.ModelCache

	rm           *registry.Manager
	cacheHandler *localcache.Handler
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	models := mgmt.LLMFactory.Ml().V1().Model()
	jobs := mgmt.BatchFactory.Batch().V1().Job()
	snapshots := mgmt.SnapshotFactory.Snapshot().V1().VolumeSnapshot()

	h := handler{
		ctx: mgmt.Ctx,

		modelClient: models,
		modelCache:  models.Cache(),
	}
	h.cacheHandler = localcache.NewHandler(mgmt)

	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	models.OnChange(mgmt.Ctx, modelOnChangeName, h.OnChange)
	models.OnRemove(mgmt.Ctx, modelOnRemoveName, h.OnRemove)
	jobs.OnChange(mgmt.Ctx, jobOnChangeName, h.jobOnChange)
	snapshots.OnChange(mgmt.Ctx, volumeSnapshotOnChangeName, h.volumeSnapshotOnChange)

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
	if err = b.CreateDirectory(h.ctx, modelRootDir); err != nil {
		return h.updateModelStatus(modelCopy, model, fmt.Errorf(registry.ErrCreateDirectory, modelRootDir, err))
	}

	modelCopy.Status.RootPath = modelRootDir

	size, err := b.GetSize(h.ctx, modelRootDir)
	if err != nil {
		return h.updateModelStatus(modelCopy, model, fmt.Errorf(registry.ErrGetSize, modelRootDir, err))
	}

	if size > 0 {
		// Because the OnChange function will update the status of the model,
		// we don't need to update the status of the model in the cache.
		if err := h.cacheHandler.ReconcileCache(localcache.NewModelCacheAdapter(modelCopy, nil), size); err != nil {
			return h.updateModelStatus(modelCopy, model, fmt.Errorf("reconcile cache failed: %w", err))
		}
	} else {
		logrus.Infof("there is no content of model %s/%s, skip reconcile cache", model.Namespace, model.Name)
	}

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

	updatedModel, updateErr := h.modelClient.Update(modelCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update model status failed: %w", updateErr)
	}
	return updatedModel, err
}

func (h *handler) jobOnChange(_ string, job *batchv1.Job) (*batchv1.Job, error) {
	return job, h.cacheHandler.ReconcileJob(job)
}

func (h *handler) volumeSnapshotOnChange(
	_ string,
	snapshot *snapshotstoragev1.VolumeSnapshot,
) (*snapshotstoragev1.VolumeSnapshot, error) {
	return snapshot, h.cacheHandler.ReconcileVolumeSnapshot(snapshot)
}
