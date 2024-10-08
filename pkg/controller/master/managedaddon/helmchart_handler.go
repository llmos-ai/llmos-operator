package managedaddon

import (
	"fmt"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

// addonHelmChartOnDelete helps to disable addon when helm chart is deleted
func (h *handler) addonHelmChartOnDelete(_ string, chart *helmv1.HelmChart) (*helmv1.HelmChart, error) {
	if chart == nil || chart.DeletionTimestamp == nil {
		return nil, nil
	}

	// only need to handle helm chart owned by managed addon
	if chart.Labels == nil || len(chart.OwnerReferences) == 0 || chart.Labels[constant.ManagedAddonLabel] != strTrue {
		return nil, nil
	}

	ownerRef := chart.OwnerReferences[0]
	addon, err := h.managedAddonCache.Get(chart.Namespace, ownerRef.Name)
	if err != nil && !errors.IsNotFound(err) {
		return chart, fmt.Errorf("failed to get addon %s/%s: %w", chart.Namespace, ownerRef.Name, err)
	}

	if addon == nil {
		logrus.Warnf("empty addon %s of helm chart %s, skip disabling", ownerRef.Name, chart.Name)
		return chart, nil
	}

	addonCpy := addon.DeepCopy()
	if addonCpy.Spec.Enabled {
		addonCpy.Spec.Enabled = false
		if _, err = h.managedAddons.Update(addonCpy); err != nil {
			return chart, fmt.Errorf("failed to update addon %s/%s: %w", chart.Namespace, ownerRef.Name, err)
		}
	}

	_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateDisabled, "", nil)

	return chart, err
}
