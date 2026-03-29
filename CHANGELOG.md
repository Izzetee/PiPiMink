# Changelog

All notable changes to PiPiMink will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Added

- `SECURITY.md` — security policy and vulnerability disclosure process
- `CONTRIBUTING.md` — contributor guide with setup, branch, and test conventions
- GitHub issue templates (bug report, feature request)
- GitHub pull request template
- Dev-credential warnings in `docker-compose.yml` and `docker-compose-db.yml`

---

## [0.5.0] — Benchmarking System

### Added

- Benchmark runner, scorer, suite, and task infrastructure (`internal/benchmark/`)
- System prompt management for benchmark tasks
- Per-model benchmark scoring stored in the database
- Benchmark scores surfaced as a secondary routing signal alongside capability tags
- Azure AI Foundry support: per-model provider entries with individual API key management

### Changed

- Routing priority clarified: capability tags (primary) → benchmark scores (secondary) → response time (tiebreaker)
- Sequence diagram in README updated to reflect benchmark score integration

---

## [0.4.0] — Azure & Multi-Endpoint Support

### Added

- Azure AI Foundry provider type with three supported endpoint patterns (Models-as-a-Service, Serverless API, Azure OpenAI)
- MLX detection: temperature parameter excluded for MLX-backed models
- o1/o3/o4-series compatibility: system messages automatically converted to user-role messages
- Multiple provider endpoints can be registered for the same underlying service

### Changed

- `providers.example.json` extended with Azure AI Foundry examples

---

## [0.3.0] — Observability & Compatibility

### Added

- OpenTelemetry tracing and metrics (`cmd/server/telemetry.go`, `http_tracing.go`, `http_metrics.go`)
- Prometheus/OpenMetrics endpoint (`GET /metrics`)
- Selection cache statistics logging and configuration (`SELECTION_CACHE_ENABLED`, `SELECTION_CACHE_TTL`, `SELECTION_CACHE_MAX_ENTRIES`)
- Multi-turn conversation support in `ChatWithModel`
- Ollama-compatible endpoints (`POST /api/chat`, `POST /api/generate`, `GET /api/tags`)
- Admin UI for model management and benchmark configuration

### Changed

- Default meta-routing model updated to `gpt-4-turbo`
- Improved model tag prompts for better routing accuracy

---

## [0.2.0] — Routing Core & Testing

### Added

- LRU + TTL routing decision cache (`internal/llm/decision_cache.go`)
- Rate limiter per provider (`internal/llm/rate_limiter.go`)
- OpenAI-compatible endpoints (`POST /v1/chat/completions`, `GET /v1/models`)
- Swagger UI (`GET /swagger/index.html`) and auto-generated OpenAPI docs
- Unit tests for model selection, cache, rate limiter, and API validation
- SQL mock-based database tests (`DATA-DOG/go-sqlmock`)
- GitHub Actions CI: `gofmt`, `go vet`, `go test`, Markdown lint, secret scan (Gitleaks)
- Dependabot configuration for Go modules and GitHub Actions

### Changed

- Models with empty capability tags (no valid strengths/weaknesses) are automatically disabled
- Non-chat models (embeddings, image generation) are deleted from the registry on refresh

---

## [0.1.0] — Initial Release

### Added

- Core routing loop: capability tagging (model self-assessment) + prompt-based model selection via a meta-model
- `POST /chat` native endpoint and `POST /models/update` admin endpoint
- PostgreSQL persistence for model metadata and capability tags
- Multi-provider support: OpenAI-compatible APIs and Anthropic Claude
- `providers.json` configuration with `providers.example.json` template
- Docker Compose stack (`docker-compose.yml`, `docker-compose-db.yml`, `docker-compose-app.yml`)
- Helper scripts in `scripts/`: `start-stack.sh`, `update_models.sh`, `test_chat_request.sh`, `generate-swagger.sh`
- MIT License
