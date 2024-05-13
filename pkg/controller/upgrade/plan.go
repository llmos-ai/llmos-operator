package upgrade

import (
	"fmt"
	"reflect"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	llmosv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
)

const (
	labelArch               = "kubernetes.io/arch"
	labelCriticalAddonsOnly = "CriticalAddonsOnly"
)

func (h *handler) reconcileUpgradeStatus(_ string, plan *upgradev1.Plan) (*upgradev1.Plan, error) {
	if plan == nil || plan.DeletionTimestamp != nil {
		return plan, nil
	}

	if plan.Labels == nil || plan.Labels[constant.LLMOSUpgradeLabel] == "" || plan.Spec.NodeSelector == nil {
		return plan, nil
	}

	upgrade, err := h.upgradeCache.Get(constant.LLMOSSystemNamespace, plan.Labels[constant.LLMOSUpgradeLabel])
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Errorf("upgrade %s not found", plan.Labels[constant.LLMOSUpgradeLabel])
			return nil, nil
		}
		return plan, err
	}

	return plan, h.syncPlanStatusToUpgrade(upgrade, plan)
}

func (h *handler) syncPlanStatusToUpgrade(upgrade *llmosv1.Upgrade, plan *upgradev1.Plan) error {
	if !reflect.DeepEqual(upgrade.Status.PlanStatus, plan.Status) {
		toUpdate := upgrade.DeepCopy()
		toUpdate.Status.PlanStatus = plan.Status
		// set image id to the one applied on the plan obj
		toUpdate.Status.ImageID = fmt.Sprintf("%s:%s", plan.Spec.Upgrade.Image, plan.Spec.Version)
		if _, err := h.upgradeClient.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func upgradePlan(upgrade *llmosv1.Upgrade) *upgradev1.Plan {
	version := upgrade.Spec.Version
	plan := &upgradev1.Plan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("llmos-%s", upgrade.Name),
			Namespace: constant.SUCNamespace,
			Labels: map[string]string{
				constant.LLMOSVersionLabel: version,
				constant.LLMOSUpgradeLabel: upgrade.Name,
			},
		},
		Spec: upgradev1.PlanSpec{
			Concurrency: int64(1),
			Version:     version,
			NodeSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      constant.LLMOSManagedLabel,
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      constant.LLMOSUpgradeLabel,
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"disabled", "false"},
					},
					{
						Key:      "node-role.kubernetes.io/control-plane",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
			},
			ServiceAccountName: upgradeServiceAccount,
			Tolerations: []v1.Toleration{
				{
					Key:      labelCriticalAddonsOnly,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      labelArch,
					Operator: v1.TolerationOpEqual,
					Effect:   v1.TaintEffectNoSchedule,
					Value:    "amd64",
				},
				{
					Key:      labelArch,
					Operator: v1.TolerationOpEqual,
					Effect:   v1.TaintEffectNoSchedule,
					Value:    "arm64",
				},
				{
					Key:      labelArch,
					Operator: v1.TolerationOpEqual,
					Effect:   v1.TaintEffectNoSchedule,
					Value:    "arm",
				},
			},
			Cordon:  true,
			Upgrade: upgrade.Spec.ContainerSpec,
		},
	}

	if upgrade.Spec.Drain != nil {
		plan.Spec.Drain = upgrade.Spec.Drain
	}
	return plan
}
