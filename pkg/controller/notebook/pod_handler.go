package notebook

import (
	"strings"
	"time"

	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	cond "github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const notebookNameLabel = "notebook-name"

// ReconcileNotebookPodOwners reconciles the owner notebook by pod
func (h *Handler) ReconcileNotebookPodOwners(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if hc, ok := obj.(*corev1.Pod); ok {
		for k, v := range hc.GetLabels() {
			if strings.ToLower(k) == notebookNameLabel {
				return []relatedresource.Key{
					{
						Name:      v,
						Namespace: hc.Namespace,
					},
				}, nil
			}
		}
	}

	return nil, nil
}

// PodCondToNotebookCond Note: this is referred to kubeflow notebook controller
// https://github.com/kubeflow/kubeflow/tree/master/components/notebook-controller
func PodCondToNotebookCond(pCond corev1.PodCondition) common.Condition {
	condition := common.Condition{}

	if len(pCond.Type) > 0 {
		condition.Type = cond.Cond(pCond.Type)
	}

	if len(pCond.Status) > 0 {
		condition.Status = metav1.ConditionStatus(pCond.Status)
	}

	if len(pCond.Message) > 0 {
		condition.Message = pCond.Message
	}

	if len(pCond.Reason) > 0 {
		condition.Reason = pCond.Reason
	}

	// check if pCond.LastProbeTime is null. If so initialize
	// the field with metav1.Now()
	check := pCond.LastProbeTime.Time.Equal(time.Time{})
	if !check {
		condition.LastUpdateTime = pCond.LastProbeTime.Format(time.RFC3339)
	} else {
		condition.LastUpdateTime = metav1.Now().UTC().Format(time.RFC3339)
	}

	// check if pCond.LastTransitionTime is null. If so initialize
	// the field with metav1.Now()
	check = pCond.LastTransitionTime.Time.Equal(time.Time{})
	if !check {
		condition.LastTransitionTime = pCond.LastTransitionTime.Format(time.RFC3339)
	} else {
		condition.LastTransitionTime = metav1.Now().UTC().Format(time.RFC3339)
	}

	return condition
}
