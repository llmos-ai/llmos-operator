package upgrade

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	gversion "github.com/hashicorp/go-version"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/discovery"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlhelmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/helm.cattle.io/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	ctlupgradev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/upgrade.cattle.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	cond "github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const (
	llmosCrdChartName      = constant.LLMOSCrdChartName
	llmosOperatorChartName = constant.LLMOSOperatorChartName

	msgWaitingForAddons   = "Waiting for managed addons to be validated"
	msgWaitingForManifest = "Waiting for HelmChart [%s] upgrade job to be complete"
	msgWaitingForNodes    = "Waiting for nodes to be upgraded to %s"
	msgWaitingForRepo     = "Waiting for upgrade repo %s to be ready"
)

var operatorUpgradeCharts = []string{
	llmosCrdChartName,
	llmosOperatorChartName,
}

var (
	upgradeControllerLock sync.Mutex
)

type upgradeHandler struct {
	upgradeClient    ctlmgmtv1.UpgradeClient
	upgradeCache     ctlmgmtv1.UpgradeCache
	helmChartClient  ctlhelmv1.HelmChartClient
	helmChartCache   ctlhelmv1.HelmChartCache
	planClient       ctlupgradev1.PlanClient
	planCache        ctlupgradev1.PlanCache
	deploymentClient ctlappsv1.DeploymentClient
	deploymentCache  ctlappsv1.DeploymentCache
	svcClient        ctlcorev1.ServiceClient
	svcCache         ctlcorev1.ServiceCache
	addonCache       ctlmgmtv1.ManagedAddonCache
	discovery        discovery.DiscoveryInterface
	*commonHandler
}

func (h *upgradeHandler) onChange(_ string, upgrade *mgmtv1.Upgrade) (*mgmtv1.Upgrade, error) {
	if upgrade == nil || upgrade.DeletionTimestamp != nil {
		return nil, nil
	}

	upgradeControllerLock.Lock()
	defer upgradeControllerLock.Unlock()

	if mgmtv1.UpgradeCompleted.IsTrue(upgrade) {
		logrus.Debug("upgrade is completed, skip processing")
		return nil, nil
	}

	var err error
	toUpdate := upgrade.DeepCopy()

	// Init upgrade status
	if mgmtv1.UpgradeCompleted.GetStatus(upgrade) == "" {
		if err := h.setLatestUpgradeLabel(toUpdate); err != nil {
			return upgrade, err
		}

		initStatus(toUpdate)
		// check if the new k8s version is supported
		version, err := h.canUpgradeK8sVersion(toUpdate)
		toUpdate.Status.PreviousKubernetesVersion = version
		if err != nil {
			return h.updateErrorCond(toUpdate, mgmtv1.UpgradeCompleted, err)
		}
		return h.upgradeClient.UpdateStatus(toUpdate)
	}

	// Setup upgrade system charts repo
	if mgmtv1.UpgradeChartsRepoReady.GetStatus(upgrade) == "" ||
		mgmtv1.UpgradeChartsRepoReady.GetStatus(upgrade) == cond.StateError {
		if _, err = h.reconcileUpgradeSystemChartsRepo(toUpdate); err != nil {
			logrus.Debugf("Failed to setup upgrade system charts repo for upgrade %s: %v", upgrade.Name, err)
			return h.updateErrorCond(toUpdate, mgmtv1.UpgradeChartsRepoReady, err)
		}
	}

	// Initialize managed addons status
	if mgmtv1.UpgradeChartsRepoReady.IsTrue(upgrade) && (mgmtv1.ManagedAddonsIsReady.GetStatus(upgrade) == "" ||
		upgrade.Status.ManagedAddonStatus == nil) {
		return h.initUpgradeAddonStatus(toUpdate)
	}

	// Reconcile manifest upgrade
	if mgmtv1.UpgradeChartsRepoReady.IsTrue(upgrade) && !mgmtv1.ManifestUpgradeComplete.IsTrue(upgrade) {
		if _, err = h.reconcileManifestUpgrade(toUpdate); err != nil {
			logrus.Debugf("Failed to reconcile operator manifest upgrade for upgrade %s: %v", upgrade.Name, err)
			return h.updateErrorCond(toUpdate, mgmtv1.ManifestUpgradeComplete, err)
		}
	}

	// Reconcile nodes upgrade plan when is not initialized or error
	if mgmtv1.ManagedAddonsIsReady.IsTrue(upgrade) && mgmtv1.ManifestUpgradeComplete.IsTrue(upgrade) &&
		(mgmtv1.NodesUpgraded.GetStatus(upgrade) == "" || mgmtv1.NodesUpgraded.GetStatus(upgrade) == cond.StateError) {
		serverPlan := serverPlan(toUpdate)
		if err = h.reconcileNodesUpgradePlan(toUpdate, serverPlan); err != nil {
			return h.updateErrorCond(toUpdate, mgmtv1.NodesUpgraded, err)
		}

		agentPlan := agentPlan(toUpdate)
		if err = h.reconcileNodesUpgradePlan(toUpdate, agentPlan); err != nil {
			return h.updateErrorCond(toUpdate, mgmtv1.NodesUpgraded, err)
		}
	}

	// Set upgrade completed when all upgrade conditions are completed
	if mgmtv1.NodesUpgraded.IsTrue(upgrade) && mgmtv1.ManifestUpgradeComplete.IsTrue(upgrade) &&
		mgmtv1.ManagedAddonsIsReady.IsTrue(upgrade) && !mgmtv1.UpgradeCompleted.IsTrue(upgrade) {
		toUpdate.Status.State = cond.StateComplete
		toUpdate.Status.CompleteTime = metav1.Now()
		return h.updateReadyCond(toUpdate, mgmtv1.UpgradeCompleted, "Upgrade completed")
	}

	return upgrade, nil
}

func (h *upgradeHandler) reconcileNodesUpgradePlan(upgrade *mgmtv1.Upgrade, plan *upgradev1.Plan) error {
	foundPlan, err := h.planCache.Get(plan.Namespace, plan.Name)
	if err != nil && errors.IsNotFound(err) {
		if _, err = h.planClient.Create(plan); err != nil {
			return err
		}

		if _, err = h.updateUpgradingCond(upgrade, mgmtv1.NodesUpgraded,
			fmt.Sprintf(msgWaitingForNodes, upgrade.Spec.Version)); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(plan.Spec, foundPlan.Spec) {
		toUpdate := foundPlan.DeepCopy()
		toUpdate.Spec = plan.Spec
		if _, err = h.planClient.Update(toUpdate); err != nil {
			return err
		}
	}

	return nil
}

func (h *upgradeHandler) setLatestUpgradeLabel(upgrade *mgmtv1.Upgrade) error {
	sets := labels.Set{
		llmosLatestUpgradeLabel: "true",
	}

	upgrades, err := h.upgradeCache.List(sets.AsSelector())
	if err != nil {
		return err
	}

	for _, u := range upgrades {
		if u.Name == upgrade.Name {
			continue
		}
		toUpdate := u.DeepCopy()
		delete(toUpdate.Labels, llmosLatestUpgradeLabel)
		if _, err = h.upgradeClient.Update(toUpdate); err != nil {
			return err
		}
	}

	if upgrade.Labels == nil {
		upgrade.Labels = make(map[string]string)
	}
	upgrade.Labels[llmosLatestUpgradeLabel] = "true"
	if _, err = h.upgradeClient.Update(upgrade); err != nil {
		return err
	}
	return nil
}

func (h *upgradeHandler) reconcileManifestUpgrade(upgrade *mgmtv1.Upgrade) (*mgmtv1.Upgrade, error) {
	// init llmos operator upgrade status
	if mgmtv1.ManifestUpgradeComplete.GetStatus(upgrade) == "" {
		jobNames := make([]string, 0)
		for _, name := range operatorUpgradeCharts {
			upgrade.Status.UpgradeJobs = append(upgrade.Status.UpgradeJobs, mgmtv1.UpgradeJobStatus{
				HelmChartName: name,
				Name:          fmt.Sprintf("helm-install-%s", name),
				Complete:      false,
			})
			jobNames = append(jobNames, name)
		}
		msg := fmt.Sprintf(msgWaitingForManifest, strings.Join(jobNames, ", "))
		return h.updateUpgradingCond(upgrade, mgmtv1.ManifestUpgradeComplete, msg)
	}

	var err error
	for _, chartName := range operatorUpgradeCharts {
		upgrade, err = h.updateLLMOSHelmChart(upgrade, chartName)
		if err != nil {
			return h.updateErrorCond(upgrade, mgmtv1.ManifestUpgradeComplete, err)
		}
	}

	return upgrade, nil
}

func (h *upgradeHandler) updateLLMOSHelmChart(upgrade *mgmtv1.Upgrade, chartName string) (*mgmtv1.Upgrade, error) {
	// Only upgrade the operator chart after the CRDs are updated
	if chartName == llmosOperatorChartName {
		if upgrade.Status.UpgradeJobs == nil {
			return upgrade, nil
		}

		for _, upgradeJob := range upgrade.Status.UpgradeJobs {
			if upgradeJob.HelmChartName == llmosCrdChartName && !upgradeJob.Complete {
				logrus.Infof("waiting for the %s upgrade job to be complete first", llmosCrdChartName)
				return upgrade, nil
			}
		}
	}

	chart, err := h.helmChartCache.Get(constant.SystemNamespaceName, chartName)
	if err != nil {
		return upgrade, err
	}

	chartCpy := chart.DeepCopy()
	chartCpy.Spec.Repo = getUpgradeSystemChartsRepoUrl()
	chartCpy.Spec.Chart = chartName
	chartCpy.Spec.Version = upgrade.Spec.Version
	if chartCpy.Labels == nil {
		chartCpy.Labels = make(map[string]string)
	}
	chartCpy.Labels[llmosUpgradeNameLabel] = upgrade.Name
	chartCpy.Labels[llmosVersionLabel] = upgrade.Spec.Version

	if !reflect.DeepEqual(chartCpy.Spec, chart.Spec) || !reflect.DeepEqual(chartCpy.Labels, chart.Labels) {
		chart, err = h.helmChartClient.Update(chartCpy)
		if err != nil {
			logrus.Debugf("Failed to update upgrade chart %s: %v", chartName, err)
			return upgrade, err
		}

		found := false
		for i, j := range upgrade.Status.UpgradeJobs {
			if j.HelmChartName == chartName {
				// Job found, update its status and exit the loop
				found = true
				upgrade.Status.UpgradeJobs[i].Name = chart.Status.JobName
				upgrade.Status.UpgradeJobs[i].HelmChartName = chart.Name
				upgrade.Status.UpgradeJobs[i].Complete = false
				break
			}
		}

		if !found {
			upgrade.Status.UpgradeJobs = append(upgrade.Status.UpgradeJobs, mgmtv1.UpgradeJobStatus{
				Name:          chart.Status.JobName,
				HelmChartName: chart.Name,
				Complete:      false,
			})
		}

		return h.upgradeClient.UpdateStatus(upgrade)
	}

	return upgrade, nil
}

func (h *upgradeHandler) onDelete(_ string, upgrade *mgmtv1.Upgrade) (*mgmtv1.Upgrade, error) {
	if upgrade == nil || upgrade.DeletionTimestamp == nil {
		return nil, nil
	}

	// delete upgrade plan
	sets := labels.Set{
		llmosUpgradeNameLabel: upgrade.Name,
	}
	plans, err := h.planCache.List(constant.SUCNamespace, sets.AsSelector())
	if err != nil {
		return upgrade, err
	}

	for _, plan := range plans {
		if err = h.planClient.Delete(plan.Namespace, plan.Name, nil); err != nil {
			return upgrade, err
		}
	}
	return nil, nil
}

func initStatus(upgrade *mgmtv1.Upgrade) {
	upgrade.Status = mgmtv1.UpgradeStatus{
		Conditions:   make([]common.Condition, 0),
		UpgradeJobs:  make([]mgmtv1.UpgradeJobStatus, 0),
		NodeStatuses: make(map[string]mgmtv1.NodeUpgradeStatus),
	}
	mgmtv1.UpgradeCompleted.SetStatusBool(upgrade, false)
	upgrade.Status.PreviousVersion = settings.ServerVersion.Get()
	upgrade.Status.AppliedVersion = upgrade.Spec.Version
	upgrade.Status.State = cond.StateUpgrading
	upgrade.Status.StartTime = metav1.Now()
}

func (h *upgradeHandler) canUpgradeK8sVersion(upgrade *mgmtv1.Upgrade) (string, error) {
	version, err := h.discovery.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get system kubernetes version: %v", err)
	}

	if upgrade.Spec.KubernetesVersion == "" {
		return version.String(), nil
	}

	currentVersion, err := gversion.NewSemver(version.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse system kubernetes version: %v", err)
	}

	upgradeVersion, err := gversion.NewSemver(upgrade.Spec.KubernetesVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse upgrade kubernetes version: %v", err)
	}

	if upgradeVersion.LessThan(currentVersion) {
		return "", fmt.Errorf("upgrade kubernetes version %s is less than current version %s",
			upgrade.Spec.KubernetesVersion, version.String())
	}

	return version.String(), nil
}

func (h *upgradeHandler) initUpgradeAddonStatus(upgrade *mgmtv1.Upgrade) (*mgmtv1.Upgrade, error) {
	selector := labels.SelectorFromSet(map[string]string{
		constant.SystemAddonLabel: "true",
	})

	systemAddons, err := h.addonCache.List(constant.SystemNamespaceName, selector)
	if err != nil {
		return upgrade, err
	}

	for _, addon := range systemAddons {
		upgrade.Status.ManagedAddonStatus = append(upgrade.Status.ManagedAddonStatus, mgmtv1.UpgradeManagedAddonStatus{
			Name:     addon.Name,
			JobName:  addon.Status.JobName,
			Disabled: !addon.Spec.Enabled,
			Complete: !addon.Spec.Enabled, // skip upgrade for disabled addon
		})
	}

	return h.updateUpgradingCond(upgrade, mgmtv1.ManagedAddonsIsReady, msgWaitingForAddons)
}
