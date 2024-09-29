package upgrade

import (
	"fmt"

	gversion "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	upgrade2 "github.com/llmos-ai/llmos-operator/pkg/controller/master/upgrade"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	werror "github.com/llmos-ai/llmos-operator/pkg/webhook/error"
)

func (v *validator) validateCanUpgradeVersion(upgrade *mgmtv1.Upgrade) (bool, error) {
	serverVersion, err := v.settingCache.Get(settings.ServerVersionName)
	if err != nil {
		return false, werror.InternalError(fmt.Sprintf("Failed to get server version: %v", err))
	}

	currentVersion, err := gversion.NewSemver(serverVersion.Value)
	if err != nil {
		return false, werror.InternalError(fmt.Sprintf("Failed to parse current server version: %v", err))
	}

	newVersion, err := v.versionCache.Get(upgrade.Spec.Version)
	if err != nil {
		return false, werror.InternalError(fmt.Sprintf("Failed to get version %q: %v",
			upgrade.Spec.Version, err))
	}

	ok, err := upgrade2.CanUpgradeVersion(currentVersion, newVersion)
	logrus.Infof("Validating can upgrade from version %s to version %+v, result:%t",
		currentVersion.Original(), newVersion.Name, ok)

	return ok, err
}

func (v *validator) checkUpgradeResources() error {
	if err := v.checkManagedAddons(); err != nil {
		return err
	}

	if err := v.checkOperatorManifest(); err != nil {
		return err
	}

	if err := v.checkNodes(); err != nil {
		return err
	}

	return nil
}

func (v *validator) checkManagedAddons() error {
	selector := labels.Set{
		constant.SystemAddonLabel: "true",
	}.AsSelector()

	addons, err := v.addonCache.List(constant.SystemNamespaceName, selector)
	if err != nil {
		return werror.InternalError(fmt.Sprintf("Failed to list managed addons: %v", err))
	}
	for _, addon := range addons {
		if addon.Spec.Enabled && !mgmtv1.AddonCondReady.IsTrue(addon) {
			return werror.BadRequest(fmt.Sprintf("Cannot upgrade while the managed addon %s is not ready",
				addon.Name))
		}
	}
	return nil
}

func (v *validator) checkNodes() error {
	nodes, err := v.nodeCache.List(labels.Everything())
	if err != nil {
		return werror.InternalError(fmt.Sprintf("Failed to list nodes: %v", err))
	}
	for _, node := range nodes {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status != corev1.ConditionTrue {
					return werror.BadRequest(fmt.Sprintf("Node %s is not ready[%s], please "+
						"resolve the condition first", node.Name, condition.Message))
				}
				break
			}
		}

		if node.Spec.Unschedulable {
			return werror.BadRequest(fmt.Sprintf("Node %s is unschedulable, please resolve the "+
				"config first", node.Name))
		}
	}

	return nil
}

func (v *validator) checkOperatorManifest() error {
	selector := labels.SelectorFromSet(map[string]string{
		constant.LabelAppName: v.releaseName,
	})
	manifestDeployments, err := v.deploymentCache.List(constant.SystemNamespaceName, selector)
	if err != nil {
		return werror.InternalError(fmt.Sprintf("Failed to list manifest deployments: %v", err))
	}

	for _, deployment := range manifestDeployments {
		if !upgrade2.DeploymentIsReady(deployment) {
			return werror.BadRequest(fmt.Sprintf("Manifest deployment %s is not ready", deployment.Name))
		}
	}

	return nil
}
