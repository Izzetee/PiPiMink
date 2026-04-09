package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/database"

	"github.com/stretchr/testify/mock"
)

func (s *ServerTestSuite) TestHandleGetBenchmarkTasks() {
	cfgs := []benchmark.BenchmarkTaskConfig{
		{TaskID: "prime-check", Category: "coding", ScoringMethod: "deterministic"},
	}
	s.mockDB.On("GetBenchmarkTaskConfigs").Return(cfgs, nil)

	req, _ := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpsertBenchmarkTask() {
	s.mockDB.On("UpsertBenchmarkTaskConfig", mock.Anything).Return(nil)

	body, _ := json.Marshal(benchmark.BenchmarkTaskConfig{TaskID: "fizzbuzz", Category: "coding", ScoringMethod: "deterministic"})
	req, _ := http.NewRequest("POST", "/admin/benchmark-tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpsertBenchmarkTask_Unauthorized() {
	body, _ := json.Marshal(benchmark.BenchmarkTaskConfig{TaskID: "fizzbuzz"})
	req, _ := http.NewRequest("POST", "/admin/benchmark-tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpsertBenchmarkTask_MissingTaskID() {
	body, _ := json.Marshal(benchmark.BenchmarkTaskConfig{TaskID: ""})
	req, _ := http.NewRequest("POST", "/admin/benchmark-tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleDeleteBenchmarkTask() {
	s.mockDB.On("DeleteBenchmarkTaskConfig", "prime-check", mock.Anything).Return(nil)

	req, _ := http.NewRequest("DELETE", "/admin/benchmark-tasks/prime-check", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleGetSystemPrompts() {
	prompts := map[string]database.SystemPromptRow{
		"tagging_system": {Key: "tagging_system", Value: "Assess capabilities", Description: "System prompt"},
	}
	s.mockDB.On("GetAllSystemPrompts").Return(prompts, nil)

	req, _ := http.NewRequest("GET", "/admin/system-prompts", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpdateSystemPrompt() {
	s.mockDB.On("SetSystemPrompt", "tagging_system", "new prompt", "updated").Return(nil)

	body, _ := json.Marshal(map[string]string{"value": "new prompt", "description": "updated"})
	req, _ := http.NewRequest("PUT", "/admin/system-prompts/tagging_system", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpdateSystemPrompt_Unauthorized() {
	body, _ := json.Marshal(map[string]string{"value": "new prompt"})
	req, _ := http.NewRequest("PUT", "/admin/system-prompts/tagging_system", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}
