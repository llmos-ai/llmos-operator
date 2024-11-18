package reconcilehelper

import (
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	cond "github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

func CopyStatefulSetFields(from, to *appsv1.StatefulSet) bool {
	requireUpdate := false
	if to.Labels == nil {
		to.Labels = make(map[string]string)
	}
	for k, v := range to.Labels {
		if from.Labels[k] != v {
			requireUpdate = true
		}
	}
	to.Labels = from.Labels

	if to.Annotations == nil {
		to.Annotations = make(map[string]string)
	}
	for k, v := range to.Annotations {
		if from.Annotations[k] != v {
			requireUpdate = true
		}
	}
	to.Annotations = from.Annotations

	if from.Spec.Replicas != to.Spec.Replicas {
		to.Spec.Replicas = from.Spec.Replicas
		requireUpdate = true
	}

	if !reflect.DeepEqual(to.Spec.Template.ObjectMeta, from.Spec.Template.ObjectMeta) {
		requireUpdate = true
	}
	to.Spec.Template.ObjectMeta = from.Spec.Template.ObjectMeta

	if !reflect.DeepEqual(to.Spec.Template.Spec, from.Spec.Template.Spec) {
		requireUpdate = true
	}
	to.Spec.Template.Spec = from.Spec.Template.Spec

	if !reflect.DeepEqual(to.Spec.UpdateStrategy, from.Spec.UpdateStrategy) {
		requireUpdate = true
	}
	to.Spec.UpdateStrategy = from.Spec.UpdateStrategy

	return requireUpdate
}

func CopyDeploymentFields(from, to *appsv1.Deployment) bool {
	requireUpdate := false
	if to.Labels == nil {
		to.Labels = make(map[string]string)
	}
	for k, v := range to.Labels {
		if from.Labels[k] != v {
			requireUpdate = true
		}
	}
	to.Labels = from.Labels

	if to.Annotations == nil {
		to.Annotations = make(map[string]string)
	}
	for k, v := range to.Annotations {
		if from.Annotations[k] != v {
			requireUpdate = true
		}
	}
	to.Annotations = from.Annotations

	if from.Spec.Replicas != to.Spec.Replicas {
		requireUpdate = true
	}
	to.Spec.Replicas = from.Spec.Replicas

	if !reflect.DeepEqual(to.Spec.Template.Spec, from.Spec.Template.Spec) {
		requireUpdate = true
	}
	to.Spec.Template.Spec = from.Spec.Template.Spec

	return requireUpdate
}

// CopyServiceFields copies the owned fields from one Service to another
func CopyServiceFields(from, to *corev1.Service) bool {
	requireUpdate := false
	for k, v := range to.Labels {
		if from.Labels[k] != v {
			requireUpdate = true
		}
	}
	to.Labels = from.Labels

	for k, v := range to.Annotations {
		if from.Annotations[k] != v {
			requireUpdate = true
		}
	}
	to.Annotations = from.Annotations

	// Don't copy the entire Spec, because some fields are immutable
	if !reflect.DeepEqual(to.Spec.Selector, from.Spec.Selector) {
		requireUpdate = true
	}
	to.Spec.Selector = from.Spec.Selector

	if !reflect.DeepEqual(to.Spec.Ports, from.Spec.Ports) {
		requireUpdate = true
	}
	to.Spec.Ports = from.Spec.Ports

	if from.Spec.Type != to.Spec.Type {
		requireUpdate = true
	}
	to.Spec.Type = from.Spec.Type

	return requireUpdate
}

func PodCondToCond(podc corev1.PodCondition) common.Condition {
	condition := common.Condition{}

	if len(podc.Type) > 0 {
		condition.Type = cond.Cond(podc.Type)
	}

	if len(podc.Status) > 0 {
		condition.Status = metav1.ConditionStatus(podc.Status)
	}

	if len(podc.Message) > 0 {
		condition.Message = podc.Message
	}

	if len(podc.Reason) > 0 {
		condition.Reason = podc.Reason
	}

	check := podc.LastProbeTime.Time.Equal(time.Time{})
	if !check {
		condition.LastUpdateTime = podc.LastProbeTime.Format(time.RFC3339)
	} else {
		condition.LastUpdateTime = metav1.Now().UTC().Format(time.RFC3339)
	}

	check = podc.LastTransitionTime.Time.Equal(time.Time{})
	if !check {
		condition.LastTransitionTime = podc.LastTransitionTime.Format(time.RFC3339)
	} else {
		condition.LastTransitionTime = metav1.Now().UTC().Format(time.RFC3339)
	}

	return condition
}
