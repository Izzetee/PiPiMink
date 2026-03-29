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

### Admin API Key (`ADMIN_API_KEY`)

- The `POST /models/update` endpoint is protected by the `X-API-Key` header.
- Set a strong, randomly generated value for `ADMIN_API_KEY` in production.
- The default fallback in `scripts/update_models.sh` (`admin-key-12345`) is for **local development only** — never use it in a networked or production environment.

### Docker / Database Credentials

- The `docker-compose.yml` files ship with example credentials (`user:password`, `admin:admin`) intended for local development only.
- For any internet-facing deployment, replace all database and pgAdmin credentials with strong, unique values — ideally passed via environment variables or Docker secrets.

### Network Exposure

- By default PiPiMink listens on `0.0.0.0:8080`. In production, place it behind a reverse proxy (nginx, Caddy, Traefik) with TLS.
- The pgAdmin interface (port 5050) should never be exposed to the public internet.

### Secret Scanning

This repository has automated secret scanning via [Gitleaks](https://github.com/gitleaks/gitleaks) on every push and pull request. If a scan blocks your PR, ensure no credentials are embedded in the diff.
