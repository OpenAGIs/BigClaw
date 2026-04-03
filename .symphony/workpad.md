# BIG-GO-1109

## Plan
- inventory the remaining `src/bigclaw/*.py` assets and choose the nearest self-contained residual slice that already has clear Go ownership
- sweep the operator-console tranche by deleting `src/bigclaw/design_system.py`, `src/bigclaw/console_ia.py`, and `src/bigclaw/ui_review.py`
- retarget active Go planning evidence from deleted Python paths to the canonical Go owners under `bigclaw-go/internal/designsystem`, `bigclaw-go/internal/consoleia`, and `bigclaw-go/internal/uireview`
- add a regression guard that asserts the deleted Python modules stay absent while the Go replacements remain present
- run targeted validation, record exact commands/results, then commit and push the scoped branch

## Acceptance
- lane coverage is explicit for `src/bigclaw/design_system.py`, `src/bigclaw/console_ia.py`, and `src/bigclaw/ui_review.py`
- the selected tranche removes real Python artifacts rather than only touching docs or tracker metadata
- `find . -name '*.py' | wc -l` is lower after the sweep
- targeted validation records exact commands and residual risks

## Validation
- `find . -name '*.py' | wc -l`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `rg -n "src/bigclaw/design_system\\.py|src/bigclaw/console_ia\\.py|src/bigclaw/ui_review\\.py" src bigclaw-go README.md docs/go-mainline-cutover-issue-pack.md`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression`

## Validation Results
- `find . -name '*.py' | wc -l` -> `14`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`, `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`, `src/bigclaw/observability.py`, `src/bigclaw/operations.py`, `src/bigclaw/planning.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/runtime.py`
- `rg -n "src/bigclaw/design_system\\.py|src/bigclaw/console_ia\\.py|src/bigclaw/ui_review\\.py" src bigclaw-go README.md docs/go-mainline-cutover-issue-pack.md` -> matches only the new regression guard and the historical planning note in `docs/go-mainline-cutover-issue-pack.md:476-479`; no active source or Go planning path still targets the deleted Python modules
- `cd bigclaw-go && go test ./internal/planning ./internal/regression` -> `ok   bigclaw-go/internal/planning 0.447s`; `ok   bigclaw-go/internal/regression 0.652s`

## Archived Workpads

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
