package upgrade

import (
	"fmt"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const (
	labelArch               = "kubernetes.io/arch"
	labelCriticalAddonsOnly = "CriticalAddonsOnly"

	upgradeRepoName       = "upgrade-repo"
	upgradeServiceAccount = "system-upgrade-controller"
	systemChartsImageName = "llmos-ai/system-charts-repo"
	nodeUpgradeImageName  = "llmos-ai/node-upgrade"

	serverComponent = "server"
	agentComponent  = "agent"

	llmosUpgradeNameLabel      = "llmos.ai/upgrade-name"
	llmosVersionLabel          = "llmos.ai/version"
	llmosUpgradeComponentLabel = "llmos.io/upgrade-component"
	llmosLatestUpgradeLabel    = "llmos.ai/latest-upgrade"
)

// upgradeJobIsCompleteAfter returns true if the job is completed after the upgrade start time
func upgradeJobIsCompleteAfter(upgrade *mgmtv1.Upgrade, job *batchv1.Job) bool {
	// use completion time since the start time will possibly be same
	if job.Status.Succeeded == 1 && job.Status.CompletionTime != nil &&
		job.Status.CompletionTime.After(upgrade.Status.StartTime.Time) {
		return true
	}
	return false
}

func serverPlan(upgrade *mgmtv1.Upgrade) *upgradev1.Plan {
	plan := basePlan(upgrade)
	plan.Name = getServerPlanName(upgrade)
	plan.Labels[llmosUpgradeComponentLabel] = serverComponent
	plan.Spec.NodeSelector.MatchExpressions = []metav1.LabelSelectorRequirement{
		{
			Key:      constant.KubeControlPlaneNodeLabelKey,
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"true"},
		},
	}
	plan.Spec.Upgrade = &upgradev1.ContainerSpec{
		Image: formatRepoImage(upgrade.Spec.Registry, nodeUpgradeImageName, upgrade.Spec.Version),
		Args:  []string{"upgrade"},
	}
	return plan
}

func getServerPlanName(upgrade *mgmtv1.Upgrade) string {
	return fmt.Sprintf("%s-server", upgrade.Name)
}

func agentPlan(upgrade *mgmtv1.Upgrade) *upgradev1.Plan {
	plan := basePlan(upgrade)
	plan.Name = fmt.Sprintf("%s-agent", upgrade.Name)
	plan.Labels[llmosUpgradeComponentLabel] = agentComponent
	plan.Spec.NodeSelector.MatchExpressions = []metav1.LabelSelectorRequirement{
		{
			Key:      constant.KubeControlPlaneNodeLabelKey,
			Operator: metav1.LabelSelectorOpDoesNotExist,
		},
	}
	plan.Spec.Prepare = &upgradev1.ContainerSpec{
		// Use prepare step in the agent-plan to wait for the server-plan to complete before they execute.
		Image: formatRepoImage(upgrade.Spec.Registry, nodeUpgradeImageName, upgrade.Spec.Version),
		Args:  []string{"prepare", getServerPlanName(upgrade)},
	}
	plan.Spec.Upgrade = &upgradev1.ContainerSpec{
		Image: formatRepoImage(upgrade.Spec.Registry, nodeUpgradeImageName, upgrade.Spec.Version),
		Args:  []string{"upgrade"},
	}
	return plan
}

func basePlan(upgrade *mgmtv1.Upgrade) *upgradev1.Plan {
	version := upgrade.Spec.Version
	plan := &upgradev1.Plan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgrade.Name,
			Namespace: constant.SUCNamespace,
			Labels: map[string]string{
				llmosUpgradeNameLabel: upgrade.Name,
				llmosVersionLabel:     version,
			},
		},
		Spec: upgradev1.PlanSpec{
			Concurrency:           int64(1),
			Version:               upgrade.Spec.KubernetesVersion,
			JobActiveDeadlineSecs: defaultTTLSecondsAfterFinished,
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					constant.LLMOSManagedLabel: "true",
				},
			},
			ServiceAccountName: upgradeServiceAccount,
			Tolerations:        getDefaultTolerations(),
			Cordon:             true,
			Upgrade: &upgradev1.ContainerSpec{
				Image: formatRepoImage(upgrade.Spec.Registry, nodeUpgradeImageName, version),
			},
		},
	}
	if upgrade.Spec.Drain != nil {
		plan.Spec.Drain = upgrade.Spec.Drain
	}

	return plan
}

// getDefaultTolerations returns the default tolerations config for upgrade workloads
func getDefaultTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:      corev1.TaintNodeUnschedulable,
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		{
			Key:      constant.KubeControlPlaneNodeLabelKey,
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoExecute,
		},
		{
			Key:      constant.KubeEtcdNodeLabelKey,
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoExecute,
		},
		{
			Key:      labelCriticalAddonsOnly,
			Operator: corev1.TolerationOpExists,
		},
		{
			Key:      corev1.TaintNodeUnreachable,
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoExecute,
		},
		{
			Key:      labelArch,
			Operator: corev1.TolerationOpEqual,
			Effect:   corev1.TaintEffectNoSchedule,
			Value:    "amd64",
		},
		{
			Key:      labelArch,
			Operator: corev1.TolerationOpEqual,
			Effect:   corev1.TaintEffectNoSchedule,
			Value:    "arm64",
		},
		{
			Key:      labelArch,
			Operator: corev1.TolerationOpEqual,
			Effect:   corev1.TaintEffectNoSchedule,
			Value:    "arm",
		},
	}
}

func formatRepoImage(registry, repo, tag string) string {
	if registry != "" {
		return fmt.Sprintf("%s/%s:%s", registry, repo, tag)
	}
	return fmt.Sprintf("%s/%s:%s", settings.GlobalSystemImageRegistry.Get(), repo, tag)
}

func getUpgradeSystemChartsRepoUrl() string {
	return fmt.Sprintf("http://%s.%s.svc", upgradeRepoName, constant.SystemNamespaceName)
}

type commonHandler struct {
	upgradeClient ctlmgmtv1.UpgradeClient
	upgradeCache  ctlmgmtv1.UpgradeCache
}

// getLatestUpgrade returns the latest upgrade
func (h *commonHandler) getLatestUpgrade(name string) (*mgmtv1.Upgrade, error) {
	if name != "" {
		return h.upgradeCache.Get(name)
	}

	upgrades, err := h.upgradeCache.List(labels.SelectorFromSet(map[string]string{
		llmosLatestUpgradeLabel: "true",
	}))

	if err != nil {
		return nil, err
	}

	if len(upgrades) == 0 {
		return nil, nil
	}

	if len(upgrades) > 1 {
		return nil, fmt.Errorf("expected exactly one latest upgrade, got %d", len(upgrades))
	}

	return upgrades[0], nil
}

func (h *commonHandler) updateErrorCond(upgrade *mgmtv1.Upgrade, cond condition.Cond,
	err error) (*mgmtv1.Upgrade, error) {
	cond.SetError(upgrade, condition.StateError, err)
	upgrade.Status.State = condition.StateError
	return h.upgradeClient.UpdateStatus(upgrade)
}

func (h *commonHandler) updateUpgradingCond(upgrade *mgmtv1.Upgrade, cond condition.Cond,
	msg string) (*mgmtv1.Upgrade, error) {
	cond.SetStatus(upgrade, condition.StateProcessing)
	cond.Reason(upgrade, condition.StateProcessing)
	cond.Message(upgrade, msg)
	upgrade.Status.State = condition.StateUpgrading
	return h.upgradeClient.UpdateStatus(upgrade)
}

func (h *commonHandler) updateReadyCond(upgrade *mgmtv1.Upgrade, cond condition.Cond,
	msg string) (*mgmtv1.Upgrade, error) {
	cond.SetStatus(upgrade, "True")
	cond.Reason(upgrade, "Ready")
	cond.Message(upgrade, msg)
	return h.upgradeClient.UpdateStatus(upgrade)
}
