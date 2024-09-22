package managedaddon

import (
	"context"
	"fmt"
	"reflect"
	"time"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	ctlbatchv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlhelmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/helm.cattle.io/v1"
	ctlmanagementv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	addOnChange            = "managedAddon.onChange"
	addonJobOnChange       = "managedAddon.JobOnChange"
	addonHelmChartOnDelete = "managedAddon.helmChartOnDelete"

	defaultWaitTime = 5 * time.Second
)

type handler struct {
	managedAddon      ctlmanagementv1.ManagedAddonController
	managedAddons     ctlmanagementv1.ManagedAddonClient
	managedAddonCache ctlmanagementv1.ManagedAddonCache
	helmCharts        ctlhelmv1.HelmChartClient
	helmChartCache    ctlhelmv1.HelmChartCache
	jobs              ctlbatchv1.JobClient
	jobCache          ctlbatchv1.JobCache
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	addon := mgmt.MgmtFactory.Management().V1().ManagedAddon()
	helm := mgmt.HelmFactory.Helm().V1().HelmChart()
	job := mgmt.BatchFactory.Batch().V1().Job()
	h := &handler{
		managedAddon:      addon,
		managedAddons:     addon,
		managedAddonCache: addon.Cache(),
		helmCharts:        helm,
		helmChartCache:    helm.Cache(),
		jobs:              job,
		jobCache:          job.Cache(),
	}

	addon.OnChange(ctx, addOnChange, h.OnChange)
	job.OnChange(ctx, addonJobOnChange, h.OnAddonJobChange)
	helm.OnRemove(ctx, addonHelmChartOnDelete, h.addonHelmChartOnDelete)

	return h.registerSystemAddons(ctx)
}

func (h *handler) OnChange(_ string, addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	if addon == nil || addon.DeletionTimestamp != nil {
		return addon, nil
	}

	// process addon if it is disabled
	if addon.Spec.Enabled {
		return h.enableManagedAddon(addon)
	}

	// disable managed addon when it is set to disabled
	return h.disabledManagedAddon(addon)
}

// Each managedAddon will have 3 conditions, and for each condition it will contain different state:
// - ChartDeployed, indicates that the chart has been enabled or disabled
//   - AddonStateEnabled: addon is enabled
//   - AddonStateDeployed: chart is deployed, but not ready
//   - AddonStateDisabled: addon is disabled
//
// - InProgress, indicates that an operation is in progress
//   - AddonStateInProgress: job is in progress
//
// - Ready, indicates that the addon is ready
//   - AddonStateComplete: chart is ready
//   - AddonStateError: chart is in error state
//   - AddonStateFailed: chart job is failed
func (h *handler) enableManagedAddon(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	// init addon state to enabled first
	if !mgmtv1.AddonCondChartDeployed.IsTrue(addon) {
		if err := ValidateChartValues(addon.Spec.ValuesContent); err != nil {
			logrus.Debugf("failed to validate chart values for addon %s: %s", addon.Name, err)
			return h.setAddonCondStatus(addon, mgmtv1.AddonStateDeployed, "", err)
		}
		return h.setAddonCondStatus(addon, mgmtv1.AddonStateEnabled, "", nil)
	}

	addonCpy := addon.DeepCopy()
	switch addonCpy.Status.State {
	case mgmtv1.AddonStateEnabled, mgmtv1.AddonStateDeployed:
		return h.enableAddonChart(addonCpy)
	case mgmtv1.AddonStateDisabled:
		return addonCpy, nil
	default:
		return h.reconcileAddonChart(addonCpy)
	}
}

func (h *handler) disabledManagedAddon(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	// check for existing chart
	chart, owned, err := h.getHelmChart(addon)
	if err != nil {
		return addon, err
	}

	if chart != nil && owned {
		if err = h.helmCharts.Delete(chart.Namespace, chart.Name, &metav1.DeleteOptions{}); err != nil {
			return addon, err
		}
	}

	return h.setAddonCondStatus(addon, mgmtv1.AddonStateDisabled, "", nil)
}

func (h *handler) enableAddonChart(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	// check for existing chart
	logrus.Debugf("check existing chart for addon %s", addon.Name)
	chart, owned, err := h.getHelmChart(addon)
	if err != nil {
		return addon, err
	}

	fullName := getChartFullName(addon)
	if chart != nil && !chart.DeletionTimestamp.IsZero() {
		logrus.Warnf("chart %s is currently under removing, will enqueue after %s", fullName, defaultWaitTime.String())
		h.managedAddon.EnqueueAfter(addon.Namespace, addon.Name, defaultWaitTime)
		return addon, nil
	}

	// return and save error message if chart exists but not owned by this addon
	if chart != nil && !owned {
		err = fmt.Errorf("chart %s exists but not owned by this addon", fullName)
		return h.setAddonCondStatus(addon, mgmtv1.AddonStateDeployed, "", err)
	}

	if chart == nil {
		if _, err = h.deployHelmChart(addon); err != nil {
			err = fmt.Errorf("failed to create helm chart %s for addon %s: %w", fullName, addon.Name, err)
			return h.setAddonCondStatus(addon, mgmtv1.AddonStateDeployed, "", err)
		}
		logrus.Debugf("helm chart %s created by addon %s", fullName, addon.Name)
	}

	return h.setAddonCondStatus(addon, mgmtv1.AddonStateInProgress, "", nil)
}

func (h *handler) deployHelmChart(addon *mgmtv1.ManagedAddon) (*helmv1.HelmChart, error) {
	logrus.Debugf("creating new helm chart %s for addon %s", getChartFullName(addon), addon.Name)
	return h.helmCharts.Create(&helmv1.HelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addon.Name,
			Namespace: addon.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: addon.APIVersion,
					Kind:       addon.Kind,
					Name:       addon.Name,
					UID:        addon.UID,
				},
			},
			Labels: map[string]string{
				constant.ManagedAddonLabel: "true",
			},
		},
		Spec: helmv1.HelmChartSpec{
			TargetNamespace: addon.Namespace,
			Chart:           addon.Spec.Chart,
			Repo:            addon.Spec.Repo,
			Version:         addon.Spec.Version,
			ValuesContent:   addon.Spec.ValuesContent,
		},
	})
}

func (h *handler) reconcileAddonChart(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	chart, _, err := h.getHelmChart(addon)
	if err != nil {
		return addon, err
	}

	fullName := getChartFullName(addon)
	if chart == nil {
		err = fmt.Errorf("helm chart %s not found", fullName)
		return h.setAddonCondStatus(addon, mgmtv1.AddonStateError, "Error", err)
	}

	chartCpy := chart.DeepCopy()
	chartCpy.Spec.ValuesContent = addon.Spec.ValuesContent
	chartCpy.Spec.Version = addon.Spec.Version
	chartCpy.Spec.Repo = addon.Spec.Repo
	chartCpy.Spec.Chart = addon.Spec.Chart

	if !reflect.DeepEqual(chartCpy.Spec, chart.Spec) {
		logrus.Debugf("updating helm chart %s spec for addon %s", fullName, addon.Name)
		if _, err = h.helmCharts.Update(chartCpy); err != nil {
			err = fmt.Errorf("failed to update helm chart %s", getChartFullName(addon))
			return h.setAddonCondStatus(addon, mgmtv1.AddonStateError, "", err)
		}
	}

	return nil, nil
}

func getChartFullName(addon *mgmtv1.ManagedAddon) string {
	return fmt.Sprintf("%s-%s", addon.Namespace, addon.Name)
}
