# BIG-GO-1091

## Plan
- inspect the remaining root `scripts/ops/*.py` workspace shims and their active repo references
- delete `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py`
- update active documentation and regression coverage to point at `bash scripts/ops/bigclawctl workspace ...` instead of the deleted Python shims
- run targeted validation covering reference cleanup, Go legacy shim tests, CLI help, and repository `.py` count reduction
- commit and push the scoped change set

## Acceptance
- the three remaining Python workspace shims under `scripts/ops` are removed from the repository
- active repo guidance and regression coverage no longer instruct users to execute those deleted Python files
- the repository `.py` file count decreases from the pre-change baseline
- targeted validation records exact commands and results

## Validation
- `rg -n "python3 scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py|scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py --help|scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py\\b" README.md docs scripts bigclaw-go -g '!docs/go-cli-script-migration-plan.md' -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go'`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "python3 scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py|scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py --help|scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py\\b" README.md docs scripts bigclaw-go -g '!docs/go-cli-script-migration-plan.md' -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/regression 0.606s`
- `bash scripts/ops/bigclawctl workspace bootstrap --help` -> exit `0`; printed `usage: bigclawctl workspace bootstrap [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l` -> `22` in `HEAD`; `19` in the worktree after deleting the three root Python shims
