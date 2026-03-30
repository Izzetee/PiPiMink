# PiPiMink — Project Context for Claude

## What this project is

PiPiMink is a Go HTTP service that routes each incoming prompt to the LLM most likely to produce the **best output** for that specific request. The primary design goal is **output quality, not token cost**.

It supports any OpenAI-compatible API (OpenAI, Gemini, OpenRouter, local models via Ollama/llama.cpp/MLX) and Anthropic Claude natively. Azure AI Foundry is supported via multiple single-model provider entries. Exposes drop-in compatible APIs for OpenAI and Ollama clients.

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
| `internal/llm/model_selection.go` | `DecideModelBasedOnCapabilities` — the meta-routing call |
| `internal/llm/model_tags.go` | `GetModelTags` — per-model self-assessment call |
| `internal/llm/chat.go` | `ChatWithModel` — forwards prompt to the selected model |
| `internal/llm/decision_cache.go` | In-memory LRU+TTL routing decision cache |
| `internal/llm/client.go` | `Client` struct, provider map, helpers |
| `internal/llm/model_list.go` | `GetModelsByProvider` — list models per provider |
| `internal/config/config.go` | `Config` + `ProviderConfig`; loads `providers.json` + env |
| `providers.example.json` | Template for provider configuration |
| `internal/database/database.go` | PostgreSQL persistence (model metadata, tags) |
| `internal/models/models.go` | Domain types: `ModelInfo`, `ChatRequest`, `ChatResponse` |
| `docs/` | Generated Swagger/OpenAPI artifacts |

Note: `internal/api/` exists but is **not** the active runtime path.

## API surface

| Endpoint | Description |
| --- | --- |
| `POST /chat` | Native PiPiMink chat — always routes automatically |
| `GET /models` | List all models and their metadata |
| `POST /models/update` | Trigger model refresh (admin, requires `X-API-Key`) |
| `POST /v1/chat/completions` | OpenAI-compatible — auto-routes if model not found in registry |
| `GET /v1/models` | OpenAI-compatible model list |
| `POST /api/chat`, `POST /api/generate`, `GET /api/tags` | Ollama-compatible endpoints |
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
| `SELECTION_CACHE_ENABLED` | `true` |
| `SELECTION_CACHE_TTL` | `2m` |
| `SELECTION_CACHE_MAX_ENTRIES` | `1000` |
| `PORT` | `8080` |

## Model refresh behavior

- `fetchAndTagModels` queries all providers for their model list, calls `GetModelTags` for each, and upserts results into PostgreSQL.
- Models that fail to return valid tags (empty strengths/weaknesses arrays) are **disabled** automatically.
- Models that are not chat-compatible (e.g. embedding or image models) are **deleted** from the registry.
- Local models behind MLX have temperature excluded from requests (MLX does not support it).
- o1/o3/o4-series OpenAI models have system messages replaced with user-role messages.

## Routing decision cache

Cache key = SHA hash of (normalized prompt + enabled model capability snapshot). If the set of enabled models changes, existing cache entries naturally become stale and will miss on next lookup.

## Testing

```bash
go test ./...          # all tests
go test -short ./...   # skip integration tests that require a live DB
go test -cover ./...   # with coverage
```

- Integration tests require a live PostgreSQL instance. Unit tests use `sqlmock`.
- Tests use `testify` suite/mock patterns — follow existing patterns, do not introduce new frameworks.
- Existing test helpers: `cmd/server/test_utils.go` and `internal/llm/test_helpers.go`. Extend these before adding new fixtures.
- For routing/model-selection changes, update tests in `internal/llm/*_test.go`, `cmd/server/*_test.go`, and `tests/integration_test.go` as appropriate.
- The Dockerfile runs `go test -v ./...` during image build — a failing test will break the Docker build.

## Helper scripts

| Script | Purpose |
| --- | --- |
| `scripts/start-stack.sh` | Starts DB + app via Docker Compose in correct order |
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

## License

Apache 2.0 with Commons Clause. See `LICENSE` for details. Repository: `github.com/Izzetee/PiPiMink`.

## What NOT to assume

- PiPiMink does not optimize for cost. Do not add cost-related routing logic unless explicitly asked.
- The model selection is intentionally done by an LLM (not rule-based). The meta-model reasons over capability tags, not hardcoded heuristics.
- `internal/api/` is legacy — new handler logic goes in `cmd/server/`.
- The `usage` token counts in OpenAI-compatible responses are rough approximations (`len / 4`), not real token counts.
