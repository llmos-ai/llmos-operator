package upgrade

import (
	"context"
	"fmt"
	"reflect"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	ctlmgmtv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai/v1"
	ctlupgradev1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/upgrade.cattle.io/v1"

	llmosv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/settings"
)

const (
	upgradeOnChange   = "upgrade.onChange"
	upgradeOnDelete   = "upgrade.delete"
	upgradeWatchPlans = "upgrade.watchPlans"

	upgradeServiceAccount   = "system-upgrade"
	stateUpgrading          = "Upgrading"
	stateFailed             = "Failed"
	llmosLatestUpgradeLabel = "llmos.ai/latest-upgrade"
)

type handler struct {
	namespace     string
	upgradeClient ctlmgmtv1.UpgradeClient
	upgradeCache  ctlmgmtv1.UpgradeCache
	planClient    ctlupgradev1.PlanClient
	planCache     ctlupgradev1.PlanCache
}

func Register(ctx context.Context, mgmt *config.Management) error {
	upgrades := mgmt.MgmtFactory.Management().V1().Upgrade()
	plans := mgmt.UpgradeFactory.Upgrade().V1().Plan()

	h := &handler{
		namespace:     mgmt.Namespace,
		upgradeClient: upgrades,
		upgradeCache:  upgrades.Cache(),
		planClient:    plans,
		planCache:     plans.Cache(),
	}

	upgrades.OnChange(ctx, upgradeOnChange, h.onChange)
	upgrades.OnRemove(ctx, upgradeOnDelete, h.onDelete)
	plans.OnChange(ctx, upgradeWatchPlans, h.reconcileUpgradeStatus)
	return nil
}

func (h *handler) onChange(_ string, upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	if upgrade == nil || upgrade.DeletionTimestamp != nil {
		return nil, nil
	}

	if llmosv1.UpgradeCompleted.IsTrue(upgrade) {
		logrus.Debug("Upgrade is completed, skip processing")
		return nil, nil
	}

	// create plans if not initialized
	toUpdate := upgrade.DeepCopy()

	if llmosv1.UpgradeCompleted.GetStatus(upgrade) == "" {
		if err := h.setLatestUpgradeLabel(upgrade.Name); err != nil {
			return upgrade, err
		}

		initStatus(toUpdate)
		plan := upgradePlan(upgrade)
		if _, err := h.planClient.Create(plan); err != nil && !errors.IsAlreadyExists(err) {
			llmosv1.UpgradeCompleted.SetError(toUpdate, stateFailed, err)
			return h.upgradeClient.UpdateStatus(toUpdate)
		}

		llmosv1.UpgradeCompleted.SetStatus(toUpdate, stateUpgrading)
		return h.upgradeClient.UpdateStatus(toUpdate)
	}

	// sync upgrade plan
	plan, err := h.syncUpgradePlan(toUpdate)
	if err != nil {
		llmosv1.UpgradeCompleted.SetError(toUpdate, stateFailed, err)
		return h.upgradeClient.UpdateStatus(toUpdate)
	}

	return nil, h.updateUpgradeStatus(plan, toUpdate)
}

func (h *handler) syncUpgradePlan(upgrade *llmosv1.Upgrade) (*upgradev1.Plan, error) {
	sets := labels.Set{
		constant.LLMOSUpgradeLabel: upgrade.Name,
	}
	plans, err := h.planCache.List(constant.SUCNamespace, sets.AsSelector())
	if err != nil {
		return nil, err
	}

	if len(plans) == 0 {
		return nil, fmt.Errorf("no plan found for upgrade %s", upgrade.Name)
	}

	plan := plans[0]

	// TODO, move to the validation webhook
	if upgradev1.PlanComplete.IsTrue(plan) {
		logrus.Debugf("Upgrade plan %s is completed, skip updating", plan.Name)
		return plan, nil
	}

	if !reflect.DeepEqual(plan.Spec.Upgrade, upgrade.Spec.ContainerSpec) ||
		!reflect.DeepEqual(plan.Spec.Drain, upgrade.Spec.Drain) ||
		plan.Spec.Version != upgrade.Spec.Version {
		logrus.Debugf("Syncing plan: %s of upgrade: %s", plan.Name, upgrade.Name)
		toUpdate := plan.DeepCopy()
		toUpdate.Spec.Version = upgrade.Spec.Version
		toUpdate.Spec.Drain = upgrade.Spec.Drain
		toUpdate.Spec.Upgrade = upgrade.Spec.ContainerSpec
		toUpdate.Labels[constant.LLMOSVersionLabel] = upgrade.Spec.Version
		if _, err = h.planClient.Update(toUpdate); err != nil {
			return plan, err
		}
	}
	return plan, nil
}

func (h *handler) updateUpgradeStatus(plan *upgradev1.Plan, upgrade *llmosv1.Upgrade) error {
	if plan == nil || upgrade == nil {
		return nil
	}

	if upgradev1.PlanComplete.IsTrue(plan) && !llmosv1.UpgradeCompleted.IsTrue(upgrade) {
		logrus.Debugf("Upgrade plan %s is completed, updating upgrade complete status", plan.Name)
		toUpdate := upgrade.DeepCopy()
		llmosv1.UpgradeCompleted.SetStatus(toUpdate, "True")
		if _, err := h.upgradeClient.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func (h *handler) setLatestUpgradeLabel(latestUpgradeName string) error {
	sets := labels.Set{
		llmosLatestUpgradeLabel: "true",
	}
	upgrades, err := h.upgradeCache.List(h.namespace, sets.AsSelector())
	if err != nil {
		return err
	}
	for _, upgrade := range upgrades {
		if upgrade.Name == latestUpgradeName {
			continue
		}
		toUpdate := upgrade.DeepCopy()
		delete(toUpdate.Labels, llmosLatestUpgradeLabel)
		if _, err := h.upgradeClient.Update(toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func initStatus(upgrade *llmosv1.Upgrade) {
	llmosv1.UpgradeCompleted.CreateUnknownIfNotExists(upgrade)
	if upgrade.Labels == nil {
		upgrade.Labels = make(map[string]string)
	}
	upgrade.Labels[llmosLatestUpgradeLabel] = "true"
	upgrade.Status.PreviousVersion = settings.ServerVersion.Get()
}

func (h *handler) onDelete(_ string, upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	if upgrade == nil || upgrade.DeletionTimestamp == nil {
		return nil, nil
	}

	// delete upgrade plan
	sets := labels.Set{
		constant.LLMOSUpgradeLabel: upgrade.Name,
	}
	plans, err := h.planCache.List(constant.SUCNamespace, sets.AsSelector())
	if err != nil {
		return upgrade, err
	}

	logrus.Debug("delete upgrade plan", "plans", plans)
	for _, plan := range plans {
		if err := h.planClient.Delete(plan.Namespace, plan.Name, nil); err != nil {
			return upgrade, err
		}
	}
	return nil, nil
}
