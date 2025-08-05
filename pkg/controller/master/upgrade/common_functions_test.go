package upgrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

func TestFormatRepoImage(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		repo     string
		tag      string
		expected string
	}{
		{
			name:     "default registry",
			registry: "ghcr.io",
			repo:     "llmos-ai/system-charts-repo",
			tag:      "v0.3.0",
			expected: "ghcr.io/llmos-ai/system-charts-repo:v0.3.0",
		},
		{
			name:     "custom registry",
			registry: "docker.io",
			repo:     "llmos-ai/node-upgrade",
			tag:      "v0.3.0-rc2",
			expected: "docker.io/llmos-ai/node-upgrade:v0.3.0-rc2",
		},
		{
			name:     "private registry",
			registry: "my-registry.com:5000",
			repo:     "llmos-ai/system-charts-repo",
			tag:      "latest",
			expected: "my-registry.com:5000/llmos-ai/system-charts-repo:latest",
		},
		{
			name:     "empty registry uses global setting",
			registry: "",
			repo:     "llmos-ai/node-upgrade",
			tag:      "v0.3.0",
			expected: "ghcr.io/llmos-ai/node-upgrade:v0.3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRepoImage(tt.registry, tt.repo, tt.tag)
			assert.Equal(t, tt.expected, result, "formatRepoImage should format correctly")
		})
	}
}

func TestServerPlan(t *testing.T) {
	upgrade := &mgmtv1.Upgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-upgrade",
		},
		Spec: mgmtv1.UpgradeSpec{
			Version:  "v0.3.0",
			Registry: "ghcr.io",
		},
	}

	plan := serverPlan(upgrade)

	assert.NotNil(t, plan, "Server plan should not be nil")
	assert.Equal(t, "test-upgrade-server", plan.Name, "Server plan name should be correct")
	assert.Equal(t, serverComponent, plan.Labels[llmosUpgradeComponentLabel], "Should have server component label")
	assert.NotNil(t, plan.Spec.Upgrade, "Should have upgrade spec")
	assert.Contains(t, plan.Spec.Upgrade.Image, "ghcr.io/llmos-ai/node-upgrade:v0.3.0", "Should use correct image")
	assert.Contains(t, plan.Spec.Upgrade.Args, "upgrade", "Should have upgrade arg")
}

func TestAgentPlan(t *testing.T) {
	upgrade := &mgmtv1.Upgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-upgrade",
		},
		Spec: mgmtv1.UpgradeSpec{
			Version:  "v0.3.0",
			Registry: "docker.io",
		},
	}

	plan := agentPlan(upgrade)

	assert.NotNil(t, plan, "Agent plan should not be nil")
	assert.Equal(t, "test-upgrade-agent", plan.Name, "Agent plan name should be correct")
	assert.Equal(t, agentComponent, plan.Labels[llmosUpgradeComponentLabel], "Should have agent component label")
	assert.NotNil(t, plan.Spec.Prepare, "Should have prepare spec")
	assert.NotNil(t, plan.Spec.Upgrade, "Should have upgrade spec")
	assert.Contains(t, plan.Spec.Prepare.Image, "docker.io/llmos-ai/node-upgrade:v0.3.0", "Should use correct prepare image")
	assert.Contains(t, plan.Spec.Upgrade.Image, "docker.io/llmos-ai/node-upgrade:v0.3.0", "Should use correct upgrade image")
	assert.Contains(t, plan.Spec.Prepare.Args, "prepare", "Should have prepare arg")
	assert.Contains(t, plan.Spec.Prepare.Args, "test-upgrade-server", "Should wait for server plan")
}

func TestGetUpgradeSystemChartsRepoUrl(t *testing.T) {
	// Test the function that gets the system charts repo URL
	url := getUpgradeSystemChartsRepoUrl()
	assert.NotEmpty(t, url, "System charts repo URL should not be empty")
	assert.Contains(t, url, "upgrade-repo", "URL should contain upgrade-repo")
	assert.Contains(t, url, "llmos-system", "URL should contain llmos-system namespace")
}

func TestUpgradeJobIsCompleteAfter(t *testing.T) {
	tests := []struct {
		name     string
		upgrade  *mgmtv1.Upgrade
		job      *batchv1.Job
		expected bool
	}{
		{
			name:     "job completed after upgrade start",
			upgrade:  newTestUpgradeBuilder().InitStatus().Build(),
			job:      newJobBuilder("test-job").Completed(fakeTime.Add(time.Hour)).Build(),
			expected: true,
		},
		{
			name:     "job completed before upgrade start",
			upgrade:  newTestUpgradeBuilder().InitStatusWithTime(fakeTime.Add(time.Hour)).Build(),
			job:      newJobBuilder("test-job").Completed(fakeTime).Build(),
			expected: false,
		},
		{
			name:     "job not completed",
			upgrade:  newTestUpgradeBuilder().InitStatus().Build(),
			job:      newJobBuilder("test-job").Running().Build(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := upgradeJobIsCompleteAfter(tt.upgrade, tt.job)
			assert.Equal(t, tt.expected, result, "upgradeJobIsCompleteAfter should return correct result")
		})
	}
}

func TestGlobalSystemImageRegistryUsage(t *testing.T) {
	// Test that the upgrade controller uses GlobalSystemImageRegistry correctly
	originalValue := settings.GlobalSystemImageRegistry.Get()
	defer func() {
		if err := settings.GlobalSystemImageRegistry.Set(originalValue); err != nil {
			t.Errorf("Failed to reset GlobalSystemImageRegistry: %v", err)
		}
	}()

	// Set a custom registry
	if err := settings.GlobalSystemImageRegistry.Set("my-custom-registry.com"); err != nil {
		t.Fatalf("Failed to set GlobalSystemImageRegistry: %v", err)
	}

	upgrade := &mgmtv1.Upgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-upgrade",
		},
		Spec: mgmtv1.UpgradeSpec{
			Version: "v0.3.0",
			// No registry specified, should use global setting
		},
	}

	// Test server plan uses the registry
	serverPlan := serverPlan(upgrade)
	assert.Contains(t, serverPlan.Spec.Upgrade.Image, "my-custom-registry.com", "Server plan should use global registry when no registry specified")

	// Test agent plan uses the registry
	agentPlan := agentPlan(upgrade)
	assert.Contains(t, agentPlan.Spec.Prepare.Image, "my-custom-registry.com", "Agent plan prepare should use global registry")
	assert.Contains(t, agentPlan.Spec.Upgrade.Image, "my-custom-registry.com", "Agent plan upgrade should use global registry")
}
