package upgrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

func Test_JobHandler_OnChanged(t *testing.T) {
	currentTime := time.Now()
	type input struct {
		key     string
		job     *batchv1.Job
		upgrade *mgmtv1.Upgrade
	}
	type output struct {
		job     *batchv1.Job
		upgrade *mgmtv1.Upgrade
		err     error
	}
	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "no update if llmos-crd if completed before upgrade",
			given: input{
				key:     llmosCrdChartName,
				job:     newJobBuilder(llmosCrdChartName).WithLabel(helmChartLabelKey, llmosCrdChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime.Add(10 * time.Minute)).Build(),
			},
			expected: output{
				job:     newJobBuilder(llmosCrdChartName).WithLabel(helmChartLabelKey, llmosCrdChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime.Add(10 * time.Minute)).Build(),
				err:     nil,
			},
		},
		{
			name: "llmos-crd chart job is complete",
			given: input{
				key:     llmosCrdChartName,
				job:     newJobBuilder(llmosCrdChartName).WithLabel(helmChartLabelKey, llmosCrdChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).Build(),
			},
			expected: output{
				job: newJobBuilder(llmosCrdChartName).WithLabel(helmChartLabelKey, llmosCrdChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).
					WithUpgradeJobStatus(llmosCrdChartName, llmosCrdChartName, currentTime).Build(),
				err: nil,
			},
		},
		{
			name: "llmos-operator chart job is complete",
			given: input{
				key:     llmosOperatorChartName,
				job:     newJobBuilder(llmosOperatorChartName).WithLabel(helmChartLabelKey, llmosOperatorChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).Build(),
			},
			expected: output{
				job: newJobBuilder(llmosOperatorChartName).WithLabel(helmChartLabelKey, llmosOperatorChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).
					WithUpgradeJobStatus(llmosOperatorChartName, llmosOperatorChartName, currentTime).Build(),
				err: nil,
			},
		},
		{
			name: "ignored chart name",
			given: input{
				key:     testJobName,
				job:     newJobBuilder(testJobName).WithLabel(helmChartLabelKey, testJobName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).Build(),
			},
			expected: output{
				job:     nil,
				upgrade: nil,
				err:     nil,
			},
		},
		{
			name: "chart without required label",
			given: input{
				key:     llmosOperatorChartName,
				job:     newJobBuilder(llmosOperatorChartName).Completed(currentTime).Build(),
				upgrade: newTestUpgradeBuilder().InitStatusWithTime(currentTime).Build(),
			},
			expected: output{
				job:     nil,
				upgrade: nil,
				err:     nil,
			},
		},
	}
	for _, tc := range testCases {
		var clientset = fake.NewSimpleClientset(tc.given.upgrade)
		commonHandler := newFakeCommonHandler(clientset)
		var handler = &jobHandler{
			upgradeClient: fakeclients.UpgradeClient(clientset.ManagementV1().Upgrades),
			upgradeCache:  fakeclients.UpgradeCache(clientset.ManagementV1().Upgrades),
			commonHandler: commonHandler,
		}

		var actual output
		actual.job, actual.err = handler.watchUpgradeJobs(tc.given.key, tc.given.job)

		if tc.expected.upgrade != nil {
			var err error
			upgrade, err := handler.upgradeCache.Get(tc.given.upgrade.Name)
			assert.Nil(t, err)

			actual.upgrade, err = handler.syncUpgradeJobStatus(tc.given.upgrade, actual.job, tc.given.key)
			assert.Nil(t, err)
			assert.Equal(t, tc.expected.upgrade, upgrade)
			assert.True(t, actual.upgrade.Status.UpgradeJobs[0].Complete)
		}

		assert.Equal(t, tc.expected.job, actual.job, "case %q", tc.name)
	}
}
