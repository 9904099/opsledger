# Open Source Checklist

Use this checklist before pushing OpsLedger to a public repository.

## Required Checks

- [ ] `git status --ignored --short` shows `data/`, `releases/`, local memory, host, env, and deployment logs as ignored or absent.
- [ ] `git grep -n "AKIA\\|ASIA\\|BEGIN .*PRIVATE\\|secret_access_key\\|password=.*\\|token=.*"` only returns code field names or documentation examples.
- [ ] No real cloud account names, private IPs, internal domains, deployment paths, or customer names are included.
- [ ] `go test ./...` passes.
- [ ] The public repository remote points to the intended GitHub account.

## Suggested Public Files

- `README.md`
- `LICENSE`
- `SECURITY.md`
- `OPEN_SOURCE_CHECKLIST.md`
- `go.mod`, `go.sum`
- `cmd/`
- `internal/`
- selected sanitized docs under `docs/`
- `data/.gitkeep`

## Files Intentionally Excluded

- `data/`
- `releases/`
- `memory.md`
- `todo.md`
- `env.md`
- `host.md`
- `docs/development-log.md`
- `docs/domain-tool-inventory.md`
- `docs/environment-tool-inventory.md`
