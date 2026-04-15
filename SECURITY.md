# Security Policy

## Supported Versions

PiPiMink is currently in active development. Security fixes are applied to the latest version on the `main` branch.

| Version | Supported |
| ------- | --------- |
| latest (`main`) | ✅ |
| older commits | ❌ |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability, please report it responsibly:

1. **Open a [GitHub Security Advisory](https://github.com/Izzetee/PiPiMink/security/advisories/new)** — this keeps the report private until a fix is released.
2. Alternatively, send an e-mail to the maintainer listed in the repository profile with the subject line `[PiPiMink] Security Vulnerability`.

Please include as much of the following as possible:

- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept
- Affected component (e.g. routing logic, admin API, Docker setup)
- Any suggested fix or mitigation

You can expect an acknowledgement within **72 hours** and a status update within **7 days**.

## Security Considerations for Operators

### API Keys

- PiPiMink reads API keys exclusively from **environment variables** (via `.env` or `docker run -e`).
- Never commit your `.env` file or a `providers.json` with real credentials.
- Use the `.env.example` and `providers.example.json` templates as starting points.

### Authentication

PiPiMink enforces authentication at three levels:

| Level | Who | How |
| ----- | --- | --- |
| **Public** | Anyone | `GET /admin/status`, `GET /auth/login`, `GET /auth/callback` — no credentials required |
| **User** | Authenticated users | Valid session cookie or `Authorization: Bearer ppm_<token>` header |
| **Admin** | Admin users / scripts | `X-API-Key: <ADMIN_API_KEY>` header, or an admin session/Bearer token |

Auth is enforced centrally by `cmd/server/auth_middleware.go` — individual handlers do not perform their own auth checks.

PiPiMink supports three authentication modes:

- **OAuth2/OIDC via Authentik** (recommended for production) — users authenticate through Authentik; sessions are managed with encrypted cookies.
- **Bearer tokens** — programmatic API access without a browser flow. Tokens use the `ppm_` prefix and are stored as SHA-256 hashes — the plaintext is shown only once at creation and is never stored. Rotate tokens regularly and revoke any that are no longer needed.
- **X-API-Key header** — backward-compatible fallback for admin endpoints and scripts. Set a strong, randomly generated value for `ADMIN_API_KEY`.
- **Passthrough** — when no OAuth provider is configured and no API key is sent, all requests pass through. This legacy mode is **not suitable for internet-facing deployments**.

**Recommended production settings:**

```env
REQUIRE_AUTH_FOR_CHAT=true   # require auth for /chat, /v1/chat/completions, /api/chat
ADMIN_API_KEY=<strong-random-value>
SESSION_SECRET=<64-byte-hex-string>
```

The default fallback in `scripts/update_models.sh` (`admin-key-12345`) is for **local development only** — never use it in a networked or production environment.

### Session Security

- Set `SESSION_SECRET` to a 64-byte hex string in production. If unset, a random key is generated at startup and sessions will not survive a restart.
- Session cookies are `HttpOnly` and `SameSite=Lax`.
- In production, place PiPiMink behind a TLS-terminating reverse proxy so that session cookies are transmitted securely.
- OIDC discovery retries up to 6 times at startup (5-second intervals), so PiPiMink tolerates slow-starting identity providers. Non-OAuth routes remain fully functional during the retry window.

### Docker / Database Credentials

- The `docker-compose.yml` files ship with example credentials (`user:password`, `admin:admin`) intended for local development only.
- The `docker-compose-authentik.yml` ships with default credentials (`AUTHENTIK_SECRET_KEY`, PostgreSQL password) that must be changed for any networked deployment.
- For any internet-facing deployment, replace all database, pgAdmin, and Authentik credentials with strong, unique values — ideally passed via environment variables or Docker secrets.

### Network Exposure

- By default PiPiMink listens on `0.0.0.0:8080`. In production, place it behind a reverse proxy (nginx, Caddy, Traefik) with TLS.
- The pgAdmin interface (port 5050) and Authentik admin interface (port 9000) should never be exposed to the public internet.

### Secret Scanning

This repository has automated secret scanning via [Gitleaks](https://github.com/gitleaks/gitleaks) on every push and pull request. If a scan blocks your PR, ensure no credentials are embedded in the diff.
