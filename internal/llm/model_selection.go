package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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

// selectionProviderURL returns the base URL and API key for the provider used
// to make routing decisions (the meta-router).
func (c *Client) selectionProviderCredentials() (baseURL, apiKey string) {
	if p, ok := c.selectionProvider(); ok {
		return p.BaseURL, p.APIKey
	}
	// Hard fallback — should not normally be reached.
	return "https://api.openai.com", ""
}

// DecideModelBasedOnCapabilities analyzes a message and determines the best model to use
// based on model capabilities (strengths and weaknesses).
func (c *Client) DecideModelBasedOnCapabilities(message string, availableModels map[string]models.ModelInfo) (string, error) {
	// Start timing the execution
	startTime := time.Now()

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
		return model, nil
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

	// Use the configured selection provider for routing decisions.
	_, selAPIKey := c.selectionProviderCredentials()
	selProvider, hasSP := c.selectionProvider()
	var selTimeout = 2 * time.Minute
	url := "https://api.openai.com/v1/chat/completions" // hard fallback, normally overridden below
	if hasSP {
		selTimeout = selProvider.Timeout
		url = selProvider.ChatCompletionsURL()
	}
	apiKey := selAPIKey
	client := &http.Client{Timeout: selTimeout}

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
		return "", fmt.Errorf("error marshalling model capabilities: %w", err)
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
	selectionModel := c.selectionModel()
	fallbackModel := c.fallbackModel()
	payload := map[string]interface{}{
		"model":       selectionModel,
		"temperature": 0.0, // Set temperature to 0 for consistent responses
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemMessage,
			},
			{
				"role":    "user",
				"content": message,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("Sending model selection request")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read and process the response
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Log the raw response for debugging purposes
	responseStr := responseBody.String()
	log.Printf("Raw API response: %s", responseStr)

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody.Bytes(), &result); err != nil {
		return "", fmt.Errorf("error decoding response JSON: %w", err)
	}

	// Extract model name from response with detailed error checking
	choices, ok := result["choices"].([]interface{})
	if !ok {
		log.Printf("Error: 'choices' field is missing or not an array. Response keys: %v", getMapKeys(result))
		return "", fmt.Errorf("error extracting choices from response: 'choices' field missing or invalid")
	}

	if len(choices) == 0 {
		log.Printf("Error: 'choices' array is empty")
		return "", fmt.Errorf("error extracting choices from response: 'choices' array is empty")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		log.Printf("Error: first choice is not an object. Type: %T", choices[0])
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

	log.Printf("Extracted model name: '%s'", content)

	// Extract JSON from the response content
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		log.Printf("Error: No valid JSON object found in response content")
		return fallbackModel, fmt.Errorf("error extracting JSON from response")
	}

	jsonContent := content[jsonStart : jsonEnd+1]
	var modelSelection map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &modelSelection); err != nil {
		log.Printf("Error parsing JSON from response: %v", err)
		return fallbackModel, fmt.Errorf("error parsing JSON from response: %w", err)
	}

	selectedModel, ok := modelSelection["modelname"].(string)
	if !ok {
		log.Printf("Error: 'modelname' field missing or not a string in JSON. JSON keys: %v", getMapKeys(modelSelection))
		return fallbackModel, fmt.Errorf("error extracting modelname from JSON")
	}

	reasonStr, ok := modelSelection["reason"].(string)
	if ok {
		log.Printf("Model selection reason: %s", reasonStr)
	} else {
		log.Printf("Warning: No reason provided for model selection")
	}

	selectedModel = strings.TrimSpace(selectedModel)
	log.Printf("Extracted model name: '%s'", selectedModel)

	// Ensure the selected model is valid
	if _, exists := availableModels[selectedModel]; !exists {
		log.Printf("Selected model '%s' is not in available models, using default", selectedModel)
		elapsedTime := time.Since(startTime)
		log.Printf("Model selection with %s took %v to evaluate the prompt", selectionModel, elapsedTime)
		if cacheKeyErr == nil {
			c.decisionCache.set(cacheKey, fallbackModel)
		}
		return fallbackModel, nil
	}

	log.Printf("Successfully selected model based on capabilities: %s", selectedModel)
	if cacheKeyErr == nil {
		c.decisionCache.set(cacheKey, selectedModel)
	}
	elapsedTime := time.Since(startTime)
	log.Printf("Model selection with %s took %v to evaluate the prompt", selectionModel, elapsedTime)
	return selectedModel, nil
}

// DecideModel is a simpler model decision function that returns a default model.
// This can be expanded with more sophisticated logic in the future.
func (c *Client) DecideModel(message string) string {
	// Default model selection logic
	// Could be expanded with more sophisticated logic
	return c.fallbackModel()
}
