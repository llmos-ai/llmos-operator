package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// TestReplaceAndLower tests the ReplaceAndLower function.
func TestReplaceAndLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello_World:This_Is_A_Test", "hello-world-this-is-a-test"},
		{"NoSpecialCharacters", "nospecialcharacters"},
		{"multiple___underscores", "multiple---underscores"},
		{"colons::everywhere", "colons--everywhere"},
		{"_LeadingAndTrailing_", "-leadingandtrailing-"},
		{":colon_at_start", "-colon-at-start"},
		{"mixed_CASE:and_Special_Characters", "mixed-case-and-special-characters"},
		{"already-lowered-and-hyphenated", "already-lowered-and-hyphenated"},
		{"", ""},
	}

	for _, test := range tests {
		result := ReplaceAndLower(test.input)
		if result != test.expected {
			t.Errorf("ReplaceAndLower(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
func Test_MergeYAML(t *testing.T) {
	type input struct {
		key           string
		defaultValues string
		valueContent  string
	}
	type output struct {
		values string
		err    error
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "merge values",
			given: input{
				key: "test",
				defaultValues: `
name: application
replicas: 3
image:
  name: myapp
  tag: v1
`,
				valueContent: `
replicas: 5
image:
  tag: v2
resources:
  limits:
    memory: "256Mi"
`,
			},
			expected: output{
				values: `
name: application
replicas: 5
image:
  name: myapp
  tag: v2
resources:
  limits:
    memory: "256Mi"
`,
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		var actual output
		actual.values, actual.err = MergeYAML(tc.given.defaultValues, tc.given.valueContent)
		var (
			result1 = map[string]interface{}{}
			result2 = map[string]interface{}{}
			err     error
		)
		err = yaml.Unmarshal([]byte(actual.values), &result1)
		assert.NoError(t, err)
		err = yaml.Unmarshal([]byte(tc.expected.values), &result2)
		assert.NoError(t, err)
		assert.Equal(t, result1, result2)
		assert.Nil(t, actual.err)
	}
}
