# Changelog

All notable changes to PiPiMink will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Added

- 3-tier authentication model: Public (unauthenticated), User (session or Bearer token), Admin (X-API-Key or admin session)
  - Centralized auth middleware enforces tiers on all routes — new handlers must not perform inline auth checks
  - Bearer token support: per-user API tokens for programmatic access (`POST /auth/tokens`, `GET /auth/tokens`, `DELETE /auth/tokens/{id}`)
  - `REQUIRE_AUTH_FOR_CHAT` config flag: when `true`, chat and API endpoints require User or Admin auth (default: `false` for backward compatibility)
- User-scoped analytics: routing decisions now track `user_id`; admins see all data, regular users see only their own decisions
- Removed inline auth checks from 20+ handlers in favor of centralized middleware (`cmd/server/auth_middleware.go`)
- React console UI replacing the old inline HTML admin pages
  - Model dashboard with discovery, tagging, benchmarking workflow
  - Provider management with per-model config editing
  - Benchmark config and tagging prompt editor
  - Settings page with global save bar and API key vault
  - Analytics dashboard with KPI summary, latency charts, and routing decision log
  - Auth & Users: provider config, user/group management, routing rules, audit log
- OAuth2/OIDC authentication via Authentik as primary auth method
- Session management with encrypted cookies (`gorilla/securecookie`)
- Auth middleware supporting three modes: OAuth session, X-API-Key, passthrough
- User management with RBAC (admin/user roles)
- Group-based routing rules (allow/deny providers and models)
- GDPR-compliant user deletion with mandatory reason and audit trail
- 5 new database tables: auth_providers, users, groups, routing_rules, audit_log
- `docker-compose-authentik.yml` for local Authentik identity provider
- `--with-authentik` flag for `scripts/start-stack.sh`
- Login page with OAuth and API key fallback
- Setup wizard for zero-config first-run experience
- `GET /admin/status` endpoint for instance state detection
- Startup log message guiding users to Console when no admin key is set
- Analytics and routing decision tracking with KPI summaries
- Model reset (`POST /models/{name}/reset`) — clears tags, benchmarks, stats; keeps model entry
- Model full delete (`DELETE /models/{name}`) — removes all data; rediscovery starts fresh
- Benchmark model responses now persisted in database and viewable in the UI
  - `response TEXT` column added to `benchmark_results`
  - Expandable response viewer per model result in Config/Benchmarks section
- Benchmark overhaul: 49 builtin tasks (up from 27)
  - `coding-security` category with 3 tasks (SQL injection, JWT vulnerabilities, path traversal)
  - 18 language-specific coding tasks (C#, Go, Rust, Java, TypeScript, Python at easy/medium/hard)
  - Creative writing uniqueness test (3 distinct short stories)
  - Hard summarization test with strict judge criteria
- `SECURITY.md` — security policy and vulnerability disclosure process
- `CONTRIBUTING.md` — contributor guide with setup, branch, and test conventions
- GitHub issue templates (bug report, feature request)
- GitHub pull request template
- Dev-credential warnings in `docker-compose.yml` and `docker-compose-db.yml`
- Comprehensive Go unit tests for all new handlers (67 tests across 10 files)
  - Auth middleware, admin status, OAuth, console SPA, auth admin CRUD
  - Analytics, config, providers, API keys
- Frontend test infrastructure with Vitest + React Testing Library
  - Hooks: useSetupStatus, useAuth, useTheme
  - API client and App routing tests (22 tests across 5 files)
- Frontend CI job: TypeScript type check + Vitest in GitHub Actions
- OIDC discovery retry logic: `initOAuth()` retries up to 6 times with 5-second intervals in a background goroutine, so PiPiMink starts immediately and non-OAuth routes work while the identity provider is still booting
- First OAuth user automatically gets the `admin` role; subsequent users get `user`
- HTTPS cookie `Secure` flag auto-detection based on `X-Forwarded-Proto` header or `OAUTH_REDIRECT_URL` scheme
- `/auth/login` returns 503 with a user-friendly message during OIDC discovery retry window instead of generic "OAuth not configured"

### Changed

- `/admin` and `/admin/config` now redirect to `/console/models` and `/console/config`
- All model management moved from inline HTML to React console at `/console/`
- Authentik docker-compose upgraded from 2024.12 to 2026.2.2; Redis removed (dropped in Authentik 2025.10), PostgreSQL upgraded to 16-alpine
- `/auth/me` response `oauthEnabled` field now uses config check (`OAuthEnabled()`) instead of runtime `oauthConfig != nil`, so the frontend shows the OAuth login button even during OIDC discovery retry

### Removed

- Old inline HTML admin UI (`admin.go`, 834 lines of embedded HTML)
- Static `/assets/` file server (logos were only used by old HTML)
- Dead code in `internal/api/api.go` — orphaned Server implementation (validators retained)

### Fixed

- Analytics latency time series query using incorrect `date_trunc` unit strings (`"1 hour"` → `"hour"`)

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
- Apache License 2.0
