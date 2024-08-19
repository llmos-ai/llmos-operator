package modelservice

import (
	"context"
	"reflect"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

const (
	modelServiceOnChange = "modelService.onChange"
	modelServiceWatchSs  = "modelService.watchStatefulSets"

	msKindName     = "ModelService"
	typeName       = "model-service"
	vllmEngineName = "vllm"
)

type handler struct {
	ModelServices     ctlmlv1.ModelServiceClient
	ModelServiceCache ctlmlv1.ModelServiceCache
	StatefulSets      ctlappsv1.StatefulSetClient
	StatefulSetCache  ctlappsv1.StatefulSetCache
	Deployments       ctlappsv1.DeploymentClient
	DeploymentCache   ctlappsv1.DeploymentCache
	Services          ctlcorev1.ServiceClient
	ServiceCache      ctlcorev1.ServiceCache
	Pods              ctlcorev1.PodClient
	PodCache          ctlcorev1.PodCache
}

func Register(ctx context.Context, mgmt *config.Management) error {
	modelService := mgmt.LLMFactory.Ml().V1().ModelService()
	statefulSet := mgmt.AppsFactory.Apps().V1().StatefulSet()
	deployment := mgmt.AppsFactory.Apps().V1().Deployment()
	service := mgmt.CoreFactory.Core().V1().Service()
	pod := mgmt.CoreFactory.Core().V1().Pod()

	h := &handler{
		ModelServices:     modelService,
		ModelServiceCache: modelService.Cache(),
		StatefulSets:      statefulSet,
		StatefulSetCache:  statefulSet.Cache(),
		Deployments:       deployment,
		DeploymentCache:   deployment.Cache(),
		Services:          service,
		ServiceCache:      service.Cache(),
		Pods:              pod,
		PodCache:          pod.Cache(),
	}

	modelService.OnChange(ctx, modelServiceOnChange, h.OnChange)
	relatedresource.Watch(ctx, modelServiceWatchSs, h.ReconcileModelServiceByStatefulSet, modelService, statefulSet)
	return nil
}

func (h *handler) OnChange(_ string, ms *mlv1.ModelService) (*mlv1.ModelService, error) {
	if ms == nil || ms.DeletionTimestamp != nil {
		return nil, nil
	}

	// reconcile model service statefulSet
	ss, err := h.reconcileModelStatefulSet(ms)
	if err != nil {
		return ms, err
	}

	// reconcile model service svc
	svc, err := h.reconcileModelService(ms)
	if err != nil {
		return ms, err
	}

	if err = h.createModelServiceGUI(ms, svc); err != nil {
		return ms, err
	}

	return ms, h.updateModelServiceStatus(ms, ss)
}

// reconcileModelStatefulSet reconciles the statefulSet of the model
func (h *handler) reconcileModelStatefulSet(ms *mlv1.ModelService) (*v1.StatefulSet, error) {
	ss := constructModelStatefulSet(ms)
	foundSs, err := h.StatefulSetCache.Get(ss.Namespace, ss.Name)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("creating new statefulSet of model %s", ms.Name)
		return h.StatefulSets.Create(ss)
	} else if err != nil {
		return nil, err
	}

	// Update the object and write the result back if there are any changes
	if reconcilehelper.CopyStatefulSetFields(ss, foundSs) {
		logrus.Debugf("updating model serive statefulSet %s/%s", foundSs.Namespace, foundSs.Name)
		toUpdate := foundSs.DeepCopy()
		return h.StatefulSets.Update(toUpdate)
	}

	return foundSs, nil
}

// reconcileModelSvc reconciles the service of the model
func (h *handler) reconcileModelService(ms *mlv1.ModelService) (*corev1.Service, error) {
	svc := constructModelSvc(ms)
	foundSvc, err := h.ServiceCache.Get(svc.Namespace, svc.Name)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("creating new service of model %s", ms.Name)
		return h.Services.Create(svc)
	} else if err != nil {
		return nil, err
	}

	if reconcilehelper.CopyServiceFields(svc, foundSvc) {
		logrus.Debugf("updating service of model %s", svc.Name)
		toUpdate := foundSvc.DeepCopy()
		return h.Services.Update(toUpdate)
	}

	return foundSvc, nil
}

func (h *handler) updateModelServiceStatus(ms *mlv1.ModelService, ss *v1.StatefulSet) error {
	if ss == nil {
		logrus.Debugf("statefulSet of model %s not found, skip updating status", ms.Name)
		return nil
	}

	podName := getPodName(ss.Name)
	pod, err := h.PodCache.Get(ss.Namespace, podName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Debugf("model serivce pod %s not found, skip updating status", podName)
		return nil
	} else if err != nil {
		return err
	}

	status := constructModelStatus(ss, pod)
	logrus.Debugf("updating status of model %s, status: %v", ms.Name, status)
	if !reflect.DeepEqual(ms.Status, status) {
		msCpy := ms.DeepCopy()
		msCpy.Status = status
		if _, err = h.ModelServices.UpdateStatus(msCpy); err != nil {
			return err
		}
	}

	return nil
}

// ReconcileModelServiceByStatefulSet reconciles the owner modelService by statefulSet
func (h *handler) ReconcileModelServiceByStatefulSet(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if ss, ok := obj.(*v1.StatefulSet); ok {
		for k, v := range ss.GetLabels() {
			if strings.Contains(k, constant.LabelModelServiceName) {
				logrus.Debugf("reconcile model service: %s/%s", ss.Namespace, v)
				return []relatedresource.Key{
					{
						Name:      v,
						Namespace: ss.Namespace,
					},
				}, nil
			}
		}
	}

	return nil, nil
}
