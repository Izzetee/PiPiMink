package llm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildResponsesInput(t *testing.T) {
	messages := []map[string]interface{}{
		{"role": "system", "content": "sys"},
		{"role": "user", "content": "hi"},
		{"content": "no-role-dropped"},
	}
	input := buildResponsesInput(messages)
	assert.Len(t, input, 2)
	assert.Equal(t, "system", input[0]["role"])
	assert.Equal(t, "sys", input[0]["content"])
	assert.Equal(t, "user", input[1]["role"])
}

func TestBuildResponsesPayload(t *testing.T) {
	messages := []map[string]interface{}{
		{"role": "user", "content": "hi"},
	}

	// Defaults omit temperature and max_output_tokens (reasoning-model safe).
	payload := buildResponsesPayload("gpt-5", messages, responsesRequestOptions{})
	assert.Equal(t, "gpt-5", payload["model"])
	_, hasMessages := payload["messages"]
	assert.False(t, hasMessages, "must not send 'messages' to the Responses API")
	_, hasInput := payload["input"]
	assert.True(t, hasInput, "must send 'input' to the Responses API")
	_, hasTemp := payload["temperature"]
	assert.False(t, hasTemp)
	_, hasMax := payload["max_output_tokens"]
	assert.False(t, hasMax)

	// Options are honoured when set.
	temp := 0.0
	payload = buildResponsesPayload("gpt-5", messages, responsesRequestOptions{temperature: &temp, maxOutputTokens: 256})
	assert.Equal(t, 0.0, payload["temperature"])
	assert.Equal(t, 256, payload["max_output_tokens"])

	// The marshalled body must contain "input" and not "messages".
	raw, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.Contains(t, string(raw), `"input"`)
	assert.NotContains(t, string(raw), `"messages"`)
}

func TestExtractResponsesContent(t *testing.T) {
	t.Run("Single message with output_text", func(t *testing.T) {
		body := []byte(`{"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello world"}]}]}`)
		content, err := extractResponsesContent(body)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", content)
	})

	t.Run("Reasoning item before message", func(t *testing.T) {
		body := []byte(`{"output":[{"type":"reasoning","summary":[]},{"type":"message","content":[{"type":"output_text","text":"answer"}]}]}`)
		content, err := extractResponsesContent(body)
		assert.NoError(t, err)
		assert.Equal(t, "answer", content)
	})

	t.Run("Multiple output_text parts concatenated", func(t *testing.T) {
		body := []byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"a"},{"type":"output_text","text":"b"}]}]}`)
		content, err := extractResponsesContent(body)
		assert.NoError(t, err)
		assert.Equal(t, "ab", content)
	})

	t.Run("Top-level output_text fallback", func(t *testing.T) {
		body := []byte(`{"output_text":"convenience","output":[]}`)
		content, err := extractResponsesContent(body)
		assert.NoError(t, err)
		assert.Equal(t, "convenience", content)
	})

	t.Run("API error surfaced", func(t *testing.T) {
		body := []byte(`{"error":{"message":"Unsupported parameter: 'messages'."}}`)
		_, err := extractResponsesContent(body)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "messages")
	})

	t.Run("Truncation via incomplete_details", func(t *testing.T) {
		body := []byte(`{"output":[],"incomplete_details":{"reason":"max_output_tokens"}}`)
		_, err := extractResponsesContent(body)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_output_tokens")
	})

	t.Run("Empty output", func(t *testing.T) {
		body := []byte(`{"output":[]}`)
		_, err := extractResponsesContent(body)
		assert.Error(t, err)
	})
}
