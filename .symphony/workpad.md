# BIG-GO-1101

## Plan
- inventory the actual remaining Python files in this worktree and confirm which candidate root wrappers are already gone versus still referenced in live docs/tests
- keep the lane scoped to root executable-wrapper cleanup plus a physically larger Python-script sweep that can reduce `find . -name '*.py' | wc -l`
- replace stale root guidance that still claims `scripts/ops/*workspace*.py` compatibility shims remain, because those wrappers are already removed in this branch history
- migrate Go planning evidence links off the frozen Python release-gate sources where the Go-native packages already own the behavior and tests
- delete the now-unreferenced release-gate Python source tranche under `src/bigclaw`: `design_system.py`, `console_ia.py`, and `ui_review.py`
- delete the now-unreferenced Python planning/governance tranche under `src/bigclaw`: `planning.py` and `governance.py`
- retarget the remaining planning evidence links off `operations.py`, `evaluation.py`, and `reports.py` so the orphaned Python operations/reporting support tranche can be removed
- delete the now-unreferenced Python operations/reporting support tranche under `src/bigclaw`: `audit_events.py`, `collaboration.py`, `deprecation.py`, `evaluation.py`, `observability.py`, `operations.py`, `reports.py`, `risk.py`, and `run_detail.py`
- update planning/unit regression expectations to point at Go-owned evidence targets instead of the deleted Python files
- run targeted validation for planning/report regressions, legacy shim compile-check, docs/reference sweeps, and repo Python file count reduction
- commit the scoped change set and push it to the remote branch

## Acceptance
- lane coverage is explicit: stale root wrapper guidance plus the removable release-gate Python tranche (`src/bigclaw/design_system.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/ui_review.py`)
- lane coverage also includes the removable orphan planning/governance tranche (`src/bigclaw/planning.py`, `src/bigclaw/governance.py`)
- lane coverage also includes the removable orphan operations/reporting support tranche (`src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`)
- the change removes or replaces real Python assets instead of only tracker/doc cosmetics
- repository guidance no longer claims the deleted workspace Python wrappers still exist
- Go planning evidence points at Go-owned implementation/test surfaces for release-gate validation
- `find . -name '*.py' | wc -l` decreases from the pre-change baseline in this worktree
- exact validation commands and outcomes are recorded below

## Validation
- `find . -name '*.py' | wc -l`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg -n "scripts/ops/\\*workspace\\*\\.py|compatibility shims over the same Go CLI|ops wrappers remain only as compatibility shims" README.md docs/go-cli-script-migration-plan.md`
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review)\\.py" README.md bigclaw-go docs scripts .github -g '!docs/go-mainline-cutover-issue-pack.md'`
- `rg -n "src/bigclaw/(design_system|console_ia|ui_review|planning|governance)\\.py|bigclaw\\.planning|bigclaw\\.governance" README.md bigclaw-go docs scripts .github src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!reports/**' -g '!local-issues.json'`
- `rg -n "src/bigclaw/(audit_events|collaboration|deprecation|evaluation|governance|observability|operations|planning|reports|risk|run_detail|design_system|console_ia|ui_review)\\.py|bigclaw\\.(planning|governance)" README.md bigclaw-go docs scripts .github src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!reports/**' -g '!local-issues.json'`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `2`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`
- `rg -n "scripts/ops/\\*workspace\\*\\.py|compatibility shims over the same Go CLI|ops wrappers remain only as compatibility shims" README.md docs/go-cli-script-migration-plan.md` -> exit `1` with no matches
- `rg -n "src/bigclaw/(audit_events|collaboration|deprecation|evaluation|governance|observability|operations|planning|reports|risk|run_detail|design_system|console_ia|ui_review)\\.py|bigclaw\\.(planning|governance)" README.md bigclaw-go docs scripts .github src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!reports/**' -g '!local-issues.json'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/planning 0.818s`; `ok   bigclaw-go/internal/regression 0.983s`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/cmd/bigclawctl (cached)`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1101/src/bigclaw/legacy_shim.py`
- `git status --short` -> modified `.symphony/workpad.md`, `bigclaw-go/internal/planning/planning.go`, `bigclaw-go/internal/planning/planning_test.go`; deleted `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`

## Archived Workpads
### BIG-GO-1096

#### Plan
- inspect the remaining packaging-era Python package surface under `src/bigclaw`, with focus on `__main__.py`, `__init__.py`, and legacy compile-check references
- remove broken package-entry residue that still implies repo-root/package execution support, while preserving non-packaging migration-only Python modules
- update Go compatibility checks and repo guidance so they validate only the frozen legacy shims that still exist
- neutralize any remaining deleted-entrypoint warning text inside frozen Python modules
- remove dead repo-root Python test/lint/bootstrap commands that still point at the deleted `tests/` lane
- update active planning/backlog generators so they stop emitting deleted `tests/...` evidence targets and validation commands
- update active docs that still prescribe deleted Python test lanes as current validation
- simplify regression checks so they assert the removed `tests/` directory rather than carrying long deleted-file manifests
- remove the last active `tests/` mentions from root README/current planning assertions
- tighten the remaining README bootstrap wording so it no longer uses the old migration-surface packaging phrasing
- run targeted validation covering reference cleanup, Go legacy-shim tests, CLI help, and repository `.py` count reduction
- commit and push the scoped change set

#### Acceptance
- packaging-only Python package residue is reduced, with repository `.py` count lower than the pre-change baseline
- no active repo guidance or validation path still points at deleted package entrypoint files or already-removed `src/bigclaw/service.py`
- Go legacy compatibility checks only cover Python shim files that still exist in the repository
- targeted validation records exact commands and results

#### Validation
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg -n "src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `bash scripts/ops/bigclawctl legacy-python --help`
- `rg -n "pytest tests|tests/test_planning\\.py|ruff check src tests scripts|PYTHONPATH=src python3 -m pytest tests" README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `rg -n "tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_ui_review\\.py|tests/test_control_center\\.py|tests/test_operations\\.py|tests/test_evaluation\\.py|tests/test_orchestration\\.py|tests/test_reports\\.py" bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go src/bigclaw/planning.py`
- `cd bigclaw-go && go test ./internal/planning`
- `rg -n "PYTHONPATH=src python3 -m pytest tests/test_|PYTHONPATH=src python3 -m pytest -q|python3 -m pytest tests/test_legacy_shim\\.py" docs/go-cli-script-migration-plan.md docs/BigClaw-AgentHub-Integration-Alignment.md`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/repo ./internal/collaboration ./internal/observability ./internal/reportstudio ./internal/governance ./internal/triage ./internal/product`
- `rg -n "tests/test_control_center\\.py|tests/test_live_shadow_bundle\\.py|tests/test_orchestration\\.py|tests/test_queue\\.py|tests/test_repo_links\\.py|tests/test_repo_collaboration\\.py|tests/test_repo_rollout\\.py|tests/test_models\\.py|tests/test_planning\\.py|tests/test_operations\\.py|tests/test_observability\\.py|tests/test_reports\\.py|tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_evaluation\\.py|tests/test_risk\\.py|tests/test_runtime_matrix\\.py|tests/test_scheduler\\.py|tests/test_ui_review\\.py|tests/reports_legacy\\.py|tests/conftest\\.py" bigclaw-go/internal/regression`
- `cd bigclaw-go && go test ./internal/regression`
- `rg -n "tests/|python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/service\\.py" README.md bigclaw-go/internal/planning/planning_test.go -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `cd bigclaw-go && go test ./internal/planning`
- `rg -n "legacy Python migration surface|repo-root packaging|python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/service\\.py|tests/" README.md -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'`
- `bash scripts/dev_bootstrap.sh`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l`

#### Validation Results
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/design_system.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`, `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/planning.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/runtime.py`, `src/bigclaw/ui_review.py`
- `rg -n "src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/legacyshim 0.843s`; `ok   bigclaw-go/internal/regression 0.858s`; `ok   bigclaw-go/cmd/bigclawctl 4.171s`
- `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python --help` -> exit `0`; printed `usage: bigclawctl legacy-python <compile-check> [flags]`
- `rg -n "pytest tests|tests/test_planning\\.py|ruff check src tests scripts|PYTHONPATH=src python3 -m pytest tests" README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1096/src/bigclaw/legacy_shim.py`
- `rg -n "tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_ui_review\\.py|tests/test_control_center\\.py|tests/test_operations\\.py|tests/test_evaluation\\.py|tests/test_orchestration\\.py|tests/test_reports\\.py" bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go src/bigclaw/planning.py` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning` -> `ok   bigclaw-go/internal/planning 0.533s`
- `rg -n "PYTHONPATH=src python3 -m pytest tests/test_|PYTHONPATH=src python3 -m pytest -q|python3 -m pytest tests/test_legacy_shim\\.py" docs/go-cli-script-migration-plan.md docs/BigClaw-AgentHub-Integration-Alignment.md` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/repo ./internal/collaboration ./internal/observability ./internal/reportstudio ./internal/governance ./internal/triage ./internal/product` -> `ok   bigclaw-go/cmd/bigclawctl (cached)`; `ok   bigclaw-go/internal/repo 0.476s`; `ok   bigclaw-go/internal/collaboration 0.904s`; `ok   bigclaw-go/internal/observability 1.333s`; `ok   bigclaw-go/internal/reportstudio 2.233s`; `ok   bigclaw-go/internal/governance 1.791s`; `ok   bigclaw-go/internal/triage 2.689s`; `ok   bigclaw-go/internal/product 3.093s`
- `rg -n "tests/test_control_center\\.py|tests/test_live_shadow_bundle\\.py|tests/test_orchestration\\.py|tests/test_queue\\.py|tests/test_repo_links\\.py|tests/test_repo_collaboration\\.py|tests/test_repo_rollout\\.py|tests/test_models\\.py|tests/test_planning\\.py|tests/test_operations\\.py|tests/test_observability\\.py|tests/test_reports\\.py|tests/test_design_system\\.py|tests/test_console_ia\\.py|tests/test_evaluation\\.py|tests/test_risk\\.py|tests/test_runtime_matrix\\.py|tests/test_scheduler\\.py|tests/test_ui_review\\.py|tests/reports_legacy\\.py|tests/conftest\\.py" bigclaw-go/internal/regression` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/regression` -> `ok   bigclaw-go/internal/regression 0.602s`
- `rg -n "tests/|python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/service\\.py" README.md bigclaw-go/internal/planning/planning_test.go -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning` -> `ok   bigclaw-go/internal/planning 0.823s`
- `rg -n "legacy Python migration surface|repo-root packaging|python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/service\\.py|tests/" README.md -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches
- `bash scripts/dev_bootstrap.sh` -> exit `0`; `ok   bigclaw-go/cmd/bigclawctl (cached)` followed by `BigClaw Go development environment is ready.`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l` -> `19` tracked `.py` files in `HEAD`; `17` `.py` files in the worktree after deleting the packaging entrypoint residue
- follow-up sweep: `rg -n "python -m bigclaw serve|src/bigclaw/service\\.py|src/bigclaw/__main__\\.py|python -m bigclaw\\b" README.md bigclaw-go .github scripts docs src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json'` -> exit `1` with no matches after updating `src/bigclaw/runtime.py`
- follow-up sweep: `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/regression (cached)`; `ok   bigclaw-go/cmd/bigclawctl 3.216s`

### BIG-GO-1104

#### Plan
- confirm the lane-owned residual Python surface under `src/bigclaw` and baseline the repository `.py` count
- verify whether `src/bigclaw/runtime.py` still has live imports, CLI entrypoints, or Go-side compatibility checks that would block removal
- delete `src/bigclaw/runtime.py` if it is unreferenced, and tighten any active repo guidance that still describes it as a retained migration-only asset
- run targeted validation covering search sweeps, Go compatibility checks, and repository `.py` count reduction
- commit and push the scoped change set

#### Acceptance
- the lane file list is explicit and this issue only touches the residual `src/bigclaw/runtime.py` surface from the provided candidate set
- a real Python asset is removed or replaced, not just tracker/doc cosmetics
- `find . -name '*.py' | wc -l` is lower after the change than before
- exact validation commands and outcomes are recorded

#### Validation
- `find . -name '*.py' | wc -l`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg -n "src/bigclaw/runtime\\.py|bigclaw\\.runtime" README.md bigclaw-go scripts docs .github src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!local-issues.json'`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`

#### Validation Results
- `find . -name '*.py' | wc -l` before -> `17`
- `find . -name '*.py' | wc -l` after -> `16`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/console_ia.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/design_system.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`, `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/planning.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/ui_review.py`
- `rg -n "src/bigclaw/runtime\\.py|bigclaw\\.runtime" README.md bigclaw-go scripts docs .github src -g '!docs/go-mainline-cutover-issue-pack.md' -g '!local-issues.json'` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1104/src/bigclaw/legacy_shim.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/legacyshim 0.417s`; `ok   bigclaw-go/internal/regression 0.824s`; `ok   bigclaw-go/cmd/bigclawctl 3.778s`
### BIG-GO-1053

#### Plan
- Reconfirm the live `bigclaw-go/scripts/e2e/` surface is Go/shell only and identify any active docs, workflow, hook, or CI references that still mention deleted tranche-2 Python helpers.
- Align the issue-local migration evidence so the archived workpad note and migration matrix reflect `BIG-GO-1053` rather than an older tranche header.
- Run targeted validation for the e2e entrypoint migration guard and the Go CLI help surfaces used by the retained operator entrypoints.
- Record exact commands and results in the issue reports and push the scoped closeout refresh.

#### Acceptance
- `bigclaw-go/scripts/e2e/` contains no tracked `.py` helpers.
- Live README/docs/workflow/hooks/CI surfaces do not reference deleted tranche-2 Python helpers.
- `bigclaw-go/docs/go-cli-script-migration.md` explicitly attributes the tranche-2 Python-free e2e surface to `BIG-GO-1053`.
- Targeted validation passes and exact commands/results are captured in `reports/BIG-GO-1053-validation.md`.

#### Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l` -> `0`
- `find . -name '*.py' | wc -l` -> `43`
- `dirs=(); for p in README.md bigclaw-go/README.md bigclaw-go/docs docs .github .githooks .husky workflow.md; do [ -e "$p" ] && dirs+=("$p"); done; rg -n "bigclaw-go/scripts/e2e/.*\\.py|scripts/e2e/.*\\.py|run_task_smoke\\.py|export_validation_bundle\\.py|validation_bundle_continuation_policy_gate\\.py|mixed_workload_matrix\\.py|cross_process_coordination_surface\\.py|subscriber_takeover_fault_matrix\\.py|external_store_validation\\.py|multi_node_shared_queue\\.py" "${dirs[@]}"` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...` -> `ok   bigclaw-go/cmd/bigclawctl 4.995s`; `ok   bigclaw-go/internal/regression 0.839s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> exit `0`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help` -> exit `0`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help` -> exit `0`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help` -> exit `0`
