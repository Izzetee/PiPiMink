// Package server provides handler functions for API endpoints
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/models"

	"github.com/gorilla/mux"
)

// writeSSEChunk writes a single SSE data line.
func writeSSEChunk(w http.ResponseWriter, data string) {
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// handleChat handles chat requests
// @Summary Process a chat request
// @Description Routes the chat request to the most appropriate AI model based on message content
// @Tags chat
// @Accept json
// @Produce json
// @Param request body models.ChatRequest true "Chat request"
// @Success 200 {object} models.ChatResponse
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /chat [post]
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received chat request")

	var chatReq models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build the messages array. If the caller supplied a full history use it;
	// otherwise wrap the single Message field for backward compatibility.
	messages := chatReq.Messages
	if len(messages) == 0 {
		if chatReq.Message == "" {
			http.Error(w, "message or messages is required", http.StatusBadRequest)
			return
		}
		messages = []map[string]interface{}{
			{"role": "user", "content": chatReq.Message},
		}
	}

	// Extract the last user message for routing decision.
	routingMessage := chatReq.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if role, ok := messages[i]["role"].(string); ok && role == "user" {
			if content, ok := messages[i]["content"].(string); ok {
				routingMessage = content
				break
			}
		}
	}

	// Get enabled models for selection
	s.modelMutex.RLock()
	enabledModels := s.modelCollection.GetEnabledModels()
	s.modelMutex.RUnlock()

	// If no enabled models found, return error
	if len(enabledModels) == 0 {
		log.Printf("No enabled models found")
		http.Error(w, "No enabled models available", http.StatusInternalServerError)
		return
	}

	// Use LLM to decide which model to use based on the message and model capabilities
	routingResult, err := s.llmClient.DecideModelBasedOnCapabilities(routingMessage, enabledModels)
	modelName := routingResult.ModelName
	if err != nil {
		log.Printf("Error deciding model: %v, falling back to default model", err)
		modelName = s.getFallbackModelName(enabledModels)
		if modelName == "" {
			http.Error(w, "No enabled models available", http.StatusInternalServerError)
			return
		}
	}

	// Get model info for the selected model
	s.modelMutex.RLock()
	modelInfo, exists := s.modelCollection.GetModel(modelName)
	s.modelMutex.RUnlock()

	if !exists {
		log.Printf("Model %s not found, using default response", modelName)
		chatRes := models.ChatResponse{
			Response: "This is a placeholder response (model not found)",
			Model:    modelName,
		}
		_ = json.NewEncoder(w).Encode(chatRes)
		return
	}

	// Call the selected model with the full message history
	log.Printf("Routing request to model: %s (history length: %d)", modelName, len(messages))
	responseStart := time.Now()
	response, err := s.llmClient.ChatWithModel(modelInfo, modelName, messages)
	responseLatencyMs := time.Since(responseStart).Milliseconds()
	status := "success"
	if err != nil {
		log.Printf("Error calling model: %v", err)
		status = "error"
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		// Still log the routing decision even on error
		s.logRoutingDecision(routingResult, routingMessage, modelInfo.Source, responseLatencyMs, status, getUserID(r))
		return
	}

	// Log the routing decision asynchronously
	s.logRoutingDecision(routingResult, routingMessage, modelInfo.Source, responseLatencyMs, status, getUserID(r))

	chatRes := models.ChatResponse{
		Response: response,
		Model:    modelName,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chatRes)
}

// handleUpdateModels handles requests to manually update the model database
// @Summary Update model information
// @Description Triggers an update of model information from all configured sources
// @Tags models,admin
// @Accept json
// @Produce json
// @Param X-API-Key header string true "Admin API Key"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/update [post]
// Auth: admin (enforced by middleware)
func (s *Server) handleUpdateModels(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to update models")

	// Start the update process
	go func() {
		log.Println("Starting model update process")
		if err := s.fetchAndTagModels(); err != nil {
			log.Printf("Error updating models: %v", err)
		} else {
			log.Println("Model update completed successfully")
		}
		// Disable any models that returned empty capability tags
		s.disableEmptyTagModels()
		// Reload models to ensure we're using the updated data
		if err := s.loadModelsFromDatabase(); err != nil {
			log.Printf("Error reloading models from database: %v", err)
		}
		s.logModels()
	}()

	// Send response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Model update process started",
	})
}

// handleListModels handles requests to list all available models
// @Summary List available models
// @Description Returns a list of all available models
// @Tags models
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /models [get]
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()

	// Convert internal model structure to a response format
	modelsList := make([]map[string]interface{}, 0, len(s.modelCollection.Models))
	for name, info := range s.modelCollection.Models {
		tagged := hasUsefulTags(info.Tags)
		modelData := map[string]interface{}{
			"name":            name,
			"source":          info.Source,
			"enabled":         info.Enabled,
			"tagged":          tagged,
			"hasReasoning":    info.HasReasoning,
			"updatedAt":       info.UpdatedAt,
			"benchmarkScores": info.BenchmarkScores,
			"avgLatencyMs":    info.AvgLatencyMs,
			"tags":            parseTags(info.Tags),
		}
		if tagged {
			modelData["taggedBy"] = "self-assessment"
		}
		modelsList = append(modelsList, modelData)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"models":         modelsList,
		"count":          len(modelsList),
		"benchmarkJudge": s.resolveBenchmarkJudgeName(),
	})
}

// handleOpenAIChatCompletions handles the OpenAI chat completions API endpoint
// @Summary OpenAI-compatible chat completions
// @Description OpenAI-compatible endpoint for chat completions
// @Tags openai,chat
// @Accept json
// @Produce json
// @Param request body object true "OpenAI-compatible chat request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/chat/completions [post]
func (s *Server) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to OpenAI-compatible /v1/chat/completions endpoint")

	// Parse OpenAI-style request
	var openAIReq struct {
		Model     string                   `json:"model"`
		Messages  []map[string]interface{} `json:"messages"`
		MaxTokens int                      `json:"max_tokens,omitempty"`
		Stream    bool                     `json:"stream,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&openAIReq); err != nil {
		log.Printf("Error decoding OpenAI-style request: %v", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	log.Printf("Received chat completion request for model: %s", openAIReq.Model)

	// Extract the actual message content from the messages array (usually the last user message)
	var userMessage string
	for i := len(openAIReq.Messages) - 1; i >= 0; i-- {
		msg := openAIReq.Messages[i]
		if role, ok := msg["role"].(string); ok && role == "user" {
			if content, ok := msg["content"].(string); ok {
				userMessage = content
				break
			}
		}
	}

	if userMessage == "" {
		log.Printf("No user message found in the request")
		http.Error(w, "No user message found", http.StatusBadRequest)
		return
	}

	// If model is specified, use it directly
	var modelName string
	var modelInfo models.ModelInfo
	var exists bool

	if openAIReq.Model != "" {
		modelName = openAIReq.Model
		s.modelMutex.RLock()
		modelInfo, exists = s.modelCollection.GetModel(modelName)
		s.modelMutex.RUnlock()
	}

	// Variables for routing decision logging (captured by defer closure below)
	var openAIResponseLatencyMs int64
	openAIResponseStatus := "success"

	// If model doesn't exist or isn't specified, route based on capabilities
	if !exists {
		// Get enabled models for selection
		s.modelMutex.RLock()
		enabledModels := s.modelCollection.GetEnabledModels()
		s.modelMutex.RUnlock()

		// If no enabled models found, return error
		if len(enabledModels) == 0 {
			log.Printf("No enabled models found")
			http.Error(w, "No enabled models available", http.StatusInternalServerError)
			return
		}

		// Select model based on capabilities
		routingResult, routeErr := s.llmClient.DecideModelBasedOnCapabilities(userMessage, enabledModels)
		modelName = routingResult.ModelName
		if routeErr != nil {
			log.Printf("Error deciding model: %v, falling back to default model", routeErr)
			modelName = s.getFallbackModelName(enabledModels)
			if modelName == "" {
				http.Error(w, "No enabled models available", http.StatusInternalServerError)
				return
			}
		}

		// Get model info for the selected model
		s.modelMutex.RLock()
		modelInfo, exists = s.modelCollection.GetModel(modelName)
		s.modelMutex.RUnlock()

		if !exists {
			log.Printf("Model %s not found, using default response", modelName)
			http.Error(w, "Model not found", http.StatusBadRequest)
			return
		}

		// Log routing decision after we get a response (deferred below)
		uid := getUserID(r)
		defer func() {
			s.logRoutingDecision(routingResult, userMessage, modelInfo.Source, openAIResponseLatencyMs, openAIResponseStatus, uid)
		}()
	}

	// Call the selected model with the full message history
	log.Printf("Routing request to model: %s (history length: %d)", modelName, len(openAIReq.Messages))
	responseStart := time.Now()
	response, err := s.llmClient.ChatWithModel(modelInfo, modelName, openAIReq.Messages)
	openAIResponseLatencyMs = time.Since(responseStart).Milliseconds()
	if err != nil {
		log.Printf("Error calling model: %v", err)
		openAIResponseStatus = "error"
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	completionID := "chatcmpl-" + generateRandomID()
	created := getCurrentUnixTimestamp()

	// Streaming response: send the full content as a single SSE chunk, then [DONE].
	// This is "fake" streaming (one chunk) but is fully spec-compliant and works with
	// all OpenAI-compatible clients including Open WebUI.
	if openAIReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// First chunk: role
		roleChunk := map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelName,
			"choices": []map[string]interface{}{
				{"index": 0, "delta": map[string]string{"role": "assistant", "content": ""}, "finish_reason": nil},
			},
		}
		if b, err := json.Marshal(roleChunk); err == nil {
			writeSSEChunk(w, string(b))
		}

		// Content chunk
		contentChunk := map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelName,
			"choices": []map[string]interface{}{
				{"index": 0, "delta": map[string]string{"content": response}, "finish_reason": nil},
			},
		}
		if b, err := json.Marshal(contentChunk); err == nil {
			writeSSEChunk(w, string(b))
		}

		// Final chunk: finish_reason
		doneChunk := map[string]interface{}{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelName,
			"choices": []map[string]interface{}{
				{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"},
			},
		}
		if b, err := json.Marshal(doneChunk); err == nil {
			writeSSEChunk(w, string(b))
		}

		writeSSEChunk(w, "[DONE]")
		return
	}

	// Non-streaming response
	openAIResponse := map[string]interface{}{
		"id":      completionID,
		"object":  "chat.completion",
		"created": created,
		"model":   modelName,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": response,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     len(userMessage) / 4,
			"completion_tokens": len(response) / 4,
			"total_tokens":      (len(userMessage) + len(response)) / 4,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(openAIResponse)
}

// handleOpenAIModels handles the OpenAI models API endpoint
// @Summary OpenAI-compatible models listing
// @Description OpenAI-compatible endpoint for listing available models
// @Tags openai,models
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /v1/models [get]
func (s *Server) handleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to OpenAI-compatible /v1/models endpoint")

	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()

	// Convert to OpenAI compatible format
	var modelsList []map[string]interface{}
	for name, info := range s.modelCollection.Models {
		if info.Enabled {
			modelsList = append(modelsList, map[string]interface{}{
				"id":       name,
				"object":   "model",
				"created":  getCurrentUnixTimestamp() - 86400, // 1 day ago as a placeholder
				"owned_by": info.Source,
			})
		}
	}

	// If no models are loaded, expose a configured default model.
	if len(modelsList) == 0 {
		defaultModel := strings.TrimSpace(s.config.DefaultChatModel)
		if defaultModel == "" {
			defaultModel = DefaultModel
		}
		modelsList = append(modelsList, map[string]interface{}{
			"id":       defaultModel,
			"object":   "model",
			"created":  getCurrentUnixTimestamp() - 86400,
			"owned_by": "openai",
		})
	}

	response := map[string]interface{}{
		"object": "list",
		"data":   modelsList,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleGetReasoningModels returns all models that support reasoning
func (s *Server) handleGetReasoningModels(w http.ResponseWriter, r *http.Request) {
	// Get all models with reasoning capability
	s.modelMutex.RLock()
	reasoningModels := s.modelCollection.GetReasoningModels()
	s.modelMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reasoningModels); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// handleGetNonReasoningModels returns all models that don't support reasoning
func (s *Server) handleGetNonReasoningModels(w http.ResponseWriter, r *http.Request) {
	// Get all models without reasoning capability
	s.modelMutex.RLock()
	nonReasoningModels := s.modelCollection.GetNonReasoningModels()
	s.modelMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(nonReasoningModels); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// handleUpdateModelReasoning updates the reasoning capability of a model
// Auth: admin (enforced by middleware)
func (s *Server) handleUpdateModelReasoning(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Model        string `json:"model"`
		Source       string `json:"source"`
		HasReasoning bool   `json:"has_reasoning"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Model == "" || request.Source == "" {
		http.Error(w, "Model name and source are required", http.StatusBadRequest)
		return
	}

	// Update in database
	if err := s.db.UpdateModelReasoning(request.Model, request.Source, request.HasReasoning); err != nil {
		log.Printf("Error updating model reasoning in database: %v", err)
		http.Error(w, "Error updating model reasoning", http.StatusInternalServerError)
		return
	}

	// Update in memory collection
	s.modelMutex.Lock()
	if modelInfo, exists := s.modelCollection.GetModel(request.Model); exists {
		modelInfo.HasReasoning = request.HasReasoning
		s.modelCollection.UpdateModel(request.Model, modelInfo)
	}
	s.modelMutex.Unlock()

	log.Printf("Updated reasoning capability for model %s (source: %s) to %v",
		request.Model, request.Source, request.HasReasoning)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Model %s reasoning capability updated to %v", request.Model, request.HasReasoning),
	})
}

// handleRunBenchmarks triggers a benchmark run for all (or filtered) models.
// @Summary Run model benchmarks
// @Description Runs capability benchmarks against enabled models. Optional query params: model, category.
// @Tags models,admin
// @Param X-API-Key header string true "Admin API Key"
// @Param model query string false "Run only for this model name"
// @Param category query string false "Run only tasks in this category (coding|reasoning|instruction-following|creative-writing|summarization|factual-qa)"
// @Success 202 {object} map[string]string
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 503 {object} map[string]string "Benchmarks disabled"
// @Router /models/benchmark [post]
// Auth: admin (enforced by middleware)
func (s *Server) handleRunBenchmarks(w http.ResponseWriter, r *http.Request) {
	if !s.config.BenchmarkEnabled {
		http.Error(w, "Benchmarks are disabled (set BENCHMARK_ENABLED=true)", http.StatusServiceUnavailable)
		return
	}

	modelFilter := r.URL.Query().Get("model")
	categoryFilter := r.URL.Query().Get("category")

	// Optional body: {"models": [{"name":"...","source":"..."}], "category": "..."}
	// Body values override query params when provided.
	var body struct {
		Models   []ModelTarget `json:"models"`
		Category string        `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		if body.Category != "" {
			categoryFilter = body.Category
		}
		// When a model list is supplied run each by name filter individually.
		if len(body.Models) > 0 {
			targets := body.Models
			go func() {
				s.benchmarkOpMutex.Lock()
				s.benchmarkOperation = &operationState{
					Status:    "running",
					Total:     len(targets),
					Completed: 0,
					StartedAt: time.Now().UTC().Format(time.RFC3339),
				}
				s.benchmarkOpMutex.Unlock()

				for i, t := range targets {
					s.benchmarkOpMutex.Lock()
					s.benchmarkOperation.CurrentModel = t.Name
					s.benchmarkOperation.CompletedTasks = 0
					s.benchmarkOperation.TotalTasks = 0
					s.benchmarkOperation.CurrentTask = ""
					s.benchmarkOpMutex.Unlock()

					s.runBenchmarks(context.Background(), t.Name, categoryFilter)

					s.benchmarkOpMutex.Lock()
					s.benchmarkOperation.Completed = i + 1
					s.benchmarkOpMutex.Unlock()
				}

				s.benchmarkOpMutex.Lock()
				s.benchmarkOperation.Status = "completed"
				s.benchmarkOperation.CurrentModel = ""
				s.benchmarkOperation.CurrentTask = ""
				s.benchmarkOpMutex.Unlock()
			}()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "accepted",
				"message": fmt.Sprintf("Benchmark run started for %d model(s)", len(targets)),
			})
			return
		}
	}

	go s.runBenchmarks(context.Background(), modelFilter, categoryFilter)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "Benchmark run started in background",
	})
}

// handleGetModelBenchmarks returns the benchmark scores for a single model.
// @Summary Get model benchmark scores
// @Description Returns average benchmark scores per category for the specified model.
// @Tags models
// @Param name path string true "Model name"
// @Param source query string false "Provider source (required when multiple providers have the same model name)"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "Model not found"
// @Router /models/{name}/benchmarks [get]
func (s *Server) handleGetModelBenchmarks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelName, _ := url.PathUnescape(vars["name"])

	// Resolve source: prefer query param, fall back to model collection.
	source := r.URL.Query().Get("source")
	if source == "" {
		s.modelMutex.RLock()
		if info, ok := s.modelCollection.GetModel(modelName); ok {
			source = info.Source
		}
		s.modelMutex.RUnlock()
	}
	if source == "" {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	scores, err := s.db.GetBenchmarkScores(modelName, source)
	if err != nil {
		log.Printf("Error fetching benchmark scores for %s: %v", modelName, err)
		http.Error(w, "error fetching benchmark scores", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model":  modelName,
		"source": source,
		"scores": scores,
	})
}

// handleGetModelBenchmarkResults returns per-task benchmark results for a single model.
// @Summary Get model benchmark results
// @Description Returns individual task-level benchmark results for the specified model.
// @Tags models
// @Param name path string true "Model name"
// @Param source query string false "Provider source"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "Model not found"
// @Router /models/{name}/benchmark-results [get]
func (s *Server) handleGetModelBenchmarkResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelName, _ := url.PathUnescape(vars["name"])

	source := r.URL.Query().Get("source")
	if source == "" {
		s.modelMutex.RLock()
		if info, ok := s.modelCollection.GetModel(modelName); ok {
			source = info.Source
		}
		s.modelMutex.RUnlock()
	}
	if source == "" {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	results, err := s.db.GetBenchmarkResults(modelName, source)
	if err != nil {
		log.Printf("Error fetching benchmark results for %s: %v", modelName, err)
		http.Error(w, "error fetching benchmark results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model":          modelName,
		"source":         source,
		"results":        results,
		"benchmarkJudge": s.resolveBenchmarkJudgeName(),
	})
}

// leaderboardEntry holds one model's aggregated benchmark data for the leaderboard response.
type leaderboardEntry struct {
	Model  string             `json:"model"`
	Source string             `json:"source"`
	Scores map[string]float64 `json:"scores"`
	Avg    float64            `json:"avg_score"`
}

// handleBenchmarkLeaderboard returns all models ranked by benchmark performance.
// @Summary Benchmark leaderboard
// @Description Returns all models with benchmark scores, sorted by average score descending. Optional ?category filter.
// @Tags models
// @Param category query string false "Filter by category"
// @Success 200 {object} map[string]interface{}
// @Router /benchmarks/leaderboard [get]
func (s *Server) handleBenchmarkLeaderboard(w http.ResponseWriter, r *http.Request) {
	categoryFilter := r.URL.Query().Get("category")

	allScores, err := s.db.GetAllBenchmarkScores()
	if err != nil {
		log.Printf("Error fetching leaderboard scores: %v", err)
		http.Error(w, "error fetching benchmark scores", http.StatusInternalServerError)
		return
	}

	entries := make([]leaderboardEntry, 0, len(allScores))
	for key, scores := range allScores {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		modelName, source := parts[0], parts[1]

		filtered := scores
		if categoryFilter != "" {
			if v, ok := scores[categoryFilter]; ok {
				filtered = map[string]float64{categoryFilter: v}
			} else {
				continue // model has no score for this category
			}
		}

		avg := 0.0
		for _, v := range filtered {
			avg += v
		}
		if len(filtered) > 0 {
			avg /= float64(len(filtered))
		}

		entries = append(entries, leaderboardEntry{
			Model:  modelName,
			Source: source,
			Scores: filtered,
			Avg:    avg,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Avg > entries[j].Avg
	})

	// Attach rank.
	type rankedEntry struct {
		leaderboardEntry
		Rank int `json:"rank"`
	}
	ranked := make([]rankedEntry, len(entries))
	for i, e := range entries {
		ranked[i] = rankedEntry{e, i + 1}
	}

	categories := make([]string, 0, len(benchmark.AllCategories()))
	for _, c := range benchmark.AllCategories() {
		categories = append(categories, string(c))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"leaderboard":    ranked,
		"categories":     categories,
		"count":          len(ranked),
		"benchmarkJudge": s.resolveBenchmarkJudgeName(),
	})
}

// handleSetModelEnabled enables or disables a model by name.
// @Summary Enable or disable a model
// @Description Sets the enabled flag for a model. Requires admin API key.
// @Tags models,admin
// @Accept json
// @Produce json
// @Param name path string true "Model name"
// @Param X-API-Key header string true "Admin API Key"
// @Param request body object true "{\"source\":\"openai\",\"enabled\":true}"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/{name}/enable [patch]
// Auth: admin (enforced by middleware)
func (s *Server) handleSetModelEnabled(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(mux.Vars(r)["name"])
	var body struct {
		Source  string `json:"source"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Source == "" {
		http.Error(w, "request body must be {\"source\":\"...\",\"enabled\":true|false}", http.StatusBadRequest)
		return
	}

	if err := s.db.EnableModel(name, body.Source, body.Enabled); err != nil {
		log.Printf("Error setting enabled=%v for model %s: %v", body.Enabled, name, err)
		http.Error(w, "Failed to update model", http.StatusInternalServerError)
		return
	}

	// Reflect change in the in-memory collection immediately.
	s.modelMutex.Lock()
	if info, ok := s.modelCollection.GetModel(name); ok {
		info.Enabled = body.Enabled
		s.modelCollection.UpdateModel(name, info)
	}
	s.modelMutex.Unlock()

	log.Printf("Model %s (%s) enabled=%v", name, body.Source, body.Enabled)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    name,
		"source":  body.Source,
		"enabled": body.Enabled,
	})
}

// handleResetModel clears a model's tags, benchmark results, and routing analytics.
// @Summary Reset a model
// @Description Clears tags, benchmarks, and analytics for a model but keeps the model entry.
// @Tags models,admin
// @Param name path string true "Model name (URL-encoded)"
// @Param X-API-Key header string true "Admin API Key"
// @Accept json
// @Produce json
// @Param body body object true "source field required" example({"source":"openai"})
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/{name}/reset [post]
// Auth: admin (enforced by middleware)
func (s *Server) handleResetModel(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(mux.Vars(r)["name"])
	var body struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Source == "" {
		http.Error(w, `request body must be {"source":"..."}`, http.StatusBadRequest)
		return
	}

	if err := s.db.ResetModel(name, body.Source); err != nil {
		log.Printf("Error resetting model %s (%s): %v", name, body.Source, err)
		http.Error(w, "Failed to reset model", http.StatusInternalServerError)
		return
	}

	// Update in-memory collection: disable and clear tags.
	s.modelMutex.Lock()
	if info, ok := s.modelCollection.GetModel(name); ok {
		info.Enabled = false
		info.Tags = `{"strengths":[],"weaknesses":[]}`
		s.modelCollection.UpdateModel(name, info)
	}
	s.modelMutex.Unlock()

	log.Printf("Model %s (%s) has been reset", name, body.Source)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "reset",
		"name":   name,
		"source": body.Source,
	})
}

// handleDeleteModel permanently deletes a model and all associated data.
// @Summary Delete a model
// @Description Permanently deletes a model and all its benchmarks, tags, and analytics.
// @Tags models,admin
// @Param name path string true "Model name (URL-encoded)"
// @Param X-API-Key header string true "Admin API Key"
// @Accept json
// @Produce json
// @Param body body object true "source field required" example({"source":"openai"})
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/{name} [delete]
// Auth: admin (enforced by middleware)
func (s *Server) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(mux.Vars(r)["name"])
	var body struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Source == "" {
		http.Error(w, `request body must be {"source":"..."}`, http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteModelFull(name, body.Source); err != nil {
		log.Printf("Error deleting model %s (%s): %v", name, body.Source, err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	// Remove from in-memory collection.
	s.modelMutex.Lock()
	s.modelCollection.RemoveModel(name)
	s.modelMutex.Unlock()

	log.Printf("Model %s (%s) has been permanently deleted", name, body.Source)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"name":   name,
		"source": body.Source,
	})
}

// handleDiscoverModels queries every provider for its model list and registers new models.
// @Summary Discover models from all providers
// @Description Queries each configured provider for its model list and registers any new models (no tagging).
// @Tags models,admin
// @Param X-API-Key header string true "Admin API Key"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/discover [post]
// Auth: admin (enforced by middleware)
func (s *Server) handleDiscoverModels(w http.ResponseWriter, r *http.Request) {
	providers, discovered, err := s.discoverModels()
	if err != nil {
		http.Error(w, "Discovery failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"providers":  providers,
		"discovered": discovered,
	})
}

// handleTagModels runs capability tagging for a user-supplied list of models.
// @Summary Tag selected models
// @Description Runs the capability self-assessment (interview) for each supplied model and persists the tags.
// @Tags models,admin
// @Accept json
// @Produce json
// @Param X-API-Key header string true "Admin API Key"
// @Param request body object true "List of models to tag: {\"models\":[{\"name\":\"...\",\"source\":\"...\"}]}"
// @Success 202 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/tag [post]
// Auth: admin (enforced by middleware)
func (s *Server) handleTagModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Models []ModelTarget `json:"models"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Models) == 0 {
		http.Error(w, "request body must be {\"models\":[{\"name\":\"...\",\"source\":\"...\"}]}", http.StatusBadRequest)
		return
	}

	targets := body.Models
	go s.tagModels(targets)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": fmt.Sprintf("Tagging started for %d model(s)", len(targets)),
	})
}

// handleTagStatus returns the current tagging operation progress.
// @Summary Get tagging operation status
// @Description Returns the progress of the current or last tagging operation.
// @Tags models,admin
// @Produce json
// @Param X-API-Key header string true "Admin API Key"
// @Success 200 {object} operationState
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/tag/status [get]
// Auth: admin (enforced by middleware)
func (s *Server) handleTagStatus(w http.ResponseWriter, r *http.Request) {
	s.tagOpMutex.Lock()
	op := s.tagOperation
	s.tagOpMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if op == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "idle"})
		return
	}
	_ = json.NewEncoder(w).Encode(op)
}

// handleBenchmarkStatus returns the current benchmark operation progress.
// @Summary Get benchmark operation status
// @Description Returns the progress of the current or last benchmark operation.
// @Tags models,admin
// @Produce json
// @Param X-API-Key header string true "Admin API Key"
// @Success 200 {object} operationState
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /models/benchmark/status [get]
// Auth: admin (enforced by middleware)
func (s *Server) handleBenchmarkStatus(w http.ResponseWriter, r *http.Request) {
	s.benchmarkOpMutex.Lock()
	op := s.benchmarkOperation
	s.benchmarkOpMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if op == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "idle"})
		return
	}
	_ = json.NewEncoder(w).Encode(op)
}

