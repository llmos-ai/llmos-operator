package notebook

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

func getNoteBookStatefulSet(notebook *mlv1.Notebook) *v1.StatefulSet {
	replicas := int32(1)
	if metav1.HasAnnotation(notebook.ObjectMeta, constant.AnnotationResourceStopped) {
		replicas = 0
	}

	selector := GetNotebookSelector(notebook)
	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedNotebookName(notebook),
			Namespace: notebook.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(notebook, notebook.GroupVersionKind()),
			},
			Labels: selector.MatchLabels,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector.MatchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      selector.MatchLabels,
					Annotations: map[string]string{},
				},
				Spec: *notebook.Spec.Template.Spec.DeepCopy(),
			},
			VolumeClaimTemplates: notebook.Spec.VolumeClaimTemplates,
		},
	}

	// copy all the notebook labels to the pod including pod default related labels
	l := &ss.Spec.Template.ObjectMeta.Labels
	for k, v := range notebook.ObjectMeta.Labels {
		(*l)[k] = v
	}

	// copy all the notebook annotations to the pod.
	a := &ss.Spec.Template.ObjectMeta.Annotations
	for k, v := range notebook.ObjectMeta.Annotations {
		if !strings.Contains(k, "kubectl") && !strings.Contains(k, "notebook") {
			(*a)[k] = v
		}
	}

	container := ss.Spec.Template.Spec.Containers[0]
	container.Name = notebook.Name
	if container.WorkingDir == "" {
		container.WorkingDir = "/home/jovyan"
	}
	if container.Ports == nil {
		container.Ports = []corev1.ContainerPort{
			{
				ContainerPort: DefaultContainerPort,
				Name:          "notebook-port",
				Protocol:      "TCP",
			},
		}
	}

	return ss
}
func getNotebookService(notebook *mlv1.Notebook) *corev1.Service {
	svcType := corev1.ServiceTypeClusterIP
	if notebook.Spec.ServiceType != "" {
		svcType = notebook.Spec.ServiceType
	}

	selector := GetNotebookSelector(notebook)
	// Define the desired Service object
	port := DefaultContainerPort
	containerPorts := notebook.Spec.Template.Spec.Containers[0].Ports
	if containerPorts != nil {
		port = containerPorts[0].ContainerPort
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedNotebookName(notebook),
			Namespace: notebook.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(notebook, notebook.GroupVersionKind()),
			},
			Labels: selector.MatchLabels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: selector.MatchLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       DefaultServingPort,
					TargetPort: intstr.FromInt32(port),
					Protocol:   "TCP",
				},
			},
		},
	}
	return svc
}

func getNotebookStatus(ss *v1.StatefulSet, pod *corev1.Pod) mlv1.NotebookStatus {
	status := mlv1.NotebookStatus{
		Conditions:     make([]common.Condition, 0),
		ReadyReplicas:  ss.Status.ReadyReplicas,
		ContainerState: corev1.ContainerState{},
		State:          "",
	}

	if reflect.DeepEqual(pod.Status, corev1.PodStatus{}) {
		logrus.Infof("notebook pod status is empty, skip updating conditions and state")
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

func GetNotebookSelector(notebook *mlv1.Notebook) *metav1.LabelSelector {
	if notebook.Spec.Selector != nil {
		selector := notebook.Spec.Selector.DeepCopy()
		if selector.MatchLabels == nil {
			selector.MatchLabels = make(map[string]string)
		}
		selector.MatchLabels[constant.LabelLLMOSMLAppName] = strings.ToLower(notebook.Kind)
		selector.MatchLabels[constant.LabelNotebookName] = notebook.Name
		return selector
	}
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			constant.LabelLLMOSMLAppName: strings.ToLower(notebook.Kind),
			constant.LabelNotebookName:   notebook.Name,
		},
	}
}

func getFormattedNotebookName(notebook *mlv1.Notebook) string {
	return fmt.Sprintf("notebook-%s", notebook.Name)
}

func getNotebookPodName(statefulSetName string) string {
	return fmt.Sprintf("%s-0", statefulSetName)
}
