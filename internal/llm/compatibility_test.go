package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsModelIncompatibleError(t *testing.T) {
	testCases := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "Audio Model Incompatibility",
			errMsg:   "This model requires that either input content or output modality contain audio",
			expected: true,
		},
		{
			name:     "Different Modality Incompatibility",
			errMsg:   "This model requires input content in a different modality",
			expected: true,
		},
		{
			name:     "No Text Input Incompatibility",
			errMsg:   "This model doesn't support text input",
			expected: true,
		},
		{
			name:     "Specific Format Incompatibility",
			errMsg:   "This model requires specific input formats",
			expected: true,
		},
		{
			name:     "Server Error",
			errMsg:   "Internal server error",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsModelIncompatibleError(tc.errMsg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
