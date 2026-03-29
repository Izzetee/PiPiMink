# Branch Protection Recommendations

Use these settings on your default branch (`main` or `master`) to align with this repository's CI guardrails.

## Required status checks

Require all of the following checks to pass before merging:

- `Quality And Tests`
- `Markdown Lint`
- `Secret Scan`
- `Go Lint`
- `Go Vulnerability Check`
- `CodeQL Analysis`

## Pull request protections

Recommended settings:

- Require a pull request before merging
- Require approvals: at least 1
- Dismiss stale pull request approvals when new commits are pushed
- Require conversation resolution before merging
- Require branches to be up to date before merging

## Merge strategy

- Enable **Squash merging** only (disable merge commits and rebase merging)
- Enable **Automatically delete head branches** after merge

## Additional protections

Recommended settings:

- Restrict who can push to matching branches
- Do not allow force pushes
- Do not allow deletions

## Code security settings

Enable under Settings > Code security and analysis:

- **Dependency graph** (default on for public repos)
- **Dependabot alerts** — CVE notifications for vulnerable dependencies
- **Dependabot security updates** — auto-PRs to fix vulnerable dependencies
- **Secret scanning** — detects leaked credentials in commits
- **Push protection** — blocks pushes containing secrets before they enter the repo

## Why these map correctly

These check names correspond exactly to the configured GitHub Actions jobs in:

- `.github/workflows/ci.yml` — `Quality And Tests`, `Markdown Lint`
- `.github/workflows/security.yml` — `Secret Scan`
- `.github/workflows/lint.yml` — `Go Lint`
- `.github/workflows/govulncheck.yml` — `Go Vulnerability Check`
- `.github/workflows/codeql.yml` — `CodeQL Analysis`
