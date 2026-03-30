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

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server represents the API server
type Server struct {
	config          *config.Config
	db              DatabaseInterface
	llmClient       LLMInterface
	router          *mux.Router
	modelCollection *models.ModelCollection
	modelMutex      sync.RWMutex
	httpMetrics     *httpMetrics
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db DatabaseInterface, llmClient LLMInterface) *Server {
	server := &Server{
		config:          cfg,
		db:              db,
		llmClient:       llmClient,
		router:          mux.NewRouter().UseEncodedPath(),
		modelCollection: models.NewModelCollection(),
		httpMetrics:     newHTTPMetrics(),
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
	// Add a redirect from root path to Swagger UI
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	// Admin UI
	s.router.HandleFunc("/admin", s.handleAdminUI).Methods("GET")
	s.router.HandleFunc("/admin/config", s.handleConfigUI).Methods("GET")

	// Admin config API — benchmark tasks
	s.router.HandleFunc("/admin/benchmark-tasks", s.handleGetBenchmarkTasks).Methods("GET")
	s.router.HandleFunc("/admin/benchmark-tasks", s.handleUpsertBenchmarkTask).Methods("POST")
	s.router.HandleFunc("/admin/benchmark-tasks/{id}", s.handleDeleteBenchmarkTask).Methods("DELETE")

	// Admin config API — system prompts
	s.router.HandleFunc("/admin/system-prompts", s.handleGetSystemPrompts).Methods("GET")
	s.router.HandleFunc("/admin/system-prompts/{key}", s.handleUpdateSystemPrompt).Methods("PUT")

	// Original PiPiMink routes
	s.router.HandleFunc("/chat", s.handleChat).Methods("POST")
	s.router.HandleFunc("/models/update", s.handleUpdateModels).Methods("POST")
	s.router.HandleFunc("/models/discover", s.handleDiscoverModels).Methods("POST")
	s.router.HandleFunc("/models/tag", s.handleTagModels).Methods("POST")
	s.router.HandleFunc("/models", s.handleListModels).Methods("GET")
	s.router.HandleFunc("/models/reasoning", s.handleGetReasoningModels).Methods("GET")
	s.router.HandleFunc("/models/non-reasoning", s.handleGetNonReasoningModels).Methods("GET")
	s.router.HandleFunc("/models/reasoning/update", s.handleUpdateModelReasoning).Methods("POST")
	s.router.HandleFunc("/models/benchmark", s.handleRunBenchmarks).Methods("POST")
	s.router.HandleFunc("/models/{name}/enable", s.handleSetModelEnabled).Methods("PATCH")
	s.router.HandleFunc("/models/{name}/benchmarks", s.handleGetModelBenchmarks).Methods("GET")
	s.router.HandleFunc("/benchmarks/leaderboard", s.handleBenchmarkLeaderboard).Methods("GET")

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

	// Add middleware for logging and authentication
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
	hasLocal := false
	for _, p := range s.config.Providers {
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
