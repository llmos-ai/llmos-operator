package upgrade

import (
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	helmChartLabelKey = "helmcharts.helm.cattle.io/chart"
)

type jobHandler struct {
	upgradeClient ctlmgmtv1.UpgradeClient
	upgradeCache  ctlmgmtv1.UpgradeCache
	*commonHandler
}

// watchUpgradeJobs watches for manifest upgrade jobs and updates the upgrade status when they are completed
func (h *jobHandler) watchUpgradeJobs(_ string, job *batchv1.Job) (*batchv1.Job, error) {
	if job == nil || job.DeletionTimestamp != nil || job.Labels == nil || job.Namespace != constant.SystemNamespaceName {
		return nil, nil
	}

	chartName := job.Labels[helmChartLabelKey]

	// only watch jobs that are part of the upgrade process
	if chartName == "" || !utils.ArrayStringContains(operatorUpgradeCharts, chartName) {
		return nil, nil
	}

	upgrade, err := h.getLatestUpgrade("")
	if err != nil {
		return nil, err
	}

	if upgrade == nil {
		logrus.Debugf("no latest upgrade found, skip syncing job status")
		return job, nil
	}

	if upgradeJobIsCompleteAfter(upgrade, job) {
		if _, err = h.syncUpgradeJobStatus(upgrade, job, chartName); err != nil {
			return job, err
		}
	}

	return job, nil
}

func (h *jobHandler) syncUpgradeJobStatus(upgrade *mgmtv1.Upgrade, job *batchv1.Job,
	chartName string) (*mgmtv1.Upgrade, error) {
	logrus.Debugf("job %s is complete after upgrade %s, updating upgrade status", job.Name, upgrade.Name)
	upgradeCpy := upgrade.DeepCopy()
	addJobStatusToUpgrade(upgradeCpy, job.Name, chartName, *job.Status.CompletionTime)

	return h.upgradeClient.UpdateStatus(upgradeCpy)
}

func addJobStatusToUpgrade(upgrade *mgmtv1.Upgrade, jobName, chartName string, completeTime metav1.Time) {
	logrus.Debugf("adding job %s to upgrade %s status", jobName, upgrade.Name)
	found := false
	jobStatus := mgmtv1.UpgradeJobStatus{
		Complete:       true,
		HelmChartName:  chartName,
		Name:           jobName,
		LastUpdateTime: completeTime,
	}

	for i, j := range upgrade.Status.UpgradeJobs {
		if j.Name == jobName || j.HelmChartName == chartName {
			upgrade.Status.UpgradeJobs[i] = jobStatus
			found = true
			break
		}
	}

	if !found {
		logrus.Debugf("job %s not found in upgrade %s status, adding it", jobName, upgrade.Name)
		upgrade.Status.UpgradeJobs = append(upgrade.Status.UpgradeJobs, jobStatus)
	}
}
