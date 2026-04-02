# BIG-GO-1096

## Plan
- inspect the remaining packaging-era Python package surface under `src/bigclaw`, with focus on `__main__.py`, `__init__.py`, and legacy compile-check references
- remove broken package-entry residue that still implies repo-root/package execution support, while preserving non-packaging migration-only Python modules
- update Go compatibility checks and repo guidance so they validate only the frozen legacy shims that still exist
- neutralize any remaining deleted-entrypoint warning text inside frozen Python modules
- remove dead repo-root Python test/lint/bootstrap commands that still point at the deleted `tests/` lane
- update active planning/backlog generators so they stop emitting deleted `tests/...` evidence targets and validation commands
- run targeted validation covering reference cleanup, Go legacy-shim tests, CLI help, and repository `.py` count reduction
- commit and push the scoped change set

## Acceptance
- packaging-only Python package residue is reduced, with repository `.py` count lower than the pre-change baseline
- no active repo guidance or validation path still points at deleted package entrypoint files or already-removed `src/bigclaw/service.py`
- Go legacy compatibility checks only cover Python shim files that still exist in the repository
- targeted validation records exact commands and results

## Validation
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg -n "src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `bash scripts/ops/bigclawctl legacy-python --help`
- `rg -n "pytest tests|tests/test_planning\\.py|ruff check src tests scripts|PYTHONPATH=src python3 -m pytest tests" README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `rg -n "tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_ui_review\\.py|tests/test_control_center\\.py|tests/test_operations\\.py|tests/test_evaluation\\.py|tests/test_orchestration\\.py|tests/test_reports\\.py" bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go src/bigclaw/planning.py`
- `cd bigclaw-go && go test ./internal/planning`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l`

## Validation Results
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/design_system.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`, `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/planning.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/runtime.py`, `src/bigclaw/ui_review.py`
- `rg -n "src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/legacyshim 0.843s`; `ok   bigclaw-go/internal/regression 0.858s`; `ok   bigclaw-go/cmd/bigclawctl 4.171s`
- `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python --help` -> exit `0`; printed `usage: bigclawctl legacy-python <compile-check> [flags]`
- `rg -n "pytest tests|tests/test_planning\\.py|ruff check src tests scripts|PYTHONPATH=src python3 -m pytest tests" README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1096/src/bigclaw/legacy_shim.py`
- `rg -n "tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_ui_review\\.py|tests/test_control_center\\.py|tests/test_operations\\.py|tests/test_evaluation\\.py|tests/test_orchestration\\.py|tests/test_reports\\.py" bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go src/bigclaw/planning.py` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning` -> `ok   bigclaw-go/internal/planning 0.533s`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l` -> `19` tracked `.py` files in `HEAD`; `17` `.py` files in the worktree after deleting the packaging entrypoint residue
- follow-up sweep: `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches after updating `src/bigclaw/runtime.py`
- follow-up sweep: `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/regression (cached)`; `ok   bigclaw-go/cmd/bigclawctl 3.216s`
