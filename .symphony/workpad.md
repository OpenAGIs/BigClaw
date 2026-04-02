# BIG-GO-1092

## Plan
- inspect the remaining `scripts/ops/*.py` wrappers, their Go replacements, and active repo references
- delete `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py`
- update targeted Go regression coverage and active migration docs so they no longer treat those Python wrappers as live compatibility entrypoints
- run targeted validation for the Go workspace commands, reference cleanup, and Python file-count reduction
- commit and push the scoped change set

## Acceptance
- the three residual Python files under `scripts/ops/` are removed from the repository
- active docs and regression coverage point operators to `bash scripts/ops/bigclawctl workspace ...` instead of the deleted Python wrappers
- the repository `.py` file count decreases from the pre-change baseline
- targeted validation passes and records exact commands plus results

## Validation
- `rg -n "python3 scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py|scripts/ops/\\*workspace\\*\\.py helpers remain compatibility shims" README.md docs bigclaw-go scripts`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "python3 scripts/ops/(bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py|scripts/ops/\\*workspace\\*\\.py helpers remain compatibility shims" README.md docs bigclaw-go scripts` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 8.005s`; `ok   bigclaw-go/internal/legacyshim 4.191s`; `ok   bigclaw-go/internal/regression 3.678s`
- `bash scripts/ops/bigclawctl workspace bootstrap --help` -> exit `0`; printed `usage: bigclawctl workspace bootstrap [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `find . -name '*.py' | wc -l` -> `19` after deletion, down from the pre-change baseline `22`
