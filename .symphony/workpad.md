# BIG-GO-1086

## Plan
- Confirm how `scripts/ops/symphony_workspace_bootstrap.py` is still referenced and whether any default runtime path still depends on it.
- Delete the Python wrapper and keep the Go-first workspace bootstrap path through `scripts/ops/bigclawctl workspace ...` as the only active entrypoint.
- Update only the focused regression/documentation surface that should no longer claim the deleted wrapper exists.
- Run targeted validation for the Go workspace command and verify the repository `.py` count decreases.
- Commit and push the scoped branch.

## Acceptance
- `scripts/ops/symphony_workspace_bootstrap.py` is deleted or no longer reachable through a default execution path.
- The remaining bootstrap entrypoint is the Go-first `scripts/ops/bigclawctl workspace ...` path.
- Repository `.py` count decreases from the pre-change baseline.
- Validation covers the surviving Go workspace command path and any targeted regression touched by the removal.
- Changes stay scoped to this issue.

## Validation
- `find . -name '*.py' | wc -l`
- `rg -n "symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap" --glob '!reports/**' --glob '!local-issues.json' .`
- `bash scripts/ops/bigclawctl workspace --help`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- `git status --short`

## Validation Results
- Pre-change `find . -name '*.py' | wc -l` -> `23`
- Post-change `find . -name '*.py' | wc -l` -> `22`
- `rg -n "symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap" --glob '!reports/**' --glob '!local-issues.json' .` -> remaining matches are the historical retirement note in `docs/go-mainline-cutover-issue-pack.md` and the new absence regression `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`
- `bash scripts/ops/bigclawctl workspace --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 5.798s`, `ok   bigclaw-go/internal/legacyshim 2.348s`, `ok   bigclaw-go/internal/regression 2.829s`
- `git status --short` -> `M .symphony/workpad.md`, `M docs/go-cli-script-migration-plan.md`, `M docs/go-mainline-cutover-issue-pack.md`, `D scripts/ops/symphony_workspace_bootstrap.py`, `?? bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`
