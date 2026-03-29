package llm

import (
	"testing"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSON(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: "https://api.openai.com", APIKey: "test-key"},
			{Name: "local", Type: config.ProviderTypeOpenAICompatible, BaseURL: "http://localhost:11434", RateLimitSeconds: 1},
		},
	}

	client := NewClient(cfg)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Direct JSON",
			input:    `{"strengths":["math","logic"],"weaknesses":["creativity"]}`,
			expected: `{"strengths":["math","logic"],"weaknesses":["creativity"]}`,
		},
		{
			name:     "JSON in Message",
			input:    `Here's what I think: {"strengths":["math","logic"],"weaknesses":["creativity"]} That's my assessment.`,
			expected: `{"strengths":["math","logic"],"weaknesses":["creativity"]}`,
		},
		{
			name:     "Invalid JSON",
			input:    "This is not JSON at all.",
			expected: "{\"strengths\":[], \"weaknesses\":[]}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.extractJSON(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
