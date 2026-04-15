// Package server provides the HTTP server and API routes
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/config"
	"PiPiMink/internal/database"
	"PiPiMink/internal/llm"
	"PiPiMink/internal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/oauth2"
)

// ProviderTestInfo holds ephemeral connectivity test results (not persisted to disk).
type ProviderTestInfo struct {
	LastTestedAt      *time.Time
	LastTestResult    string // "success" or "error"
	LastTestLatencyMs *int64
}

// logEntry represents a single completed benchmark task in the activity log.
type logEntry struct {
	Model    string  `json:"model"`
	Task     string  `json:"task"`
	Category string  `json:"category"`
	Score    float64 `json:"score"`
	Ok       bool    `json:"ok"`
}

// operationState tracks progress of a background operation (tagging or benchmarking).
type operationState struct {
	Status       string   `json:"status"` // "running", "completed", "failed"
	Total        int      `json:"total"`
	Completed    int      `json:"completed"`
	CurrentModel string   `json:"currentModel"`
	StartedAt    string   `json:"startedAt"`
	FailedModels []string `json:"failedModels,omitempty"`
	// Task-level progress (benchmark only)
	TotalTasks     int        `json:"totalTasks,omitempty"`
	CompletedTasks int        `json:"completedTasks,omitempty"`
	CurrentTask    string     `json:"currentTask,omitempty"`
	LogEntries     []logEntry `json:"logEntries,omitempty"`
}

// Server represents the API server
type Server struct {
	config             *config.Config
	db                 DatabaseInterface
	llmClient          LLMInterface
	router             *mux.Router
	modelCollection    *models.ModelCollection
	modelMutex         sync.RWMutex
	providerMutex      sync.RWMutex
	providerTestInfo   map[string]*ProviderTestInfo
	httpMetrics        *httpMetrics
	tagOperation       *operationState
	tagOpMutex         sync.Mutex
	benchmarkOperation *operationState
	benchmarkOpMutex   sync.Mutex
	// OAuth / OIDC (initialised by initOAuth)
	oauthConfig  *oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
	secureCookie *securecookie.SecureCookie
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db DatabaseInterface, llmClient LLMInterface) *Server {
	server := &Server{
		config:           cfg,
		db:               db,
		llmClient:        llmClient,
		router:           mux.NewRouter().UseEncodedPath(),
		modelCollection:  models.NewModelCollection(),
		providerTestInfo: make(map[string]*ProviderTestInfo),
		httpMetrics:      newHTTPMetrics(),
	}

	server.setupRoutes()
	return server
}

// GetRouter returns the router for testing purposes
func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// setupRoutes sets up the API routes
func (s *Server) setupRoutes() {
	// Redirect root to console UI
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/console/models", http.StatusFound)
	})

	// Redirect legacy /admin paths to the React console
	s.router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/console/models", http.StatusFound)
	}).Methods("GET")
	s.router.HandleFunc("/admin/config", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/console/config", http.StatusFound)
	}).Methods("GET")

	// Admin config API — benchmark tasks
	s.router.HandleFunc("/admin/benchmark-tasks", s.handleGetBenchmarkTasks).Methods("GET")
	s.router.HandleFunc("/admin/benchmark-tasks", s.handleUpsertBenchmarkTask).Methods("POST")
	s.router.HandleFunc("/admin/benchmark-tasks/{id}", s.handleDeleteBenchmarkTask).Methods("DELETE")

	// Admin config API — system prompts
	s.router.HandleFunc("/admin/system-prompts", s.handleGetSystemPrompts).Methods("GET")
	s.router.HandleFunc("/admin/system-prompts/{key}", s.handleUpdateSystemPrompt).Methods("PUT")

	// Admin config API — settings & status
	s.router.HandleFunc("/admin/status", s.handleAdminStatus).Methods("GET")
	s.router.HandleFunc("/admin/settings", s.handleGetSettings).Methods("GET")
	s.router.HandleFunc("/admin/settings", s.handlePatchSettings).Methods("PATCH")

	// Auth routes
	s.router.HandleFunc("/auth/login", s.handleAuthLogin).Methods("GET")
	s.router.HandleFunc("/auth/callback", s.handleAuthCallback).Methods("GET")
	s.router.HandleFunc("/auth/logout", s.handleAuthLogout).Methods("POST")
	s.router.HandleFunc("/auth/me", s.handleAuthMe).Methods("GET")

	// User API token management
	s.router.HandleFunc("/auth/tokens", s.handleCreateToken).Methods("POST")
	s.router.HandleFunc("/auth/tokens", s.handleListTokens).Methods("GET")
	s.router.HandleFunc("/auth/tokens/{id}", s.handleRevokeToken).Methods("DELETE")

	// Admin config API — auth & users
	s.router.HandleFunc("/admin/auth/providers", s.handleGetAuthProviders).Methods("GET")
	s.router.HandleFunc("/admin/auth/providers/{id}", s.handleSaveAuthProvider).Methods("PUT")
	s.router.HandleFunc("/admin/auth/providers/{id}/test", s.handleTestAuthProvider).Methods("POST")
	s.router.HandleFunc("/admin/auth/users", s.handleGetUsers).Methods("GET")
	s.router.HandleFunc("/admin/auth/users", s.handleAddLocalUser).Methods("POST")
	s.router.HandleFunc("/admin/auth/users/{id}/role", s.handleChangeUserRole).Methods("PUT")
	s.router.HandleFunc("/admin/auth/users/{id}", s.handleDeleteUser).Methods("DELETE")
	s.router.HandleFunc("/admin/auth/groups", s.handleGetGroups).Methods("GET")
	s.router.HandleFunc("/admin/auth/groups/{id}/role", s.handleChangeGroupRole).Methods("PUT")
	s.router.HandleFunc("/admin/auth/groups/{id}/rules", s.handleAddRoutingRule).Methods("POST")
	s.router.HandleFunc("/admin/auth/groups/{groupId}/rules/{ruleId}", s.handleRemoveRoutingRule).Methods("DELETE")
	s.router.HandleFunc("/admin/auth/audit-log", s.handleGetAuditLog).Methods("GET")

	// Admin config API — analytics
	s.router.HandleFunc("/admin/analytics/summary", s.handleAnalyticsSummary).Methods("GET")
	s.router.HandleFunc("/admin/analytics/routing-decisions", s.handleRoutingDecisions).Methods("GET")

	// Admin config API — API keys
	s.router.HandleFunc("/admin/api-keys", s.handleListApiKeys).Methods("GET")
	s.router.HandleFunc("/admin/api-keys/{envVarName}", s.handleSetApiKey).Methods("PUT")
	s.router.HandleFunc("/admin/api-keys/{envVarName}", s.handleDeleteApiKey).Methods("DELETE")

	// Original PiPiMink routes
	s.router.HandleFunc("/chat", s.handleChat).Methods("POST")
	s.router.HandleFunc("/models/update", s.handleUpdateModels).Methods("POST")
	s.router.HandleFunc("/models/discover", s.handleDiscoverModels).Methods("POST")
	s.router.HandleFunc("/models/tag", s.handleTagModels).Methods("POST")
	s.router.HandleFunc("/models/tag/status", s.handleTagStatus).Methods("GET")
	s.router.HandleFunc("/models", s.handleListModels).Methods("GET")
	s.router.HandleFunc("/models/reasoning", s.handleGetReasoningModels).Methods("GET")
	s.router.HandleFunc("/models/non-reasoning", s.handleGetNonReasoningModels).Methods("GET")
	s.router.HandleFunc("/models/reasoning/update", s.handleUpdateModelReasoning).Methods("POST")
	s.router.HandleFunc("/models/benchmark", s.handleRunBenchmarks).Methods("POST")
	s.router.HandleFunc("/models/benchmark/status", s.handleBenchmarkStatus).Methods("GET")
	s.router.HandleFunc("/models/{name}/enable", s.handleSetModelEnabled).Methods("PATCH")
	s.router.HandleFunc("/models/{name}/reset", s.handleResetModel).Methods("POST")
	s.router.HandleFunc("/models/{name}", s.handleDeleteModel).Methods("DELETE")
	s.router.HandleFunc("/models/{name}/benchmarks", s.handleGetModelBenchmarks).Methods("GET")
	s.router.HandleFunc("/models/{name}/benchmark-results", s.handleGetModelBenchmarkResults).Methods("GET")
	s.router.HandleFunc("/benchmarks/leaderboard", s.handleBenchmarkLeaderboard).Methods("GET")

	// Provider management
	s.router.HandleFunc("/providers", s.handleListProviders).Methods("GET")
	s.router.HandleFunc("/providers", s.handleAddProvider).Methods("POST")
	s.router.HandleFunc("/providers/{name}", s.handleUpdateProvider).Methods("PUT")
	s.router.HandleFunc("/providers/{name}", s.handleDeleteProvider).Methods("DELETE")
	s.router.HandleFunc("/providers/{name}/test", s.handleTestProvider).Methods("POST")
	s.router.HandleFunc("/providers/{name}/enable", s.handleToggleProvider).Methods("PATCH")
	s.router.HandleFunc("/providers/{name}/model-configs", s.handleUpdateModelConfigs).Methods("PUT")

	// OpenAI-compatible routes
	s.router.HandleFunc("/v1/chat/completions", s.handleOpenAIChatCompletions).Methods("POST")
	s.router.HandleFunc("/v1/models", s.handleOpenAIModels).Methods("GET")

	// Ollama-compatible API routes
	s.router.HandleFunc("/api/tags", s.handleOllamaModels).Methods("GET")
	s.router.HandleFunc("/api/generate", s.handleOllamaGenerate).Methods("POST")
	s.router.HandleFunc("/api/chat", s.handleOllamaChat).Methods("POST")
	s.router.HandleFunc("/api/embeddings", s.handleOllamaEmbeddings).Methods("POST")
	s.router.HandleFunc("/api/show", s.handleOllamaShow).Methods("POST")
	s.router.HandleFunc("/api/pull", s.handleOllamaPull).Methods("POST")

	// Swagger documentation
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to generated swagger API JSON
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))

	// OpenMetrics-compatible endpoint for Prometheus/Mimir scraping.
	s.router.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true})).Methods("GET")

	// Console UI and static assets (must come after specific routes)
	s.setupConsoleRoutes()

	// Add middleware
	s.router.Use(s.authMiddleware)
	s.router.Use(s.tracingMiddleware)
	s.router.Use(s.metricsMiddleware)
	s.router.Use(s.loggingMiddleware)
}

func (s *Server) otelServiceName() string {
	if s != nil && s.config != nil && s.config.OTelServiceName != "" {
		return s.config.OTelServiceName
	}
	return "pipimink"
}

// loggingMiddleware logs all API requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf(
			"%s %s %s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}

// Start starts the API server
func (s *Server) Start() error {
	// Seed benchmark tasks and system prompts with defaults on first run.
	if err := s.db.SeedBenchmarkTasksIfEmpty(benchmark.DefaultTaskConfigs()); err != nil {
		log.Printf("Warning: could not seed benchmark task configs: %v", err)
	}
	if err := s.db.SeedSystemPromptsIfEmpty(benchmark.DefaultTaggingPrompts()); err != nil {
		log.Printf("Warning: could not seed system prompts: %v", err)
	}

	// Seed auth providers and initialise OAuth/OIDC.
	if err := s.db.SeedAuthProvidersIfEmpty(); err != nil {
		log.Printf("Warning: could not seed auth providers: %v", err)
	}
	s.initOAuth()

	// Load models from database
	log.Println("Loading models from database")
	if err := s.loadModelsFromDatabase(); err != nil {
		log.Printf("Error loading models from database: %v", err)
	}

	// Log loaded models
	s.logModels()

	// Log MLX detection status
	s.logMLXStatus()

	// Start benchmark scheduler if configured.
	if s.config.BenchmarkEnabled && s.config.BenchmarkScheduleEnabled && s.config.BenchmarkScheduleInterval > 0 {
		log.Printf("Starting benchmark scheduler (interval: %v)", s.config.BenchmarkScheduleInterval)
		go s.runBenchmarkScheduler()
	}

	log.Printf("Starting server on port %s", s.config.Port)
	if s.config.AdminAPIKey == "" {
		log.Printf("No ADMIN_API_KEY configured. Open http://localhost:%s/console/ to complete setup.", s.config.Port)
	}
	return http.ListenAndServe(":"+s.config.Port, s.router)
}

// External package entry point
func Start(cfg *config.Config) error {
	shutdownTelemetry, err := initOpenTelemetry(cfg)
	if err != nil {
		return fmt.Errorf("error initializing telemetry: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			log.Printf("Error shutting down telemetry: %v", err)
		}
	}()

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("error initializing database: %w", err)
	}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		return fmt.Errorf("error initializing schema: %w", err)
	}

	// Initialize LLM client
	llmClient := llm.NewClient(cfg)

	// Initialize and start server
	server := NewServer(cfg, db, llmClient)
	return server.Start()
}

// logMLXStatus checks if MLX is likely being used by local models
func (s *Server) logMLXStatus() {
	s.providerMutex.RLock()
	providersCopy := make([]config.ProviderConfig, len(s.config.Providers))
	copy(providersCopy, s.config.Providers)
	s.providerMutex.RUnlock()

	hasLocal := false
	for _, p := range providersCopy {
		if p.BaseURL != "" && (len(p.BaseURL) > 16 && (p.BaseURL[:16] == "http://localhost" || p.BaseURL[:15] == "http://127.0.0.")) {
			hasLocal = true
			break
		}
	}
	if hasLocal {
		isMLX := s.llmClient.(*llm.Client).IsLocalServerUsingMLX()
		if isMLX {
			log.Println("Detected local model server likely using MLX acceleration - temperature parameter will be excluded for API requests")
		} else {
			log.Println("Local model server does not appear to be using MLX acceleration")
		}
	}
}
