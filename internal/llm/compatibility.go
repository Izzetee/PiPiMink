package llm

import (
	"strings"
)

// IsModelIncompatibleError checks if the error message indicates that the model is incompatible with text requests
// and should be disabled
func IsModelIncompatibleError(errorMessage string) bool {
	// Check for known incompatibility messages
	incompatibleMessages := []string{
		"This model requires that either input content or output modality contain audio",
		"This model requires input content in a different modality",
		"This model doesn't support text input",
		"This model requires specific input formats",
	}

	for _, msg := range incompatibleMessages {
		if strings.Contains(errorMessage, msg) {
			return true
		}
	}
	return false
}
