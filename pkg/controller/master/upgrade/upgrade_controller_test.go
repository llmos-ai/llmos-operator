package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

var fakeManagedAddons = []*mgmtv1.ManagedAddon{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-addon1",
			Namespace: constant.SystemNamespaceName,
			Labels: map[string]string{
				constant.SystemAddonLabel: "true",
			},
		},
		Spec: mgmtv1.ManagedAddonSpec{
			Enabled: true,
			Version: "1.0.0",
		},
		Status: mgmtv1.ManagedAddonStatus{
			JobName: "test-addon1-job",
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-addon2",
			Namespace: constant.SystemNamespaceName,
			Labels: map[string]string{
				constant.SystemAddonLabel: "true",
			},
		},
		Spec: mgmtv1.ManagedAddonSpec{
			Enabled: true,
			Version: "1.0.0",
		},
		Status: mgmtv1.ManagedAddonStatus{
			JobName: "test-addon2-job",
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-addon3",
			Namespace: constant.SystemNamespaceName,
			Labels: map[string]string{
				constant.SystemAddonLabel: "true",
			},
		},
		Spec: mgmtv1.ManagedAddonSpec{
			Enabled: false,
			Version: "1.0.0",
		},
		Status: mgmtv1.ManagedAddonStatus{
			JobName: "test-addon3-job",
		},
	},
}

func TestHandler_OnUpgradeChanged(t *testing.T) {
	type input struct {
		key     string
		upgrade *mgmtv1.Upgrade
		//plans   []*upgradev1.Plan
		addons []*mgmtv1.ManagedAddon
	}
	type output struct {
		upgrade *mgmtv1.Upgrade
		//plan    *upgradev1.Plan
		err error
	}
	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "upgrade with condition check - charts repo processing",
			given: input{
				key:     testUpgradeName,
				upgrade: newTestUpgradeBuilder().InitStatus().Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().WithChartsRepoCondition().Build(),
				err:     nil,
			},
		},
		{
			name: "create new upgrade",
			given: input{
				key:     testUpgradeName,
				upgrade: newTestUpgradeBuilder().Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().Build(),
				err:     nil,
			},
		},
		{
			name: "new upgrade must have charts repo",
			given: input{
				key:     testUpgradeName,
				upgrade: newTestUpgradeBuilder().InitStatus().Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().WithChartsRepoCondition().Build(),
				err:     nil,
			},
		},
		{
			name: "new upgrade must initialize managed-addons status",
			given: input{
				key:    testUpgradeName,
				addons: fakeManagedAddons,
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithManagedAddonStatus(fakeManagedAddons).
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).Build(),
				err: nil,
			},
		},
		{
			name: "new upgrade must initialize manifest upgrade status",
			given: input{
				key:    testUpgradeName,
				addons: fakeManagedAddons,
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).
					WithManagedAddonStatus(fakeManagedAddons).
					WithConditionReady(mgmtv1.ManagedAddonsIsReady).Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).
					WithManagedAddonStatus(fakeManagedAddons).
					WithConditionReady(mgmtv1.ManagedAddonsIsReady).
					WithManifestStatus().Build(),
				err: nil,
			},
		},
		{
			name: "new upgrade must create node plans",
			given: input{
				key:    testUpgradeName,
				addons: fakeManagedAddons,
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithManagedAddonStatus(fakeManagedAddons).
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).
					WithConditionReady(mgmtv1.ManagedAddonsIsReady).
					WithConditionReady(mgmtv1.ManifestUpgradeComplete).Build(),
			},
			expected: output{
				upgrade: newTestUpgradeBuilder().InitStatus().
					WithManagedAddonStatus(fakeManagedAddons).
					WithConditionReady(mgmtv1.UpgradeChartsRepoReady).
					WithConditionReady(mgmtv1.ManagedAddonsIsReady).
					WithConditionReady(mgmtv1.ManifestUpgradeComplete).
					WithNodesPlanStatus().Build(),
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		fakeClient := fake.NewSimpleClientset()

		var k8sclientset = k8sfake.NewSimpleClientset()
		if tc.given.upgrade != nil {
			var err = fakeClient.Tracker().Add(tc.given.upgrade)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}

		if tc.given.addons != nil {
			for _, addon := range tc.given.addons {
				var err = fakeClient.Tracker().Add(addon)
				assert.Nil(t, err, "mock resource %s should add into fake controller tracker", addon.Name)
			}
		}

		h := &upgradeHandler{
			upgradeClient:    fakeclients.UpgradeClient(fakeClient.ManagementV1().Upgrades),
			upgradeCache:     fakeclients.UpgradeCache(fakeClient.ManagementV1().Upgrades),
			helmChartClient:  fakeclients.HelmChartClient(fakeClient.HelmV1().HelmCharts),
			helmChartCache:   fakeclients.HelmChartCache(fakeClient.HelmV1().HelmCharts),
			planClient:       fakeclients.PlanClient(fakeClient.UpgradeV1().Plans),
			planCache:        fakeclients.PlanCache(fakeClient.UpgradeV1().Plans),
			deploymentClient: fakeclients.DeploymentClient(k8sclientset.AppsV1().Deployments),
			deploymentCache:  fakeclients.DeploymentCache(k8sclientset.AppsV1().Deployments),
			svcClient:        fakeclients.ServiceClient(k8sclientset.CoreV1().Services),
			svcCache:         fakeclients.ServiceCache(k8sclientset.CoreV1().Services),
			addonCache:       fakeclients.ManagedAddonCache(fakeClient.ManagementV1().ManagedAddons),
			discovery:        k8sclientset.Discovery(),
			commonHandler:    newFakeCommonHandler(fakeClient),
		}

		var actual output
		_, actual.err = h.onChange(tc.given.key, tc.given.upgrade)
		if tc.expected.upgrade != nil {
			var err error
			actual.upgrade, err = h.upgradeClient.Get(tc.expected.upgrade.Name, metav1.GetOptions{})
			actual.upgrade.Status = sanitizeUpgradeStatus(actual.upgrade.Status)
			assert.Nil(t, err)
		}

		assert.Equal(t, tc.expected, actual, "case %q", tc.name)
	}
}

func sanitizeUpgradeStatus(status mgmtv1.UpgradeStatus) mgmtv1.UpgradeStatus {
	toUpdate := status.DeepCopy()
	toUpdate.PreviousKubernetesVersion = ""
	toUpdate.StartTime = metav1.Time{}
	return *toUpdate
}
