package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// ValidationError represents an error during request validation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult holds the result of a validation operation
type ValidationResult struct {
	Valid  bool              `json:"-"`
	Errors []ValidationError `json:"errors"`
}

// NewValidationResult creates a new empty validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}
}

// AddError adds an error to the validation result
func (v *ValidationResult) AddError(field, message string) {
	v.Valid = false
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if the validation result has errors
func (v *ValidationResult) HasErrors() bool {
	return !v.Valid
}

// ErrorResponse writes a validation error response to the HTTP response writer
func (v *ValidationResult) ErrorResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"errors": v.Errors,
	})
}

// ValidateRequestBody validates the request body against the given constraints
func ValidateRequestBody(r *http.Request, maxSize int64) ([]byte, *ValidationResult) {
	result := NewValidationResult()

	// Check content type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		result.AddError("content-type", "Content-Type must be application/json")
		return nil, result
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			result.AddError("body", "Request body too large")
		} else {
			result.AddError("body", "Error reading request body")
		}
		return nil, result
	}

	// Check if body is empty
	if len(body) == 0 {
		result.AddError("body", "Request body cannot be empty")
		return nil, result
	}

	// Check if body is valid JSON
	var js json.RawMessage
	if err := json.Unmarshal(body, &js); err != nil {
		result.AddError("body", "Request body must be valid JSON")
		return nil, result
	}

	return body, result
}

// ValidateField validates a string field against the given constraints
func ValidateField(result *ValidationResult, field, value string, required bool, minLen, maxLen int) {
	if required && value == "" {
		result.AddError(field, "Field is required")
		return
	}

	if value == "" {
		return
	}

	if minLen > 0 && len(value) < minLen {
		result.AddError(field, "Must be at least "+string(rune('0'+minLen))+" characters")
	}

	if maxLen > 0 && len(value) > maxLen {
		result.AddError(field, "Must be at most "+string(rune('0'+maxLen))+" characters")
	}
}

// ValidateAuthKey validates the API key in the request header
func ValidateAuthKey(r *http.Request, expectedKey, headerName string) *ValidationResult {
	result := NewValidationResult()

	apiKey := r.Header.Get(headerName)
	if apiKey == "" {
		result.AddError("auth", "API key is required")
		return result
	}

	if apiKey != expectedKey {
		result.AddError("auth", "Invalid API key")
		return result
	}

	return result
}
