package upgrade

import (
	"fmt"

	"github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	ctlugpradev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/upgrade.cattle.io/v1"
)

const (
	// keep jobs for 7 days
	defaultTTLSecondsAfterFinished = 604800
)

// planHandler syncs on plan completions
// When a server plan completes, it will sync the plan status to upgrade.
// When an agent plan completes, it set the NodesUpgraded condition of upgrade CRD to be true.
type planHandler struct {
	upgradeClient ctlmgmtv1.UpgradeClient
	upgradeCache  ctlmgmtv1.UpgradeCache
	planClient    ctlugpradev1.PlanClient
	nodeCache     ctlcorev1.NodeCache
	*commonHandler
}

// watchUpgradePlans will update the upgrade status with the latest plan status
func (h *planHandler) watchUpgradePlans(_ string, plan *upgradev1.Plan) (*upgradev1.Plan, error) {
	if plan == nil || plan.DeletionTimestamp != nil || plan.Labels == nil ||
		(plan.Namespace != constant.SUCNamespace && plan.Namespace != constant.SystemNamespaceName) {
		return nil, nil
	}

	upgradeName := plan.Labels[llmosUpgradeNameLabel]
	component := plan.Labels[llmosUpgradeComponentLabel]
	if upgradeName == "" || component == "" || plan.Spec.NodeSelector == nil {
		return nil, nil
	}

	// wait for plan to be completed first
	if !upgradev1.PlanComplete.IsTrue(plan) {
		return nil, nil
	}

	// create select requirements of the current plan
	requirementPlanNotLatest, err := labels.NewRequirement(upgrade.LabelPlanName(plan.Name),
		selection.NotIn, []string{"disabled", plan.Status.LatestHash})
	if err != nil {
		return plan, err
	}
	selector, err := metav1.LabelSelectorAsSelector(plan.Spec.NodeSelector)
	if err != nil {
		return plan, err
	}
	selector = selector.Add(*requirementPlanNotLatest)

	nodes, err := h.nodeCache.List(selector)
	if err != nil {
		return plan, err
	}
	if len(nodes) != 0 {
		return plan, nil
	}

	// All plan nodes are upgraded at this stage
	upgrade, err := h.getLatestUpgrade(upgradeName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Debugf("upgrade %s not found, skipping upgrade plan %s", upgradeName, plan.Name)
		return plan, nil
	} else if err != nil {
		return plan, err
	}

	if err = h.syncPlanStatusToUpgrade(upgrade, plan, component); err != nil {
		return plan, err
	}

	return plan, nil
}

func (h *planHandler) syncPlanStatusToUpgrade(upgrade *mgmtv1.Upgrade, plan *upgradev1.Plan, component string) error {
	logrus.Debugf("sync upgrade plan %s status to upgrade %s", plan.Name, upgrade.Name)
	toUpdate := upgrade.DeepCopy()
	if toUpdate.Status.PlanStatus == nil {
		toUpdate.Status.PlanStatus = make([]mgmtv1.UpgradePlanStatus, 0)
	}

	found := false
	planStatus := mgmtv1.UpgradePlanStatus{
		Name:           plan.Name,
		LatestVersion:  plan.Status.LatestVersion,
		LatestHash:     plan.Status.LatestHash,
		LastUpdateTime: metav1.Now(),
		Complete:       true,
	}

	for i, ps := range toUpdate.Status.PlanStatus {
		if ps.Name == plan.Name {
			found = true
			toUpdate.Status.PlanStatus[i] = planStatus
		}
	}

	if !found {
		toUpdate.Status.PlanStatus = append(toUpdate.Status.PlanStatus, planStatus)
	}

	// all nodes are upgraded if agent plan is completed
	if !mgmtv1.NodesUpgraded.IsTrue(upgrade) && component == agentComponent {
		msg := fmt.Sprintf("All nodes upgraded to version %s(%s)", upgrade.Spec.Version,
			upgrade.Spec.KubernetesVersion)
		if _, err := h.updateReadyCond(toUpdate, mgmtv1.NodesUpgraded, msg); err != nil {
			return err
		}
	}

	if _, err := h.upgradeClient.UpdateStatus(toUpdate); err != nil {
		return err
	}

	return nil
}
