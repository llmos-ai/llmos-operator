package raycluster

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	clusterctl "github.com/llmos-ai/llmos-operator/pkg/controller/master/raycluster"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

const (
	strTrue = "true"
)

type mutator struct {
	admission.DefaultMutator
	releaseName string
}

var _ admission.Mutator = &mutator{}

func NewMutator(mgmt *config.Management) admission.Mutator {
	return &mutator{
		releaseName: mgmt.ReleaseName,
	}
}

func (m *mutator) Create(_ *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	cluster := newObj.(*rayv1.RayCluster)
	logrus.Debugf("raycluster %s is created", cluster.Name)

	// skip updating the object if it is originated from RayService
	if isOwnedByRayService(cluster) {
		logrus.Debugln("cluster is originated from rayService, skip mutating")
		return nil, nil
	}

	patchOps := make([]admission.PatchOp, 0)

	var gcsEnabled = false

	// patch fault-tolerant annotation if GCS is enabled
	if val, ok := cluster.Annotations[constant.AnnotationRayFTEnabledKey]; ok && val == strTrue {
		gcsEnabled = true
	}
	// patch head group spec
	patchOps = append(patchOps, patchHeadGroupSpec(cluster, gcsEnabled))

	// patch worker group specs
	if len(cluster.Spec.WorkerGroupSpecs) > 0 {
		patchOps = append(patchOps, patchWorkerGroupSpecs(cluster))
	}

	return patchOps, nil
}

func (m *mutator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) (admission.Patch, error) {
	oldCluster := oldObj.(*rayv1.RayCluster)
	cluster := newObj.(*rayv1.RayCluster)
	logrus.Debugf("raycluster %s is updated", cluster.Name)

	// skip updating the object if it is originated from RayService
	if isOwnedByRayService(cluster) {
		logrus.Debugln("cluster is originated from rayService, skip mutating")
		return nil, nil
	}

	val, ok := cluster.Annotations[constant.AnnotationRayFTEnabledKey]
	// turn off GCS config is not allowed
	if valOld, okOld := oldCluster.Annotations[constant.AnnotationRayFTEnabledKey]; okOld && valOld == strTrue {
		if ok && val != strTrue {
			return nil, fmt.Errorf("GCS is not allowed to be disabled once is enabled")
		}
	}

	patchOps := make([]admission.PatchOp, 0)
	var gcsEnabled = false

	// patch fault-tolerant annotation if GCS is enabled
	if ok && val == strTrue {
		gcsEnabled = true
	}

	patchOps = append(patchOps, patchHeadGroupSpec(cluster, gcsEnabled))
	if len(cluster.Spec.WorkerGroupSpecs) > 0 {
		patchOps = append(patchOps, patchWorkerGroupSpecs(cluster))
	}

	return patchOps, nil
}

func patchHeadGroupSpec(cluster *rayv1.RayCluster, gcsEnabled bool) admission.PatchOp {
	headGroupSpec := cluster.Spec.HeadGroupSpec
	headGroupSpec.RayStartParams = patchHeadGroupStartParams(cluster, gcsEnabled)
	headGroupSpec.Template.Spec.Containers[0].Ports = patchHeadGroupPorts(cluster)
	headGroupSpec.Template.Spec.Containers[0].Resources.Limits = patchResourceLimits(headGroupSpec.Template)
	headGroupSpec.Template.Spec.Containers[0].Lifecycle = patchContainerLifecycle(headGroupSpec.Template)
	if gcsEnabled {
		headGroupSpec.Template.Spec.Containers[0].Env = getHeadGroupEnvPath(cluster)
	}

	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/headGroupSpec",
		Value: headGroupSpec,
	}
}

func patchHeadGroupStartParams(cluster *rayv1.RayCluster, gcsEnabled bool) map[string]string {
	rayStartParams := cluster.Spec.HeadGroupSpec.RayStartParams
	if rayStartParams == nil {
		rayStartParams = map[string]string{
			"dashboard-host": "0.0.0.0",
			"block":          strTrue,
		}
	} else {
		rayStartParams["block"] = strTrue
		rayStartParams["dashboard-host"] = "0.0.0.0"
	}

	if gcsEnabled {
		rayStartParams["redis-password"] = "$REDIS_PASSWORD"
	}

	return rayStartParams
}

func patchHeadGroupPorts(cluster *rayv1.RayCluster) []corev1.ContainerPort {
	ports := cluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Ports
	if len(ports) > 0 {
		return cluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Ports
	}
	return []corev1.ContainerPort{
		{
			Name:          "gcs",
			ContainerPort: 6379,
		},
		{
			Name:          "client",
			ContainerPort: 10001,
		},
		{
			Name:          "dashboard",
			ContainerPort: 8265,
		},
		{
			Name:          "serve",
			ContainerPort: 8000,
		},
	}
}

// patchResourceLimits patch the resource limits by using the requests if the limits are not set
func patchResourceLimits(spec corev1.PodTemplateSpec) corev1.ResourceList {
	requests := spec.Spec.Containers[0].Resources.Requests
	limits := spec.Spec.Containers[0].Resources.Limits
	if !limits.Cpu().IsZero() && !limits.Memory().IsZero() {
		return limits
	}

	newLimits := corev1.ResourceList{}
	if !limits.Cpu().IsZero() {
		newLimits[corev1.ResourceCPU] = *resource.NewQuantity(limits.Cpu().Value(), limits.Cpu().Format)
	} else if limits.Cpu().IsZero() && !requests.Cpu().IsZero() {
		newLimits[corev1.ResourceCPU] = *resource.NewQuantity(requests.Cpu().Value(), requests.Cpu().Format)
	}

	if !limits.Memory().IsZero() {
		newLimits[corev1.ResourceMemory] = *resource.NewQuantity(limits.Memory().Value(), limits.Memory().Format)
	} else if limits.Memory().IsZero() && !requests.Memory().IsZero() {
		newLimits[corev1.ResourceMemory] = *resource.NewQuantity(requests.Memory().Value(), requests.Memory().Format)
	}

	// copy the rest of the resources
	for k, v := range requests {
		if k != corev1.ResourceCPU && k != corev1.ResourceMemory {
			newLimits[k] = v
		}
	}

	return newLimits
}

func getHeadGroupEnvPath(cluster *rayv1.RayCluster) []corev1.EnvVar {
	headGroupEnv := cluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env
	redisEnvConfig := clusterctl.GetHeadNodeRedisEnvConfig(cluster.Namespace)
	if len(headGroupEnv) == 0 {
		headGroupEnv = redisEnvConfig
	} else {
		for _, redisEnv := range redisEnvConfig {
			found := false
			for _, hEnv := range headGroupEnv {
				if redisEnv.Name == hEnv.Name {
					found = true
					continue
				}
			}
			if !found {
				headGroupEnv = append(headGroupEnv, redisEnv)
			}
		}
	}

	return headGroupEnv
}

func patchWorkerGroupSpecs(cluster *rayv1.RayCluster) admission.PatchOp {
	workerGroupSpecs := patchWorkerGroupStartParams(cluster)
	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/workerGroupSpecs",
		Value: workerGroupSpecs,
	}
}

func patchWorkerGroupStartParams(cluster *rayv1.RayCluster) []rayv1.WorkerGroupSpec {
	workerGroupSpecs := cluster.Spec.WorkerGroupSpecs
	for i, workerGroupSpec := range workerGroupSpecs {
		rayStartParams := workerGroupSpec.RayStartParams
		if rayStartParams == nil {
			rayStartParams = map[string]string{
				"block": "true",
			}
		} else {
			rayStartParams["block"] = "true"
		}
		workerGroupSpecs[i].RayStartParams = rayStartParams
		workerGroupSpecs[i].Template.Spec.Containers[0].Resources.Limits = patchResourceLimits(workerGroupSpec.Template)
		workerGroupSpecs[i].Template.Spec.Containers[0].Env = pathWorkerNodeEnvConf(workerGroupSpec)
		workerGroupSpecs[i].Template.Spec.Containers[0].Lifecycle = patchContainerLifecycle(workerGroupSpec.Template)
	}
	return workerGroupSpecs
}
func pathWorkerNodeEnvConf(spec rayv1.WorkerGroupSpec) []corev1.EnvVar {
	env := spec.Template.Spec.Containers[0].Env
	if len(env) == 0 {
		env = []corev1.EnvVar{
			{
				Name:  "RAY_gcs_rpc_server_reconnect_timeout_s",
				Value: "300",
			},
		}
	} else {
		found := false
		for _, e := range env {
			if e.Name == "RAY_gcs_rpc_server_reconnect_timeout_s" {
				found = true
				break
			}
		}
		if !found {
			env = append(env, corev1.EnvVar{
				Name:  "RAY_gcs_rpc_server_reconnect_timeout_s",
				Value: "300",
			})
		}
	}

	return env
}

func patchContainerLifecycle(spec corev1.PodTemplateSpec) *corev1.Lifecycle {
	lifecycle := spec.Spec.Containers[0].Lifecycle
	if lifecycle == nil {
		lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/sh", "-c", "ray stop"},
				},
			},
		}
	}
	return lifecycle
}

func isOwnedByRayService(cluster *rayv1.RayCluster) bool {
	if len(cluster.OwnerReferences) == 0 {
		return false
	}
	for _, owner := range cluster.OwnerReferences {
		if owner.Kind == constant.RayServiceKind {
			return true
		}
	}
	return false
}

func (m *mutator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"rayclusters"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   rayv1.SchemeGroupVersion.Group,
		APIVersion: rayv1.SchemeGroupVersion.Version,
		ObjectType: &rayv1.RayCluster{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
