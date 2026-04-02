# BIG-GO-1095

## Plan
- remove the remaining repo-root workspace Python compatibility shims under `scripts/ops/`
- update repo guidance and migration docs to point directly at `bash scripts/ops/bigclawctl workspace ...`
- add regression coverage that locks the deleted Python files out of the tree and proves the Go workspace replacement remains present
- run targeted validation and record the exact commands and results
- commit and push the scoped branch changes

## Acceptance
- `scripts/ops/bigclaw_workspace_bootstrap.py` is deleted
- `scripts/ops/symphony_workspace_bootstrap.py` is deleted
- `scripts/ops/symphony_workspace_validate.py` is deleted
- active repo guidance no longer describes those Python shims as retained compatibility entrypoints
- regression coverage asserts those Python files stay absent and the Go workspace command surface remains present
- repository `.py` file count drops from the pre-change baseline

## Validation
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/regression 0.700s`; `ok   bigclaw-go/internal/legacyshim (cached)`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]` with the expected Go flags including `-issues`, `-report`, and `-cleanup`
- `find . -name '*.py' | wc -l` -> `19` after deletion, down from the pre-change baseline `22`
