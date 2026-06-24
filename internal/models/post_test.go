package models

import (
	"testing"
)

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "Simple first line",
			body:     "This is the first line.\nThis is the second line.",
			expected: "This is the first line.",
		},
		{
			name:     "Markdown styling removed",
			body:     "**Bold** and _italic_ text here.\nNext line.",
			expected: "Bold and italic text here.",
		},
		{
			name:     "Skipping empty lines and headers",
			body:     "\n\n## Header\n---\nActual content start here.\nMore content.",
			expected: "Actual content start here.",
		},
		{
			name:     "Truncating long first line",
			body:     "This is an exceptionally long first line of the post that should definitely be truncated because it exceeds forty characters.",
			expected: "This is an exceptionally long first l...",
		},
		{
			name:     "Thread post with dividers and tweet headers",
			body:     "## Tweet 1\nHello world!\n---\n## Tweet 2\nAnother tweet.",
			expected: "Hello world!",
		},
		{
			name:     "Empty body fallback",
			body:     "",
			expected: "Unbenannter Beitrag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveTitle(tt.body)
			if got != tt.expected {
				t.Errorf("DeriveTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}
