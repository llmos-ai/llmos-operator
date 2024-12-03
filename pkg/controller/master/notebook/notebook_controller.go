package notebook

import (
	"context"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

// Note: the notebook controller is referred to the kubeflow's notebook controller
// https://github.com/kubeflow/kubeflow/tree/master/components/notebook-controller

const (
	DefaultContainerPort = int32(8888)
	DefaultServingPort   = 80
)

type Handler struct {
	statefulSets     ctlappsv1.StatefulSetClient
	statefulSetCache ctlappsv1.StatefulSetCache
	services         ctlcorev1.ServiceClient
	serviceCache     ctlcorev1.ServiceCache
	pvcHandler       *utils.PVCHandler
}

const (
	notebookOnChange              = "notebook.onChange"
	notebookOnDelete              = "notebook.onDelete"
	notebookStatefulSetOnChange   = "notebook.statefulSetOnChange"
	notebookControllerWatchSsPods = "notebook.statefulSetWatchPods"
)

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	notebooks := mgmt.LLMFactory.Ml().V1().Notebook()
	ss := mgmt.AppsFactory.Apps().V1().StatefulSet()
	services := mgmt.CoreFactory.Core().V1().Service()
	pods := mgmt.CoreFactory.Core().V1().Pod()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()

	h := Handler{
		statefulSets:     ss,
		statefulSetCache: ss.Cache(),
		services:         services,
		serviceCache:     services.Cache(),
		pvcHandler:       utils.NewPVCHandler(pvcs),
	}
	notebooks.OnChange(ctx, notebookOnChange, h.OnChanged)
	notebooks.OnRemove(ctx, notebookOnDelete, h.OnDelete)

	ssHandler := &statefulSetHandler{
		statefulSetCache: ss.Cache(),
		notebooks:        notebooks,
		notebookCache:    notebooks.Cache(),
		podCache:         pods.Cache(),
	}

	ss.OnChange(ctx, notebookStatefulSetOnChange, ssHandler.OnChange)
	relatedresource.Watch(ctx, notebookControllerWatchSsPods, h.syncNotebookStatusByPod, ss, pods)
	return nil
}

func (h *Handler) OnChanged(_ string, notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	if notebook == nil || notebook.DeletionTimestamp != nil {
		return nil, nil
	}

	// reconcile notebook statefulSet
	if _, err := h.reconcileStatefulSet(notebook); err != nil {
		return notebook, err
	}

	// reconcile notebook service
	if err := h.reconcileNotebookService(notebook); err != nil {
		return notebook, err
	}

	// TODO: only handle pvcs clean up on delete
	// NOTE: this is a workaround to clean up pvcs on delete while update reconcile is called simultaneously
	strVolumes := notebook.Annotations[constant.AnnotationOnDeleteVolumes]
	if strVolumes != "" {
		volumes := strings.Split(strVolumes, ",")
		if err := h.pvcHandler.DeletePVCs(notebook.Namespace, volumes); err != nil {
			return notebook, err
		}
	}

	return notebook, nil
}

func (h *Handler) reconcileStatefulSet(notebook *mlv1.Notebook) (*v1.StatefulSet, error) {
	ss := getNoteBookStatefulSet(notebook)
	foundSs, err := h.statefulSetCache.Get(ss.Namespace, ss.Name)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("creating new statefulset for notebook %s/%s", notebook.Namespace, notebook.Name)
		return h.statefulSets.Create(ss)
	} else if err != nil {
		return nil, err
	}

	if reconcilehelper.CopyStatefulSetFields(ss, foundSs) {
		logrus.Debugf("updating notebook statefulset %s/%s", notebook.Namespace, notebook.Name)
		toUpdate := foundSs.DeepCopy()
		return h.statefulSets.Update(toUpdate)
	}

	return foundSs, nil
}

func (h *Handler) reconcileNotebookService(notebook *mlv1.Notebook) error {
	svc := getNotebookService(notebook)
	foundSvc, err := h.serviceCache.Get(svc.Namespace, svc.Name)
	createNew := false
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("creating new notebook service %s/%s", notebook.Namespace, svc.Name)
		if _, err = h.services.Create(svc); err != nil {
			return err
		}
		createNew = true
	} else if err != nil {
		return err
	}

	if !createNew && reconcilehelper.CopyServiceFields(svc, foundSvc) {
		toUpdate := foundSvc.DeepCopy()
		if _, err = h.services.Update(toUpdate); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) OnDelete(_ string, notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	if notebook == nil || notebook.DeletionTimestamp == nil {
		return nil, nil
	}

	// Clean up on-delete pvcs if specified
	strVolumes := notebook.Annotations[constant.AnnotationOnDeleteVolumes]
	if strVolumes != "" {
		volumes := strings.Split(strVolumes, ",")
		logrus.Debugf("cleaning up pvcs %s on notebook %s/%s", volumes, notebook.Namespace, notebook.Name)
		if err := h.pvcHandler.DeletePVCs(notebook.Namespace, volumes); err != nil {
			return notebook, err
		}
	}
	return notebook, nil
}
