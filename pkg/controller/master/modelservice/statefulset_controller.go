package modelservice

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
	statefulSetCache  ctlappsv1.StatefulSetCache
	modelService      ctlmlv1.ModelServiceClient
	modelServiceCache ctlmlv1.ModelServiceCache
	podCache          ctlcorev1.PodCache
}

func (h *statefulSetHandler) OnChange(_ string, statefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	if statefulSet == nil || statefulSet.DeletionTimestamp != nil || statefulSet.Labels == nil {
		return nil, nil
	}

	msName := statefulSet.Labels[constant.LabelModelServiceName]
	if msName == "" {
		return nil, nil
	}

	modelService, err := h.modelServiceCache.Get(statefulSet.Namespace, msName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Debugf("modelService %s not found for statefulSet %s", msName, statefulSet.Name)
		return statefulSet, nil
	} else if err != nil {
		return statefulSet, err
	}

	return h.updateModelServiceStatus(statefulSet, modelService)
}

func (h *statefulSetHandler) updateModelServiceStatus(ss *appsv1.StatefulSet, modelService *mlv1.ModelService) (
	*appsv1.StatefulSet, error) {
	podName := getDefaultPodName(ss.Name)
	pod, err := h.podCache.Get(ss.Namespace, podName)
	if err != nil {
		if errors.IsNotFound(err) && *ss.Spec.Replicas == 0 {
			return ss, nil
		}
		return ss, err
	}

	status := constructModelStatus(ss, pod)
	if !reflect.DeepEqual(modelService.Status, status) {
		msCpy := modelService.DeepCopy()
		msCpy.Status = status
		if _, err = h.modelService.UpdateStatus(msCpy); err != nil {
			return ss, err
		}
	}

	return ss, nil
}

// syncServiceStatusByPod reconciles the owner statefulSet by pod event
func (h *statefulSetHandler) syncServiceStatusByPod(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if pod, ok := obj.(*corev1.Pod); ok {
		for _, ref := range pod.GetOwnerReferences() {
			if ref.Kind == "StatefulSet" && strings.Contains(ref.Name, msPrefix) {
				logrus.Debugf("reconcile modelService: %s/%s", pod.Namespace, ref.Name)
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
