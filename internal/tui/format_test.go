package tui

import (
	"strings"
	"testing"
)

func TestFormatCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single line message",
			input:    "feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Already properly formatted",
			input:    "feat: add new feature\n\n- Add functionality A\n- Add functionality B",
			expected: "feat: add new feature\n\n- Add functionality A\n- Add functionality B",
		},
		{
			name:     "Missing blank line",
			input:    "feat: add new feature\n- Add functionality A\n- Add functionality B",
			expected: "feat: add new feature\n\n- Add functionality A\n- Add functionality B",
		},
		{
			name:     "Multiple trailing empty lines",
			input:    "feat: add new feature\n\n- Add functionality A\n- Add functionality B\n\n\n",
			expected: "feat: add new feature\n\n- Add functionality A\n- Add functionality B",
		},
		{
			name:     "Only subject and empty lines",
			input:    "feat: add new feature\n\n\n",
			expected: "feat: add new feature",
		},
		{
			name:     "Subject with immediate body (no blank line)",
			input:    "fix: resolve issue\nThis fixes the problem with XYZ",
			expected: "fix: resolve issue\n\nThis fixes the problem with XYZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCommitMessage(tt.input)
			if result != tt.expected {
				t.Errorf("formatCommitMessage() = %q, want %q", result, tt.expected)

				// Show detailed comparison
				resultLines := strings.Split(result, "\n")
				expectedLines := strings.Split(tt.expected, "\n")

				t.Logf("Result lines (%d):", len(resultLines))
				for i, line := range resultLines {
					t.Logf("  [%d]: %q", i, line)
				}

				t.Logf("Expected lines (%d):", len(expectedLines))
				for i, line := range expectedLines {
					t.Logf("  [%d]: %q", i, line)
				}
			}
		})
	}
}
