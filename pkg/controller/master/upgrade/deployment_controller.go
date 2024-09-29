package upgrade

import (
	"fmt"

	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
)

type deploymentHandler struct {
	releaseName     string
	deploymentCache ctlappsv1.DeploymentCache
	upgradeClient   ctlmgmtv1.UpgradeClient
	upgradeCache    ctlmgmtv1.UpgradeCache
	*commonHandler
}

// watchDeployment watches upgrade deployments and sync upgrade repo and manifest upgrade
func (h *deploymentHandler) watchDeployment(_ string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	if deployment == nil || deployment.DeletionTimestamp != nil {
		return nil, nil
	}

	if deployment.Labels == nil || deployment.Namespace != constant.SystemNamespaceName {
		return nil, nil
	}

	component := deployment.Labels[llmosUpgradeComponentLabel]
	appName := deployment.Labels[constant.LabelAppName]
	appVersion := deployment.Labels[constant.LabelAppVersion]
	upgradeName := deployment.Labels[llmosUpgradeNameLabel]

	// Skip upgrade sync if:
	// 1. deployment is not ready
	// 2. deployment is not an upgrade repo and not an operator manifest
	if !DeploymentIsReady(deployment) || (component == "" && appName != h.releaseName) {
		return nil, nil
	}

	upgrade, err := h.getLatestUpgrade(upgradeName)
	if err != nil || upgrade == nil {
		logrus.Infof("No upgrade found for deployment %s/%s", deployment.Namespace, deployment.Name)
		return deployment, nil
	} else if err != nil {
		return deployment, err
	}

	if mgmtv1.UpgradeCompleted.IsTrue(upgrade) || upgrade.Spec.Version != appVersion {
		return deployment, nil
	}

	// sync upgrade repo
	if component == upgradeRepoName {
		if err := h.syncUpgradeRepoStatus(deployment, upgrade); err != nil {
			return deployment, err
		}
	}

	// sync manifest upgrade
	if appName == h.releaseName {
		return deployment, h.syncManifestUpgrade(deployment, upgrade)
	}

	return deployment, nil
}

func (h *deploymentHandler) syncUpgradeRepoStatus(deployment *appsv1.Deployment, upgrade *mgmtv1.Upgrade) error {
	logrus.Debugf("syncing upgrade repo status %v, for upgrade %s", deployment.Status.Conditions, upgrade.Name)
	msg := fmt.Sprintf("upgrade repo %s(%s) is ready", upgradeRepoName, upgrade.Spec.Version)
	if _, err := h.updateReadyCond(upgrade, mgmtv1.UpgradeChartsRepoReady, msg); err != nil {
		return err
	}

	return nil
}

func (h *deploymentHandler) syncManifestUpgrade(deployment *appsv1.Deployment, upgrade *mgmtv1.Upgrade) error {
	// Do not sync if manifest upgrade is not initialized by the upgrade controller
	if mgmtv1.ManifestUpgradeComplete.GetStatus(upgrade) == "" {
		return nil
	}
	selector := labels.SelectorFromSet(map[string]string{
		constant.LabelAppName:    h.releaseName,
		constant.LabelAppVersion: upgrade.Spec.Version,
	})

	manifestDeployments, err := h.deploymentCache.List(constant.SystemNamespaceName, selector)
	if err != nil {
		return err
	}

	chartVersion := deployment.Labels[constant.LabelAppVersion]
	for _, manifestDeployment := range manifestDeployments {
		if !DeploymentIsReady(manifestDeployment) {
			return nil
		}
		if !isLatestChartVersion(chartVersion, upgrade.Spec.Version) {
			return nil
		}
	}

	if _, err := h.updateReadyCond(upgrade, mgmtv1.ManifestUpgradeComplete, "Manifest upgrade is ready"); err != nil {
		return err
	}

	return nil
}

func DeploymentIsReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.ReadyReplicas == deployment.Status.Replicas
}

func isLatestChartVersion(chartVersion, upgradeVersion string) bool {
	return chartVersion == upgradeVersion
}
