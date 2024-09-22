package raycluster

import (
	"context"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlkuberayv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ray.io/v1"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	kubeRayControllerSyncCluster = "rayCluster.syncCluster"
	kubeRayControllerOnDelete    = "rayCluster.onDelete"
	kubeRayControllerCreatePVC   = "rayCluster.createPVCFromAnnotation"
)

// handler reconcile the user's clusterRole and clusterRoleBinding
type handler struct {
	releaseName  string
	rayClusters  ctlkuberayv1.RayClusterController
	services     ctlcorev1.ServiceClient
	serviceCache ctlcorev1.ServiceCache
	secrets      ctlcorev1.SecretClient
	secretsCache ctlcorev1.SecretCache
	pvcs         ctlcorev1.PersistentVolumeClaimClient
	pvcCache     ctlcorev1.PersistentVolumeClaimCache
	configmap    ctlcorev1.ConfigMapClient
}

func Register(ctx context.Context, mgmt *config.Management, opts config.Options) error {
	clusters := mgmt.KubeRayFactory.Ray().V1().RayCluster()
	services := mgmt.CoreFactory.Core().V1().Service()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	configmaps := mgmt.CoreFactory.Core().V1().ConfigMap()

	h := &handler{
		releaseName:  opts.ReleaseName,
		rayClusters:  clusters,
		services:     services,
		serviceCache: services.Cache(),
		secrets:      secrets,
		secretsCache: secrets.Cache(),
		pvcs:         pvcs,
		pvcCache:     pvcs.Cache(),
		configmap:    configmaps,
	}

	clusters.OnChange(ctx, kubeRayControllerSyncCluster, h.OnChanged)
	clusters.OnChange(ctx, kubeRayControllerCreatePVC, h.createPVCFromAnnotation)
	clusters.OnRemove(ctx, kubeRayControllerOnDelete, h.OnDelete)
	return nil
}

func (h *handler) OnChanged(_ string, cluster *rayv1.RayCluster) (*rayv1.RayCluster, error) {
	if cluster == nil || cluster.DeletionTimestamp != nil {
		return cluster, nil
	}

	// sync GCS Redis secret to the cluster namespace
	return nil, h.syncGCSRedisSecretToNamespace(h.releaseName, cluster)
}

func (h *handler) OnDelete(_ string, cluster *rayv1.RayCluster) (*rayv1.RayCluster, error) {
	if cluster == nil || cluster.DeletionTimestamp == nil {
		return nil, nil
	}

	logrus.Debugf("on delete ray cluster: %s/%s", cluster.Name, cluster.Namespace)
	// wait for the redis clean up job finished first since it will mount all volumes and configmaps to it
	for _, f := range cluster.Finalizers {
		if f == constant.RayRedisCleanUpFinalizer {
			h.rayClusters.Enqueue(cluster.Namespace, cluster.Name)
			return cluster, nil
		}
	}

	return nil, nil
}
