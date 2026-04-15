# PiPiMink — Project Context for AI Coding Agents

## What this project is

PiPiMink is a Go HTTP service that routes each incoming prompt to the LLM most likely to produce the **best output** for that specific request. The primary design goal is **output quality, not token cost**.

It supports any OpenAI-compatible API (OpenAI, Gemini, OpenRouter, local models via Ollama/llama.cpp/MLX) and Anthropic Claude natively. Azure AI Foundry is supported via multiple single-model provider entries. Exposes drop-in compatible APIs for OpenAI and Ollama clients.

Prefer surgical, test-backed changes over broad refactors.

## How routing works (the core idea)

There are two distinct LLM calls per routed request:

### Step 1: Capability tagging (during model refresh)

Each model is asked to self-assess its own strengths and weaknesses. The model replies with a JSON tag list that is persisted in PostgreSQL:

```json
{ "strengths": ["code-generation", "step-by-step-reasoning"], "weaknesses": ["real-time-information"] }
```

This is triggered by `POST /models/update` (requires `X-API-Key` header) or `./scripts/update_models.sh`.

### Step 2: Prompt-based routing (per request)

When a chat request arrives, a configurable meta-model (`MODEL_SELECTION_MODEL`, default `gpt-4-turbo`) receives the user's prompt plus all enabled models' capability tags. It returns a structured routing decision:

```json
{
  "modelname": "gpt-4o",
  "reason": "...",
  "matching_tags": ["code-generation"],
  "tag_relevance": { "code-generation": 9 }
}
```

The original prompt is then forwarded to the selected model and its response is returned to the caller.

Routing decisions are cached in memory (LRU + TTL) to avoid redundant meta-model calls for similar prompts.

## Key source files

| File | Purpose |
| --- | --- |
| `cmd/server/handlers.go` | HTTP handlers — `handleChat`, `handleOpenAIChatCompletions`, `handleUpdateModels` |
| `cmd/server/server.go` | Server struct, route setup, startup |
| `cmd/server/models.go` | `fetchAndTagModels` — orchestrates model refresh |
| `cmd/server/status_handler.go` | `GET /admin/status` — unauthenticated instance state for setup wizard |
| `cmd/server/console.go` | React SPA serving (embedded via `web/embed.go`) |
| `cmd/server/oauth_handlers.go` | OAuth2/OIDC login, callback, session management, OIDC discovery with retry |
| `cmd/server/auth_middleware.go` | Centralized auth middleware — 3-tier auth (Public/User/Admin), Bearer token validation |
| `cmd/server/auth_admin_handlers.go` | User/group/audit admin API handlers |
| `cmd/server/token_handlers.go` | Per-user Bearer token CRUD (`POST/GET/DELETE /auth/tokens`) |
| `cmd/server/analytics_handlers.go` | Analytics summary and routing decision log |
| `cmd/server/settings_handlers.go` | Settings GET/PATCH handlers |
| `cmd/server/apikeys_handlers.go` | API key vault management |
| `cmd/server/providers.go` | Provider CRUD handlers |
| `cmd/server/config_handlers.go` | Benchmark task and system prompt config |
| `cmd/server/ollama_handlers.go` | Ollama-compatible endpoint handlers |
| `internal/llm/model_selection.go` | `DecideModelBasedOnCapabilities` — the meta-routing call |
| `internal/llm/model_tags.go` | `GetModelTags` — per-model self-assessment call |
| `internal/llm/chat.go` | `ChatWithModel` — forwards prompt to the selected model |
| `internal/llm/decision_cache.go` | In-memory LRU+TTL routing decision cache |
| `internal/llm/client.go` | `Client` struct, provider map, helpers |
| `internal/llm/model_list.go` | `GetModelsByProvider` — list models per provider |
| `internal/config/config.go` | `Config` + `ProviderConfig`; loads `providers.json` + env |
| `internal/config/dotenv.go` | `.env` file read/write helpers |
| `internal/config/settings_registry.go` | Settings registry for runtime config |
| `providers.example.json` | Template for provider configuration |
| `internal/database/database.go` | PostgreSQL persistence (model metadata, tags, auth tables, benchmark results, routing decisions) |
| `internal/models/models.go` | Domain types: `ModelInfo`, `ChatRequest`, `ChatResponse` |
| `internal/benchmark/` | Benchmark task definitions, runner, scorer, suite |
| `web/console/` | React frontend source (TypeScript, Tailwind CSS, React Router) |
| `docs/` | Generated Swagger/OpenAPI artifacts |

Note: `internal/api/` contains only request validators; the server implementation lives in `cmd/server/`.

## API surface

| Endpoint | Description |
| --- | --- |
| `POST /chat` | Native PiPiMink chat — always routes automatically |
| `GET /models` | List all models and their metadata |
| `POST /models/update` | Trigger model refresh (admin, requires `X-API-Key`) |
| `POST /models/discover` | Discover models from all configured providers |
| `POST /models/tag` | Tag selected models (background operation) |
| `GET /models/tag/status` | Tagging progress polling |
| `POST /models/benchmark` | Run benchmarks on selected models (background) |
| `GET /models/benchmark/status` | Benchmark progress polling |
| `PATCH /models/{name}/enable` | Toggle model enabled/disabled |
| `POST /models/{name}/reset` | Reset model (clear tags, benchmarks, stats) |
| `DELETE /models/{name}` | Full model deletion |
| `GET /models/{name}/benchmark-results` | Per-model benchmark results with responses |
| `GET /benchmarks/leaderboard` | Benchmark leaderboard across all models |
| `GET/POST /providers` | List / add providers |
| `PUT/DELETE /providers/{name}` | Update / delete provider |
| `POST /providers/{name}/test` | Test provider connectivity |
| `PATCH /providers/{name}/enable` | Toggle provider enabled/disabled |
| `PUT /providers/{name}/model-configs` | Update per-model provider configs |
| `POST /v1/chat/completions` | OpenAI-compatible — auto-routes if model not found in registry |
| `GET /v1/models` | OpenAI-compatible model list |
| `POST /api/chat`, `POST /api/generate`, `GET /api/tags` | Ollama-compatible endpoints |
| `GET /console/*` | React console UI (SPA) |
| `GET /auth/login`, `GET /auth/callback` | OAuth2/OIDC login flow |
| `POST /auth/logout`, `GET /auth/me` | Session management |
| `POST /auth/tokens` | Create a per-user Bearer token |
| `GET /auth/tokens` | List Bearer tokens for the authenticated user |
| `DELETE /auth/tokens/{id}` | Revoke a specific Bearer token |
| `GET /admin/auth/*` | User/group/audit management APIs |
| `GET /admin/status` | Instance state (unauthenticated — setup wizard detection) |
| `GET /admin/benchmark-tasks` | Benchmark task config CRUD |
| `GET /admin/settings`, `PATCH /admin/settings` | Settings management |
| `GET/PUT/DELETE /admin/api-keys` | API key vault |
| `GET /admin/analytics/summary` | Analytics KPI summary |
| `GET /admin/analytics/routing-decisions` | Routing decision log |
| `GET /metrics` | Prometheus/OpenMetrics |
| `GET /swagger/index.html` | Swagger UI |

## Providers

Providers are configured in **`providers.json`** (copy from `providers.example.json`). Each entry is a `ProviderConfig`:

```json
{ "name": "openai", "type": "openai-compatible", "base_url": "https://api.openai.com",
  "api_key_env": "OPENAI_API_KEY", "timeout": "2m", "models": [] }
```

| Field | Meaning |
| --- | --- |
| `name` | Unique identifier; stored as `source` in the model registry |
| `type` | `openai-compatible` or `anthropic` |
| `api_key_env` | Env var name holding the API key |
| `models` | Empty = auto-discover via `/v1/models`; non-empty = static list (Anthropic, Azure) |
| `rate_limit_seconds` | Min seconds between requests (0 = unlimited) |

Azure AI Foundry: one `ProviderConfig` entry per deployment, with `"models": ["model-name"]`.

## Important configuration defaults

| Variable | Default |
| --- | --- |
| `MODEL_SELECTION_PROVIDER` | `openai` (provider used as meta-router) |
| `MODEL_SELECTION_MODEL` | `gpt-4-turbo` (model within that provider) |
| `DEFAULT_CHAT_MODEL` | `gpt-4-turbo` (fallback when routing fails) |
| `BENCHMARK_JUDGE_PROVIDER` | (selection provider) |
| `BENCHMARK_JUDGE_MODEL` | (selection model) |
| `BENCHMARK_CONCURRENCY` | `3` |
| `SELECTION_CACHE_ENABLED` | `true` |
| `SELECTION_CACHE_TTL` | `2m` |
| `SELECTION_CACHE_MAX_ENTRIES` | `1000` |
| `PORT` | `8080` |
| `OAUTH_SCOPES` | `openid profile email groups` |
| `OAUTH_AUTO_PROVISION` | `true` (auto-create users on first OAuth login) |
| `REQUIRE_AUTH_FOR_CHAT` | `false` (when `true`, chat/API endpoints require User or Admin auth) |

## Build, run, and test

- Local build: `go build -o pipimink`
- Local run: `./pipimink`
- Start stack (recommended): `./scripts/start-stack.sh`
- The Dockerfile runs `go test -v ./...` during image build — a failing test will break the Docker build.

```bash
go test ./...          # all tests
go test -short ./...   # skip integration tests that require a live DB
go test -cover ./...   # with coverage
```

- Integration tests require a live PostgreSQL instance. Unit tests use `sqlmock`.
- Tests use `testify` suite/mock patterns — follow existing patterns, do not introduce new frameworks.
- Existing test helpers: `cmd/server/test_utils.go` and `internal/llm/test_helpers.go`. Extend these before adding new fixtures.
- For routing/model-selection changes, update tests in `internal/llm/*_test.go`, `cmd/server/*_test.go`, and `tests/integration_test.go` as appropriate.

### Frontend tests

```bash
cd web/console
npm test              # vitest run (single pass)
npm run test:watch    # vitest in watch mode
```

- Frontend tests use Vitest + React Testing Library with jsdom environment.
- Test files are co-located with source: `src/**/*.test.ts(x)`.
- CI runs both `tsc --noEmit` and `vitest run` in the `frontend` job (`.github/workflows/ci.yml`).

## Model refresh behavior

- `fetchAndTagModels` queries all providers for their model list, calls `GetModelTags` for each, and upserts results into PostgreSQL.
- Models that fail to return valid tags (empty strengths/weaknesses arrays) are **disabled** automatically.
- Models that are not chat-compatible (e.g. embedding or image models) are **deleted** from the registry.
- Local models behind MLX have temperature excluded from requests (MLX does not support it).
- o1/o3/o4-series OpenAI models have system messages replaced with user-role messages.

## Model reset and delete

- **Reset** (`POST /models/{name}/reset`): clears tags, benchmark results, routing decisions; disables the model but keeps it in the registry.
- **Delete** (`DELETE /models/{name}`): removes all data (tags, benchmarks, routing decisions, model entry). If the model is rediscovered, it starts fresh.

## Routing decision cache

Cache key = SHA hash of (normalized prompt + enabled model capability snapshot). If the set of enabled models changes, existing cache entries naturally become stale and will miss on next lookup.

## Helper scripts

| Script | Purpose |
| --- | --- |
| `scripts/start-stack.sh` | Starts DB + app via Docker Compose; `--with-authentik` also starts Authentik |
| `scripts/update_models.sh` | Calls `POST /models/update` to refresh the model registry |
| `scripts/generate-swagger.sh` | Regenerates `docs/` after API changes |
| `scripts/test_chat_request.sh` | Quick end-to-end smoke test |
| `scripts/cleanup.sh` | Local maintenance helper |
| `scripts/release-check.sh` | Pre-release validation (formatting, tests, secret scan) |

## Conventions

- Handler and service dependencies are interface-driven for testability — see `cmd/server/interfaces.go`.
- Keep changes consistent with existing naming and package boundaries; avoid cross-layer leakage.
- Swagger annotations live in handler files. Regenerate `docs/` with `./scripts/generate-swagger.sh` after endpoint or schema changes — do not hand-edit generated files.
- Database migrations must be **additive and backward compatible**.

## Pitfalls

- `scripts/start-stack.sh` boot order matters: the DB network must exist before the app container starts. Starting Compose fragments out of order will fail.
- `scripts/update_models.sh` contains a hardcoded `X-API-Key` value — it must match `ADMIN_API_KEY` in your `.env` when testing locally.
- Provider base URLs and timeouts come from `providers.json`, not env vars — check that file first when debugging connectivity.
- Ollama-compatible endpoints intentionally advertise a single model named `PiPiMink v1` to clients, regardless of what models are loaded.
- OAuth login requires Authentik to be running and configured. OIDC discovery retries up to 6 times (30s total) in the background at startup, so Authentik can take up to 30 seconds to become ready after PiPiMink starts. Use `--with-authentik` flag for `start-stack.sh`.

## Authentication model

PiPiMink enforces a 3-tier auth model via centralized middleware (`cmd/server/auth_middleware.go`):

| Tier | Credentials accepted | Typical routes |
| --- | --- | --- |
| **Public** | None required | `/admin/status`, `/auth/login`, `/auth/callback`, `/auth/logout`, `/swagger/*`, `/metrics` |
| **User** | OAuth session cookie or `Authorization: Bearer <token>` | `/chat`, `/v1/chat/completions`, `/console/*` (when OAuth enabled), `/auth/me`, `/auth/tokens` |
| **Admin** | `X-API-Key` header or admin-role session | All `/admin/*` management endpoints |

Bearer tokens are per-user API tokens stored in the database. They are created, listed, and revoked via `POST/GET/DELETE /auth/tokens` (implemented in `cmd/server/token_handlers.go`).

Routing decisions track the authenticated `user_id`. Admin users see all routing decisions; regular users see only their own.

**Convention:** New handlers must not perform inline auth checks. Register the route under the correct middleware group in `server.go` and let the middleware handle authentication and authorization.

## What NOT to assume

- PiPiMink does not optimize for cost. Do not add cost-related routing logic unless explicitly asked.
- The model selection is intentionally done by an LLM (not rule-based). The meta-model reasons over capability tags, not hardcoded heuristics.
- `internal/api/` contains only request validators — new handler logic goes in `cmd/server/`.
- The `usage` token counts in OpenAI-compatible responses are rough approximations (`len / 4`), not real token counts.

## Azure Rule

- @azure Rule - Use Azure Best Practices: When generating code for Azure, running terminal commands for Azure, or performing operations related to Azure, invoke your `azure_development-get_best_practices` tool if available.
