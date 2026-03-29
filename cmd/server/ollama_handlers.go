// Package server provides Ollama-compatible handlers
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"PiPiMink/internal/api"
)

// Ollama API response types
type OllamaModelResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaModel struct {
	Name        string   `json:"name"`
	Modified    int64    `json:"modified_at"`
	Size        int64    `json:"size"`
	Digest      string   `json:"digest,omitempty"`
	ModelFormat string   `json:"model_format,omitempty"`
	Parameters  string   `json:"parameters,omitempty"`
	Template    string   `json:"template,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type OllamaGenerateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	System    string `json:"system,omitempty"`
	Template  string `json:"template,omitempty"`
	Context   []int  `json:"context,omitempty"`
	Stream    bool   `json:"stream"`
	Raw       bool   `json:"raw,omitempty"`
	Format    string `json:"format,omitempty"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type OllamaGenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

type OllamaChatRequest struct {
	Model     string                 `json:"model"`
	Messages  []OllamaChatMessage    `json:"messages"`
	Stream    bool                   `json:"stream"`
	Format    string                 `json:"format,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type OllamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatResponse struct {
	Model     string            `json:"model"`
	CreatedAt string            `json:"created_at"`
	Message   OllamaChatMessage `json:"message"`
	Done      bool              `json:"done"`
}

type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// handleOllamaModels handles the GET /api/tags endpoint
func (s *Server) handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/tags endpoint")

	// Return a single router model for Ollama-compatible clients.
	// This keeps client selection simple while routing internally across all enabled models.
	var modelsList []OllamaModel
	now := getCurrentUnixTimestamp()

	// Always return just the single "PiPiMink v1" model regardless of what's actually configured
	modelsList = append(modelsList, OllamaModel{
		Name:        "PiPiMink v1",
		Modified:    now,
		Size:        1000000000, // 1GB placeholder
		Digest:      "sha256:" + generateRandomID(),
		ModelFormat: "gguf",
		Tags:        []string{"router"},
	})

	response := OllamaModelResponse{
		Models: modelsList,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleOllamaGenerate handles the POST /api/generate endpoint
func (s *Server) handleOllamaGenerate(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/generate endpoint")

	// Validate request body
	bodyBytes, validationResult := api.ValidateRequestBody(r, 1024*1024) // 1MB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var generateReq OllamaGenerateRequest
	if err := json.Unmarshal(bodyBytes, &generateReq); err != nil {
		log.Printf("Error parsing Ollama generate request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := api.NewValidationResult()
	if generateReq.Prompt == "" {
		result.AddError("prompt", "Prompt is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Get model info
	s.modelMutex.RLock()
	modelInfo, exists := s.modelCollection.GetModel(generateReq.Model)
	s.modelMutex.RUnlock()

	// If model doesn't exist, try to find a suitable one
	if !exists {
		s.modelMutex.RLock()
		enabledModels := s.modelCollection.GetEnabledModels()
		s.modelMutex.RUnlock()

		if len(enabledModels) == 0 {
			log.Printf("No enabled models found")
			http.Error(w, "No enabled models available", http.StatusInternalServerError)
			return
		}

		// Select model based on capabilities
		var err error
		selectedModel, err := s.llmClient.DecideModelBasedOnCapabilities(generateReq.Prompt, enabledModels)
		if err != nil {
			log.Printf("Error deciding model: %v, falling back to default model", err)
			selectedModel = s.getFallbackModelName(enabledModels)
			if selectedModel == "" {
				http.Error(w, "No enabled models available", http.StatusInternalServerError)
				return
			}
		}

		generateReq.Model = selectedModel
		s.modelMutex.RLock()
		modelInfo, _ = s.modelCollection.GetModel(selectedModel)
		s.modelMutex.RUnlock()
	}

	// Wrap the single prompt in a messages slice for ChatWithModel
	generateMessages := []map[string]interface{}{
		{"role": "user", "content": generateReq.Prompt},
	}

	// Handle streaming if requested
	if generateReq.Stream {
		// Set appropriate headers for streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// In a real implementation, you would stream chunks back to the client
		// For now, we'll return the entire response at once in a format similar to streaming

		response, err := s.llmClient.ChatWithModel(modelInfo, generateReq.Model, generateMessages)
		if err != nil {
			log.Printf("Error calling model: %v", err)
			http.Error(w, "Error processing request", http.StatusInternalServerError)
			return
		}

		streamResponse := OllamaGenerateResponse{
			Model:         generateReq.Model,
			CreatedAt:     time.Now().Format(time.RFC3339),
			Response:      response,
			Done:          true,
			TotalDuration: 1000, // Placeholder durations
			EvalCount:     100,
		}

		// Encode and send the response
		_ = json.NewEncoder(w).Encode(streamResponse)
		return
	}

	// Non-streaming response
	response, err := s.llmClient.ChatWithModel(modelInfo, generateReq.Model, generateMessages)
	if err != nil {
		log.Printf("Error calling model: %v", err)
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	generateResponse := OllamaGenerateResponse{
		Model:         generateReq.Model,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Response:      response,
		Done:          true,
		TotalDuration: 1000, // Placeholder durations
		EvalCount:     100,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(generateResponse)
}

// handleOllamaChat handles the POST /api/chat endpoint
func (s *Server) handleOllamaChat(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/chat endpoint")

	// Validate request body
	bodyBytes, validationResult := api.ValidateRequestBody(r, 1024*1024) // 1MB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var chatReq OllamaChatRequest
	if err := json.Unmarshal(bodyBytes, &chatReq); err != nil {
		log.Printf("Error parsing Ollama chat request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := api.NewValidationResult()
	if len(chatReq.Messages) == 0 {
		result.AddError("messages", "At least one message is required")
	}

	// Validate message format
	var lastUserMessage string
	var foundUserMessage bool

	for i, msg := range chatReq.Messages {
		if msg.Role == "" {
			result.AddError("messages", "Message at index "+strconv.Itoa(i)+" has empty role")
		} else if msg.Role != "user" && msg.Role != "system" && msg.Role != "assistant" {
			result.AddError("messages", "Message role must be 'user', 'system', or 'assistant'")
		}

		if msg.Content == "" {
			result.AddError("messages", "Message at index "+strconv.Itoa(i)+" has empty content")
		}

		if msg.Role == "user" {
			lastUserMessage = msg.Content
			foundUserMessage = true
		}
	}

	if !foundUserMessage {
		result.AddError("messages", "At least one user message is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Get model info
	s.modelMutex.RLock()
	modelInfo, exists := s.modelCollection.GetModel(chatReq.Model)
	s.modelMutex.RUnlock()

	// If model doesn't exist, try to find a suitable one
	if !exists {
		s.modelMutex.RLock()
		enabledModels := s.modelCollection.GetEnabledModels()
		s.modelMutex.RUnlock()

		if len(enabledModels) == 0 {
			log.Printf("No enabled models found")
			http.Error(w, "No enabled models available", http.StatusInternalServerError)
			return
		}

		// Select model based on capabilities
		var err error
		selectedModel, err := s.llmClient.DecideModelBasedOnCapabilities(lastUserMessage, enabledModels)
		if err != nil {
			log.Printf("Error deciding model: %v, falling back to default model", err)
			selectedModel = s.getFallbackModelName(enabledModels)
			if selectedModel == "" {
				http.Error(w, "No enabled models available", http.StatusInternalServerError)
				return
			}
		}

		chatReq.Model = selectedModel
		s.modelMutex.RLock()
		modelInfo, _ = s.modelCollection.GetModel(selectedModel)
		s.modelMutex.RUnlock()
	}

	// Convert Ollama message structs to the generic messages format used by ChatWithModel
	chatMessages := make([]map[string]interface{}, len(chatReq.Messages))
	for i, msg := range chatReq.Messages {
		chatMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	// Handle streaming if requested
	if chatReq.Stream {
		// Set appropriate headers for streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// In a real implementation, you would stream chunks back to the client
		// For now, we'll return the entire response at once in a format similar to streaming

		response, err := s.llmClient.ChatWithModel(modelInfo, chatReq.Model, chatMessages)
		if err != nil {
			log.Printf("Error calling model: %v", err)
			http.Error(w, "Error processing request", http.StatusInternalServerError)
			return
		}

		streamResponse := OllamaChatResponse{
			Model:     chatReq.Model,
			CreatedAt: time.Now().Format(time.RFC3339),
			Message: OllamaChatMessage{
				Role:    "assistant",
				Content: response,
			},
			Done: true,
		}

		// Encode and send the response
		_ = json.NewEncoder(w).Encode(streamResponse)
		return
	}

	// Non-streaming response
	response, err := s.llmClient.ChatWithModel(modelInfo, chatReq.Model, chatMessages)
	if err != nil {
		log.Printf("Error calling model: %v", err)
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	chatResponse := OllamaChatResponse{
		Model:     chatReq.Model,
		CreatedAt: time.Now().Format(time.RFC3339),
		Message: OllamaChatMessage{
			Role:    "assistant",
			Content: response,
		},
		Done: true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chatResponse)
}

// handleOllamaEmbeddings handles the POST /api/embeddings endpoint
func (s *Server) handleOllamaEmbeddings(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/embeddings endpoint")

	// Validate request body
	bodyBytes, validationResult := api.ValidateRequestBody(r, 512*1024) // 512KB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var embeddingReq OllamaEmbeddingRequest
	if err := json.Unmarshal(bodyBytes, &embeddingReq); err != nil {
		log.Printf("Error parsing Ollama embedding request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := api.NewValidationResult()
	if embeddingReq.Prompt == "" {
		result.AddError("prompt", "Prompt is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Since we don't actually compute embeddings, we'll return a placeholder embedding
	// A real implementation would use the model to generate actual embeddings
	embeddingDim := 1536 // Standard OpenAI embedding dimension
	embedding := make([]float32, embeddingDim)

	// Fill with some placeholder values
	for i := 0; i < embeddingDim; i++ {
		embedding[i] = float32(i) / float32(embeddingDim)
	}

	embeddingResponse := OllamaEmbeddingResponse{
		Embedding: embedding,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(embeddingResponse)
}

// handleOllamaShow handles the POST /api/show endpoint
func (s *Server) handleOllamaShow(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/show endpoint")

	// Validate request body
	bodyBytes, validationResult := api.ValidateRequestBody(r, 64*1024) // 64KB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var showReq struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bodyBytes, &showReq); err != nil {
		log.Printf("Error parsing Ollama show request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := api.NewValidationResult()
	if showReq.Name == "" {
		result.AddError("name", "Model name is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Always respond with the router facade model regardless of which model was requested.
	modelDetails := OllamaModel{
		Name:        "PiPiMink v1",
		Modified:    getCurrentUnixTimestamp(),
		Size:        1000000000, // 1GB placeholder
		Digest:      "sha256:" + generateRandomID(),
		ModelFormat: "gguf",
		Tags:        []string{"router"},
		Parameters:  "{}",
		Template:    "{{ .Prompt }}",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(modelDetails)
}

// handleOllamaPull handles the POST /api/pull endpoint
func (s *Server) handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to Ollama-compatible /api/pull endpoint")

	// Validate request body
	bodyBytes, validationResult := api.ValidateRequestBody(r, 64*1024) // 64KB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var pullReq struct {
		Name     string `json:"name"`
		Insecure bool   `json:"insecure,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &pullReq); err != nil {
		log.Printf("Error parsing Ollama pull request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := api.NewValidationResult()
	if pullReq.Name == "" {
		result.AddError("name", "Model name is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Always respond as if the router facade model is already available.
	response := struct {
		Status string `json:"status"`
	}{
		Status: "Model PiPiMink v1 is already available",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
