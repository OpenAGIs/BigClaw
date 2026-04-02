# BIG-GO-1095

## Plan
- remove the remaining repo-root workspace Python compatibility shims under `scripts/ops/`
- retire the legacy Python UI specification trio `src/bigclaw/design_system.py`, `src/bigclaw/console_ia.py`, and `src/bigclaw/ui_review.py`
- retire the tiny legacy helper modules `src/bigclaw/deprecation.py` and `src/bigclaw/audit_events.py` by folding their frozen compatibility behavior into the remaining legacy modules
- retire `src/bigclaw/governance.py` by folding the frozen governance model and reporting helpers into `src/bigclaw/planning.py`
- retire `src/bigclaw/run_detail.py` by folding the frozen run-detail presentation helpers into `src/bigclaw/reports.py`
- retire `src/bigclaw/legacy_shim.py` and tighten the Go `legacy-python compile-check` surface to the remaining frozen entrypoint
- retire `src/bigclaw/__main__.py` and repoint the Go `legacy-python compile-check` surface at the remaining frozen runtime module
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
- `src/bigclaw/deprecation.py` is deleted
- `src/bigclaw/audit_events.py` is deleted
- `src/bigclaw/governance.py` is deleted
- `src/bigclaw/run_detail.py` is deleted
- `src/bigclaw/legacy_shim.py` is deleted
- `src/bigclaw/__main__.py` is deleted
- active repo guidance no longer describes those Python shims as retained compatibility entrypoints
- planning metadata no longer points release-control evidence at deleted Python UI assets or deleted Python tests
- regression coverage asserts those Python files stay absent and the Go replacements remain present
- repository `.py` file count drops from the pre-change baseline

## Validation
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github`
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review)\\.py|tests/test_(design_system|console_ia|ui_review)\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py bigclaw-go/internal/planning/planning.go`
- `rg -n "src/bigclaw/(deprecation|audit_events)\\.py" src/bigclaw/__main__.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py`
- `rg -n "from \\.governance|import \\.governance|src/bigclaw/governance\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py README.md workflow.md`
- `rg -n "from \\.run_detail|import \\.run_detail|src/bigclaw/run_detail\\.py" src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/__init__.py README.md workflow.md`
- `rg -n "src/bigclaw/legacy_shim\\.py|tests/test_legacy_shim\\.py" README.md docs/go-cli-script-migration-plan.md src bigclaw-go/scripts`
- `rg -n "src/bigclaw/__main__\\.py|python -m bigclaw" README.md docs/go-mainline-cutover-handoff.md src bigclaw-go`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim`
- `cd bigclaw-go && go test ./internal/planning ./internal/designsystem ./internal/uireview ./internal/regression`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim ./internal/observability`
- `cd bigclaw-go && go test ./internal/planning ./internal/governance ./internal/regression`
- `cd bigclaw-go && go test ./internal/regression ./internal/evaluation ./internal/contract`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/observability ./internal/planning`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/planning.py`
- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py`
- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "bigclaw_workspace_bootstrap\\.py|symphony_workspace_bootstrap\\.py|symphony_workspace_validate\\.py" README.md workflow.md docs/symphony-repo-bootstrap-template.md docs/reports/bootstrap-cache-validation.md scripts .githooks .github` -> exit `1` with no matches
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review)\\.py|tests/test_(design_system|console_ia|ui_review)\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py bigclaw-go/internal/planning/planning.go` -> exit `1` with no matches
- `rg -n "src/bigclaw/(deprecation|audit_events)\\.py" src/bigclaw/__main__.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/__init__.py` -> exit `1` with no matches
- `rg -n "from \\.governance|import \\.governance|src/bigclaw/governance\\.py" src/bigclaw/__init__.py src/bigclaw/planning.py README.md workflow.md` -> exit `1` with no matches
- `rg -n "from \\.run_detail|import \\.run_detail|src/bigclaw/run_detail\\.py" src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/__init__.py README.md workflow.md` -> exit `1` with no matches
- `rg -n "src/bigclaw/legacy_shim\\.py|tests/test_legacy_shim\\.py" README.md docs/go-cli-script-migration-plan.md src bigclaw-go/scripts` -> pending
- `rg -n "src/bigclaw/legacy_shim\\.py|tests/test_legacy_shim\\.py" README.md docs/go-cli-script-migration-plan.md src bigclaw-go/scripts` -> exit `1` with no matches
- `rg -n "src/bigclaw/__main__\\.py|python -m bigclaw" README.md docs/go-mainline-cutover-handoff.md src bigclaw-go` -> exit `0`; matches are limited to retired-entrypoint docs wording, runtime warning surface strings, and expected regression/manifests coverage
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/regression 0.700s`; `ok   bigclaw-go/internal/legacyshim (cached)`
- `cd bigclaw-go && go test ./internal/planning ./internal/designsystem ./internal/uireview ./internal/regression` -> `ok   bigclaw-go/internal/planning 0.547s`; `ok   bigclaw-go/internal/designsystem (cached)`; `ok   bigclaw-go/internal/uireview (cached)`; `ok   bigclaw-go/internal/regression 0.936s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/planning.py` -> exit `0`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression ./internal/legacyshim ./internal/observability` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/regression 1.479s`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/observability 1.012s`
- `cd bigclaw-go && go test ./internal/planning ./internal/governance ./internal/regression` -> `ok   bigclaw-go/internal/planning (cached)`; `ok   bigclaw-go/internal/governance 0.924s`; `ok   bigclaw-go/internal/regression 1.227s`
- `cd bigclaw-go && go test ./internal/regression ./internal/evaluation ./internal/contract` -> `ok   bigclaw-go/internal/regression 1.106s`; `ok   bigclaw-go/internal/evaluation 1.491s`; `ok   bigclaw-go/internal/contract 1.904s`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/observability ./internal/planning` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/observability (cached)`; `ok   bigclaw-go/internal/planning (cached)`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> pending
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 5.489s`; `ok   bigclaw-go/internal/legacyshim 2.520s`; `ok   bigclaw-go/internal/regression 1.497s`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 4.123s`; `ok   bigclaw-go/internal/legacyshim 1.397s`; `ok   bigclaw-go/internal/regression 0.974s`
- `cd bigclaw-go && go test ./internal/observability ./internal/planning ./internal/evaluation ./internal/contract` -> `ok   bigclaw-go/internal/observability (cached)`; `ok   bigclaw-go/internal/planning (cached)`; `ok   bigclaw-go/internal/evaluation (cached)`; `ok   bigclaw-go/internal/contract (cached)`
- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py` -> exit `0`
- `python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/planning.py src/bigclaw/evaluation.py src/bigclaw/__init__.py` -> exit `0`
- `python3 -m py_compile src/bigclaw/reports.py src/bigclaw/evaluation.py` -> exit `0`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]` with the expected Go flags including `-issues`, `-report`, and `-cleanup`
- `find . -name '*.py' | wc -l` -> `11` after the legacy-shim tranche, before the `__main__.py` tranche
- `find . -name '*.py' | wc -l` -> `10` after deleting `src/bigclaw/__main__.py`
