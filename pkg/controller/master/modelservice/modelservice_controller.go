package modelservice

import (
	"context"
	"fmt"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

const (
	modelServiceOnChange  = "modelService.onChange"
	modelServiceOnDelete  = "modelService.onDelete"
	msStatefulSetOnChange = "modelService.statefulSetOnChange"
	msSyncStatusByPod     = "modelService.syncStatusByPod"
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
	pvcHandler        *utils.PVCHandler
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	modelService := mgmt.LLMFactory.Ml().V1().ModelService()
	statefulSet := mgmt.AppsFactory.Apps().V1().StatefulSet()
	deployment := mgmt.AppsFactory.Apps().V1().Deployment()
	service := mgmt.CoreFactory.Core().V1().Service()
	pod := mgmt.CoreFactory.Core().V1().Pod()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()

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
		pvcHandler:        utils.NewPVCHandler(pvcs),
	}
	modelService.OnChange(ctx, modelServiceOnChange, h.OnChange)
	modelService.OnRemove(ctx, modelServiceOnDelete, h.OnDelete)

	ssHandler := &statefulSetHandler{
		statefulSetCache:  statefulSet.Cache(),
		modelService:      modelService,
		modelServiceCache: modelService.Cache(),
		pods:              pod,
		podCache:          pod.Cache(),
	}
	statefulSet.OnChange(ctx, msStatefulSetOnChange, ssHandler.OnChange)
	relatedresource.Watch(ctx, msSyncStatusByPod, ssHandler.syncServiceStatusByPod, statefulSet, pod)
	return nil
}

func (h *handler) OnChange(_ string, ms *mlv1.ModelService) (*mlv1.ModelService, error) {
	if ms == nil || ms.DeletionTimestamp != nil {
		return nil, nil
	}
	var err error
	// reconcile model service statefulSet
	if _, err = h.reconcileModelStatefulSet(ms); err != nil {
		return ms, err
	}

	// reconcile model service svc
	if _, err = h.reconcileModelService(ms); err != nil {
		return ms, err
	}

	// TODO: only handle pvcs clean up on delete
	// NOTE: this is a workaround to clean up pvcs on delete while update reconcile is called simultaneously
	strVolumes := ms.Annotations[constant.AnnotationOnDeleteVolumes]
	if strVolumes != "" {
		volumes := strings.Split(strVolumes, ",")
		logrus.Debugf("cleaning up pvcs %s on modelservice %s/%s", volumes, ms.Namespace, ms.Name)
		if err := h.pvcHandler.DeletePVCs(ms.Namespace, volumes); err != nil {
			return ms, err
		}
	}

	return ms, nil
}

// reconcileModelStatefulSet reconciles the statefulSet of the model
func (h *handler) reconcileModelStatefulSet(ms *mlv1.ModelService) (*appsv1.StatefulSet, error) {
	ss := constructModelStatefulSet(ms)
	foundSs, err := h.StatefulSetCache.Get(ss.Namespace, ss.Name)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("creating new statefulSet of model %s", ms.Name)
		return h.StatefulSets.Create(ss)
	} else if err != nil {
		return nil, err
	}

	// Update the object and write the result back if there are any changes
	toUpdate, toRedeploy := reconcilehelper.CopyStatefulSetFields(ss, foundSs)
	if toUpdate {
		logrus.Debugf("updating modelSerive statefulSet %s/%s", foundSs.Namespace, foundSs.Name)
		ssCopy := foundSs.DeepCopy()
		if ss, err = h.StatefulSets.Update(ssCopy); err != nil {
			return ss, err
		}
	}

	if toRedeploy {
		if err = h.deleteStatefulSetPods(foundSs); err != nil {
			return foundSs, err
		}
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

func (h *handler) OnDelete(_ string, ms *mlv1.ModelService) (*mlv1.ModelService, error) {
	if ms == nil || ms.DeletionTimestamp == nil {
		return nil, nil
	}

	// Clean up on-delete pvcs if specified
	strVolumes := ms.Annotations[constant.AnnotationOnDeleteVolumes]
	if strVolumes != "" {
		volumes := strings.Split(strVolumes, ",")
		logrus.Debugf("cleaning up pvcs %s on modelservice %s/%s", volumes, ms.Namespace, ms.Name)
		if err := h.pvcHandler.DeletePVCs(ms.Namespace, volumes); err != nil {
			return ms, err
		}
	}
	return ms, nil
}

func (h *handler) deleteStatefulSetPods(sts *appsv1.StatefulSet) error {
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return fmt.Errorf("failed to convert LabelSelector: %v", err)
	}

	// List the pods matching the label selector
	pods, err := h.PodCache.List(sts.Namespace, selector)
	if err != nil {
		return fmt.Errorf("failed to list modelService pods: %w", err)
	}

	// Delete each pod
	for _, pod := range pods {
		err = h.Pods.Delete(pod.Namespace, pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete modelService pod %s: %w", pod.Name, err)
		}
	}

	return nil
}
