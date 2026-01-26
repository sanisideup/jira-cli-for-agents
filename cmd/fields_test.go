package cmd

import (
	"testing"
)

// TestIsValidAlias tests the alias validation logic
func TestIsValidAlias(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		expected bool
	}{
		// Valid cases
		{"lowercase with underscore", "story_points", true},
		{"CamelCase", "StoryPoints", true},
		{"with numbers", "field_123", true},
		{"all lowercase", "epiclink", true},
		{"all uppercase", "EPICLINK", true},
		{"single char", "a", true},
		{"single underscore word", "story", true},

		// Invalid cases
		{"empty string", "", false},
		{"with hyphen", "story-points", false},
		{"with space", "story points", false},
		{"with dot", "story.points", false},
		{"with special chars", "story@points", false},
		{"starts with number", "123field", true}, // numbers are allowed anywhere
		{"only underscores", "___", true},        // technically valid per our rules
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAlias(tt.alias)
			if result != tt.expected {
				t.Errorf("isValidAlias(%q) = %v, want %v", tt.alias, result, tt.expected)
			}
		})
	}
}

// TestIsValidAlias_EdgeCases tests edge cases for alias validation
func TestIsValidAlias_EdgeCases(t *testing.T) {
	// Test with unicode characters
	invalidAliases := []string{
		"café",          // accented characters
		"story_points™", // trademark symbol
		"story\npoints", // newline
		"story\tpoints", // tab
	}

	for _, alias := range invalidAliases {
		if isValidAlias(alias) {
			t.Errorf("isValidAlias(%q) should be false for unicode/special characters", alias)
		}
	}
}

// BenchmarkIsValidAlias benchmarks the alias validation function
func BenchmarkIsValidAlias(b *testing.B) {
	testCases := []string{
		"story_points",
		"epic_link",
		"customfield_10016",
		"a",
		"invalid-alias",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, alias := range testCases {
			isValidAlias(alias)
		}
	}
}
