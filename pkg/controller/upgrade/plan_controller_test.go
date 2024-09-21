package upgrade

import (
	"testing"

	"github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

func Test_PlanHandler_OnChanged(t *testing.T) {
	type input struct {
		key     string
		plan    *upgradeapiv1.Plan
		upgrade *mgmtv1.Upgrade
		nodes   []*v1.Node
	}
	type output struct {
		plan    *upgradeapiv1.Plan
		upgrade *mgmtv1.Upgrade
		err     error
	}
	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "server upgrade plan is not complete",
			given: input{
				key:     testPlanName,
				plan:    newServerPLan().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-1").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-2").Managed().ControlPlane().Build(),
					newNodeBuilder("node-3").Managed().ControlPlane().Build(),
				},
			},
			expected: output{
				plan:    nil,
				upgrade: newTestUpgradeBuilder().Build(),
				err:     nil,
			},
		},
		{
			name: "server upgrade plan is complete but has pending nodes",
			given: input{
				key:     testPlanName,
				plan:    newServerPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-1").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-2").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-3").Managed().ControlPlane().Build(),
				},
			},
			expected: output{
				plan:    newServerPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				err:     nil,
			},
		},
		{
			name: "server upgrade plan is complete",
			given: input{
				key:     testPlanName,
				plan:    newServerPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-s1").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-s2").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-s3").Managed().ControlPlane().WithLabel(upgrade.LabelPlanName(newServerPLan().Build().Name), testServerPlanHash).Build(),
					newNodeBuilder("node-w1").Managed().Build(),
				},
			},
			expected: output{
				plan:    newServerPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().WithPlanStatus(newServerPLan().Complete().Build()).Build(),
				err:     nil,
			},
		},
		{
			name: "agent upgrade plan is running",
			given: input{
				key:     testPlanName,
				plan:    newAgentPLan().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-s1").Managed().ControlPlane().Build(),
					newNodeBuilder("node-w1").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
					newNodeBuilder("node-w2").Managed().Build(),
					newNodeBuilder("node-w3").Managed().Build(),
				},
			},
			expected: output{
				plan:    nil,
				upgrade: newTestUpgradeBuilder().Build(),
				err:     nil,
			},
		},
		{
			name: "agent upgrade plan has pending nodes",
			given: input{
				key:     testPlanName,
				plan:    newAgentPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-s1").Managed().ControlPlane().Build(),
					newNodeBuilder("node-w1").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
					newNodeBuilder("node-w2").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
					newNodeBuilder("node-w3").Managed().Build(),
				},
			},
			expected: output{
				plan:    newAgentPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				err:     nil,
			},
		},
		{
			name: "agent upgrade plan is complete",
			given: input{
				key:     testPlanName,
				plan:    newAgentPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().Build(),
				nodes: []*v1.Node{
					newNodeBuilder("node-s1").Managed().ControlPlane().Build(),
					newNodeBuilder("node-w1").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
					newNodeBuilder("node-w2").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
					newNodeBuilder("node-w3").Managed().WithLabel(upgrade.LabelPlanName(newAgentPLan().Build().Name), testAgentPlanHash).Build(),
				},
			},
			expected: output{
				plan:    newAgentPLan().Complete().Build(),
				upgrade: newTestUpgradeBuilder().WithPlanStatus(newAgentPLan().Complete().Build()).WithNodesUpgradedCondition().Build(),
				err:     nil,
			},
		},
	}
	for _, tc := range testCases {
		var clientset = fake.NewSimpleClientset(tc.given.plan, tc.given.upgrade)
		var nodes []runtime.Object
		for _, node := range tc.given.nodes {
			nodes = append(nodes, node)
		}
		var k8sclientset = k8sfake.NewSimpleClientset(nodes...)
		var commonHandler = newFakeCommonHandler(clientset)
		var h = &planHandler{
			upgradeClient: fakeclients.UpgradeClient(clientset.ManagementV1().Upgrades),
			upgradeCache:  fakeclients.UpgradeCache(clientset.ManagementV1().Upgrades),
			planClient:    fakeclients.PlanClient(clientset.UpgradeV1().Plans),
			nodeCache:     fakeclients.NodeCache(k8sclientset.CoreV1().Nodes),
			commonHandler: commonHandler,
		}
		var actual output
		var err error
		actual.plan, actual.err = h.watchUpgradePlans(tc.given.key, tc.given.plan)
		assert.Nil(t, actual.err)

		if tc.expected.upgrade != nil {
			actual.upgrade, err = h.upgradeCache.Get(tc.given.upgrade.Name)
			sanitizePlanStatus(actual.upgrade.Status, tc.given.plan.Name)
			assert.Nil(t, err)
		}
		assert.Equal(t, tc.expected, actual, "case %q", tc.name)
	}
}

// sanitizePlanStatus resets the plan status timestamps that may fail the comparison
func sanitizePlanStatus(status mgmtv1.UpgradeStatus, planName string) {
	for i, cond := range status.PlanStatus {
		if cond.Name == planName {
			status.PlanStatus[i].LastUpdateTime = metav1.NewTime(fakeTime)
		}
	}
}
