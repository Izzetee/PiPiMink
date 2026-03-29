package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRequestBody(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "application/json")

		body, result := ValidateRequestBody(req, 1024)
		assert.False(t, result.HasErrors())
		assert.NotEmpty(t, body)
	})

	t.Run("invalid content-type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "text/plain")

		_, result := ValidateRequestBody(req, 1024)
		assert.True(t, result.HasErrors())
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(nil))
		req.Header.Set("Content-Type", "application/json")

		_, result := ValidateRequestBody(req, 1024)
		assert.True(t, result.HasErrors())
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":`))
		req.Header.Set("Content-Type", "application/json")

		_, result := ValidateRequestBody(req, 1024)
		assert.True(t, result.HasErrors())
	})

	t.Run("body too large", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"123456789"}`))
		req.Header.Set("Content-Type", "application/json")

		_, result := ValidateRequestBody(req, 5)
		assert.True(t, result.HasErrors())
	})
}

func TestValidationResultAndAuthKey(t *testing.T) {
	t.Run("error response", func(t *testing.T) {
		res := NewValidationResult()
		res.AddError("field", "message")

		rr := httptest.NewRecorder()
		res.ErrorResponse(rr)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "field")
	})

	t.Run("auth key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		result := ValidateAuthKey(req, "expected", "X-API-Key")
		assert.True(t, result.HasErrors())

		req.Header.Set("X-API-Key", "wrong")
		result = ValidateAuthKey(req, "expected", "X-API-Key")
		assert.True(t, result.HasErrors())

		req.Header.Set("X-API-Key", "expected")
		result = ValidateAuthKey(req, "expected", "X-API-Key")
		assert.False(t, result.HasErrors())
	})
}

func TestValidateField(t *testing.T) {
	res := NewValidationResult()
	ValidateField(res, "name", "", true, 1, 10)
	assert.True(t, res.HasErrors())

	res = NewValidationResult()
	ValidateField(res, "name", "ok", true, 1, 10)
	assert.False(t, res.HasErrors())

	res = NewValidationResult()
	ValidateField(res, "name", "x", false, 2, 10)
	assert.True(t, res.HasErrors())
}
