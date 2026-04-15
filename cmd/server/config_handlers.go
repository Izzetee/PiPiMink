package server

import (
	"encoding/json"
	"net/http"

	"PiPiMink/internal/benchmark"

	"github.com/gorilla/mux"
)

// handleGetBenchmarkTasks returns all benchmark task configs from the DB.
func (s *Server) handleGetBenchmarkTasks(w http.ResponseWriter, r *http.Request) {
	cfgs, err := s.db.GetBenchmarkTaskConfigs()
	if err != nil {
		http.Error(w, `{"error":"failed to load benchmark tasks"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfgs)
}

// handleUpsertBenchmarkTask creates or updates a benchmark task config.
// Auth: admin (enforced by middleware)
func (s *Server) handleUpsertBenchmarkTask(w http.ResponseWriter, r *http.Request) {
	var cfg benchmark.BenchmarkTaskConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if cfg.TaskID == "" {
		http.Error(w, `{"error":"task_id is required"}`, http.StatusBadRequest)
		return
	}

	if err := s.db.UpsertBenchmarkTaskConfig(cfg); err != nil {
		http.Error(w, `{"error":"failed to save task"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteBenchmarkTask deletes a task config (builtin tasks are reset to defaults).
// Auth: admin (enforced by middleware)
func (s *Server) handleDeleteBenchmarkTask(w http.ResponseWriter, r *http.Request) {
	taskID := mux.Vars(r)["id"]
	if taskID == "" {
		http.Error(w, `{"error":"task id is required"}`, http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteBenchmarkTaskConfig(taskID, benchmark.DefaultTaskConfigs()); err != nil {
		http.Error(w, `{"error":"failed to delete task"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetSystemPrompts returns all system prompts (tagging prompts) from the DB.
func (s *Server) handleGetSystemPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := s.db.GetAllSystemPrompts()
	if err != nil {
		http.Error(w, `{"error":"failed to load system prompts"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(prompts)
}

// handleUpdateSystemPrompt updates the value of a single system prompt by key.
// Auth: admin (enforced by middleware)
func (s *Server) handleUpdateSystemPrompt(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if key == "" {
		http.Error(w, `{"error":"key is required"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := s.db.SetSystemPrompt(key, body.Value, body.Description); err != nil {
		http.Error(w, `{"error":"failed to save prompt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
