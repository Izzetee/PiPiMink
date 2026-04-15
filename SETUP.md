# Setup Guide

This guide walks you through configuring and running PiPiMink. For an overview of what PiPiMink does and how routing works, see the [README](README.md).

## Prerequisites

- Go 1.25+
- Docker and Docker Compose v2
- At least one LLM provider API key (OpenAI, Anthropic, Gemini, or a local server)

## Option A: Console Setup (recommended)

The fastest way to get started — no config files needed:

```bash
./scripts/start-stack.sh
```

Open `http://localhost:8080` and the setup wizard will walk you through:

1. Setting an admin API key to secure the instance
1. Adding your first LLM provider and API key
1. Discovering available models

All settings are stored in `.env` and `providers.json` automatically. You can change everything later in the Console UI under **Settings** and **Providers**.

## Option B: File-based Configuration

For CI/CD, Kubernetes, or if you prefer editing files directly, use the example templates:

```bash
cp providers.example.json providers.json   # edit with your provider URLs & env var names
cp .env.example .env                        # fill in API keys and admin key
```

Minimum `.env`:

```env
OPENAI_API_KEY=your_openai_api_key
ADMIN_API_KEY=your_admin_api_key
DATABASE_URL=postgres://user:password@pipimink-postgres:5432/mydatabase?sslmode=disable

MODEL_SELECTION_PROVIDER=openai   # provider used to make routing decisions
MODEL_SELECTION_MODEL=gpt-4-turbo # model within that provider
DEFAULT_CHAT_MODEL=gpt-4-turbo    # fallback if routing fails

BENCHMARK_JUDGE_PROVIDER=openai   # provider used to score LLM-judge benchmark tasks
BENCHMARK_JUDGE_MODEL=gpt-4o      # should be a capable chat model
PORT=8080

# OAuth / OIDC (optional — omit for API-key-only mode)
OAUTH_ISSUER_URL=http://localhost:9000/application/o/pipimink/
OAUTH_CLIENT_ID=pipimink-console
OAUTH_CLIENT_SECRET=
OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
OAUTH_AUTO_PROVISION=true
SESSION_SECRET=
```

## 2. Configure providers

Providers are configured in **`providers.json`** (copy from `providers.example.json`). Each entry declares the API type, base URL, env var holding the API key, timeout, rate limit, and a model list.

| Type | Examples |
| --- | --- |
| `openai-compatible` | OpenAI, Gemini, OpenRouter, LM Studio, any local server (Ollama, llama.cpp, MLX) |
| `anthropic` | Anthropic Claude (uses the native Messages API) |

Standard providers with auto-discovery or a simple static model list:

```json
[
  {"name":"openai",    "type":"openai-compatible","base_url":"https://api.openai.com",       "api_key_env":"OPENAI_API_KEY",    "timeout":"2m","models":[]},
  {"name":"anthropic", "type":"anthropic",        "base_url":"https://api.anthropic.com",    "api_key_env":"ANTHROPIC_API_KEY", "timeout":"2m","models":["claude-opus-4-6","claude-sonnet-4-6"]},
  {"name":"gemini",    "type":"openai-compatible","base_url":"https://generativelanguage.googleapis.com/v1beta/openai","api_key_env":"GEMINI_API_KEY","timeout":"2m","models":[]},
  {"name":"lm-studio", "type":"openai-compatible","base_url":"http://localhost:1234",         "api_key_env":"",                 "timeout":"5m","models":[]},
  {"name":"ollama",    "type":"openai-compatible","base_url":"http://localhost:11434",        "api_key_env":"",                 "timeout":"5m","rate_limit_seconds":2,"models":[]}
]
```

See `providers.example.json` for a full template.

### Microsoft Azure AI Foundry

Azure AI Foundry lets you host dozens or hundreds of models — including OpenAI, Anthropic, Mistral, Llama, Phi, Cohere, DeepSeek, and models from Hugging Face — under a single resource URL. The challenge is that **every model has its own API key** and models fall into one of three different endpoint patterns.

PiPiMink handles this with a single provider entry and a `model_configs` array: one element per model, each with its own key and path override. There is no limit on how many models you can add.

#### The three endpoint patterns

```mermaid
flowchart TD
    R[Your Foundry resource\nhttps://MY-RESOURCE.services.ai.azure.com]

    R -->|Serverless-API models\nMistral · Llama · Phi · Cohere · DeepSeek| P1["/models/chat/completions\n?api-version=2024-05-01-preview"]
    R -->|Anthropic models\nClaude family| P2["/anthropic/v1/messages"]
    R -->|Project-scoped OpenAI models\nGPT-4o · o-series| P3["/api/projects/MY-PROJECT\n/openai/v1/chat/completions"]
```

| Pattern | Models | `type` | Path in `chat_path` |
| --- | --- | --- | --- |
| **Serverless-API** | Mistral, Llama, Phi, Cohere, DeepSeek, … | `openai-compatible` | `/models/chat/completions?api-version=2024-05-01-preview` |
| **Anthropic** | Claude family | `anthropic` | *(not needed — set `base_url` to `…/anthropic`)* |
| **Project-scoped** | GPT-4o, o-series | `openai-compatible` | `/api/projects/YOUR-PROJECT/openai/v1/chat/completions` |

#### Configuration

**Step 1 — Add one `model_configs` entry per model in `providers.json`:**

```json
{
  "name": "az-foundry",
  "type": "openai-compatible",
  "base_url": "https://MY-RESOURCE.services.ai.azure.com",
  "timeout": "3m",
  "api_key_env": "",
  "model_configs": [
    {
      "name": "Mistral-large-3",
      "chat_path": "/models/chat/completions?api-version=2024-05-01-preview",
      "api_key_env": "AZURE_FOUNDRY_MISTRAL_LARGE_API_KEY"
    },
    {
      "name": "Phi-4",
      "chat_path": "/models/chat/completions?api-version=2024-05-01-preview",
      "api_key_env": "AZURE_FOUNDRY_PHI4_API_KEY"
    },
    {
      "name": "Llama-3.3-70B-Instruct",
      "chat_path": "/models/chat/completions?api-version=2024-05-01-preview",
      "api_key_env": "AZURE_FOUNDRY_LLAMA_API_KEY"
    },
    {
      "name": "claude-opus-4-5",
      "type": "anthropic",
      "base_url": "https://MY-RESOURCE.services.ai.azure.com/anthropic",
      "api_key_env": "AZURE_FOUNDRY_CLAUDE_OPUS_API_KEY"
    },
    {
      "name": "gpt-4o",
      "chat_path": "/api/projects/MY-PROJECT/openai/v1/chat/completions",
      "api_key_env": "AZURE_FOUNDRY_GPT4O_API_KEY"
    }
  ]
}
```

The `base_url` and `type` at the top level are the defaults. Each entry in `model_configs` only needs to set the fields that differ from those defaults. To add another model, append one more object to the array and add one line to `.env`.

**Step 2 — Add one env var per model in `.env`:**

```env
AZURE_FOUNDRY_MISTRAL_LARGE_API_KEY=5JhyBf1q...
AZURE_FOUNDRY_PHI4_API_KEY=xK9mNpLw...
AZURE_FOUNDRY_LLAMA_API_KEY=tR3vQsYz...
AZURE_FOUNDRY_CLAUDE_OPUS_API_KEY=aB7cDeFg...
AZURE_FOUNDRY_GPT4O_API_KEY=hI2jKlMn...
```

**Step 3 — Discover and tag as usual:**

Open the console at `/console/models`, click **Discover Models** — PiPiMink reads the `model_configs` list and registers all models. Then select them and click **Tag Selected** to run the capability interview. Each model call automatically uses its own API key and endpoint path.

#### Finding your endpoint URLs and API keys

In the [Azure AI Foundry portal](https://ai.azure.com):

- **Serverless-API models** (Mistral, Llama, Phi, etc.): go to **Models + endpoints**, select the deployment — the endpoint URL and key are shown on the detail page.
- **Anthropic models**: same location; the URL ends with `/anthropic/v1/messages`. Use everything before `/v1/messages` as `base_url` in the model config.
- **Project-scoped models** (GPT, o-series): go to **My assets → Models + endpoints** inside your project — copy the endpoint. The path starts with `/api/projects/YOUR-PROJECT/openai/v1/`.

## 3. Start the stack

```bash
./scripts/start-stack.sh                    # DB + PiPiMink
./scripts/start-stack.sh --with-authentik   # also start Authentik for OAuth
```

Service URLs:

- PiPiMink Console: `http://localhost:8080/console/`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- pgAdmin: `http://localhost:5050`
- Authentik (if started): `http://localhost:9000`

## 4. Set up your model registry

Open `http://localhost:8080/console/models` and:

1. Click **Discover Models** — finds all models across your configured providers
2. Select the models you want to use, click **Tag Selected** — runs the capability interview
3. Optionally click **Benchmark Selected** — measures actual performance on your tasks

Models are now ready for routing.

## Console UI

PiPiMink ships with a React console at `/console/` covering models, providers, config, settings, analytics, and user management.

### Model lifecycle (`/console/models`)

The full model lifecycle from provider discovery to routing-ready runs in three sequential steps:

```mermaid
flowchart TD
    A([Start]) --> B[Discover Models\n/models/discover]
    B --> C{Models found?}
    C -- No --> D[Check providers.json\nand API keys]
    D --> B
    C -- Yes --> E[Select models to tag\ncheck Tag checkboxes]
    E --> F[Tag Selected\n/models/tag]
    F --> G[Each selected model\nanswers capability interview]
    G --> H{Tags returned?}
    H -- Empty / no strengths --> I[Model disabled automatically\nshown as 'disabled' badge]
    H -- Valid tags --> J[Model enabled\nshown as 'tagged' badge]
    J --> K[Optionally: select models\ncheck Benchmark checkboxes]
    K --> L[Benchmark Selected\n/models/benchmark]
    L --> M[Judge model scores each task\nresults stored per model]
    M --> N([Models ready for routing])
    I --> N
```

**Step-by-step:**

1. **Sign in** — if OAuth is configured, click "Sign in with Authentik". Otherwise, enter your Admin API Key (matches `ADMIN_API_KEY` in your `.env`).
2. **Click Discover Models** — queries every configured provider for its model list. This is instant and makes no LLM calls. Newly found models appear with a yellow `discovered` badge.
3. **Select models for tagging** — use the `Tag` checkboxes in each row, or "Select all (Tag)" to pick all at once. Ignore models you don't want to route to.
4. **Click Tag Selected** — sends the capability interview to each model in the background. Reload the page after a moment to see results. Models that return valid capability tags get a green `tagged` badge and are enabled for routing. Models that return no strengths are automatically disabled.
5. *(Optional)* **Select models for benchmarking** — use the `Benchmark` checkboxes, then click **Benchmark Selected**. This runs your benchmark tasks against each model and stores scores. Scores appear as coloured pills per category in the table.
6. Use the **On/Off toggle** in any row to manually enable or disable a model at any time without re-tagging.
7. *(Optional)* **Reset a model** — clears all tags, benchmark results, and statistics while keeping the model entry. Useful for re-evaluating a model from scratch.
8. *(Optional)* **Delete a model** — fully removes the model and all associated data. If the model is rediscovered later, it starts fresh with no history.

> Discovery, tagging, and benchmarking are fully decoupled — you can run any step independently and at your own pace.

### Configuration (`/console/config`)

This page controls the two inputs that shape how routing is personalized for you.

```mermaid
flowchart LR
    subgraph Tagging prompts
        TP1[System prompt]
        TP2[User prompt\nwith system role]
        TP3[User prompt\nno system role]
    end
    subgraph Benchmark tasks
        BT1[Builtin tasks\nedit / disable / reset]
        BT2[Custom tasks\nadd new]
    end
    TP1 & TP2 & TP3 -->|saved to DB| DB[(PostgreSQL)]
    BT1 & BT2 -->|saved to DB| DB
    DB -->|loaded before each run| TAG[Tagging run\nPOST /models/tag]
    DB -->|loaded before each run| BENCH[Benchmark run\nPOST /models/benchmark]
    TAG --> CAPS[Capability tags\nper model]
    BENCH --> SCORES[Benchmark scores\nper model]
    CAPS & SCORES -->|routing inputs| ROUTER[Meta-model routing decision]
```

#### Editing tagging prompts

The three prompts define exactly what each model is asked during the capability interview:

| Prompt | When it is used |
| --- | --- |
| **System Prompt** | Sent as the system message for providers that support a system role |
| **User Prompt (with system)** | Sent as the user turn when a system message was also sent |
| **User Prompt (no system)** | Sent as the sole message for models/providers that don't support a system role |

To change a prompt: edit the text area and click **Save** next to it. Changes take effect on the next tagging run — no restart needed.

> Tip: if models are returning unhelpful or too-generic tags, try making the prompts more specific about what capability dimensions matter to you (e.g. add "focus on: code-review, german-language, data-analysis").

#### Managing benchmark tasks

Each task row shows its ID, category, scoring method, and whether it is enabled.

| Action | How |
| --- | --- |
| **Edit a task** | Click **Edit** — change the prompt, expected answer, or judge criteria, then Save |
| **Disable a task** | Click **Edit**, uncheck Enabled, Save — the task is skipped on the next run |
| **Reset a builtin task** | Click **Reset** — reverts prompt and criteria to the compiled-in defaults |
| **Delete a custom task** | Click **Delete** — permanently removes the task |
| **Add a custom task** | Click **+ New Task** — fill in ID, category, prompt, and scoring method |

Scoring methods:

- **`deterministic`** — the model response must contain the expected answer string (case-insensitive). Good for math and factual questions.
- **`llm-judge`** — the judge model scores each named criterion 0–10 independently; final score is the average. Good for code quality, writing, and summarization.
- **`format`** — a built-in structural validator checks the response (e.g. exact word count, valid JSON). Only available for builtin tasks — custom tasks cannot use this method.

## Configuration Reference

### Routing cache

| Variable | Default | Description |
| --- | --- | --- |
| `SELECTION_CACHE_ENABLED` | `true` | Enable/disable the cache |
| `SELECTION_CACHE_TTL` | `2m` | How long a cached decision is valid |
| `SELECTION_CACHE_MAX_ENTRIES` | `1000` | Maximum entries before LRU eviction |
| `SELECTION_CACHE_STATS_LOG_INTERVAL` | `1m` | How often to log hit/miss/eviction summary |

### Benchmarking

| Variable | Default | Description |
| --- | --- | --- |
| `BENCHMARK_ENABLED` | `false` | Enable benchmark endpoints |
| `BENCHMARK_JUDGE_PROVIDER` | (selection provider) | Provider for LLM judge |
| `BENCHMARK_JUDGE_MODEL` | (selection model) | Model used to score subjective tasks — use a capable chat model |
| `BENCHMARK_CONCURRENCY` | `3` | Max models benchmarked in parallel |
| `BENCHMARK_SCHEDULE_ENABLED` | `false` | Run benchmarks automatically on a schedule |
| `BENCHMARK_SCHEDULE_INTERVAL` | `24h` | Interval between scheduled benchmark runs |

## Observability

- Prometheus metrics: `GET /metrics`
- OpenTelemetry tracing: OTLP export via `OTEL_EXPORTER_OTLP_ENDPOINT`

For Grafana stack integration:

- Tempo: configure an OTLP receiver (port `4318` for HTTP) and set `OTEL_ENABLED=true`
- Mimir/Prometheus: scrape `http://<pipimink-host>:8080/metrics`
- Loki: forward container stdout/stderr with your log agent

## Authentication

PiPiMink supports two authentication modes:

- **OAuth2/OIDC via Authentik** — recommended for production and multi-user environments
- **API-key-only** — legacy mode when no OAuth provider is configured; all requests pass through

### Setting up Authentik

1. Start the stack with Authentik: `./scripts/start-stack.sh --with-authentik` (uses Authentik 2026.2)
1. Open `http://localhost:9000/if/flow/initial-setup/` and create an Authentik admin account
1. In the Authentik admin panel, create an **OAuth2/OpenID Provider** for PiPiMink:
   - **Redirect URI**: `http://localhost:8080/auth/callback`
   - **Scopes**: `openid profile email groups`
1. Create an **Application** linked to that provider
1. Copy the **Client ID** and **Client Secret** into your `.env`:

```env
OAUTH_ISSUER_URL=http://localhost:9000/application/o/pipimink/
OAUTH_CLIENT_ID=<your-client-id>
OAUTH_CLIENT_SECRET=<your-client-secret>
OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
OAUTH_AUTO_PROVISION=true
SESSION_SECRET=<128-hex-char-string>   # generate with: openssl rand -hex 64
```

1. Restart PiPiMink — visiting `/console/` will redirect to Authentik for login

> **Note:** PiPiMink retries OIDC discovery up to 6 times (5-second intervals) in the background at startup. If Authentik takes longer than usual to start, OAuth will connect automatically as soon as it becomes available — no manual restart needed.

### OAuth environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `OAUTH_ISSUER_URL` | *(empty)* | OIDC issuer URL (e.g. Authentik application URL) |
| `OAUTH_CLIENT_ID` | *(empty)* | OAuth2 client ID |
| `OAUTH_CLIENT_SECRET` | *(empty)* | OAuth2 client secret |
| `OAUTH_REDIRECT_URL` | *(empty)* | Callback URL (`http://host:port/auth/callback`) |
| `OAUTH_SCOPES` | `openid profile email groups` | Space-separated OIDC scopes |
| `OAUTH_AUTO_PROVISION` | `true` | Auto-create users on first OAuth login |
| `SESSION_SECRET` | *(random)* | 64-byte hex key for session cookie encryption |

OAuth is enabled when `OAUTH_ISSUER_URL`, `OAUTH_CLIENT_ID`, and `OAUTH_CLIENT_SECRET` are all set. Otherwise PiPiMink runs in API-key-only mode.

### User and group management

Navigate to `/console/users` to manage:

- **Auth Providers** — view and configure OAuth and LDAP provider settings, test connectivity
- **Users** — view all users, change roles (admin/user), add local users, delete users (GDPR-compliant with mandatory reason)
- **Groups** — synced from the identity provider; assign roles and configure per-group routing rules (allow/deny specific providers or models)
- **Audit Log** — chronological record of all auth-related actions

## Securing API Access

### Requiring authentication for chat endpoints

By default, chat and inference endpoints (`/chat`, `/v1/chat/completions`, `/api/chat`) accept requests without authentication, so existing clients work without any changes. To enforce authentication for these endpoints:

```env
REQUIRE_AUTH_FOR_CHAT=true
```

When this is set, every chat request must include either a valid session cookie (browser/OAuth flow) or a Bearer token. Unauthenticated requests receive a `401 Unauthorized` response.

This setting has no effect on admin endpoints, which always require either an `X-API-Key` header or an admin session.

### Creating Bearer tokens for programmatic access

Bearer tokens let scripts and API clients authenticate without going through the OAuth browser flow. They carry the `ppm_` prefix so they are easy to identify in logs and secret scanners.

**Create a token** (requires an active session or an existing Bearer token with admin rights):

```bash
curl -X POST http://localhost:8080/auth/tokens \
  -H "Authorization: Bearer ppm_your-existing-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline", "expires_in": "8760h"}'
```

The plaintext token is returned **once** at creation and never stored — copy it immediately.

**Use the token in requests:**

```bash
# OpenAI-compatible endpoint
curl -H "Authorization: Bearer ppm_your-token" \
  http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'

# Admin endpoint — X-API-Key still works too
curl -H "X-API-Key: your-admin-api-key" \
  http://localhost:8080/models
```

### Token management

Navigate to `/console/users` → **API Keys** to view, rotate, and revoke tokens through the UI. Revoked tokens are rejected immediately.

---

## Local Development

Build and run:

```bash
go build -o pipimink
./pipimink
```

Testing:

```bash
go test ./...
go test -short ./...   # skip integration tests (no DB required)
go test -cover ./...
```

Frontend testing:

```bash
cd web/console
npm test              # single run
npm run test:watch    # watch mode
```

## Helper Scripts

| Script | Purpose |
| --- | --- |
| `scripts/start-stack.sh` | Starts the database and application; use `--with-authentik` to also start the Authentik identity provider |
| `scripts/generate-swagger.sh` | Regenerates OpenAPI docs after API changes |
| `scripts/test_chat_request.sh` | Quick end-to-end smoke test |
| `scripts/cleanup.sh` | Local maintenance and lint helper |
| `scripts/release-check.sh` | Pre-release validation (formatting, tests, secret scan) |
