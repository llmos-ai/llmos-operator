package modelservice

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

const (
	msPrefix       = "modelservice"
	typeName       = "model-service"
	vllmEngineName = "vllm"
)

func constructModelStatefulSet(ms *mlv1.ModelService) *v1.StatefulSet {
	selector := GetModelServiceSelector(ms)
	replicas := ms.Spec.Replicas
	if metav1.HasAnnotation(ms.ObjectMeta, constant.AnnotationResourceStopped) {
		replicas = 0
	}
	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedMSName(ms.Name, ""),
			Namespace: ms.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, ms.GroupVersionKind()),
			},
			Labels: map[string]string{
				constant.LabelLLMOSMLType:             typeName,
				constant.LabelModelServiceName:        ms.Name,
				constant.LabelModelServiceServeEngine: vllmEngineName,
			},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: ptr.To(replicas),
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      selector.MatchLabels,
					Annotations: map[string]string{},
				},
				Spec: *ms.Spec.Template.Spec.DeepCopy(),
			},
			UpdateStrategy:       *ms.Spec.UpdateStrategy.DeepCopy(),
			VolumeClaimTemplates: ms.Spec.VolumeClaimTemplates,
		},
	}

	container := &ss.Spec.Template.Spec.Containers[0]
	container.Args = constructVllmArgs(ms, ss.Spec.Template.Spec.Containers[0].Args)
	containerPort := container.Ports[0].ContainerPort

	if container.LivenessProbe == nil {
		ss.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt32(containerPort),
				},
			},
			PeriodSeconds:    60,
			FailureThreshold: 3,
		}
	}

	if container.StartupProbe == nil {
		ss.Spec.Template.Spec.Containers[0].StartupProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt32(containerPort),
				},
			},
			InitialDelaySeconds: 30,
			FailureThreshold:    720,
			PeriodSeconds:       10,
		}
	}

	// Copy all the selector to the pod
	ls := &ss.Spec.Template.ObjectMeta.Labels
	for k, v := range ms.ObjectMeta.Labels {
		(*ls)[k] = v
	}

	// Copy all the annotations to the pod
	annos := &ss.Spec.Template.ObjectMeta.Annotations
	for k, v := range ms.ObjectMeta.Annotations {
		if !strings.Contains(k, "kubectl") && !strings.Contains(k, "notebook") {
			(*annos)[k] = v
		}
	}

	return ss
}
func constructModelSvc(ms *mlv1.ModelService) *corev1.Service {
	selector := GetModelServiceSelector(ms)

	svcPorts := make([]corev1.ServicePort, 0)
	for _, port := range ms.Spec.Template.Spec.Containers[0].Ports {
		svcPorts = append(svcPorts, corev1.ServicePort{
			Name: port.Name,
			Port: port.ContainerPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: port.Name,
			},
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedMSName(ms.Name, ""),
			Namespace: ms.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, ms.GroupVersionKind()),
			},
			Labels: selector.MatchLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector.MatchLabels,
			Type:     ms.Spec.ServiceType,
			Ports:    svcPorts,
		},
	}

	return service
}

func constructModelStatus(ss *v1.StatefulSet, pod *corev1.Pod) mlv1.ModelServiceStatus {
	status := mlv1.ModelServiceStatus{
		Conditions:     make([]common.Condition, 0),
		ReadyReplicas:  ss.Status.ReadyReplicas,
		ContainerState: corev1.ContainerState{},
		State:          "",
	}

	// Skip updating the status if the pod's status is empty
	if reflect.DeepEqual(pod.Status, corev1.PodStatus{}) {
		logrus.Infof("model service pod status is empty, skip updating conditions and state")
		return status
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		cState := pod.Status.ContainerStatuses[0].State
		status.ContainerState = cState
		if cState.Running != nil {
			status.State = "Running"
		} else if cState.Waiting != nil {
			status.State = "Waiting"
		} else if cState.Terminated != nil {
			status.State = "Terminated"
		} else {
			status.State = "Unknown"
		}
	}

	// Mirror the pod conditions to the ModelService conditions
	for i := range pod.Status.Conditions {
		condition := reconcilehelper.PodCondToCond(pod.Status.Conditions[i])
		status.Conditions = append(status.Conditions, condition)
	}

	return status
}

func constructVllmArgs(ms *mlv1.ModelService, args []string) []string {
	if args == nil {
		args = make([]string, 0)
	}

	specArgs := map[string]string{
		"--model":             ms.Spec.ModelName,
		"--served-model-name": ms.Spec.ServedModelName,
	}

	for k, v := range specArgs {
		// Skip empty values
		if v == "" {
			continue
		}
		found := false
		for i, arg := range args {
			if strings.Contains(arg, k) {
				args[i] = fmt.Sprintf("%s=%s", k, v)
				found = true
				break
			}
		}

		if !found {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return args
}

func GetModelServiceSelector(ms *mlv1.ModelService) *metav1.LabelSelector {
	if ms.Spec.Selector != nil {
		selector := ms.Spec.Selector.DeepCopy()
		if selector.MatchLabels == nil {
			selector.MatchLabels = make(map[string]string)
		}
		selector.MatchLabels[constant.LabelModelServiceName] = ms.Name
		selector.MatchLabels[constant.LabelLLMOSMLType] = typeName
		return selector
	}
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			constant.LabelLLMOSMLType:      typeName,
			constant.LabelModelServiceName: ms.Name,
		},
	}
}

func getFormattedMSName(name string, appendix string) string {
	name = strings.ReplaceAll(name, ".", "-")
	if appendix == "" {
		return fmt.Sprintf("%s-%s", msPrefix, name)
	}
	return fmt.Sprintf("%s-%s-%s", msPrefix, name, appendix)
}

func getDefaultPodName(statefulSetName string) string {
	return fmt.Sprintf("%s-0", statefulSetName)
}
