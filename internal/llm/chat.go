package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

// ChatWithModel sends a conversation to the specified model and returns the response.
// messages is the full OpenAI-format messages array (role + content objects).
// The provider is looked up from modelInfo.Source; the API format is determined by the provider type.
func (c *Client) ChatWithModel(modelInfo models.ModelInfo, model string, messages []map[string]interface{}) (string, error) {
	if !modelInfo.Enabled {
		return "This model is currently disabled", nil
	}

	p, ok := c.findProvider(modelInfo.Source)
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", modelInfo.Source)
	}

	// Apply any per-model overrides (endpoint path, API key, type) for this model.
	p = p.ForModel(model)

	log.Printf("Sending chat request to model %s (provider: %s, type: %s, history: %d messages)", model, p.Name, p.Type, len(messages))

	if rl := c.rateLimiterFor(p.Name); rl != nil {
		rl.Wait()
		defer rl.UpdateLastRequestTime()
	}

	switch p.Type {
	case config.ProviderTypeAnthropic:
		return c.chatWithAnthropicModel(p, model, messages)
	default:
		return c.chatWithOpenAICompatibleModel(p, model, messages)
	}
}

// chatWithOpenAICompatibleModel sends a chat request to an OpenAI-compatible endpoint.
func (c *Client) chatWithOpenAICompatibleModel(p config.ProviderConfig, model string, messages []map[string]interface{}) (string, error) {
	url := p.ChatCompletionsURL()
	client := &http.Client{Timeout: p.Timeout}

	isLocal := strings.HasPrefix(p.BaseURL, "http://localhost") || strings.HasPrefix(p.BaseURL, "http://127.0.0.1")
	runningWithMLX := isLocal && c.isLocalServerUsingMLX()
	if runningWithMLX {
		log.Printf("Detected local model %s running with MLX acceleration, excluding temperature parameter", model)
	}

	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if !isReasoningModelNoSysMsg(model) && !runningWithMLX {
		payload["temperature"] = 0.7
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	log.Printf("Sending request to %s for model %s", url, model)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	log.Printf("Received response with status code: %d", resp.StatusCode)

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	log.Printf("Raw API response: %s", responseBody.String())

	// If the model rejects the temperature parameter, retry once without it.
	if resp.StatusCode == http.StatusBadRequest {
		if msg := extractAPIErrorMessage(responseBody.Bytes()); isTemperatureError(msg) {
			log.Printf("Model %s rejected temperature parameter, retrying without it", model)
			delete(payload, "temperature")
			retryPayload, _ := json.Marshal(payload)
			retryReq, rErr := http.NewRequest("POST", url, bytes.NewBuffer(retryPayload))
			if rErr == nil {
				retryReq.Header.Set("Content-Type", "application/json")
				if p.APIKey != "" {
					retryReq.Header.Set("Authorization", "Bearer "+p.APIKey)
				}
				retryResp, rErr := client.Do(retryReq)
				if rErr == nil {
					defer func() { _ = retryResp.Body.Close() }()
					responseBody.Reset()
					if _, rErr = responseBody.ReadFrom(retryResp.Body); rErr != nil {
						log.Printf("Error reading retry response body: %v", rErr)
					}
					log.Printf("Retry response status: %d", retryResp.StatusCode)
				}
			}
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody.Bytes(), &result); err != nil {
		return "", fmt.Errorf("error decoding response JSON: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("missing or empty choices in response. Keys: %v", getMapKeys(result))
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format")
	}
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing message in choice")
	}
	content, ok := msg["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing content in message")
	}

	log.Printf("Successfully extracted content from response (%d characters)", len(content))
	return content, nil
}

// chatWithAnthropicModel sends a chat request using the Anthropic Messages API.
// It converts the OpenAI-format messages array into Anthropic's format:
// system messages are extracted to the top-level "system" field; all other
// messages are kept in the messages array with only "user" and "assistant" roles.
func (c *Client) chatWithAnthropicModel(p config.ProviderConfig, model string, messages []map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/v1/messages", p.BaseURL)
	client := &http.Client{Timeout: p.Timeout}

	// Separate system messages from the conversation.
	var systemParts []string
	var anthropicMessages []map[string]interface{}

	for _, m := range messages {
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		switch role {
		case "system":
			systemParts = append(systemParts, content)
		case "user", "assistant":
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    role,
				"content": content,
			})
		}
	}

	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"messages":   anthropicMessages,
	}
	if len(systemParts) > 0 {
		payload["system"] = strings.Join(systemParts, "\n")
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling Anthropic payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error creating Anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	log.Printf("Sending Anthropic chat request for model %s", model)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making Anthropic request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	log.Printf("Received Anthropic response with status code: %d", resp.StatusCode)

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("error reading Anthropic response body: %w", err)
	}
	log.Printf("Raw Anthropic response: %s", responseBody.String())

	content, err := extractAnthropicContent(responseBody.Bytes())
	if err != nil {
		return "", err
	}

	log.Printf("Successfully extracted Anthropic content (%d characters)", len(content))
	return content, nil
}
