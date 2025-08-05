package managedaddon

import (
	"fmt"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const (
	HelmKindName = "HelmChart"
)

// OnAddonJobChange helps to sync associated helm job status to addon status
func (h *AddonHandler) OnAddonJobChange(_ string, job *batchv1.Job) (*batchv1.Job, error) {
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

func (h *AddonHandler) syncJobStatusToAddon(job *batchv1.Job, name string) error {
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
	addonCpy.Status.JobName = job.Name

	if isJobCompleted(job) {
		addonCpy.Status.CompletionTime = job.Status.CompletionTime
		_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateComplete, condition.StateComplete, nil)
		return err
	} else if job.Status.Failed > 0 {
		addonCpy.Status.CompletionTime = nil
		_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateFailed, condition.StateError,
			fmt.Errorf("helm chart job %s failed", job.Name))
		return err
	}

	addonCpy.Status.CompletionTime = nil
	_, err = h.setAddonCondStatus(addonCpy, mgmtv1.AddonStateInProgress, condition.StateProcessing, nil)
	return err
}

func isJobCompleted(job *batchv1.Job) bool {
	return job.Status.Succeeded == 1 && job.Status.CompletionTime != nil
}
