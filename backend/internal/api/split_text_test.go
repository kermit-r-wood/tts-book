package api_test

import (
	"testing"
	"tts-book/backend/internal/api"
)

func TestSplitText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		limit   int
		wantLen int
	}{
		{
			name:    "Small text, large limit",
			input:   "Hello world",
			limit:   100,
			wantLen: 1,
		},
		{
			name:    "Exact limit",
			input:   "1234567890",
			limit:   10,
			wantLen: 1,
		},
		{
			name:    "Just over limit",
			input:   "12345678901",
			limit:   10,
			wantLen: 2,
		},
		{
			name:    "Split by newlines",
			input:   "Line 1\n\nLine 2\n\nLine 3",
			limit:   10,
			wantLen: 3, // "Line 1\n\n" (8), "Line 2\n\n" (8), "Line 3" (6). All < 10.
		},
		{
			name:    "Zero limit (should be handled by caller, but testing here for safety)",
			input:   "12345",
			limit:   1, // Using 1 as minimal valid limit since 0 might loop or be invalid
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := api.SplitText(tt.input, tt.limit)

			// Flexible check for newline one
			if tt.name == "Split by newlines" {
				if len(got) != 3 {
					t.Errorf("SplitText() chunks = %d, want 3", len(got))
				}
			} else {
				if len(got) != tt.wantLen {
					t.Errorf("SplitText() = %v chunks, want %d", len(got), tt.wantLen)
				}
			}

			// Verify total content reconstruction
			reconstructed := ""
			for _, s := range got {
				reconstructed += s
			}
			if reconstructed != tt.input {
				t.Errorf("SplitText() content mismatch. Reconstructed len %d != Input len %d", len(reconstructed), len(tt.input))
			}
		})
	}
}
