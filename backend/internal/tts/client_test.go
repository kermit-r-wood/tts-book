package tts

import (
	"testing"
	"tts-book/backend/internal/config"
)

func TestClient_sanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "Variant replacement (晩 -> 晚)",
			input:    "今晩的月色真美",
			expected: "今晚的月色真美",
		},
		{
			name:     "NFKC Normalization (Full-width -> Half-width)",
			input:    "１２３",
			expected: "123",
		},
		{
			name:     "NFKC Normalization (Circled -> Normal)",
			input:    "①",
			expected: "1",
		},
		{
			name:     "Mixed replacement and normalization",
			input:    "今晩１２３",
			expected: "今晚123",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	cfg := &config.Config{}
	client := NewClient(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.sanitizeText(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeText() = %v, want %v", got, tt.expected)
			}
		})
	}
}
