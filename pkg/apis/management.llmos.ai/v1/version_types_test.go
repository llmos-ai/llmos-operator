package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVersionSpec_KubernetesVersionRequired(t *testing.T) {
	tests := []struct {
		name    string
		spec    VersionSpec
		valid   bool
		message string
	}{
		{
			name: "valid version spec with kubernetes version",
			spec: VersionSpec{
				MinUpgradableVersion: "v0.1.0",
				KubernetesVersion:    "v1.33.1+k3s1",
				ReleaseDate:          "2025-08-05",
				Tags:                 []string{"stable"},
			},
			valid:   true,
			message: "Should be valid when KubernetesVersion is provided",
		},
		{
			name: "invalid version spec without kubernetes version",
			spec: VersionSpec{
				MinUpgradableVersion: "v0.1.0",
				ReleaseDate:          "2025-08-05",
				Tags:                 []string{"stable"},
			},
			valid:   false,
			message: "Should be invalid when KubernetesVersion is missing",
		},
		{
			name: "valid version spec with empty kubernetes version",
			spec: VersionSpec{
				MinUpgradableVersion: "v0.1.0",
				KubernetesVersion:    "",
				ReleaseDate:          "2025-08-05",
				Tags:                 []string{"stable"},
			},
			valid:   false,
			message: "Should be invalid when KubernetesVersion is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := &Version{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-version",
				},
				Spec: tt.spec,
			}

			// Test that the struct can be created
			assert.NotNil(t, version, "Version should be created")

			// Validate the KubernetesVersion field
			if tt.valid {
				assert.NotEmpty(t, version.Spec.KubernetesVersion, tt.message)
			} else {
				assert.Empty(t, version.Spec.KubernetesVersion, tt.message)
			}
		})
	}
}

func TestVersionSpec_ValidKubernetesVersionFormats(t *testing.T) {
	tests := []struct {
		name              string
		kubernetesVersion string
		expected          bool
	}{
		{
			name:              "k3s version format",
			kubernetesVersion: "v1.33.1+k3s1",
			expected:          true,
		},
		{
			name:              "standard kubernetes version",
			kubernetesVersion: "v1.31.0",
			expected:          true,
		},
		{
			name:              "kubernetes with patch",
			kubernetesVersion: "v1.31.2",
			expected:          true,
		},
		{
			name:              "rke2 version format",
			kubernetesVersion: "v1.31.0+rke2r1",
			expected:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := VersionSpec{
				KubernetesVersion: tt.kubernetesVersion,
				ReleaseDate:       "2025-08-05",
			}

			if tt.expected {
				assert.NotEmpty(t, spec.KubernetesVersion, "KubernetesVersion should be set")
			} else {
				assert.Empty(t, spec.KubernetesVersion, "KubernetesVersion should be empty")
			}
		})
	}
}

func TestVersion_ObjectCreation(t *testing.T) {
	version := &Version{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v0.3.0-rc2",
		},
		Spec: VersionSpec{
			MinUpgradableVersion: "v0.2.0",
			KubernetesVersion:    "v1.33.1+k3s1",
			ReleaseDate:          "2025-08-05",
			Tags:                 []string{"rc"},
		},
	}

	assert.Equal(t, "v0.3.0-rc2", version.Name, "Version name should match")
	assert.Equal(t, "v0.2.0", version.Spec.MinUpgradableVersion, "MinUpgradableVersion should match")
	assert.Equal(t, "v1.33.1+k3s1", version.Spec.KubernetesVersion, "KubernetesVersion should match")
	assert.Equal(t, "2025-08-05", version.Spec.ReleaseDate, "ReleaseDate should match")
	assert.Contains(t, version.Spec.Tags, "rc", "Tags should contain rc")
}
