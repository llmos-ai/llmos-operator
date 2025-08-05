package managedaddon

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestModifyImageRegistry(t *testing.T) {
	tests := []struct {
		name          string
		inputYAML     string
		newRegistry   string
		expectedYAML  string
		expectedError bool
	}{
		{
			name:          "Empty YAML",
			inputYAML:     "",
			newRegistry:   "myregistry.io",
			expectedYAML:  "global:\n  imageRegistry: myregistry.io\n",
			expectedError: false,
		},
		{
			name: "Valid YAML",
			inputYAML: `
global:
  imageRegistry: "ghcr.io"
  someOtherConfig: "value"
`,
			newRegistry: "myregistry.io",
			expectedYAML: `global:
  imageRegistry: myregistry.io
  someOtherConfig: value
`,
			expectedError: false,
		},
		{
			name:          "Invalid YAML",
			inputYAML:     "global:\n  imageRegistry: ghcr.io\n  invalidKey",
			newRegistry:   "myregistry.io",
			expectedYAML:  "",
			expectedError: true,
		},
		{
			name: "Update repository with ghcr.io",
			inputYAML: `
repository: ghcr.io/llmos-ai/test-image
`,
			newRegistry: "myregistry.io",
			expectedYAML: `
global:
  imageRegistry: myregistry.io
repository: myregistry.io/llmos-ai/test-image
`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputYAML, err := ModifyImageRegistry(tt.inputYAML, tt.newRegistry)

			// Check for errors
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				return
			}

			// If an error is expected, no need to compare YAML
			if tt.expectedError {
				return
			}

			// Normalize both expected and output YAML by unmarshaling and marshaling again
			var expectedMap, outputMap map[string]interface{}
			if err := yaml.Unmarshal([]byte(tt.expectedYAML), &expectedMap); err != nil {
				t.Fatalf("error unmarshaling expected YAML: %v", err)
			}
			if err := yaml.Unmarshal([]byte(outputYAML), &outputMap); err != nil {
				t.Fatalf("error unmarshaling output YAML: %v", err)
			}

			// Compare the maps
			if !reflect.DeepEqual(expectedMap, outputMap) {
				t.Errorf("expected YAML:\n%v\ngot:\n%v", tt.expectedYAML, outputYAML)
			}
		})
	}
}
