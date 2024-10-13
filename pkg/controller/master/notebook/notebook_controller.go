package notebook

import (
	"context"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
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
}

const (
	notebookControllerOnChange    = "notebook.onChange"
	notebookStatefulSetOnChange   = "notebook.statefulSetOnChange"
	notebookControllerWatchSsPods = "notebook.statefulSetWatchPods"
)

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	notebooks := mgmt.LLMFactory.Ml().V1().Notebook()
	ss := mgmt.AppsFactory.Apps().V1().StatefulSet()
	services := mgmt.CoreFactory.Core().V1().Service()
	pods := mgmt.CoreFactory.Core().V1().Pod()

	h := Handler{
		statefulSets:     ss,
		statefulSetCache: ss.Cache(),
		services:         services,
		serviceCache:     services.Cache(),
	}
	notebooks.OnChange(ctx, notebookControllerOnChange, h.OnChanged)

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
