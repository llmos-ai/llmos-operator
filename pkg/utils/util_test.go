package utils

import (
	"testing"
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
