package modelservice

import (
	"context"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

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
