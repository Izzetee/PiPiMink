package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/database"
	"PiPiMink/internal/llm"
	"PiPiMink/internal/models"

	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	config          *config.Config
	db              *database.DB
	llmClient       *llm.Client
	router          *mux.Router
	modelCollection *models.ModelCollection
	modelMutex      sync.RWMutex
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *database.DB, llmClient *llm.Client) *Server {
	server := &Server{
		config:          cfg,
		db:              db,
		llmClient:       llmClient,
		router:          mux.NewRouter(),
		modelCollection: models.NewModelCollection(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up the API routes
func (s *Server) setupRoutes() {
	// Original PiPiMink routes
	s.router.HandleFunc("/chat", s.handleChat).Methods("POST")
	s.router.HandleFunc("/models/update", s.handleUpdateModels).Methods("POST")
	s.router.HandleFunc("/models", s.handleListModels).Methods("GET")

	// OpenAI-compatible routes
	s.router.HandleFunc("/v1/chat/completions", s.handleOpenAIChatCompletions).Methods("POST")
	s.router.HandleFunc("/v1/models", s.handleOpenAIModels).Methods("GET")
}

// OpenAI compatible handlers

// handleOpenAIChatCompletions handles the OpenAI chat completions API endpoint
func (s *Server) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to OpenAI-compatible /v1/chat/completions endpoint")

	// Validate request body
	bodyBytes, validationResult := ValidateRequestBody(r, 1024*1024) // 1MB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse OpenAI-style request
	var openAIReq struct {
		Model     string                   `json:"model"`
		Messages  []map[string]interface{} `json:"messages"`
		MaxTokens int                      `json:"max_tokens,omitempty"`
		Stream    bool                     `json:"stream,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &openAIReq); err != nil {
		log.Printf("Error parsing OpenAI-style request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := NewValidationResult()
	if len(openAIReq.Messages) == 0 {
		result.AddError("messages", "At least one message is required")
	}

	// Find user message
	var userMessage string
	var foundUserMessage bool

	// First check for specific message requirements
	for i := len(openAIReq.Messages) - 1; i >= 0; i-- {
		msg := openAIReq.Messages[i]
		role, hasRole := msg["role"].(string)
		content, hasContent := msg["content"].(string)

		if !hasRole {
			result.AddError("messages", "Every message must have a 'role' field")
		} else if role != "user" && role != "system" && role != "assistant" {
			result.AddError("messages", "Message role must be 'user', 'system', or 'assistant'")
		}

		if !hasContent {
			result.AddError("messages", "Every message must have a 'content' field")
		} else if content == "" {
			result.AddError("messages", "Message content cannot be empty")
		}

		if hasRole && hasContent && role == "user" && !foundUserMessage {
			userMessage = content
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

	log.Printf("Received chat completion request for model: %s", openAIReq.Model)

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

	// If model doesn't exist or isn't specified, route based on capabilities
	if !exists {
		// Get enabled models for selection
		s.modelMutex.RLock()
		enabledModels := make(map[string]models.ModelInfo)
		for name, info := range s.modelCollection.Models {
			if info.Enabled {
				enabledModels[name] = info
			}
		}
		s.modelMutex.RUnlock()

		// If no enabled models found, return error
		if len(enabledModels) == 0 {
			log.Printf("No enabled models found")
			http.Error(w, "No enabled models available", http.StatusInternalServerError)
			return
		}

		// Select model based on capabilities
		var err error
		modelName, err = s.llmClient.DecideModelBasedOnCapabilities(userMessage, enabledModels)
		if err != nil {
			log.Printf("Error deciding model: %v, falling back to default model", err)
			modelName = "gpt-4-turbo" // Fallback to default model
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
	}

	// Call the selected model with the full message history
	log.Printf("Routing request to model: %s", modelName)
	response, err := s.llmClient.ChatWithModel(modelInfo, modelName, openAIReq.Messages)
	if err != nil {
		log.Printf("Error calling model: %v", err)
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	// Format response in OpenAI compatible format
	openAIResponse := map[string]interface{}{
		"id":      "chatcmpl-" + generateRandomID(),
		"object":  "chat.completion",
		"created": getCurrentUnixTimestamp(),
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
			"prompt_tokens":     len(userMessage) / 4, // Rough approximation
			"completion_tokens": len(response) / 4,    // Rough approximation
			"total_tokens":      (len(userMessage) + len(response)) / 4,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(openAIResponse)
}

// handleOpenAIModels handles the OpenAI models API endpoint
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

	// If no models, add a default gpt-4-turbo
	if len(modelsList) == 0 {
		modelsList = append(modelsList, map[string]interface{}{
			"id":       "gpt-4-turbo",
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

// Helper functions for OpenAI compatibility
func generateRandomID() string {
	// Use crypto/rand for cryptographically secure random numbers
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fall back to a less secure method in case of error
		log.Printf("Error generating secure random ID: %v, using fallback method", err)
		return generateFallbackRandomID()
	}

	// Convert to a URL-safe base64 string
	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)
	// Trim to desired length (10 characters as in original function)
	if len(encoded) > 10 {
		encoded = encoded[:10]
	}
	return encoded
}

// Fallback method in case crypto/rand fails
func generateFallbackRandomID() string {
	// Simple random ID generator with timestamp for uniqueness
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 10)
	timestamp := time.Now().UnixNano()
	for i := range result {
		result[i] = charset[timestamp%int64(len(charset))]
		timestamp /= int64(len(charset))
	}
	return string(result)
}

func getCurrentUnixTimestamp() int64 {
	return time.Now().Unix()
}

// Helper functions for model processing

// Tag refresh interval in hours - Don't process models more than once per 24 hours
const tagRefreshInterval = 24 * time.Hour

// shouldProcessModel checks if a model should be processed based on its last update time
func shouldProcessModel(updatedAt string, refreshInterval time.Duration) bool {
	if updatedAt == "" {
		return true
	}

	lastUpdate, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		// If we can't parse the time, assume we should process
		return true
	}

	return time.Since(lastUpdate) > refreshInterval
}

// getCurrentTimeString returns the current time as RFC3339 string
func getCurrentTimeString() string {
	return time.Now().Format(time.RFC3339)
}

// Original PiPiMink handlers

// handleChat handles chat requests
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received chat request")

	// Validate request body
	bodyBytes, validationResult := ValidateRequestBody(r, 512*1024) // 512KB limit
	if validationResult.HasErrors() {
		validationResult.ErrorResponse(w)
		return
	}

	// Parse the request
	var chatReq models.ChatRequest
	if err := json.Unmarshal(bodyBytes, &chatReq); err != nil {
		log.Printf("Error parsing chat request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	result := NewValidationResult()
	if chatReq.Message == "" {
		result.AddError("message", "Message is required")
	}

	if result.HasErrors() {
		result.ErrorResponse(w)
		return
	}

	// Get enabled models for selection
	s.modelMutex.RLock()
	enabledModels := make(map[string]models.ModelInfo)
	for name, info := range s.modelCollection.Models {
		if info.Enabled {
			enabledModels[name] = info
		}
	}
	s.modelMutex.RUnlock()

	// If no enabled models found, return error
	if len(enabledModels) == 0 {
		log.Printf("No enabled models found")
		http.Error(w, "No enabled models available", http.StatusInternalServerError)
		return
	}

	// Use the configured OpenAI router model to decide which model to use based on the message and model capabilities
	modelName, err := s.llmClient.DecideModelBasedOnCapabilities(chatReq.Message, enabledModels)
	if err != nil {
		log.Printf("Error deciding model: %v, falling back to default model", err)
		modelName = "gpt-4-turbo" // Fallback to default model
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chatRes)
		return
	}

	// Call the selected model, wrapping the single message in a slice
	log.Printf("Routing request to model: %s", modelName)
	response, err := s.llmClient.ChatWithModel(modelInfo, modelName, []map[string]interface{}{
		{"role": "user", "content": chatReq.Message},
	})
	if err != nil {
		log.Printf("Error calling model: %v", err)
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	chatRes := models.ChatResponse{
		Response: response,
		Model:    modelName,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chatRes)
}

// handleUpdateModels handles requests to manually update the model database
func (s *Server) handleUpdateModels(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to update models")

	// Validate the authentication header
	validationResult := ValidateAuthKey(r, s.config.AdminAPIKey, "X-API-Key")
	if validationResult.HasErrors() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"errors": validationResult.Errors,
		})
		log.Printf("Unauthorized model update attempt")
		return
	}

	// Ensure proper request method
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Method not allowed, use POST",
		})
		return
	}

	// Start the update process
	go func() {
		log.Println("Starting model update process")
		if err := s.fetchAndTagModels(); err != nil {
			log.Printf("Error updating models: %v", err)
		} else {
			log.Println("Model update completed successfully")
		}
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
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()

	// Convert internal model structure to a response format
	modelsList := make([]map[string]interface{}, 0, len(s.modelCollection.Models))
	for name, info := range s.modelCollection.Models {
		modelData := map[string]interface{}{
			"name":      name,
			"source":    info.Source,
			"enabled":   info.Enabled,
			"updatedAt": info.UpdatedAt,
		}
		modelsList = append(modelsList, modelData)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"models": modelsList,
		"count":  len(modelsList),
	})
}

// Start starts the API server
func (s *Server) Start() error {
	// Load models from database
	log.Println("Loading models from database")
	if err := s.loadModelsFromDatabase(); err != nil {
		log.Printf("Error loading models from database: %v", err)
	}

	// Check if database has models
	hasModels, err := s.db.HasModels()
	if err != nil {
		log.Printf("Error checking if database has models: %v", err)
	}

	// If no models loaded, fetch from APIs
	s.modelMutex.RLock()
	modelsCount := len(s.modelCollection.Models)
	s.modelMutex.RUnlock()

	if modelsCount == 0 || !hasModels {
		log.Println("No models found in database, fetching from APIs")
		if err := s.fetchAndTagModels(); err != nil {
			log.Printf("Error fetching models: %v", err)
		}
	}

	// Log loaded models
	s.logModels()

	log.Printf("Starting server on port %s", s.config.Port)
	return http.ListenAndServe(":"+s.config.Port, s.router)
}

// loadModelsFromDatabase loads models from the database
func (s *Server) loadModelsFromDatabase() error {
	dbModels, err := s.db.GetAllModels()
	if err != nil {
		return err
	}

	s.modelMutex.Lock()
	defer s.modelMutex.Unlock()

	s.modelCollection.FromDatabaseMap(dbModels)
	return nil
}

// fetchAndTagModels fetches models from all configured providers and tags them.
func (s *Server) fetchAndTagModels() error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	type pendingModel struct {
		name string
		p    config.ProviderConfig
	}
	var pending []pendingModel

	for _, provider := range s.config.Providers {
		modelNames, err := s.llmClient.GetModelsByProvider(provider)
		if err != nil {
			log.Printf("Error fetching models from provider %s: %v", provider.Name, err)
			continue
		}
		for _, name := range modelNames {
			s.modelMutex.Lock()
			s.modelCollection.AddModel(name, models.ModelInfo{Source: provider.Name, Tags: "{}", Enabled: true})
			s.modelMutex.Unlock()
			mu.Lock()
			pending = append(pending, pendingModel{name: name, p: provider})
			mu.Unlock()
		}
	}

	// Tag all collected models concurrently
	for _, pm := range pending {
		wg.Add(1)
		go func(name string, p config.ProviderConfig) {
			defer wg.Done()

			// Re-read the cached info for this model
			s.modelMutex.RLock()
			info, _ := s.modelCollection.GetModel(name)
			s.modelMutex.RUnlock()

			if !shouldProcessModel(info.UpdatedAt, tagRefreshInterval) {
				return
			}

			tags, shouldDisable, shouldDelete, err := s.llmClient.GetModelTags(name, p)
			if err != nil {
				log.Printf("Error getting tags for model %s: %v", name, err)
				tags = "{}"
			}

			if shouldDelete {
				log.Printf("Model %s is not a chat model and will be deleted", name)
				s.modelMutex.Lock()
				s.modelCollection.RemoveModel(name)
				s.modelMutex.Unlock()
				if err := s.db.DeleteModel(name, p.Name); err != nil {
					log.Printf("Error deleting model %s: %v", name, err)
				}
				return
			}

			enabled := !shouldDisable && tags != ""
			hasReasoning := models.IsReasoningModel(name)
			s.modelMutex.Lock()
			s.modelCollection.UpdateModel(name, models.ModelInfo{
				Source:       p.Name,
				Tags:         tags,
				Enabled:      enabled,
				HasReasoning: hasReasoning,
				UpdatedAt:    getCurrentTimeString(),
			})
			s.modelMutex.Unlock()
			if err := s.db.SaveModel(name, p.Name, tags, enabled, hasReasoning); err != nil {
				log.Printf("Error saving model %s: %v", name, err)
			}
		}(pm.name, pm.p)
	}
	wg.Wait()
	return nil
}

// logModels logs the loaded models
func (s *Server) logModels() {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()

	log.Println("Loaded models:")
	for modelName, modelInfo := range s.modelCollection.Models {
		log.Printf("Model: %s, Source: %s, Enabled: %t", modelName, modelInfo.Source, modelInfo.Enabled)
	}
}
