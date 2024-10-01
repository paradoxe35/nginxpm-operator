package util

import "testing"

func TestExtractRootDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple domain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "Subdomain",
			input:    "sub.example.com",
			expected: "example.com",
		},
		{
			name:     "Multiple subdomains",
			input:    "a.b.c.example.com",
			expected: "b.c.example.com",
		},
		{
			name:     "Subdomain with two-part TLD",
			input:    "sub.example.co.uk",
			expected: "example.co.uk",
		},
		{
			name:     "Single-part domain",
			input:    "localhost",
			expected: "localhost",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Domain with trailing dot",
			input:    "example.com.",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRootDomain(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractRootDomain(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
