package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalSystemImageRegistryDefault(t *testing.T) {
	// Test that GlobalSystemImageRegistry has the correct default value
	assert.Equal(t, "ghcr.io", GlobalSystemImageRegistry.Default, "GlobalSystemImageRegistry should default to ghcr.io")
}

func TestSetDefaultNotebookImages(t *testing.T) {
	tests := []struct {
		name             string
		registryValue    string
		expectedContains string
	}{
		{
			name:             "default registry",
			registryValue:    "ghcr.io",
			expectedContains: "ghcr.io/oneblock-ai/jupyter-scipy",
		},
		{
			name:             "custom registry",
			registryValue:    "docker.io",
			expectedContains: "docker.io/oneblock-ai/jupyter-scipy",
		},
		{
			name:             "private registry",
			registryValue:    "my-registry.com",
			expectedContains: "my-registry.com/oneblock-ai/jupyter-scipy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the registry value
			if err := GlobalSystemImageRegistry.Set(tt.registryValue); err != nil {
				t.Fatalf("Failed to set GlobalSystemImageRegistry: %v", err)
			}

			// Get the notebook images
			images := setDefaultNotebookImages()

			// Verify the registry is used correctly
			assert.Contains(t, images, tt.expectedContains, "Notebook images should use the configured registry")

			// Reset to default for other tests
			if err := GlobalSystemImageRegistry.Set("ghcr.io"); err != nil {
				t.Errorf("Failed to reset GlobalSystemImageRegistry: %v", err)
			}
		})
	}
}

func TestNotebookImagesStructure(t *testing.T) {
	// Test that the notebook images JSON structure is valid
	images := setDefaultNotebookImages()

	// Should not be empty
	assert.NotEmpty(t, images, "Notebook images should not be empty")

	// Should contain jupyter images
	assert.Contains(t, images, "jupyter", "Should contain jupyter images")

	// Should contain code-server images
	assert.Contains(t, images, "code-server", "Should contain code-server images")

	// Should be valid JSON
	assert.True(t, isValidJSON(images), "Should be valid JSON")
}

func isValidJSON(s string) bool {
	// Simple check to see if string looks like JSON
	return len(s) > 0 && s[0] == '{' && s[len(s)-1] == '}'
}
