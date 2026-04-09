package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

const (
	defaultSelectionModel = "gpt-4-turbo"
)

func (c *Client) selectionModel() string {
	if c != nil && c.Config != nil && strings.TrimSpace(c.Config.ModelSelectionModel) != "" {
		return strings.TrimSpace(c.Config.ModelSelectionModel)
	}
	return defaultSelectionModel
}

func (c *Client) fallbackModel() string {
	if c != nil && c.Config != nil && strings.TrimSpace(c.Config.DefaultChatModel) != "" {
		return strings.TrimSpace(c.Config.DefaultChatModel)
	}
	return c.selectionModel()
}

// resolvedSelectionProvider returns the ProviderConfig for the meta-router
// with per-model overrides applied (base_url, api_key, type, chat_path).
func (c *Client) resolvedSelectionProvider() (config.ProviderConfig, bool) {
	p, ok := c.selectionProvider()
	if !ok {
		return p, false
	}
	return p.ForModel(c.selectionModel()), true
}

// DecideModelBasedOnCapabilities analyzes a message and determines the best model to use
// based on model capabilities (strengths and weaknesses).
func (c *Client) DecideModelBasedOnCapabilities(message string, availableModels map[string]models.ModelInfo) (models.RoutingResult, error) {
	// Start timing the execution
	startTime := time.Now()

	selectionModel := c.selectionModel()

	cacheKey, cacheKeyErr := buildDecisionCacheKey(message, availableModels)
	if cacheKeyErr != nil {
		log.Printf("Could not build routing decision cache key: %v", cacheKeyErr)
	} else if model, ok, status := c.decisionCache.getWithStatus(cacheKey); ok {
		log.Printf("Routing decision cache %s for prompt", status)
		if stats, emit := c.decisionCache.maybeStatsSummary(); emit {
			totalLookups := stats.Hits + stats.Misses
			hitRate := 0.0
			if totalLookups > 0 {
				hitRate = float64(stats.Hits) * 100 / float64(totalLookups)
			}
			log.Printf("Routing decision cache summary: hits=%d misses=%d expired=%d sets=%d evictions=%d hit_rate=%.2f%%", stats.Hits, stats.Misses, stats.Expired, stats.Sets, stats.Evictions, hitRate)
		}
		return models.RoutingResult{
			ModelName:        model,
			CacheHit:         true,
			EvaluatorModel:   selectionModel,
			EvaluationTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	} else {
		log.Printf("Routing decision cache %s for prompt", status)
		if stats, emit := c.decisionCache.maybeStatsSummary(); emit {
			totalLookups := stats.Hits + stats.Misses
			hitRate := 0.0
			if totalLookups > 0 {
				hitRate = float64(stats.Hits) * 100 / float64(totalLookups)
			}
			log.Printf("Routing decision cache summary: hits=%d misses=%d expired=%d sets=%d evictions=%d hit_rate=%.2f%%", stats.Hits, stats.Misses, stats.Expired, stats.Sets, stats.Evictions, hitRate)
		}
	}

	log.Printf("Starting model decision process for message: %.100s...", message)
	log.Printf("Available models count: %d", len(availableModels))

	errResult := func(err error) (models.RoutingResult, error) {
		return models.RoutingResult{EvaluatorModel: selectionModel}, err
	}
	fallbackResult := func(err error) (models.RoutingResult, error) {
		fb := c.fallbackModel()
		return models.RoutingResult{
			ModelName:        fb,
			EvaluatorModel:   selectionModel,
			EvaluationTimeMs: time.Since(startTime).Milliseconds(),
			FallbackUsed:     true,
		}, err
	}

	// Use the configured selection provider for routing decisions (with per-model overrides).
	selProvider, hasSP := c.resolvedSelectionProvider()
	if !hasSP {
		return errResult(fmt.Errorf("no selection provider configured"))
	}
	client := &http.Client{Timeout: selProvider.Timeout}

	// Prepare model capabilities information
	hasBenchmarkScores := false
	hasLatency := false
	modelCapabilities := make(map[string]map[string]interface{})
	for name, info := range availableModels {
		if !info.Enabled {
			continue
		}

		var tags map[string]interface{}
		if err := json.Unmarshal([]byte(info.Tags), &tags); err != nil {
			log.Printf("Error parsing tags for model %s: %v", name, err)
			continue
		}

		entry := map[string]interface{}{
			"source": info.Source,
			"tags":   tags,
		}
		if len(info.BenchmarkScores) > 0 {
			entry["benchmark_scores"] = info.BenchmarkScores
			hasBenchmarkScores = true
		}
		if info.AvgLatencyMs != nil {
			entry["avg_latency_ms"] = *info.AvgLatencyMs
			hasLatency = true
		}
		modelCapabilities[name] = entry
	}
	log.Printf("Prepared capabilities for %d enabled models (benchmark scores: %v, latency: %v)", len(modelCapabilities), hasBenchmarkScores, hasLatency)

	modelCapabilitiesJSON, err := json.Marshal(modelCapabilities)
	if err != nil {
		return errResult(fmt.Errorf("error marshalling model capabilities: %w", err))
	}

	// Build the system message. Mention benchmark scores and latency when present.
	benchmarkNote := ""
	if hasBenchmarkScores {
		benchmarkNote = "Where available, models also include 'benchmark_scores': an object mapping task categories " +
			"(e.g. \"coding\", \"reasoning\", \"creative-writing\") to empirically measured average scores in [0.0, 1.0]. " +
			"Higher scores indicate better measured performance on that category. " +
			"Capability tags are the PRIMARY signal. Benchmark scores are the SECONDARY signal — prefer higher scores for the relevant category when tags are otherwise equal. "
	}
	latencyNote := ""
	if hasLatency {
		latencyNote = "Some models include 'avg_latency_ms': the measured average response time in milliseconds. " +
			"Latency is the TERTIARY signal — use it only as a tiebreaker when capability tags and benchmark scores point to equally suitable models. " +
			"When quality signals are equal, prefer the model with the lowest avg_latency_ms. "
	}

	systemMessage := fmt.Sprintf(
		"You are a model selection system with NO PRIOR KNOWLEDGE about any AI models. Your ONLY task is to match user requests to the most appropriate model based EXCLUSIVELY on the data provided to you. "+
			"You MUST IGNORE any knowledge you might have about AI models and ONLY use the explicit capability data provided. "+
			"Each model entry has 'tags' (self-reported strengths and weaknesses) and optionally 'benchmark_scores' and 'avg_latency_ms'. "+
			"DECISION PRIORITY: (1) capability tags — most important, (2) benchmark scores — secondary, (3) avg_latency_ms — tiebreaker only. "+
			benchmarkNote+
			latencyNote+
			"Analyse the user message and select the model whose strengths and benchmark scores best match the requirements, using latency only to break ties. "+
			"IMPORTANT: You MUST return a valid JSON object with the following fields: "+
			"'modelname' (string name of the selected model, must be one of the available models), "+
			"'reason' (a detailed explanation of why you chose this model), "+
			"'matching_tags' (an array of specific tags from the selected model that match the requirements in the user message), and "+
			"'tag_relevance' (an object with tag names as keys and relevance scores from 0-10 as values, indicating how relevant each tag is to the user's request). "+
			"Example response format: {\"modelname\": \"gpt-4\", \"reason\": \"Selected because the request requires advanced reasoning capabilities\", \"matching_tags\": [\"complex-reasoning\"], \"tag_relevance\": {\"complex-reasoning\": 9}}. "+
			"Available models with their capabilities: %s",
		string(modelCapabilitiesJSON),
	)

	// Create the request payload with temperature=0 for consistent responses
	fallbackModel := c.fallbackModel()

	var jsonPayload []byte
	var endpoint string

	switch selProvider.Type {
	case config.ProviderTypeAnthropic:
		payload := map[string]interface{}{
			"model":      selectionModel,
			"max_tokens": 4096,
			"system":     systemMessage,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
		}
		jsonPayload, err = json.Marshal(payload)
		endpoint = selProvider.BaseURL + "/v1/messages"
	default:
		payload := map[string]interface{}{
			"model":       selectionModel,
			"temperature": 0.0,
			"messages": []map[string]string{
				{"role": "system", "content": systemMessage},
				{"role": "user", "content": message},
			},
		}
		jsonPayload, err = json.Marshal(payload)
		endpoint = selProvider.ChatCompletionsURL()
	}
	if err != nil {
		return errResult(fmt.Errorf("error marshalling payload: %w", err))
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return errResult(fmt.Errorf("error creating request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	switch selProvider.Type {
	case config.ProviderTypeAnthropic:
		if selProvider.APIKey != "" {
			req.Header.Set("x-api-key", selProvider.APIKey)
		}
		req.Header.Set("anthropic-version", anthropicVersion)
	default:
		if selProvider.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+selProvider.APIKey)
		}
	}

	log.Printf("Sending model selection request to %s (type: %s)", endpoint, selProvider.Type)
	resp, err := client.Do(req)
	if err != nil {
		return errResult(fmt.Errorf("error making request: %w", err))
	}
	defer func() { _ = resp.Body.Close() }()

	// Read and process the response
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		return errResult(fmt.Errorf("error reading response body: %w", err))
	}

	// Log the raw response for debugging purposes
	responseStr := responseBody.String()
	log.Printf("Raw API response: %s", responseStr)

	// Extract text content from the response based on provider type.
	var content string
	switch selProvider.Type {
	case config.ProviderTypeAnthropic:
		content, err = extractAnthropicContent(responseBody.Bytes())
		if err != nil {
			return errResult(fmt.Errorf("error extracting content from Anthropic response: %w", err))
		}
	default:
		content, err = extractOpenAISelectionContent(responseBody.Bytes())
		if err != nil {
			return errResult(err)
		}
	}

	log.Printf("Extracted model selection content: '%s'", content)

	// Extract JSON from the response content
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		log.Printf("Error: No valid JSON object found in response content")
		return fallbackResult(fmt.Errorf("error extracting JSON from response"))
	}

	jsonContent := content[jsonStart : jsonEnd+1]
	var modelSelection map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &modelSelection); err != nil {
		log.Printf("Error parsing JSON from response: %v", err)
		return fallbackResult(fmt.Errorf("error parsing JSON from response: %w", err))
	}

	selectedModel, ok := modelSelection["modelname"].(string)
	if !ok {
		log.Printf("Error: 'modelname' field missing or not a string in JSON. JSON keys: %v", getMapKeys(modelSelection))
		return fallbackResult(fmt.Errorf("error extracting modelname from JSON"))
	}

	reasonStr, _ := modelSelection["reason"].(string)
	if reasonStr != "" {
		log.Printf("Model selection reason: %s", reasonStr)
	} else {
		log.Printf("Warning: No reason provided for model selection")
	}

	// Extract matching_tags and tag_relevance from the response
	var matchingTags []string
	if rawTags, ok := modelSelection["matching_tags"].([]interface{}); ok {
		for _, t := range rawTags {
			if s, ok := t.(string); ok {
				matchingTags = append(matchingTags, s)
			}
		}
	}
	tagRelevance := make(map[string]float64)
	if rawRelevance, ok := modelSelection["tag_relevance"].(map[string]interface{}); ok {
		for k, v := range rawRelevance {
			if f, ok := v.(float64); ok {
				tagRelevance[k] = f
			}
		}
	}

	selectedModel = strings.TrimSpace(selectedModel)
	log.Printf("Extracted model name: '%s'", selectedModel)

	evalTimeMs := time.Since(startTime).Milliseconds()

	// Ensure the selected model is valid
	if _, exists := availableModels[selectedModel]; !exists {
		log.Printf("Selected model '%s' is not in available models, using default", selectedModel)
		log.Printf("Model selection with %s took %v to evaluate the prompt", selectionModel, time.Since(startTime))
		if cacheKeyErr == nil {
			c.decisionCache.set(cacheKey, fallbackModel)
		}
		return fallbackResult(nil)
	}

	log.Printf("Successfully selected model based on capabilities: %s", selectedModel)
	if cacheKeyErr == nil {
		c.decisionCache.set(cacheKey, selectedModel)
	}
	log.Printf("Model selection with %s took %v to evaluate the prompt", selectionModel, time.Since(startTime))

	return models.RoutingResult{
		ModelName:        selectedModel,
		Reason:           reasonStr,
		MatchingTags:     matchingTags,
		TagRelevance:     tagRelevance,
		EvaluatorModel:   selectionModel,
		EvaluationTimeMs: evalTimeMs,
	}, nil
}

// extractOpenAISelectionContent extracts the text content from an OpenAI-compatible
// chat completions response, with detailed error logging for debugging.
func extractOpenAISelectionContent(body []byte) (string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error decoding response JSON: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok {
		log.Printf("Error: 'choices' field is missing or not an array. Response keys: %v", getMapKeys(result))
		return "", fmt.Errorf("error extracting choices from response: 'choices' field missing or invalid")
	}
	if len(choices) == 0 {
		return "", fmt.Errorf("error extracting choices from response: 'choices' array is empty")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error extracting first choice from response")
	}

	messageObj, ok := choice["message"].(map[string]interface{})
	if !ok {
		log.Printf("Error: 'message' field missing or invalid in choice. Choice keys: %v", getMapKeys(choice))
		return "", fmt.Errorf("error extracting message from response")
	}

	content, ok := messageObj["content"].(string)
	if !ok {
		log.Printf("Error: 'content' field missing or not a string in message. Message keys: %v", getMapKeys(messageObj))
		return "", fmt.Errorf("error extracting content from message")
	}
	return content, nil
}

// DecideModel is a simpler model decision function that returns a default model.
// This can be expanded with more sophisticated logic in the future.
func (c *Client) DecideModel(message string) string {
	// Default model selection logic
	// Could be expanded with more sophisticated logic
	return c.fallbackModel()
}
