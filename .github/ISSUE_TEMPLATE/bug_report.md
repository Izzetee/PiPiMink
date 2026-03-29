---
name: Bug Report
about: Report a reproducible bug or unexpected behaviour
title: "[Bug] "
labels: bug
assignees: ""
---

## Describe the Bug

A clear and concise description of what the bug is.

## Steps to Reproduce

1. Configure providers with ...
2. Send request to `POST /...` with body `...`
3. Observe ...

## Expected Behaviour

What you expected to happen.

## Actual Behaviour

What actually happened. Include error messages, stack traces, or log output.

```text
paste logs here
```

## Environment

| Field | Value |
| ----- | ----- |
| PiPiMink version / commit | `git rev-parse --short HEAD` |
| Go version | `go version` |
| OS | e.g. Ubuntu 24.04 / macOS 15 / Windows 11 |
| Deployment | Docker Compose / bare Go binary / other |
| Providers involved | e.g. OpenAI, Anthropic, local Ollama |

## Additional Context

Add any other context, screenshots, or request/response payloads that may help diagnose the issue. Redact API keys and personal data before posting.
