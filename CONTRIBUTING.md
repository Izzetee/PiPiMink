# Contributing to PiPiMink

Thank you for your interest in contributing! This document explains how to get started.

## Table of Contents

- [License](#license)
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

## License

By intentionally submitting a contribution for inclusion in PiPiMink — whether via pull request, patch, or any other mechanism — you agree that your contribution will be licensed under the [Apache License 2.0](LICENSE), the same license that covers the project, unless you explicitly state otherwise.

You retain copyright to your own contributions. No separate Contributor License Agreement (CLA) is required.

> **Optional note for future consideration:** The project may adopt [Developer Certificate of Origin (DCO)](https://developercertificate.org/) sign-off (`git commit -s`) in the future to formalize contribution provenance. This is not currently required.

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

- Go 1.25+
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

**Run the frontend dev server (React console UI):**

```bash
cd web/console
npm install
npm run dev    # starts Vite on port 5173 with hot reload
```

The dev server proxies API requests to `http://localhost:8080`, so the Go backend must be running.

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
  - `Frontend` (tsc --noEmit, vitest run)
  - `Markdown Lint`
  - `Secret Scan` (Gitleaks)
  - `Go Lint` (golangci-lint)
  - `Govulncheck`
  - `CodeQL`
- A maintainer review is required before merge.

---

## Code Style

- Run `gofmt -w .` before committing — the CI enforces it.
- Follow the existing package structure. New handler logic goes in `cmd/server/`, not `internal/api/`.
- Keep changes consistent with the interface-driven design in `cmd/server/interfaces.go`.
- **Authentication must not be checked inline inside handlers.** Auth is enforced centrally by `cmd/server/auth_middleware.go`. New handlers must rely on the middleware — never add `if !isAdmin` or token validation logic inside a handler function.
- Do not hand-edit generated files in `docs/` — regenerate with `./scripts/generate-swagger.sh`.
- Frontend code lives in `web/console/`. It uses TypeScript strict mode, Tailwind CSS v4, and React 19. Run `npm run build` in `web/console/` to verify the frontend compiles before submitting.

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
- Frontend:

  ```bash
  cd web/console
  npm test              # single run (vitest run)
  npm run test:watch    # watch mode
  npx tsc --noEmit      # type check only
  ```

  Tests use Vitest + React Testing Library (jsdom). Test files live next to their source (`*.test.ts` / `*.test.tsx`).

---

## Swagger / OpenAPI Docs

If you add or change an API endpoint, regenerate the docs:

```bash
./scripts/generate-swagger.sh
```

Swagger annotations live in the handler files (`cmd/server/handlers.go`, etc.). Commit the regenerated `docs/` directory together with your handler changes.
