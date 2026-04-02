# BIG-GO-1095

## Plan
- remove the remaining repo-root workspace Python compatibility shims under `scripts/ops/`
- retire the legacy Python UI specification trio `src/bigclaw/design_system.py`, `src/bigclaw/console_ia.py`, and `src/bigclaw/ui_review.py`
- update repo guidance and planning metadata to point at the Go-native `bigclaw-go/internal/designsystem` and `bigclaw-go/internal/uireview` surfaces instead of deleted Python sources/tests
- add regression coverage that locks the deleted Python files out of the tree and proves the Go replacements remain present
- run targeted validation and record the exact commands and results
- commit and push the scoped branch changes

## Acceptance
- `scripts/ops/bigclaw_workspace_bootstrap.py` is deleted
- `scripts/ops/symphony_workspace_bootstrap.py` is deleted
- `scripts/ops/symphony_workspace_validate.py` is deleted
- `src/bigclaw/design_system.py` is deleted
- `src/bigclaw/console_ia.py` is deleted
- `src/bigclaw/ui_review.py` is deleted
- active repo guidance no longer describes those Python shims as retained compatibility entrypoints
- planning metadata no longer points release-control evidence at deleted Python UI assets or deleted Python tests
- regression coverage asserts those Python files stay absent and the Go replacements remain present
- repository `.py` file count drops from the pre-change baseline

## Validation
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github`
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review)\\.py|tests/test_(design_system|console_ia|ui_review)\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py bigclaw-go/internal/planning/planning.go`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim`
- `cd bigclaw-go && go test ./internal/planning ./internal/designsystem ./internal/uireview ./internal/regression`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/planning.py`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github` -> exit `1` with no matches
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review)\\.py|tests/test_(design_system|console_ia|ui_review)\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py bigclaw-go/internal/planning/planning.go` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/regression 0.700s`; `ok   bigclaw-go/internal/legacyshim (cached)`
- `cd bigclaw-go && go test ./internal/planning ./internal/designsystem ./internal/uireview ./internal/regression` -> `ok   bigclaw-go/internal/planning 0.547s`; `ok   bigclaw-go/internal/designsystem (cached)`; `ok   bigclaw-go/internal/uireview (cached)`; `ok   bigclaw-go/internal/regression 0.936s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/planning.py` -> exit `0`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]` with the expected Go flags including `-issues`, `-report`, and `-cleanup`
- `find . -name '*.py' | wc -l` -> `16` after both deletion tranches, down from the pre-change baseline `22`
