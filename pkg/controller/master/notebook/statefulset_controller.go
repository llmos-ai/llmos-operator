package notebook

import (
	"reflect"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
)

type statefulSetHandler struct {
	statefulSetCache ctlappsv1.StatefulSetCache
	notebooks        ctlmlv1.NotebookClient
	notebookCache    ctlmlv1.NotebookCache
	podCache         ctlcorev1.PodCache
}

func (h *statefulSetHandler) OnChange(_ string, statefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	if statefulSet == nil || statefulSet.DeletionTimestamp != nil || statefulSet.Labels == nil {
		return nil, nil
	}

	nbName := statefulSet.Labels[constant.LabelNotebookName]
	if nbName == "" {
		return nil, nil
	}

	notebook, err := h.notebookCache.Get(statefulSet.Namespace, nbName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Debugf("notebook %s not found by statefulSet %s", nbName, statefulSet.Name)
		return statefulSet, nil
	} else if err != nil {
		return statefulSet, err
	}

	if err = h.updateNotebookStatus(statefulSet, notebook); err != nil {
		return statefulSet, err
	}

	return statefulSet, nil
}

func (h *statefulSetHandler) updateNotebookStatus(ss *appsv1.StatefulSet, notebook *mlv1.Notebook) error {
	podName := getNotebookPodName(ss.Name)
	pod, err := h.podCache.Get(ss.Namespace, podName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("notebook pod %s not found, skipp updating and waiting for reconcile", podName)
		return nil
	} else if err != nil {
		return err
	}

	status := getNotebookStatus(ss, pod)
	if !reflect.DeepEqual(notebook.Status, status) {
		nbCpy := notebook.DeepCopy()
		nbCpy.Status = status
		if _, err = h.notebooks.UpdateStatus(nbCpy); err != nil {
			return err
		}
	}

	return nil
}

// syncNotebookStatusByPod reconciles the owner statefulSet by watching pods
func (h *Handler) syncNotebookStatusByPod(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if pod, ok := obj.(*corev1.Pod); ok {
		for _, ref := range pod.GetOwnerReferences() {
			if ref.Kind == "StatefulSet" && strings.Contains(ref.Name, NamePrefix) {
				logrus.Debugf("reconcile notebook by: %s/%s", pod.Namespace, ref.Name)
				return []relatedresource.Key{
					{
						Name:      ref.Name,
						Namespace: pod.Namespace,
					},
				}, nil
			}
		}
	}
	return nil, nil
}
