package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/config"
	"PiPiMink/internal/database"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// errTest is a sentinel error for mock expectations that need to return a failure.
var errTest = errors.New("test error")

// Ensure mock types satisfy interfaces at compile time.
var _ LLMInterface = (*MockLLMClient)(nil)
var _ DatabaseInterface = (*MockDB)(nil)

// MockDB implements DatabaseInterface for testing
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetAllModels() (map[string]map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]map[string]interface{}), args.Error(1)
}

func (m *MockDB) SaveModel(name, source, tags string, enabled bool, hasReasoning bool) error {
	args := m.Called(name, source, tags, enabled, hasReasoning)
	return args.Error(0)
}

func (m *MockDB) RegisterDiscoveredModel(name, source string) error {
	args := m.Called(name, source)
	return args.Error(0)
}

func (m *MockDB) UpdateModelReasoning(name, source string, hasReasoning bool) error {
	args := m.Called(name, source, hasReasoning)
	return args.Error(0)
}

func (m *MockDB) HasModels() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockDB) EnableModel(name, source string, enabled bool) error {
	args := m.Called(name, source, enabled)
	return args.Error(0)
}

// Add DeleteModel to MockDB
func (m *MockDB) DeleteModel(name, source string) error {
	args := m.Called(name, source)
	return args.Error(0)
}

func (m *MockDB) ResetModel(name, source string) error {
	args := m.Called(name, source)
	return args.Error(0)
}

func (m *MockDB) DeleteModelFull(name, source string) error {
	args := m.Called(name, source)
	return args.Error(0)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64, judgeModel, response string) error {
	args := m.Called(modelName, source, category, taskID, score, latencyMs, judgeModel, response)
	return args.Error(0)
}

func (m *MockDB) GetBenchmarkScores(modelName, source string) (map[string]float64, error) {
	args := m.Called(modelName, source)
	return args.Get(0).(map[string]float64), args.Error(1)
}

func (m *MockDB) GetBenchmarkResults(modelName, source string) ([]database.BenchmarkResult, error) {
	args := m.Called(modelName, source)
	return args.Get(0).([]database.BenchmarkResult), args.Error(1)
}

func (m *MockDB) GetAllBenchmarkScores() (map[string]map[string]float64, error) {
	args := m.Called()
	return args.Get(0).(map[string]map[string]float64), args.Error(1)
}

func (m *MockDB) GetAllModelLatencies() (map[string]int64, error) {
	args := m.Called()
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockDB) SeedBenchmarkTasksIfEmpty(defaults []benchmark.BenchmarkTaskConfig) error {
	args := m.Called(defaults)
	return args.Error(0)
}

func (m *MockDB) GetBenchmarkTaskConfigs() ([]benchmark.BenchmarkTaskConfig, error) {
	args := m.Called()
	return args.Get(0).([]benchmark.BenchmarkTaskConfig), args.Error(1)
}

func (m *MockDB) UpsertBenchmarkTaskConfig(cfg benchmark.BenchmarkTaskConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockDB) DeleteBenchmarkTaskConfig(taskID string, defaultCfgs []benchmark.BenchmarkTaskConfig) error {
	args := m.Called(taskID, defaultCfgs)
	return args.Error(0)
}

func (m *MockDB) SeedSystemPromptsIfEmpty(defaults map[string]string) error {
	args := m.Called(defaults)
	return args.Error(0)
}

func (m *MockDB) GetSystemPrompt(key string) (string, bool, error) {
	args := m.Called(key)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockDB) SetSystemPrompt(key, value, description string) error {
	args := m.Called(key, value, description)
	return args.Error(0)
}

func (m *MockDB) GetAllSystemPrompts() (map[string]database.SystemPromptRow, error) {
	args := m.Called()
	return args.Get(0).(map[string]database.SystemPromptRow), args.Error(1)
}

func (m *MockDB) SaveRoutingDecision(rd database.RoutingDecisionRow) error {
	args := m.Called(rd)
	return args.Error(0)
}

func (m *MockDB) GetRoutingDecisions(start, end time.Time, limit, offset int) ([]database.RoutingDecisionRow, int, error) {
	args := m.Called(start, end, limit, offset)
	return args.Get(0).([]database.RoutingDecisionRow), args.Int(1), args.Error(2)
}

func (m *MockDB) GetKpiSummary(start, end time.Time) (database.KpiSummary, error) {
	args := m.Called(start, end)
	return args.Get(0).(database.KpiSummary), args.Error(1)
}

func (m *MockDB) GetModelUsage(start, end time.Time) ([]database.ModelUsageRow, error) {
	args := m.Called(start, end)
	return args.Get(0).([]database.ModelUsageRow), args.Error(1)
}

func (m *MockDB) GetLatencyPerModel(start, end time.Time) ([]database.LatencyPerModelRow, error) {
	args := m.Called(start, end)
	return args.Get(0).([]database.LatencyPerModelRow), args.Error(1)
}

func (m *MockDB) GetLatencyTimeSeries(start, end time.Time) ([]database.LatencyTimeSeriesRow, error) {
	args := m.Called(start, end)
	return args.Get(0).([]database.LatencyTimeSeriesRow), args.Error(1)
}

func (m *MockDB) GetLatencyPercentiles(start, end time.Time) ([]database.LatencyPercentilesRow, error) {
	args := m.Called(start, end)
	return args.Get(0).([]database.LatencyPercentilesRow), args.Error(1)
}

// Auth & Users mock methods

func (m *MockDB) SeedAuthProvidersIfEmpty() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDB) GetAuthProviders() ([]database.AuthProviderRow, error) {
	args := m.Called()
	return args.Get(0).([]database.AuthProviderRow), args.Error(1)
}

func (m *MockDB) SaveAuthProvider(p database.AuthProviderRow) error {
	args := m.Called(p)
	return args.Error(0)
}

func (m *MockDB) GetUsers() ([]database.UserRow, error) {
	args := m.Called()
	return args.Get(0).([]database.UserRow), args.Error(1)
}

func (m *MockDB) GetUserByEmail(email string) (*database.UserRow, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.UserRow), args.Error(1)
}

func (m *MockDB) UpsertUser(u database.UserRow) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockDB) ChangeUserRole(userID, role string) error {
	args := m.Called(userID, role)
	return args.Error(0)
}

func (m *MockDB) DeleteUser(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockDB) GetGroups() ([]database.GroupRow, error) {
	args := m.Called()
	return args.Get(0).([]database.GroupRow), args.Error(1)
}

func (m *MockDB) SaveGroup(g database.GroupRow) error {
	args := m.Called(g)
	return args.Error(0)
}

func (m *MockDB) ChangeGroupRole(groupID, role string) error {
	args := m.Called(groupID, role)
	return args.Error(0)
}

func (m *MockDB) SaveRoutingRule(groupID string, r database.RoutingRuleRow) error {
	args := m.Called(groupID, r)
	return args.Error(0)
}

func (m *MockDB) DeleteRoutingRule(groupID, ruleID string) error {
	args := m.Called(groupID, ruleID)
	return args.Error(0)
}

func (m *MockDB) GetAuditLog() ([]database.AuditEntryRow, error) {
	args := m.Called()
	return args.Get(0).([]database.AuditEntryRow), args.Error(1)
}

func (m *MockDB) SaveAuditEntry(e database.AuditEntryRow) error {
	args := m.Called(e)
	return args.Error(0)
}

// User API token mock methods

func (m *MockDB) CreateUserAPIToken(userID, name string) (string, string, error) {
	args := m.Called(userID, name)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockDB) GetUserByAPIToken(tokenHash string) (*database.UserRow, error) {
	args := m.Called(tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.UserRow), args.Error(1)
}

func (m *MockDB) ListUserAPITokens(userID string) ([]database.UserAPITokenRow, error) {
	args := m.Called(userID)
	return args.Get(0).([]database.UserAPITokenRow), args.Error(1)
}

func (m *MockDB) RevokeUserAPIToken(tokenID, userID string) error {
	args := m.Called(tokenID, userID)
	return args.Error(0)
}

// User-scoped (filtered) analytics mock methods

func (m *MockDB) GetRoutingDecisionsFiltered(start, end time.Time, limit, offset int, userID string) ([]database.RoutingDecisionRow, int, error) {
	args := m.Called(start, end, limit, offset, userID)
	return args.Get(0).([]database.RoutingDecisionRow), args.Int(1), args.Error(2)
}

func (m *MockDB) GetKpiSummaryFiltered(start, end time.Time, userID string) (database.KpiSummary, error) {
	args := m.Called(start, end, userID)
	return args.Get(0).(database.KpiSummary), args.Error(1)
}

func (m *MockDB) GetModelUsageFiltered(start, end time.Time, userID string) ([]database.ModelUsageRow, error) {
	args := m.Called(start, end, userID)
	return args.Get(0).([]database.ModelUsageRow), args.Error(1)
}

func (m *MockDB) GetLatencyPerModelFiltered(start, end time.Time, userID string) ([]database.LatencyPerModelRow, error) {
	args := m.Called(start, end, userID)
	return args.Get(0).([]database.LatencyPerModelRow), args.Error(1)
}

func (m *MockDB) GetLatencyTimeSeriesFiltered(start, end time.Time, userID string) ([]database.LatencyTimeSeriesRow, error) {
	args := m.Called(start, end, userID)
	return args.Get(0).([]database.LatencyTimeSeriesRow), args.Error(1)
}

func (m *MockDB) GetLatencyPercentilesFiltered(start, end time.Time, userID string) ([]database.LatencyPercentilesRow, error) {
	args := m.Called(start, end, userID)
	return args.Get(0).([]database.LatencyPercentilesRow), args.Error(1)
}

// MockLLMClient implements LLMInterface for testing
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) ChatWithModel(modelInfo models.ModelInfo, model string, messages []map[string]interface{}) (string, error) {
	args := m.Called(modelInfo, model, messages)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) DecideModelBasedOnCapabilities(message string, availableModels map[string]models.ModelInfo) (models.RoutingResult, error) {
	args := m.Called(message, availableModels)
	return args.Get(0).(models.RoutingResult), args.Error(1)
}

func (m *MockLLMClient) GetModelTags(model string, p config.ProviderConfig) (string, bool, bool, error) {
	args := m.Called(model, p)
	return args.String(0), args.Bool(1), args.Bool(2), args.Error(3)
}

func (m *MockLLMClient) GetModelsByProvider(p config.ProviderConfig) ([]string, error) {
	args := m.Called(p)
	return args.Get(0).([]string), args.Error(1)
}

// Implement IsLocalServerUsingMLX to satisfy the LLMInterface
func (m *MockLLMClient) IsLocalServerUsingMLX() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLLMClient) UpdateTaggingPrompts(systemPrompt, userPrompt, userNoSysPrompt string) {
	m.Called(systemPrompt, userPrompt, userNoSysPrompt)
}

// ServerTestSuite is a test suite for the Server
type ServerTestSuite struct {
	suite.Suite
	server   *Server
	mockDB   *MockDB
	mockLLM  *MockLLMClient
	config   *config.Config
	recorder *httptest.ResponseRecorder
}

// SetupTest is called before each test
func (s *ServerTestSuite) SetupTest() {
	s.config = &config.Config{
		AdminAPIKey: "test-admin-key",
		Port:        "8080",
	}

	s.mockDB = new(MockDB)
	s.mockLLM = new(MockLLMClient)

	// Set up expectations for IsLocalServerUsingMLX (added for MLX detection feature)
	// Using .Maybe() to indicate this method can be called any number of times
	s.mockLLM.On("IsLocalServerUsingMLX").Return(false).Maybe()
	// GetAllBenchmarkScores and GetAllModelLatencies are called by loadModelsFromDatabase on every model reload.
	s.mockDB.On("GetAllBenchmarkScores").Return(map[string]map[string]float64{}, nil).Maybe()
	s.mockDB.On("GetAllModelLatencies").Return(map[string]int64{}, nil).Maybe()
	// EnableModel may be called by handleSetModelEnabled.
	s.mockDB.On("EnableModel", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	// RegisterDiscoveredModel may be called by handleDiscoverModels.
	s.mockDB.On("RegisterDiscoveredModel", mock.Anything, mock.Anything).Return(nil).Maybe()
	// SaveAuditEntry is called asynchronously by auth admin handlers.
	s.mockDB.On("SaveAuditEntry", mock.Anything).Return(nil).Maybe()
	// SaveRoutingDecision is called asynchronously after chat routing.
	s.mockDB.On("SaveRoutingDecision", mock.Anything).Return(nil).Maybe()

	s.server = NewServer(s.config, s.mockDB, s.mockLLM)
	s.recorder = httptest.NewRecorder()
}

// TestHandleChat tests the chat endpoint
func (s *ServerTestSuite) TestHandleChat() {
	// Setup test models
	modelMap := map[string]map[string]interface{}{
		"gpt-4-turbo": {
			"source":     "openai",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model description",
			"enabled":    true,
			"updated_at": "2023-05-01",
		},
	}

	// Configure mock behavior
	s.mockDB.On("GetAllModels").Return(modelMap, nil)
	s.mockLLM.On("DecideModelBasedOnCapabilities", "Hello, how are you?", mock.Anything).Return(models.RoutingResult{ModelName: "gpt-4-turbo"}, nil)

	// We need to specify modelInfo parameter more precisely for ChatWithModel
	modelInfo := models.ModelInfo{
		Source:    "openai",
		Tags:      `{"capabilities":["general"]}`,
		Enabled:   true,
		Response:  "Model description",
		UpdatedAt: "2023-05-01",
	}
	expectedMessages := []map[string]interface{}{{"role": "user", "content": "Hello, how are you?"}}
	s.mockLLM.On("ChatWithModel", modelInfo, "gpt-4-turbo", expectedMessages).Return("I'm doing well, thank you for asking!", nil)

	// Load models into server
	// First we need to properly load the models into the server's modelCollection
	collection := models.NewModelCollection()
	collection.FromDatabaseMap(modelMap)
	s.server.modelCollection = collection

	// Create the request
	reqBody := map[string]string{
		"message": "Hello, how are you?",
	}
	reqBytes, err := json.Marshal(reqBody)
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", "/chat", bytes.NewBuffer(reqBytes))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Process the request
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]string
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("I'm doing well, thank you for asking!", response["response"])
	s.Equal("gpt-4-turbo", response["model"])

	// Verify mock expectations
	s.mockLLM.AssertExpectations(s.T())
}

// Run the test suite
func TestServerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite.Run(t, new(ServerTestSuite))
}
