# Contributing to PiPiMink

Thank you for your interest in contributing! This document explains how to get started.

## Table of Contents

- [AI-Assisted Development](#ai-assisted-development)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Branch Conventions](#branch-conventions)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Code Style](#code-style)
- [Tests](#tests)
- [Swagger / OpenAPI Docs](#swagger--openapi-docs)

---

## AI-Assisted Development

This project was built with the help of AI coding tools (Claude Code, GitHub Copilot). Using AI assistants for contributions is welcome, but please keep these guidelines in mind:

- **Understand what you submit.** Review every line of AI-generated code before committing. You are responsible for the correctness, security, and maintainability of your contribution — not the AI.
- **Test thoroughly.** AI-generated code can look plausible while being subtly wrong. Run the full test suite (`go test ./...`) and verify edge cases manually.
- **Respect the architecture.** AI tools sometimes ignore existing patterns or invent unnecessary abstractions. Follow the conventions described in this guide and in `CLAUDE.md`.
- **Do not blindly paste.** If you do not understand why a piece of generated code works, do not include it. Ask questions in the PR instead.
- **Disclose when helpful.** You are not required to label every AI-assisted line, but if a PR contains substantial AI-generated logic, a brief note in the PR description helps reviewers calibrate their review.

---

## Getting Started

1. **Fork** the repository and clone your fork.
2. Create a **new branch** from `main` for your change (see [Branch Conventions](#branch-conventions)).
3. Make your changes, write or update tests, and ensure all checks pass locally.
4. Open a **Pull Request** against `main`.

---

## Development Setup

**Prerequisites:**

- Go 1.22+
- Docker & Docker Compose
- A PostgreSQL instance (or use the provided compose stack)

**Run the full stack locally:**

```bash
cp .env.example .env          # fill in your API keys
cp providers.example.json providers.json   # adjust providers
./scripts/start-stack.sh      # starts DB + app via Docker Compose
```

**Run only the database (for local Go development):**

```bash
docker compose -f docker-compose-db.yml up -d
go run main.go
```

**Trigger a model refresh:**

```bash
./scripts/update_models.sh
```

---

## Branch Conventions

| Prefix | Use case |
| ------ | -------- |
| `feat/` | New features |
| `fix/` | Bug fixes |
| `refactor/` | Code refactoring without behaviour changes |
| `docs/` | Documentation only |
| `chore/` | Build, CI, dependency updates |
| `test/` | Tests only |

Example: `feat/add-gemini-streaming`

---

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```text
<type>(<optional scope>): <short description>

[optional body]
```

Examples:

```text
feat(routing): add support for streaming responses
fix(cache): correct TTL expiry on cache miss
docs: update provider configuration examples
```

---

## Pull Requests

- Keep PRs focused — one feature or fix per PR.
- Fill in the PR template (description, test plan, checklist).
- All CI checks must pass before merging:
  - `Quality And Tests` (gofmt, go vet, go test)
  - `Markdown Lint`
  - `Secret Scan` (Gitleaks)
- A maintainer review is required before merge.

---

## Code Style

- Run `gofmt -w .` before committing — the CI enforces it.
- Follow the existing package structure. New handler logic goes in `cmd/server/`, not `internal/api/`.
- Keep changes consistent with the interface-driven design in `cmd/server/interfaces.go`.
- Do not hand-edit generated files in `docs/` — regenerate with `./scripts/generate-swagger.sh`.

---

## Tests

```bash
go test ./...           # all tests
go test -short ./...    # skip DB integration tests
go test -cover ./...    # with coverage
```

- Integration tests require a live PostgreSQL instance (use the compose stack).
- Unit tests use `sqlmock` — see existing patterns in `internal/database/`.
- Extend `cmd/server/test_utils.go` and `internal/llm/test_helpers.go` before adding new fixtures.
- The Docker build runs `go test -v ./...` — a failing test will block the image build.

---

## Swagger / OpenAPI Docs

If you add or change an API endpoint, regenerate the docs:

```bash
./scripts/generate-swagger.sh
```

Swagger annotations live in the handler files (`cmd/server/handlers.go`, etc.). Commit the regenerated `docs/` directory together with your handler changes.
