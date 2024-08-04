package managedaddon

import (
	"fmt"
	"reflect"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

const (
	HelmKindName = "HelmChart"
)

// OnAddonJobChange helps to sync associated helm job status to addon status
func (h *handler) OnAddonJobChange(_ string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil || job.DeletionTimestamp != nil {
		return nil, nil
	}

	for _, ownerRef := range job.OwnerReferences {
		if ownerRef.Kind == HelmKindName && ownerRef.APIVersion == helmv1.SchemeGroupVersion.String() {
			return job, h.syncJobStatusToAddon(job, ownerRef.Name)
		}
	}
	return job, nil
}

func (h *handler) syncJobStatusToAddon(job *batchv1.Job, name string) error {
	logrus.Debugf("syncing job %s status to addon %s", job.Name, name)
	addon, err := h.managedAddonCache.Get(job.Namespace, name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get addon %s/%s: %w", job.Namespace, name, err)
	}

	if addon == nil {
		logrus.Debugf("empty addon %s of job %s, skip syncing status", name, job.Name)
		return nil
	}

	addonCpy := addon.DeepCopy()
	addonCpy.Status.Ready = job.Status.Ready
	addonCpy.Status.Succeeded = job.Status.Succeeded

	if job.Status.CompletionTime != nil {
		addonCpy.Status.CompletionTime = job.Status.CompletionTime
		_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateComplete, "", nil)
		return err
	} else if job.Status.Failed > 0 {
		_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateFailed, "", fmt.Errorf("helm chart job %s failed", job.Name))
		return err
	}

	if !reflect.DeepEqual(addonCpy.Status, addon.Status) {
		if _, err := h.managedAddons.UpdateStatus(addonCpy); err != nil {
			return fmt.Errorf("failed to update addon %s/%s status: %w", job.Namespace, name, err)
		}
	}

	return nil
}

// addonHelmChartOnDelete helps to disable addon when helm chart is deleted
func (h *handler) addonHelmChartOnDelete(_ string, chart *helmv1.HelmChart) (*helmv1.HelmChart, error) {
	if chart == nil || chart.DeletionTimestamp == nil {
		return nil, nil
	}

	// only need to handle helm chart owned by managed addon
	if chart.Labels == nil || len(chart.OwnerReferences) == 0 || chart.Labels[constant.ManagedAddonLabel] != "true" {
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
