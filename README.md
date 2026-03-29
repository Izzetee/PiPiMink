<p align="center">
  <img src="assets/pipimink_cropped_1.png" alt="PiPiMink" width="400">
</p>

<p align="center">
  <strong>Route every prompt to the best model — for you specifically.</strong>
</p>

<p align="center">
  <a href="https://github.com/Izzetee/PiPiMink/actions/workflows/ci.yml"><img src="https://github.com/Izzetee/PiPiMink/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/Izzetee/PiPiMink/actions/workflows/security.yml"><img src="https://github.com/Izzetee/PiPiMink/actions/workflows/security.yml/badge.svg" alt="Security"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0%20%2B%20Commons%20Clause-blue" alt="License"></a>
</p>

PiPiMink is a Go service that routes each prompt to the LLM most likely to produce the best response — **for you specifically**.

## The Core Idea

Most AI routers use generic, one-size-fits-all benchmarks (MMLU, HumanEval, etc.) that measure average performance across millions of anonymous prompts. PiPiMink takes the opposite approach: **it learns which models work best for your actual use cases**, based on benchmarks you define and results you observe yourself.

The routing is intentionally subjective. A model that scores 90% on a generic coding benchmark may still give worse answers than a smaller local model for *your* specific coding style, domain, or workflow. PiPiMink lets you measure exactly that.

**There is no global leaderboard that defines what "best" means. You do.**

## How It Works

### Request flow

Every chat request goes through two steps: a routing decision, then the actual model call.

```mermaid
sequenceDiagram
    participant Client
    participant PiPiMink
    participant Cache
    participant MetaModel as Meta-model<br/>(routing LLM)
    participant TargetModel as Target model<br/>(selected LLM)

    Client->>PiPiMink: POST /v1/chat/completions<br/>{ "messages": [...] }

    PiPiMink->>Cache: lookup(hash(prompt + model_capabilities))
    alt cache hit
        Cache-->>PiPiMink: cached model name
    else cache miss
        PiPiMink->>MetaModel: prompt + capability tags<br/>+ benchmark scores of all enabled models
        MetaModel-->>PiPiMink: { "modelname": "gpt-4o",<br/>  "reason": "...",<br/>  "matching_tags": [...] }
        PiPiMink->>Cache: store(decision)
    end

    PiPiMink->>TargetModel: original messages
    TargetModel-->>PiPiMink: response
    PiPiMink-->>Client: response
```

The routing priority inside the meta-model call is:

1. **Capability tags** — primary signal (what the model says it excels at)
2. **Benchmark scores** — secondary signal (how the model actually performed on *your* tasks)
3. **Average response time** — tiebreaker only

### 1. Self-reported capability tags

During a model refresh, PiPiMink asks every available model to assess its own strengths and weaknesses. Each model replies with a structured JSON tag list:

```json
{
  "strengths": ["code-generation", "step-by-step-reasoning", "multilingual"],
  "weaknesses": ["real-time-information", "image-generation"]
}
```

These tags are stored in PostgreSQL alongside each model's metadata. The exact prompts used for this interview are editable in the admin config page — you can steer which capability dimensions get reported.

### 2. Prompt-driven routing

When a chat request arrives, PiPiMink sends the user's prompt — along with the capability tags and benchmark scores of all enabled models — to a configurable meta-model. The meta-model returns a structured routing decision:

```json
{
  "modelname": "gpt-4o",
  "reason": "The request requires deep reasoning and code review",
  "matching_tags": ["code-generation", "step-by-step-reasoning"],
  "tag_relevance": { "code-generation": 9, "step-by-step-reasoning": 8 }
}
```

PiPiMink then forwards the original prompt to the selected model and returns its response.

### 3. Your benchmarks, your signal

The built-in benchmark suite evaluates models across coding, reasoning, instruction-following, creative writing, summarization, and factual QA. Scores are measured by an LLM judge you configure — meaning the evaluation reflects your standards, not an industry average.

Because benchmark results feed directly into the routing decision, the more you benchmark with tasks relevant to your workflow, the better the routing gets — for you.

### 4. Routing decision cache

Routing decisions are cached in memory using a hash of the normalized prompt and the current capability snapshot. Cache entries expire by TTL and are evicted by LRU when the size limit is reached, so the router stays fast for repeated or similar prompts.

## Supported Providers

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

See `providers.example.json` for a full template. Azure AI Foundry is documented in detail below.

---

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

Open `/admin`, click **Discover Models** — PiPiMink reads the `model_configs` list and registers all models. Then select them and click **Tag Selected** to run the capability interview. Each model call automatically uses its own API key and endpoint path.

#### Finding your endpoint URLs and API keys

In the [Azure AI Foundry portal](https://ai.azure.com):

- **Serverless-API models** (Mistral, Llama, Phi, etc.): go to **Models + endpoints**, select the deployment → the endpoint URL and key are shown on the detail page.
- **Anthropic models**: same location; the URL ends with `/anthropic/v1/messages`. Use everything before `/v1/messages` as `base_url` in the model config.
- **Project-scoped models** (GPT, o-series): go to **My assets → Models + endpoints** inside your project → copy the endpoint. The path starts with `/api/projects/YOUR-PROJECT/openai/v1/`.

---

## Admin UI

PiPiMink ships with two admin pages, both under `/admin`.

### Model lifecycle (`/admin`)

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

1. **Enter your Admin API Key** in the field at the top (matches `ADMIN_API_KEY` in your `.env`).
2. **Click Discover Models** — queries every configured provider for its model list. This is instant and makes no LLM calls. Newly found models appear with a yellow `discovered` badge.
3. **Select models for tagging** — use the `Tag` checkboxes in each row, or "Select all (Tag)" to pick all at once. Ignore models you don't want to route to.
4. **Click Tag Selected** — sends the capability interview to each model in the background. Reload the page after a moment to see results. Models that return valid capability tags get a green `tagged` badge and are enabled for routing. Models that return no strengths are automatically disabled.
5. *(Optional)* **Select models for benchmarking** — use the `Benchmark` checkboxes, then click **Benchmark Selected**. This runs your benchmark tasks against each model and stores scores. Scores appear as coloured pills per category in the table.
6. Use the **On/Off toggle** in any row to manually enable or disable a model at any time without re-tagging.

> Discovery, tagging, and benchmarking are fully decoupled — you can run any step independently and at your own pace.

### Configuration (`/admin/config`)

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

## Exposed APIs

PiPiMink is a drop-in proxy. Existing clients require no changes:

| Endpoint | Description |
| --- | --- |
| `POST /chat` | Native PiPiMink — always auto-routes |
| `GET /models` | List all models with tags, scores, and latency |
| `POST /v1/chat/completions` | OpenAI-compatible (streaming supported) |
| `GET /v1/models` | OpenAI-compatible model list |
| `POST /api/chat`, `GET /api/tags` | Ollama-compatible |
| `GET /admin` | Model management UI |
| `GET /admin/config` | Benchmark task + tagging prompt editor |
| `GET /metrics` | Prometheus/OpenMetrics |

Clients like **Open WebUI** can connect directly using either the OpenAI-compatible or Ollama-compatible endpoint. PiPiMink will appear as a single model and route each request internally.

## Quick Start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose v2
- At least one configured provider API key

### 1. Configure providers and environment

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
```

### 2. Start stack

```bash
./scripts/start-stack.sh
```

Service URLs:

- PiPiMink API + Admin UI: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- pgAdmin: `http://localhost:5050`

### 3. Set up your model registry

Open `http://localhost:8080/admin` and:

1. Click **Discover Models** — finds all models across your configured providers
2. Select the models you want to use, click **Tag Selected** — runs the capability interview
3. Optionally click **Benchmark Selected** — measures actual performance on your tasks

Models are now ready for routing.

## Benchmarks

The benchmark suite is the mechanism through which routing becomes personalized. Categories include:

| Category | Scoring |
| --- | --- |
| `coding` | LLM judge — multi-criteria (correctness, efficiency, clarity, edge cases) |
| `reasoning` | Deterministic — exact numeric answer |
| `instruction-following` | Format validator — structural checks |
| `creative-writing` | LLM judge — multi-criteria (imagery, originality, structure, tone) |
| `summarization` | LLM judge — multi-criteria (coverage, accuracy, conciseness, format) |
| `factual-qa` | Deterministic — substring match |

LLM-judge tasks use a configurable judge model (`BENCHMARK_JUDGE_PROVIDER` / `BENCHMARK_JUDGE_MODEL`). Each criterion is scored independently on a 0–10 scale; the final score is the average across all criteria. This gives fine-grained, continuous scores rather than binary pass/fail.

Benchmark scores feed directly into routing decisions as the secondary signal after capability tags, pushing traffic toward models that empirically perform better on the types of tasks you actually run.

## Observability

- Prometheus metrics: `GET /metrics`
- OpenTelemetry tracing: OTLP export via `OTEL_EXPORTER_OTLP_ENDPOINT`

For Grafana stack integration:

- Tempo: configure an OTLP receiver (port `4318` for HTTP) and set `OTEL_ENABLED=true`
- Mimir/Prometheus: scrape `http://<pipimink-host>:8080/metrics`
- Loki: forward container stdout/stderr with your log agent

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

## Architecture

| Path | Purpose |
| --- | --- |
| `main.go` | Entry point |
| `cmd/server/` | HTTP server, handlers, routing logic, admin UI |
| `internal/llm/` | Provider clients, capability tagging, model selection, routing cache |
| `internal/benchmark/` | Benchmark task definitions, runner, scorer |
| `internal/database/` | PostgreSQL persistence and schema migration |
| `internal/models/` | Domain types |
| `docs/` | Generated OpenAPI / Swagger artifacts |

## Helper Scripts

| Script | Purpose |
| --- | --- |
| `scripts/start-stack.sh` | Starts the database and application in the correct order |
| `scripts/generate-swagger.sh` | Regenerates OpenAPI docs after API changes |
| `scripts/test_chat_request.sh` | Quick end-to-end smoke test |
| `scripts/cleanup.sh` | Local maintenance and lint helper |
| `scripts/release-check.sh` | Pre-release validation (formatting, tests, secret scan) |

## AI-Assisted Development

This project was developed with the assistance of AI coding tools. Contributions that use AI are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md#ai-assisted-development) for guidelines.

## License

Apache 2.0 with [Commons Clause](LICENSE). Free to use, modify, and self-host. Commercial reselling requires prior permission — see the [LICENSE](LICENSE) for details.
