package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"PiPiMink/internal/config"
)

const (
	anthropicVersion = "2023-06-01"
	tagsMaxTokens    = 1024

	// tagsSystemPrompt instructs the model on how to produce routing-useful capability tags.
	// Key design decisions:
	//   - Explains WHY tags are needed (prompt routing) so models give actionable answers.
	//   - Enforces lowercase-hyphenated format for consistent tag matching downstream.
	//   - Provides concrete examples to anchor vocabulary and avoid generic responses.
	//   - Prefers 5–15 high-quality tags over padding to hit a minimum count.
	//   - Explicitly handles non-text-generation models (return empty strengths array).
	tagsSystemPrompt = `You are a routing capabilities assessor for an LLM gateway. Your output will be used to automatically route user prompts to the most suitable language model. Accurate, specific tags lead to better routing decisions.

Your task: describe THIS model's specific task strengths and limitations using lowercase-hyphenated tags.

RULES (follow exactly):
1. Reply ONLY with a valid JSON object — no explanation, no markdown fences, no preamble.
2. Format: {"strengths":["tag1","tag2"],"weaknesses":["tag1","tag2"]}
3. All tags MUST be lowercase-hyphenated (e.g. "code-generation", "step-by-step-reasoning").
4. Tags represent TASK TYPES a user might request — not abstract capability names.
5. Be SPECIFIC to this model's actual capabilities compared to other LLMs.
6. Provide 5–15 strengths and 3–15 weaknesses. Quality over quantity — no padding.
7. If this model does not generate text (e.g. image generation, embeddings, audio-only), return exactly: {"strengths":[],"weaknesses":["not-a-text-generation-model"]}`

	// tagsUserPrompt is the follow-up turn for models that support system messages.
	tagsUserPrompt = `Assess THIS model's specific capabilities for task routing. Return only the JSON object.

Example tags (use similar style): "complex-reasoning", "mathematical-problem-solving", "multi-step-code-generation", "code-debugging", "long-context-analysis", "creative-writing", "factual-qa", "text-summarization", "document-extraction", "multilingual-translation", "instruction-following", "structured-data-analysis", "scientific-research", "real-time-information", "function-calling", "vision-understanding", "latex-math", "sql-query-writing".

What does THIS model specifically excel at compared to other LLMs? What tasks does it handle poorly or refuse?`

	// tagsUserPromptNoSys combines both instructions into a single user turn for models
	// that do not support system messages (e.g. o1/o3/o4-series, some fine-tuned models).
	tagsUserPromptNoSys = `You are a routing capabilities assessor for an LLM gateway. Your output routes user prompts to the best model. Return ONLY a JSON object — no markdown, no explanation.

Format: {"strengths":["tag1","tag2"],"weaknesses":["tag1","tag2"]}
Rules: lowercase-hyphenated tags only; 5–15 strengths; 3–15 weaknesses; tags = task types a user might request; if not a text-generation model return {"strengths":[],"weaknesses":["not-a-text-generation-model"]}.

Example tags: "complex-reasoning", "mathematical-problem-solving", "multi-step-code-generation", "code-debugging", "long-context-analysis", "creative-writing", "factual-qa", "text-summarization", "multilingual-translation", "structured-data-analysis", "real-time-information", "function-calling", "vision-understanding".

Assess THIS model's specific strengths and weaknesses for task routing. What does this model excel at compared to other LLMs? What does it handle poorly?`
)

// GetModelTags asks a model to self-assess its capabilities and returns the result as
// a JSON string of strengths and weaknesses.
//
// Returns: tags, shouldDisable, shouldDelete, error
//   - shouldDisable: model is incompatible with text chat (e.g. audio-only)
//   - shouldDelete:  model is not a chat model at all (e.g. embedding model)
func (c *Client) GetModelTags(model string, p config.ProviderConfig) (string, bool, bool, error) {
	// Apply any per-model overrides (endpoint path, API key, type) for this model.
	p = p.ForModel(model)

	log.Printf("Getting tags for model %q (provider: %s, type: %s)", model, p.Name, p.Type)

	if rl := c.rateLimiterFor(p.Name); rl != nil {
		rl.Wait()
	}

	var tags string
	var shouldDisable, shouldDelete bool
	var err error

	switch p.Type {
	case config.ProviderTypeAnthropic:
		tags, shouldDisable, shouldDelete, err = c.getTagsAnthropic(model, p)
	default:
		tags, shouldDisable, shouldDelete, err = c.getTagsOpenAICompatible(model, p)
	}

	if rl := c.rateLimiterFor(p.Name); rl != nil {
		rl.UpdateLastRequestTime()
	}

	return tags, shouldDisable, shouldDelete, err
}

// isKnownNonChatModel returns true for model names that are definitively not chat-completion
// models (image generation, embeddings, speech, etc.). These are deleted rather than tagged.
func isKnownNonChatModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "dall-e") ||
		strings.Contains(m, "gpt-image") ||
		strings.Contains(m, "sora") ||
		strings.Contains(m, "text-embedding") ||
		strings.HasPrefix(m, "embedding") ||
		strings.HasPrefix(m, "whisper") ||
		strings.HasPrefix(m, "tts-") ||
		strings.Contains(m, "moderation")
}

// isTemperatureError returns true when the API rejected the request solely because of
// the temperature=0 parameter (some newer/reasoning models only accept default temperature).
func isTemperatureError(msg string) bool {
	return strings.Contains(msg, "temperature' does not support 0") ||
		strings.Contains(msg, "incompatible request argument supplied: temperature") ||
		strings.Contains(msg, "Unsupported value: 'temperature'") ||
		strings.Contains(msg, "temperature is not supported")
}

// activeTaggingSystemPrompt returns the in-use system prompt (DB override or default constant).
func (c *Client) activeTaggingSystemPrompt() string {
	if c.taggingSystemPrompt != "" {
		return c.taggingSystemPrompt
	}
	return tagsSystemPrompt
}

// activeTaggingUserPrompt returns the in-use user prompt.
func (c *Client) activeTaggingUserPrompt() string {
	if c.taggingUserPrompt != "" {
		return c.taggingUserPrompt
	}
	return tagsUserPrompt
}

// activeTaggingUserNoSysPrompt returns the in-use no-sys user prompt.
func (c *Client) activeTaggingUserNoSysPrompt() string {
	if c.taggingUserNoSysPrompt != "" {
		return c.taggingUserNoSysPrompt
	}
	return tagsUserPromptNoSys
}

// getTagsOpenAICompatible fetches tags via the /v1/chat/completions endpoint.
func (c *Client) getTagsOpenAICompatible(model string, p config.ProviderConfig) (string, bool, bool, error) {
	// Skip API call entirely for models we know cannot handle chat completions.
	if isKnownNonChatModel(model) {
		log.Printf("Model %s identified as non-chat model by name — deleting", model)
		return "", false, true, nil
	}

	url := p.ChatCompletionsURL()
	client := &http.Client{Timeout: p.Timeout}

	// MLX check only applies to locally-running servers.
	isLocal := strings.HasPrefix(p.BaseURL, "http://localhost") || strings.HasPrefix(p.BaseURL, "http://127.0.0.1")
	runningWithMLX := isLocal && c.isLocalServerUsingMLX()
	if runningWithMLX {
		log.Printf("Detected local model %s running with MLX acceleration, excluding temperature parameter", model)
	}

	var payload map[string]interface{}

	// o1/o3/o4-series models do not support system messages or temperature.
	if isReasoningModelNoSysMsg(model) {
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": c.activeTaggingUserNoSysPrompt()},
			},
		}
	} else {
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": c.activeTaggingSystemPrompt()},
				{"role": "user", "content": c.activeTaggingUserPrompt()},
			},
		}
		if !runningWithMLX {
			payload["temperature"] = 0.0
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", false, false, fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", false, false, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	log.Printf("Sending tags request to %s for model %s", url, model)
	resp, err := client.Do(req)
	if err != nil {
		return "", false, false, fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", false, false, fmt.Errorf("error reading response body: %w", err)
	}
	responseStr := responseBody.String()
	log.Printf("Raw tags response for model %s: %s", model, responseStr)

	if resp.StatusCode >= 400 {
		msg := extractAPIErrorMessage(responseBody.Bytes())
		if msg != "" {
			log.Printf("API error for model %s (HTTP %d): %s", model, resp.StatusCode, msg)

			// Model only supports the newer /v1/responses endpoint — treat as deleted.
			if strings.Contains(msg, "only supported in v1/responses") {
				return "", false, true, nil
			}
			// Not a chat model at all (e.g. completion-only, image, embedding).
			if strings.Contains(msg, "This is not a chat model") ||
				strings.Contains(msg, "not supported in the v1/chat/completions endpoint") {
				return "", false, true, nil
			}
			// System role not accepted — retry with user-role-only messages.
			if strings.Contains(msg, "role' does not support 'system'") {
				return c.getTagsOpenAICompatibleUserRoleOnly(model, p)
			}
			// temperature=0 not accepted — retry without temperature parameter.
			if isTemperatureError(msg) {
				log.Printf("Model %s does not accept temperature=0 — retrying without temperature", model)
				return c.getTagsOpenAICompatibleNoTemp(model, p)
			}
			// Audio/vision/modality-only models.
			if IsModelIncompatibleError(msg) {
				return `{"strengths":["unavailable"],"weaknesses":["text-incompatible"]}`, true, false, nil
			}
		}
		// Unknown 4xx error — fall through to attempt extraction; will likely fail and return an error.
	}

	return c.extractTagsFromOpenAIResponse(responseBody.Bytes(), model)
}

// extractAPIErrorMessage extracts the human-readable error string from an OpenAI-compatible
// error response body. Handles both {"error": {"message": "..."}} and {"error": "..."} formats.
func extractAPIErrorMessage(body []byte) string {
	var errResp map[string]interface{}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return ""
	}
	switch v := errResp["error"].(type) {
	case map[string]interface{}:
		if msg, ok := v["message"].(string); ok {
			return msg
		}
	case string:
		return v
	}
	return ""
}

// getTagsOpenAICompatibleNoTemp retries the tagging request without a temperature parameter
// for models that reject temperature=0 (e.g. search-preview, some o-series variants).
func (c *Client) getTagsOpenAICompatibleNoTemp(model string, p config.ProviderConfig) (string, bool, bool, error) {
	log.Printf("Retrying tag generation for model %s without temperature parameter", model)

	url := p.ChatCompletionsURL()
	client := &http.Client{Timeout: p.Timeout}

	var payload map[string]interface{}
	if isReasoningModelNoSysMsg(model) {
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": c.activeTaggingUserNoSysPrompt()},
			},
		}
	} else {
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": c.activeTaggingSystemPrompt()},
				{"role": "user", "content": c.activeTaggingUserPrompt()},
			},
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", false, false, fmt.Errorf("error marshalling no-temp payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", false, false, fmt.Errorf("error creating no-temp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", false, false, fmt.Errorf("error making no-temp request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", false, false, fmt.Errorf("error reading no-temp response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := extractAPIErrorMessage(responseBody.Bytes())
		if msg != "" {
			log.Printf("No-temp retry API error for model %s: %s", model, msg)
			if strings.Contains(msg, "This is not a chat model") ||
				strings.Contains(msg, "not supported in the v1/chat/completions endpoint") ||
				strings.Contains(msg, "only supported in v1/responses") {
				return "", false, true, nil
			}
			if IsModelIncompatibleError(msg) {
				return `{"strengths":["unavailable"],"weaknesses":["text-incompatible"]}`, true, false, nil
			}
		}
	}

	return c.extractTagsFromOpenAIResponse(responseBody.Bytes(), model)
}

// getTagsOpenAICompatibleUserRoleOnly retries tag fetching using only user-role messages
// for models that do not support the system role.
func (c *Client) getTagsOpenAICompatibleUserRoleOnly(model string, p config.ProviderConfig) (string, bool, bool, error) {
	log.Printf("Retrying tag generation for model %s with user role only", model)

	url := p.ChatCompletionsURL()
	client := &http.Client{Timeout: p.Timeout}

	isLocal := strings.HasPrefix(p.BaseURL, "http://localhost") || strings.HasPrefix(p.BaseURL, "http://127.0.0.1")
	runningWithMLX := isLocal && c.isLocalServerUsingMLX()

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": c.activeTaggingUserNoSysPrompt()},
		},
	}
	if !isReasoningModelNoSysMsg(model) && !runningWithMLX {
		payload["temperature"] = 0.0
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", false, false, fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", false, false, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", false, false, fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", false, false, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := extractAPIErrorMessage(responseBody.Bytes())
		if msg != "" {
			log.Printf("User-role-only retry API error for model %s: %s", model, msg)
			if strings.Contains(msg, "This is not a chat model") ||
				strings.Contains(msg, "not supported in the v1/chat/completions endpoint") ||
				strings.Contains(msg, "only supported in v1/responses") {
				return "", false, true, nil
			}
			if isTemperatureError(msg) {
				return c.getTagsOpenAICompatibleNoTemp(model, p)
			}
			if IsModelIncompatibleError(msg) {
				return `{"strengths":["unavailable"],"weaknesses":["text-incompatible"]}`, true, false, nil
			}
		}
	}

	return c.extractTagsFromOpenAIResponse(responseBody.Bytes(), model)
}

// getTagsAnthropic fetches capability tags using the Anthropic Messages API.
func (c *Client) getTagsAnthropic(model string, p config.ProviderConfig) (string, bool, bool, error) {
	url := fmt.Sprintf("%s/v1/messages", p.BaseURL)
	client := &http.Client{Timeout: p.Timeout}

	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": tagsMaxTokens,
		"system":     c.activeTaggingSystemPrompt(),
		"messages": []map[string]string{
			{"role": "user", "content": c.activeTaggingUserPrompt()},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", false, false, fmt.Errorf("error marshalling Anthropic payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", false, false, fmt.Errorf("error creating Anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	log.Printf("Sending Anthropic tags request for model %s", model)
	resp, err := client.Do(req)
	if err != nil {
		return "", false, false, fmt.Errorf("error making Anthropic request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var responseBody bytes.Buffer
	if _, err = responseBody.ReadFrom(resp.Body); err != nil {
		return "", false, false, fmt.Errorf("error reading Anthropic response body: %w", err)
	}
	log.Printf("Raw Anthropic tags response for model %s: %s", model, responseBody.String())

	content, err := extractAnthropicContent(responseBody.Bytes())
	if err != nil {
		return "", false, false, err
	}

	result := c.extractJSON(content)
	log.Printf("Model %s (anthropic) extracted tags: %s", model, result)
	return result, false, false, nil
}

// extractTagsFromOpenAIResponse parses an OpenAI-format response and returns the tags JSON.
func (c *Client) extractTagsFromOpenAIResponse(body []byte, model string) (string, bool, bool, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, false, fmt.Errorf("error decoding JSON response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", false, false, fmt.Errorf("missing or empty choices in response")
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", false, false, fmt.Errorf("invalid choice format")
	}
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", false, false, fmt.Errorf("missing message in choice")
	}
	content, ok := msg["content"].(string)
	if !ok {
		return "", false, false, fmt.Errorf("missing content in message")
	}

	log.Printf("Model %s raw tags content: %s", model, content)
	extracted := c.extractJSON(content)
	log.Printf("Model %s extracted tags: %s", model, extracted)
	return extracted, false, false, nil
}

// extractAnthropicContent extracts the text from an Anthropic Messages API response.
func extractAnthropicContent(body []byte) (string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error decoding Anthropic response: %w", err)
	}

	contentArr, ok := result["content"].([]interface{})
	if !ok || len(contentArr) == 0 {
		return "", fmt.Errorf("missing or empty content in Anthropic response")
	}
	block, ok := contentArr[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content block format")
	}
	text, ok := block["text"].(string)
	if !ok {
		return "", fmt.Errorf("missing text in Anthropic content block")
	}
	return text, nil
}

// isReasoningModelNoSysMsg returns true for o-series models that do not support
// system messages or temperature=0. This includes o1, o3, and o4 families.
func isReasoningModelNoSysMsg(model string) bool {
	m := strings.ToLower(model)
	// o1 family
	if m == "o1" || m == "o1-mini" || m == "o1-2024-12-17" || strings.HasPrefix(m, "o1-") {
		return true
	}
	// o3 family (mini and full, with or without date suffix)
	if m == "o3" || m == "o3-mini" || m == "o3-mini-2025-01-31" || strings.HasPrefix(m, "o3-") {
		return true
	}
	// o4 family
	if m == "o4" || m == "o4-mini" || strings.HasPrefix(m, "o4-") {
		return true
	}
	return false
}

// DisableModelsWithEmptyTags disables all models whose tags are empty or invalid.
// Call this after a full model refresh to clean up models that failed tagging.
func (c *Client) DisableModelsWithEmptyTags(modelDB interface {
	GetAllModels() (map[string]map[string]interface{}, error)
	EnableModel(name, source string, enabled bool) error
}) error {
	log.Println("Checking for models with empty tags to disable them")

	dbModels, err := modelDB.GetAllModels()
	if err != nil {
		return fmt.Errorf("error getting models from database: %w", err)
	}

	disabledCount := 0
	for name, modelMap := range dbModels {
		source, ok := modelMap["source"].(string)
		if !ok {
			log.Printf("Warning: Model %s has invalid source, skipping", name)
			continue
		}

		tagsStr, ok := modelMap["tags"].(string)
		if !ok {
			log.Printf("Warning: Model %s has invalid tags format, disabling", name)
			if err := modelDB.EnableModel(name, source, false); err != nil {
				log.Printf("Error disabling model %s: %v", name, err)
			} else {
				disabledCount++
			}
			continue
		}

		emptyTags := tagsStr == "{}" || tagsStr == `{"strengths":[], "weaknesses":[]}`
		var tags map[string]interface{}
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			log.Printf("Warning: Model %s has invalid JSON tags, disabling", name)
			if err := modelDB.EnableModel(name, source, false); err != nil {
				log.Printf("Error disabling model %s: %v", name, err)
			} else {
				disabledCount++
			}
			continue
		}

		strengths, hasStrengths := tags["strengths"].([]interface{})
		weaknesses, hasWeaknesses := tags["weaknesses"].([]interface{})

		if emptyTags || !hasStrengths || !hasWeaknesses || (len(strengths) == 0 && len(weaknesses) == 0) {
			log.Printf("Model %s has empty capability tags, disabling", name)
			if err := modelDB.EnableModel(name, source, false); err != nil {
				log.Printf("Error disabling model %s: %v", name, err)
			} else {
				disabledCount++
			}
		}
	}

	log.Printf("Disabled %d models with empty or invalid tags", disabledCount)
	return nil
}
