package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	t.Run("NewRateLimiter", func(t *testing.T) {
		limiter := NewRateLimiter(2)
		assert.NotNil(t, limiter)
		assert.Equal(t, 2, limiter.waitSeconds)
	})

	// Note: Testing Wait and UpdateLastRequestTime would require time mocking
	// which is beyond the scope of this simple test
}
