package upgrade

import (
	"fmt"
	"strings"
	"time"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

const (
	testJobName        = "test-job"
	testPlanName       = "test-plan"
	testUpgradeName    = "test-upgrade"
	testVersion        = "test-version"
	testServerPlanHash = "test-server-hash"
	testAgentPlanHash  = "test-agent-hash"
)

// jobBuilder helps to build a job object
type jobBuilder struct {
	job *batchv1.Job
}

func newJobBuilder(name string) *jobBuilder {
	return &jobBuilder{
		job: &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: constant.SystemNamespaceName,
			},
		},
	}
}

func (j *jobBuilder) WithLabel(key, value string) *jobBuilder {
	if j.job.Labels == nil {
		j.job.Labels = make(map[string]string)
	}
	j.job.Labels[key] = value
	return j
}

func (j *jobBuilder) Running() *jobBuilder {
	j.job.Status.Active = 1
	return j
}

func (j *jobBuilder) Completed(time2 time.Time) *jobBuilder {
	j.job.Status.Succeeded = 1
	j.job.Status.Conditions = append(j.job.Status.Conditions, batchv1.JobCondition{
		Type:   batchv1.JobComplete,
		Status: "True",
	})
	j.job.Status.CompletionTime = &metav1.Time{
		Time: time2.Add(10 * time.Second),
	}
	return j
}

func (j *jobBuilder) Failed(reason, message string) *jobBuilder {
	j.job.Status.Failed = 1
	j.job.Status.Conditions = append(j.job.Status.Conditions, batchv1.JobCondition{
		Type:    batchv1.JobFailed,
		Status:  "True",
		Reason:  reason,
		Message: message,
	})
	j.job.Status.CompletionTime = nil
	j.job.Status.Succeeded = 0
	return j
}

func (j *jobBuilder) Build() *batchv1.Job {
	return j.job
}

// upgradeBuilder helps to build an upgrade object
type upgradeBuilder struct {
	upgrade *mgmtv1.Upgrade
}

func newTestUpgradeBuilder() *upgradeBuilder {
	return newUpgradeBuilder(testUpgradeName).
		WithLabel(llmosLatestUpgradeLabel, "true").
		Version(testVersion)
}

func newUpgradeBuilder(name string) *upgradeBuilder {
	return &upgradeBuilder{
		upgrade: &mgmtv1.Upgrade{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func (p *upgradeBuilder) WithLabel(key, value string) *upgradeBuilder {
	if p.upgrade.Labels == nil {
		p.upgrade.Labels = make(map[string]string)
	}
	p.upgrade.Labels[key] = value
	return p
}

func (p *upgradeBuilder) Version(version string) *upgradeBuilder {
	p.upgrade.Spec.Version = version
	return p
}

func (p *upgradeBuilder) KubeVersion(version string) *upgradeBuilder {
	p.upgrade.Spec.KubernetesVersion = version
	return p
}

func (p *upgradeBuilder) InitStatus() *upgradeBuilder {
	initStatus(p.upgrade)
	p.upgrade.Status.StartTime = metav1.Time{}
	return p
}

func (p *upgradeBuilder) InitStatusWithTime(time time.Time) *upgradeBuilder {
	initStatus(p.upgrade)
	p.upgrade.Status.StartTime = metav1.Time{
		Time: time,
	}
	return p
}

func (p *upgradeBuilder) WithUpgradeJobStatus(jobName, chartName string, time2 time.Time) *upgradeBuilder {
	if p.upgrade.Status.UpgradeJobs == nil {
		p.upgrade.Status.UpgradeJobs = make([]mgmtv1.UpgradeJobStatus, 0)
	}

	addJobStatusToUpgrade(p.upgrade, jobName, chartName, metav1.Time{
		Time: time2.Add(10 * time.Second),
	})
	return p
}

var fakeTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

func (p *upgradeBuilder) WithPlanStatus(plan *upgradev1.Plan) *upgradeBuilder {
	p.upgrade.Status.PlanStatus = append(p.upgrade.Status.PlanStatus, mgmtv1.UpgradePlanStatus{
		Name:           plan.Name,
		LatestVersion:  plan.Status.LatestVersion,
		LatestHash:     plan.Status.LatestHash,
		LastUpdateTime: metav1.NewTime(fakeTime),
		Complete:       true,
	})
	return p
}

func (p *upgradeBuilder) WithConditionReady(cond condition.Cond) *upgradeBuilder {
	cond.True(p.upgrade)
	cond.Reason(p.upgrade, "Ready")
	cond.SetStatusBool(p.upgrade, true)
	cond.Message(p.upgrade, "")
	return p
}

func (p *upgradeBuilder) WithConditionNotReady(cond condition.Cond) *upgradeBuilder {
	cond.False(p.upgrade)
	cond.Reason(p.upgrade, "NotReady")
	cond.Message(p.upgrade, "Condition not ready")
	return p
}

func (p *upgradeBuilder) WithManagedAddonStatus(addons []*mgmtv1.ManagedAddon) *upgradeBuilder {
	mgmtv1.UpgradeChartsRepoReady.True(p.upgrade)
	mgmtv1.ManagedAddonsIsReady.Reason(p.upgrade, condition.StateProcessing)
	mgmtv1.ManagedAddonsIsReady.SetStatus(p.upgrade, condition.StateProcessing)
	mgmtv1.ManagedAddonsIsReady.Message(p.upgrade, msgWaitingForAddons)
	for _, addon := range addons {
		p.upgrade.Status.ManagedAddonStatus = append(p.upgrade.Status.ManagedAddonStatus, mgmtv1.UpgradeManagedAddonStatus{
			Name:     addon.Name,
			JobName:  addon.Status.JobName,
			Disabled: !addon.Spec.Enabled,
			Complete: !addon.Spec.Enabled,
		})
	}
	return p
}

func (p *upgradeBuilder) WithManifestStatus() *upgradeBuilder {
	jobNames := make([]string, 0)
	for _, name := range operatorUpgradeCharts {
		p.upgrade.Status.UpgradeJobs = append(p.upgrade.Status.UpgradeJobs, mgmtv1.UpgradeJobStatus{
			HelmChartName: name,
			Name:          fmt.Sprintf("helm-install-%s", name),
			Complete:      false,
		})
		jobNames = append(jobNames, name)
	}
	msg := fmt.Sprintf(msgWaitingForManifest, strings.Join(jobNames, ", "))
	mgmtv1.ManifestUpgradeComplete.True(p.upgrade)
	mgmtv1.ManifestUpgradeComplete.Reason(p.upgrade, condition.StateProcessing)
	mgmtv1.ManifestUpgradeComplete.SetStatus(p.upgrade, condition.StateProcessing)
	mgmtv1.ManifestUpgradeComplete.Message(p.upgrade, msg)
	return p
}

func (p *upgradeBuilder) WithNodesPlanStatus() *upgradeBuilder {
	mgmtv1.NodesUpgraded.SetStatus(p.upgrade, condition.StateProcessing)
	mgmtv1.NodesUpgraded.Reason(p.upgrade, condition.StateProcessing)
	mgmtv1.NodesUpgraded.Message(p.upgrade, fmt.Sprintf(msgWaitingForNodes, testVersion))
	return p
}

func (p *upgradeBuilder) WithChartsRepoCondition() *upgradeBuilder {
	mgmtv1.UpgradeChartsRepoReady.SetStatus(p.upgrade, condition.StateProcessing)
	mgmtv1.UpgradeChartsRepoReady.Reason(p.upgrade, condition.StateProcessing)
	mgmtv1.UpgradeChartsRepoReady.Message(p.upgrade, fmt.Sprintf(msgWaitingForRepo, testVersion))
	return p
}

func (p *upgradeBuilder) WithNodesUpgradedCondition() *upgradeBuilder {
	mgmtv1.NodesUpgraded.SetStatus(p.upgrade, "True")
	mgmtv1.NodesUpgraded.Reason(p.upgrade, "Ready")
	mgmtv1.NodesUpgraded.Message(p.upgrade, fmt.Sprintf("All nodes upgraded to version %s()", testVersion))
	return p
}

func (p *upgradeBuilder) Build() *mgmtv1.Upgrade {
	return p.upgrade
}

// planBuilder helps to build a plan object
type planBuilder struct {
	plan *upgradev1.Plan
}

func newServerPLan() *planBuilder {
	plan := serverPlan(newTestUpgradeBuilder().Build())
	plan.Status.LatestHash = testServerPlanHash
	plan.Status.LatestVersion = testVersion
	return &planBuilder{
		plan: plan,
	}
}

func newAgentPLan() *planBuilder {
	plan := agentPlan(newTestUpgradeBuilder().Build())
	plan.Status.LatestHash = testAgentPlanHash
	plan.Status.LatestVersion = testVersion
	return &planBuilder{
		plan: plan,
	}
}

func (p *planBuilder) WithLabel(key, value string) *planBuilder {
	if p.plan.Labels == nil {
		p.plan.Labels = make(map[string]string)
	}
	p.plan.Labels[key] = value
	return p
}

func (p *planBuilder) Version(version string) *planBuilder {
	p.plan.Spec.Version = version
	return p
}

func (p *planBuilder) Hash(hash string) *planBuilder {
	p.plan.Status.LatestHash = hash
	return p
}

func (p *planBuilder) Complete() *planBuilder {
	upgradev1.PlanComplete.True(p.plan)
	return p
}

func (p *planBuilder) Build() *upgradev1.Plan {
	return p.plan
}

// nodeBuilder helps to build a node object
type nodeBuilder struct {
	node *v1.Node
}

func newNodeBuilder(name string) *nodeBuilder {
	return &nodeBuilder{
		node: &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func (n *nodeBuilder) ControlPlane() *nodeBuilder {
	n.WithLabel(constant.KubeControlPlaneNodeLabelKey, "true")
	return n
}

func (n *nodeBuilder) Managed() *nodeBuilder {
	n.WithLabel(constant.LLMOSManagedLabel, "true")
	return n
}

func (n *nodeBuilder) WithLabel(key, value string) *nodeBuilder {
	if n.node.Labels == nil {
		n.node.Labels = make(map[string]string)
	}
	n.node.Labels[key] = value
	return n
}

func (n *nodeBuilder) Build() *v1.Node {
	return n.node
}

func newFakeCommonHandler(clientset *fake.Clientset) *commonHandler {
	return &commonHandler{
		upgradeClient: fakeclients.UpgradeClient(clientset.ManagementV1().Upgrades),
		upgradeCache:  fakeclients.UpgradeCache(clientset.ManagementV1().Upgrades),
	}
}
