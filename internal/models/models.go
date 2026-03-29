package models

import (
	"encoding/json"
	"strings"
)

// ChatRequest represents a chat request from a client.
// For single-turn use, set Message. For multi-turn conversations, set Messages
// (OpenAI messages format). If both are provided, Messages takes precedence.
type ChatRequest struct {
	Message  string                   `json:"message"`
	Messages []map[string]interface{} `json:"messages,omitempty"`
}

// ChatResponse represents a chat response to the client
type ChatResponse struct {
	Response string `json:"response"`
	Model    string `json:"model"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	Source          string             `json:"source"`                     // Source of the model (e.g., "openai", "local")
	Tags            string             `json:"tags"`                       // JSON string containing model capabilities/tags
	Response        string             `json:"response"`                   // Raw model response for capabilities
	Enabled         bool               `json:"enabled"`                    // Whether the model is available for use
	HasReasoning    bool               `json:"has_reasoning"`              // Whether the model supports reasoning capabilities
	UpdatedAt       string             `json:"updated_at"`                 // Timestamp of last update
	BenchmarkScores map[string]float64 `json:"benchmark_scores,omitempty"` // Average score per benchmark category
	AvgLatencyMs    *int64             `json:"avg_latency_ms,omitempty"`   // Average response latency across all benchmark tasks (ms)
}

// ValidateJSONTags checks if tags is valid JSON and returns a default if not
func ValidateJSONTags(tags string) string {
	if tags == "" {
		return "{}"
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(tags), &js); err != nil {
		return "{}"
	}
	return tags
}

// ModelCollection represents a collection of models
type ModelCollection struct {
	Models map[string]ModelInfo
}

// NewModelCollection creates a new model collection
func NewModelCollection() *ModelCollection {
	return &ModelCollection{
		Models: make(map[string]ModelInfo),
	}
}

// AddModel adds a model to the collection
func (c *ModelCollection) AddModel(name string, info ModelInfo) {
	c.Models[name] = info
}

// GetModel gets a model from the collection
func (c *ModelCollection) GetModel(name string) (ModelInfo, bool) {
	model, ok := c.Models[name]
	return model, ok
}

// UpdateModel updates a model in the collection
func (c *ModelCollection) UpdateModel(name string, info ModelInfo) {
	c.Models[name] = info
}

// RemoveModel removes a model from the collection
func (c *ModelCollection) RemoveModel(name string) {
	delete(c.Models, name)
}

// FromDatabaseMap converts a database map to a model collection
func (c *ModelCollection) FromDatabaseMap(dbModels map[string]map[string]interface{}) {
	for name, model := range dbModels {
		// Initialize default values
		source := ""
		tags := "{}"
		response := ""
		enabled := false
		hasReasoning := false
		updatedAt := ""

		// Safely extract values with nil checks
		if src, ok := model["source"]; ok && src != nil {
			source = src.(string)
		}

		if t, ok := model["tags"]; ok && t != nil {
			tags = t.(string)
		}

		if resp, ok := model["response"]; ok && resp != nil {
			response = resp.(string)
		}

		if en, ok := model["enabled"]; ok && en != nil {
			enabled = en.(bool)
		}

		if reasoning, ok := model["has_reasoning"]; ok && reasoning != nil {
			hasReasoning = reasoning.(bool)
		}

		if updated, ok := model["updated_at"]; ok && updated != nil {
			updatedAt = updated.(string)
		}

		c.Models[name] = ModelInfo{
			Source:       source,
			Tags:         tags,
			Response:     response,
			Enabled:      enabled,
			HasReasoning: hasReasoning,
			UpdatedAt:    updatedAt,
		}
	}
}

// GetEnabledModels returns only the enabled models from the collection
func (c *ModelCollection) GetEnabledModels() map[string]ModelInfo {
	enabled := make(map[string]ModelInfo)
	for name, info := range c.Models {
		if info.Enabled {
			enabled[name] = info
		}
	}
	return enabled
}

// GetReasoningModels returns only the models that support reasoning
func (c *ModelCollection) GetReasoningModels() map[string]ModelInfo {
	reasoning := make(map[string]ModelInfo)
	for name, info := range c.Models {
		if info.HasReasoning && info.Enabled {
			reasoning[name] = info
		}
	}
	return reasoning
}

// GetNonReasoningModels returns only the models that don't support reasoning
func (c *ModelCollection) GetNonReasoningModels() map[string]ModelInfo {
	nonReasoning := make(map[string]ModelInfo)
	for name, info := range c.Models {
		if !info.HasReasoning && info.Enabled {
			nonReasoning[name] = info
		}
	}
	return nonReasoning
}

// ModelCount returns the number of models in the collection
func (c *ModelCollection) ModelCount() int {
	return len(c.Models)
}

// IsReasoningModel determines if a model has reasoning capabilities
// based on known model patterns and names
func IsReasoningModel(modelName string) bool {
	// Convert to lowercase for case-insensitive comparison
	model := strings.ToLower(modelName)

	// Known reasoning models from OpenAI
	if model == "o1-mini" || strings.HasPrefix(model, "o1-") ||
		strings.HasPrefix(model, "o1-preview") || model == "o1" ||
		model == "o3-mini-2025-01-31" || model == "o3-mini" ||
		model == "o1-2024-12-17" || model == "o4" ||
		model == "o4-mini" || model == "o4-mini-2025-04-16" {
		return true
	}

	// Check for other known reasoning model patterns
	// Add more patterns as new reasoning models are released
	if strings.Contains(model, "reasoning") ||
		strings.Contains(model, "think") ||
		strings.Contains(model, "cot") { // Chain of Thought
		return true
	}

	return false
}
