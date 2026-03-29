// Package server provides the HTTP server and API routes
package server

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"strings"
	"time"

	"PiPiMink/internal/models"
)

// generateRandomID creates a secure random ID for OpenAI compatibility
func generateRandomID() string {
	// Use crypto/rand for secure random ID
	bytes := make([]byte, 16)
	_, err := cryptorand.Read(bytes)
	if err != nil {
		// Fallback to less secure method
		return generateFallbackRandomID()
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// generateFallbackRandomID provides a fallback method for ID generation
// using math/rand if crypto/rand fails
func generateFallbackRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 16
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// getCurrentUnixTimestamp returns the current Unix timestamp
func getCurrentUnixTimestamp() int64 {
	return time.Now().Unix()
}

// getFallbackModelName returns a safe fallback model name.
// Preference order: configured default model -> first enabled model -> empty.
func (s *Server) getFallbackModelName(enabledModels map[string]models.ModelInfo) string {
	if s != nil && s.config != nil {
		configured := strings.TrimSpace(s.config.DefaultChatModel)
		if configured != "" {
			if _, exists := enabledModels[configured]; exists {
				return configured
			}
		}
	}

	for modelName := range enabledModels {
		return modelName
	}

	return ""
}
