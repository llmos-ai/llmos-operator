package managedaddon

import (
	"bytes"
	"fmt"
	"reflect"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

// getHelmChart helps to get helm chart object managed by the addon
func (h *handler) getHelmChart(addon *mgmtv1.ManagedAddon) (*helmv1.HelmChart, bool, error) {
	chart, err := h.helmChartCache.Get(addon.Namespace, addon.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logrus.Debugf("helm chart %s/%s not found", addon.Namespace, addon.Name)
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to find helm chart %s/%s: %v", addon.Namespace, addon.Name, err)
	}

	owned := false
	for _, v := range chart.GetOwnerReferences() {
		if v.Kind == addon.Kind && v.APIVersion == addon.APIVersion && v.UID == addon.UID {
			owned = true
			break
		}
	}
	return chart, owned, nil
}

// setAddonCondStatus helps to set addon condition status & state
func (h *handler) setAddonCondStatus(addon *mgmtv1.ManagedAddon, state mgmtv1.AddonState,
	reason string, err error) (*mgmtv1.ManagedAddon, error) {
	addonCpy := addon.DeepCopy()
	addonCpy.Status.State = state

	switch state {
	case mgmtv1.AddonStateEnabled, mgmtv1.AddonStateDeployed:
		// enabled & deployed state will be treated as deployed condition
		mgmtv1.AddonCondChartDeployed.SetError(addonCpy, reason, err)
	case mgmtv1.AddonStateDisabled:
		// disabled & deleting state will erase all conditions
		addonCpy.Status = mgmtv1.ManagedAddonStatus{
			State: state,
		}
		mgmtv1.AddonCondChartDeployed.SetStatusBool(addonCpy, false)
	case mgmtv1.AddonStateInProgress:
		// in progress state will be treated as in progress condition
		mgmtv1.AddonCondInProgress.SetError(addonCpy, reason, err)
		mgmtv1.AddonCondReady.SetStatusBool(addonCpy, false)
	case mgmtv1.AddonStateComplete, mgmtv1.AddonStateError, mgmtv1.AddonStateFailed:
		// all other state will be treated as ready condition
		mgmtv1.AddonCondReady.SetError(addonCpy, reason, err)
		if err == nil {
			mgmtv1.AddonCondInProgress.SetStatusBool(addonCpy, false)
		} else {
			mgmtv1.AddonCondInProgress.SetStatusBool(addonCpy, true)
		}
	}

	if !reflect.DeepEqual(addon.Status, addonCpy.Status) {
		return h.managedAddons.UpdateStatus(addonCpy)
	}

	return addonCpy, nil
}

func ValidateChartValues(valuesContent string) error {
	if valuesContent != "" {
		values := make(map[string]interface{})
		buf := bytes.NewBufferString(valuesContent)
		if err := yaml.NewDecoder(buf).Decode(values); err != nil {
			return fmt.Errorf("invalid chart valuesContent: %w", err)
		}
		logrus.Debugf("chart values: %+v", values)
	}
	return nil
}
