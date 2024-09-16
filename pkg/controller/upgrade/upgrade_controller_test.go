package upgrade

import (
	"testing"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	llmosv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

var newUpgrade = &llmosv1.Upgrade{
	ObjectMeta: metav1.ObjectMeta{
		Name: "new-upgrade",
	},
	Spec: llmosv1.UpgradeSpec{
		Version: "v0.2.0-rc1",
	},
}

func TestHandler_OnUpgradeChanged(t *testing.T) {
	type input struct {
		key     string
		Upgrade *llmosv1.Upgrade
		Plan    *upgradev1.Plan
	}
	type output struct {
		Upgrade *llmosv1.Upgrade
		Plan    *upgradev1.Plan
		err     error
	}
	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "deleted resource",
			given: input{
				key: "llmos-system/upgrade-delete",
				Upgrade: &llmosv1.Upgrade{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "llmos-system",
						Name:              "upgrade-delete",
						DeletionTimestamp: &metav1.Time{},
					},
				},
			},
			expected: output{
				Upgrade: nil,
				err:     nil,
			},
		},
		{
			name: "create upgrade",
			given: input{
				key:     "llmos-system/new-upgrade",
				Upgrade: newUpgrade,
			},
			expected: output{
				Upgrade: &llmosv1.Upgrade{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: newUpgrade.Namespace,
						Name:      newUpgrade.Name,
						Labels: map[string]string{
							llmosLatestUpgradeLabel: "true",
						},
					},
					Spec: newUpgrade.Spec,
				},
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		fakeClient := fake.NewSimpleClientset()
		if tc.given.Upgrade != nil {
			var err = fakeClient.Tracker().Add(tc.given.Upgrade)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}

		h := &upgradeHandler{
			upgradeClient: fakeclients.UpgradeClient(fakeClient.ManagementV1().Upgrades),
			upgradeCache:  fakeclients.UpgradeCache(fakeClient.ManagementV1().Upgrades),
			planClient:    fakeclients.PlanClient(fakeClient.UpgradeV1().Plans),
			planCache:     fakeclients.PlanCache(fakeClient.UpgradeV1().Plans),
		}
		var actual output
		actual.Upgrade, actual.err = h.onChange(tc.given.key, tc.given.Upgrade)
		assert.Nil(t, actual.err, "case %q", tc.name)
	}
}
