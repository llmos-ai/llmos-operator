package upgrade

import (
	"reflect"

	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
)

type addonHandler struct {
	addonClient   ctlmgmtv1.ManagedAddonClient
	addonCache    ctlmgmtv1.ManagedAddonCache
	upgradeClient ctlmgmtv1.UpgradeClient
	*commonHandler
}

// onAddonChange updates the upgrade status of the managed-addons
func (h *addonHandler) onAddonChange(_ string, addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	if addon == nil || addon.DeletionTimestamp != nil || addon.Labels == nil ||
		addon.Namespace != constant.SystemNamespaceName {
		return nil, nil
	}

	systemAddon := addon.Labels[constant.SystemAddonLabel]
	version := addon.Labels[constant.LLMOSServerVersionLabel]
	if systemAddon == "" || version == "" {
		return addon, nil
	}

	upgrade, err := h.getLatestUpgrade("")
	if err != nil || upgrade == nil {
		return addon, err
	}

	// skip update upgrade status if upgrade is completed
	if mgmtv1.UpgradeCompleted.IsTrue(upgrade) || !isManagedAddonReady(addon) || version != upgrade.Spec.Version {
		return addon, nil
	}

	logrus.Debugf("Updating upgrade status for managed addon %s", addon.Name)
	return addon, h.updateManagedAddonStatus(addon, upgrade)
}

func (h *addonHandler) updateManagedAddonStatus(addon *mgmtv1.ManagedAddon, upgrade *mgmtv1.Upgrade) error {
	toUpdate := upgrade.DeepCopy()

	allComplete := true
	for i, status := range toUpdate.Status.ManagedAddonStatus {
		if status.Name == addon.Name {
			upgrade.Status.ManagedAddonStatus[i] = mgmtv1.UpgradeManagedAddonStatus{
				Name:     addon.Name,
				JobName:  addon.Status.JobName,
				Complete: true,
			}
		}
		if !status.Complete {
			allComplete = false
		}
	}

	if allComplete {
		msg := "All managed system addons are ready"
		if _, err := h.updateReadyCond(toUpdate, mgmtv1.ManagedAddonsIsReady, msg); err != nil {
			return err
		}
	} else if !reflect.DeepEqual(upgrade.Status, toUpdate.Status) {
		if _, err := h.upgradeClient.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}

	return nil
}

func isManagedAddonReady(addon *mgmtv1.ManagedAddon) bool {
	for _, cond := range addon.Status.Conditions {
		if cond.Type == mgmtv1.AddonCondReady && cond.Status != "True" {
			return false
		}
	}

	return addon.Status.CompletionTime != nil && addon.Status.Succeeded >= 1 &&
		addon.Status.State == mgmtv1.AddonStateComplete
}
