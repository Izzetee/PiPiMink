package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// mlxDetectionResult caches the result of MLX detection to avoid repeated API calls
var mlxDetectionCache struct {
	isMLX         bool
	lastChecked   time.Time
	cacheDuration time.Duration
	mutex         sync.RWMutex
}

func init() {
	// Initialize cache with a 1-hour duration
	mlxDetectionCache.cacheDuration = 1 * time.Hour
}

// IsLocalServerUsingMLX determines if the local LLM server is using MLX acceleration
// by combining system checks and API-based detection
// This is a public wrapper around the internal function for use by the server
func (c *Client) IsLocalServerUsingMLX() bool {
	return c.isLocalServerUsingMLX()
}

// isLocalServerUsingMLX determines if the local LLM server is using MLX acceleration
// by combining system checks and API-based detection
func (c *Client) isLocalServerUsingMLX() bool {
	// First check the cache
	mlxDetectionCache.mutex.RLock()
	if time.Since(mlxDetectionCache.lastChecked) < mlxDetectionCache.cacheDuration {
		result := mlxDetectionCache.isMLX
		mlxDetectionCache.mutex.RUnlock()
		return result
	}
	mlxDetectionCache.mutex.RUnlock()

	// Need to do a fresh check
	mlxDetectionCache.mutex.Lock()
	defer mlxDetectionCache.mutex.Unlock()

	// Double-check in case another goroutine updated while we were waiting for the write lock
	if time.Since(mlxDetectionCache.lastChecked) < mlxDetectionCache.cacheDuration {
		return mlxDetectionCache.isMLX
	}

	// Start with system-level check
	isMLXSystem := isSystemUsingMLX()

	// Then try API-based detection
	isMLXAPI := c.detectMLXViaAPI()

	// Combined decision:
	// - If API detection was conclusive, use that result
	// - Otherwise fall back to system-level detection
	result := isMLXAPI || isMLXSystem

	// Update cache
	mlxDetectionCache.isMLX = result
	mlxDetectionCache.lastChecked = time.Now()

	log.Printf("MLX detection result: %v (System: %v, API: %v)", result, isMLXSystem, isMLXAPI)
	return result
}

// isSystemUsingMLX checks if the system is running on macOS with MLX acceleration
// This is the original implementation, now used as a fallback
func isSystemUsingMLX() bool {
	// First check if we're on macOS
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check for MLX installation by running "pip list | grep mlx"
	cmd := exec.Command("bash", "-c", "pip list | grep -i mlx")
	output, err := cmd.Output()
	if err != nil {
		// If we can't run the command or MLX isn't found, return false
		return false
	}

	// If output contains "mlx", assume MLX is being used
	return strings.Contains(string(output), "mlx")
}

// detectMLXViaAPI attempts to detect MLX usage by querying the local LLM server's API
func (c *Client) detectMLXViaAPI() bool {
	// First, try Ollama-specific API detection
	if ollamaResult, conclusive := c.detectMLXViaOllamaAPI(); conclusive {
		return ollamaResult
	}

	// Then, try LM Studio-specific API detection
	if lmStudioResult, conclusive := c.detectMLXViaLMStudioAPI(); conclusive {
		return lmStudioResult
	}

	// As a last resort, try a generic detection approach by sending a test request
	// without temperature and seeing if it works
	return c.detectMLXViaGenericTest()
}

// detectMLXViaOllamaAPI attempts to detect MLX by querying Ollama's API
func (c *Client) detectMLXViaOllamaAPI() (bool, bool) {
	// Try to get information about available models from Ollama
	// First check if this is an Ollama server by trying a known Ollama-specific endpoint
	url := fmt.Sprintf("%s/api/tags", c.localProviderBaseURL())

	// Create HTTP client with appropriate timeout
	client := &http.Client{Timeout: 5 * time.Second}

	// Create and send the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for Ollama API: %v", err)
		return false, false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error querying Ollama API: %v", err)
		return false, false // Not conclusive
	}
	defer func() { _ = resp.Body.Close() }()

	// If we got a non-200 status code, this might not be Ollama
	if resp.StatusCode != 200 {
		log.Printf("Ollama API returned non-200 status: %d", resp.StatusCode)
		return false, false // Not conclusive
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding Ollama API response: %v", err)
		return false, false // Not conclusive
	}

	// Check if models key exists, confirming this is Ollama
	models, ok := result["models"].([]interface{})
	if !ok {
		// This might not be Ollama API format
		return false, false
	}

	log.Printf("Confirmed Ollama API, checking for MLX indicators")

	// Look through models for any indication of MLX
	for _, model := range models {
		modelInfo, ok := model.(map[string]interface{})
		if !ok {
			continue
		}

		// Check model name for MLX indicator
		name, ok := modelInfo["name"].(string)
		if ok && strings.Contains(strings.ToLower(name), "mlx") {
			log.Printf("Found MLX indicator in Ollama model name: %s", name)
			return true, true
		}

		// Check for details that might indicate MLX
		details, ok := modelInfo["details"].(map[string]interface{})
		if ok {
			// Look for any MLX indicators in details
			detailsJSON, _ := json.Marshal(details)
			if strings.Contains(strings.ToLower(string(detailsJSON)), "mlx") {
				log.Printf("Found MLX indicator in Ollama model details")
				return true, true
			}
		}
	}

	// Now try a more specific Ollama endpoint that might contain system info
	infoURL := fmt.Sprintf("%s/api/version", c.localProviderBaseURL())

	req, err = http.NewRequest("GET", infoURL, nil)
	if err != nil {
		log.Printf("Failed to create request for Ollama version API: %v", err)
		return false, false
	}

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error querying Ollama version API: %v", err)
		return false, false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return false, false
	}

	var versionInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return false, false
	}

	// Check if this is running on macOS
	platform, hasPlatform := versionInfo["platform"].(string)
	if hasPlatform && strings.Contains(strings.ToLower(platform), "darwin") {
		// Check for any MLX indicators in the version info
		versionJSON, _ := json.Marshal(versionInfo)
		if strings.Contains(strings.ToLower(string(versionJSON)), "mlx") {
			log.Printf("Found MLX indicator in Ollama version info")
			return true, true
		}

		// If we're on macOS (Darwin) and using Ollama, also check the model list
		// using a different endpoint that might have more details
		modelsURL := fmt.Sprintf("%s/api/models", c.localProviderBaseURL())
		req, err = http.NewRequest("GET", modelsURL, nil)
		if err != nil {
			return false, false
		}

		resp, err = client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return false, false
		}
		defer func() { _ = resp.Body.Close() }()

		var modelsInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&modelsInfo); err != nil {
			return false, false
		}

		// Check for MLX indicators in model info
		modelsJSON, _ := json.Marshal(modelsInfo)
		if strings.Contains(strings.ToLower(string(modelsJSON)), "mlx") {
			log.Printf("Found MLX indicator in Ollama models API")
			return true, true
		}

		// If we're on macOS and it's Ollama, there's a good chance MLX is being used
		// Perform a test query without temperature
		testResult := c.testOllamaModelWithoutTemperature()
		if testResult {
			log.Printf("Ollama on macOS responding to queries without temperature parameter - likely using MLX")
			return true, true
		}
	}

	// If we got this far, we're querying Ollama but found no MLX indicators
	log.Printf("This appears to be Ollama, but no clear MLX indicators found")
	return false, true // Return conclusive=true as we know it's Ollama
}

// detectMLXViaLMStudioAPI attempts to detect MLX by querying LM Studio's API
func (c *Client) detectMLXViaLMStudioAPI() (bool, bool) {
	// Check if this is LM Studio by looking for its OpenAI compatible endpoints
	// LM Studio follows OpenAI API format with some differences

	// Try LM Studio's model list endpoint (standard OpenAI endpoint)
	url := fmt.Sprintf("%s/v1/models", c.localProviderBaseURL())

	// Create HTTP client with appropriate timeout
	client := &http.Client{Timeout: 5 * time.Second}

	// Create and send the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for LM Studio API: %v", err)
		return false, false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error querying LM Studio API: %v", err)
		return false, false // Not conclusive
	}
	defer func() { _ = resp.Body.Close() }()

	// If endpoint doesn't exist or returns error, not conclusive
	if resp.StatusCode != 200 {
		return false, false
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, false // Not conclusive
	}

	// Check for OpenAI-style "data" array which LM Studio uses
	data, ok := result["data"].([]interface{})
	if !ok {
		return false, false // Not LM Studio
	}

	// This is likely LM Studio since it's returning a standard OpenAI models response
	// but didn't match Ollama's API pattern
	log.Printf("Detected possible LM Studio API")

	// Check for explicit MLX indicators in model names
	for _, model := range data {
		modelInfo, ok := model.(map[string]interface{})
		if !ok {
			continue
		}

		// Check model ID/name for MLX indicator
		id, ok := modelInfo["id"].(string)
		if ok && (strings.Contains(strings.ToLower(id), "mlx") ||
			strings.Contains(strings.ToLower(id), "mac") ||
			strings.Contains(strings.ToLower(id), "apple")) {
			log.Printf("Found MLX indicator in LM Studio model id: %s", id)
			return true, true
		}
	}

	// If we're on macOS, LM Studio is likely to use MLX if available
	// Try to do a specific check for macOS + LM Studio
	if runtime.GOOS == "darwin" {
		// LM Studio on Mac often uses MLX. Let's do a test call without temperature
		log.Printf("LM Studio on macOS detected - testing for MLX compatibility")

		// Find a model to test with
		var testModelID string
		if len(data) > 0 {
			if model, ok := data[0].(map[string]interface{}); ok {
				if id, ok := model["id"].(string); ok {
					testModelID = id
				}
			}
		}

		if testModelID != "" {
			// Test with a request without temperature
			chatURL := fmt.Sprintf("%s/v1/chat/completions", c.localProviderBaseURL())

			// Create test payload without temperature parameter
			payloadWithoutTemp := map[string]interface{}{
				"model": testModelID,
				"messages": []map[string]string{
					{"role": "user", "content": "hello"},
				},
				// deliberately omit temperature
			}

			payloadBytes, err := json.Marshal(payloadWithoutTemp)
			if err != nil {
				log.Printf("Error marshalling LM Studio test payload: %v", err)
				return false, true
			}

			// Send the test request
			req, err = http.NewRequest("POST", chatURL, strings.NewReader(string(payloadBytes)))
			if err != nil {
				log.Printf("Failed to create LM Studio test request: %v", err)
				return false, true
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err = client.Do(req)
			if err != nil {
				log.Printf("Error sending LM Studio test request: %v", err)
				return false, true
			}
			defer func() { _ = resp.Body.Close() }()

			// If request succeeded without temperature parameter, it's likely using MLX
			isSuccess := resp.StatusCode == 200

			log.Printf("LM Studio MLX test result: Success=%v (StatusCode=%d)", isSuccess, resp.StatusCode)
			return isSuccess, true
		}
	}

	// Not conclusive about MLX, but we know it's LM Studio
	return false, true
}

// detectMLXViaGenericTest performs a generic test by sending a chat completion request
// without temperature parameter and checking if it succeeds
func (c *Client) detectMLXViaGenericTest() bool {
	// Only proceed with this test if we're on macOS
	if runtime.GOOS != "darwin" {
		log.Printf("Not on macOS, skipping generic MLX detection")
		return false
	}

	log.Printf("Performing generic MLX detection test via API")

	// First, check if MLX python package is installed
	// This increases the likelihood that MLX is being used
	cmd := exec.Command("bash", "-c", "pip list | grep -i mlx")
	output, err := cmd.Output()
	mlxInstalled := err == nil && strings.Contains(string(output), "mlx")

	if !mlxInstalled {
		// Also try pip3
		cmd = exec.Command("bash", "-c", "pip3 list | grep -i mlx")
		output, err = cmd.Output()
		mlxInstalled = err == nil && strings.Contains(string(output), "mlx")
	}

	if mlxInstalled {
		log.Printf("MLX Python package is installed on the system")
	}

	// Get a list of models from OpenAI-compatible endpoint
	url := fmt.Sprintf("%s/v1/models", c.localProviderBaseURL())
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for models list: %v", err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting models list: %v", err)
		return false
	}

	// If models endpoint doesn't exist, skip this test
	if resp.StatusCode != 200 {
		log.Printf("Models endpoint returned non-200 status: %d", resp.StatusCode)
		return false
	}

	defer func() { _ = resp.Body.Close() }()

	// Parse the response to get a model to test with
	var modelsResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		log.Printf("Error decoding models response: %v", err)
		return false
	}

	// Extract the first model ID to test with
	var testModelID string
	if data, ok := modelsResponse["data"].([]interface{}); ok && len(data) > 0 {
		if model, ok := data[0].(map[string]interface{}); ok {
			if id, ok := model["id"].(string); ok {
				testModelID = id
			}
		}
	}

	if testModelID == "" {
		log.Printf("Could not find a model ID to test with")
		return false
	}

	log.Printf("Testing model %s without temperature parameter", testModelID)

	// Now test the model with a request that deliberately omits temperature
	chatURL := fmt.Sprintf("%s/v1/chat/completions", c.localProviderBaseURL())

	// Create test payload without temperature
	payloadWithoutTemp := map[string]interface{}{
		"model": testModelID,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
		// deliberately omit temperature
	}

	payloadBytes, err := json.Marshal(payloadWithoutTemp)
	if err != nil {
		log.Printf("Error marshalling test payload: %v", err)
		return false
	}

	// Send the test request
	req, err = http.NewRequest("POST", chatURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("Failed to create test request: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error sending test request: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	// If request succeeded without temperature parameter, it might be MLX
	isSuccess := resp.StatusCode == 200

	// If we're on macOS, MLX is installed, and the request succeeded without temperature,
	// it's very likely MLX is being used
	if isSuccess && mlxInstalled {
		log.Printf("High confidence MLX detection: macOS + MLX installed + API works without temperature")
		return true
	}

	// If we're on macOS and the request succeeded without temperature, it's likely MLX
	if isSuccess {
		log.Printf("Medium confidence MLX detection: macOS + API works without temperature")
		return true
	}

	// If MLX is installed but the request failed, it might still be using MLX but with a different setup
	if mlxInstalled {
		log.Printf("Low confidence MLX detection: macOS + MLX installed but API requires temperature")
		// Return true with lower confidence if MLX is installed but API test failed
		// This defaults to omitting temperature, which is safer
		return true
	}

	log.Printf("Generic MLX test result: No evidence of MLX (Success=%v, MLX installed=%v)", isSuccess, mlxInstalled)
	return false
}

// testOllamaModelWithoutTemperature performs a quick test to see if Ollama models work without temperature parameter
// This can indicate MLX support as MLX doesn't need/use temperature
func (c *Client) testOllamaModelWithoutTemperature() bool {
	log.Printf("Testing Ollama model without temperature parameter")

	// Get a list of models first to find one to test with
	url := fmt.Sprintf("%s/api/models", c.localProviderBaseURL())
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for Ollama models list: %v", err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting Ollama models list: %v", err)
		return false
	}

	// If models endpoint doesn't exist, skip this test
	if resp.StatusCode != 200 {
		log.Printf("Ollama models endpoint returned non-200 status: %d", resp.StatusCode)
		return false
	}

	defer func() { _ = resp.Body.Close() }()

	// Parse the response to get a model to test with
	var modelsResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		log.Printf("Error decoding Ollama models response: %v", err)
		return false
	}

	// Extract the first model name to test with
	var testModelName string
	if models, ok := modelsResponse["models"].([]interface{}); ok && len(models) > 0 {
		if model, ok := models[0].(map[string]interface{}); ok {
			if name, ok := model["name"].(string); ok {
				testModelName = name
			}
		}
	}

	if testModelName == "" {
		log.Printf("Could not find an Ollama model to test with")
		return false
	}

	// Now test the model with a request that deliberately omits temperature
	// Ollama uses a different API endpoint format
	chatURL := fmt.Sprintf("%s/api/chat", c.localProviderBaseURL())

	// Create test payload without temperature
	payloadWithoutTemp := map[string]interface{}{
		"model": testModelName,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
		// deliberately omit temperature
	}

	payloadBytes, err := json.Marshal(payloadWithoutTemp)
	if err != nil {
		log.Printf("Error marshalling Ollama test payload: %v", err)
		return false
	}

	// Send the test request
	req, err = http.NewRequest("POST", chatURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("Failed to create Ollama test request: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error sending Ollama test request: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	// If request succeeded without temperature parameter, it might be MLX
	isSuccess := resp.StatusCode == 200

	log.Printf("Ollama MLX test result: Success=%v (StatusCode=%d)", isSuccess, resp.StatusCode)
	return isSuccess
}
