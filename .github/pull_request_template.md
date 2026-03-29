# Pull Request

## Description

<!-- What does this PR do? Link any related issues with "Closes #123" or "Fixes #123". -->

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Refactor (no behaviour change)
- [ ] Documentation
- [ ] Dependency update
- [ ] CI / tooling

## How Was This Tested?

<!-- Describe how you verified the change works. Include unit tests added/updated, manual steps, or test commands run. -->

```bash
go test ./...
```

## Checklist

- [ ] `gofmt -w .` applied — code passes `gofmt` check
- [ ] `go vet ./...` passes with no warnings
- [ ] All existing tests pass (`go test ./...`)
- [ ] New tests added for new behaviour
- [ ] No secrets, credentials, or internal hostnames are included in the diff
- [ ] Swagger docs regenerated (`./scripts/generate-swagger.sh`) if API surface changed
- [ ] README or CHANGELOG updated if relevant
