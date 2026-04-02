# BIG-GO-1065 Validation

Date: 2026-04-03

## Scope

Issue: `BIG-GO-1065`

Title: `tests residual sweep B`

This tranche covers the remaining Python test assets named in the issue scope.
In this checkout those files were already absent before implementation started, so
the scoped work for `BIG-GO-1065` is to make the inventory explicit, pin that
absence with a Go regression guard, and record exact validation and count evidence.

## Delivered

- added `bigclaw-go/internal/regression/top_level_module_purge_tranche29_test.go`
  to pin the full 40-file Python test asset inventory listed in the issue
- verified the tranche remains backed by Go-owned replacement surfaces spanning:
  - `bigclaw-go/internal/governance/*`
  - `bigclaw-go/internal/issuearchive/*`
  - `bigclaw-go/internal/observability/*`
  - `bigclaw-go/internal/policy/*`
  - `bigclaw-go/internal/product/*`
  - `bigclaw-go/internal/queue/*`
  - `bigclaw-go/internal/refill/*`
  - `bigclaw-go/internal/repo/*`
  - `bigclaw-go/internal/risk/*`
  - `bigclaw-go/internal/scheduler/*`
  - `bigclaw-go/internal/service/*`
  - `bigclaw-go/internal/worker/*`
  - `bigclaw-go/internal/workflow/*`
  - `bigclaw-go/internal/regression/*`
  - `scripts/ops/bigclawctl`
- recorded the repo-wide Python file count impact and residual risk for this batch

## Python Asset Inventory

The following Python test files are the scoped tranche and are verified absent:

- `tests/test_github_sync.py`
- `tests/test_governance.py`
- `tests/test_issue_archive.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_live_shadow_scorecard.py`
- `tests/test_mapping.py`
- `tests/test_memory.py`
- `tests/test_models.py`
- `tests/test_observability.py`
- `tests/test_operations.py`
- `tests/test_orchestration.py`
- `tests/test_parallel_refill.py`
- `tests/test_parallel_validation_bundle.py`
- `tests/test_pilot.py`
- `tests/test_planning.py`
- `tests/test_queue.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_triage.py`
- `tests/test_reports.py`
- `tests/test_risk.py`
- `tests/test_roadmap.py`
- `tests/test_runtime.py`
- `tests/test_runtime_matrix.py`
- `tests/test_saved_views.py`
- `tests/test_scheduler.py`
- `tests/test_service.py`
- `tests/test_shadow_matrix_corpus.py`
- `tests/test_subscriber_takeover_harness.py`
- `tests/test_ui_review.py`
- `tests/test_validation_bundle_continuation_policy_gate.py`
- `tests/test_validation_bundle_continuation_scorecard.py`
- `tests/test_validation_policy.py`
- `tests/test_workflow.py`
- `tests/test_workspace_bootstrap.py`

## Validation

### Asset absence check

Command:

```bash
for f in tests/test_github_sync.py tests/test_governance.py tests/test_issue_archive.py tests/test_live_shadow_bundle.py tests/test_live_shadow_scorecard.py tests/test_mapping.py tests/test_memory.py tests/test_models.py tests/test_observability.py tests/test_operations.py tests/test_orchestration.py tests/test_parallel_refill.py tests/test_parallel_validation_bundle.py tests/test_pilot.py tests/test_planning.py tests/test_queue.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_rollout.py tests/test_repo_triage.py tests/test_reports.py tests/test_risk.py tests/test_roadmap.py tests/test_runtime.py tests/test_runtime_matrix.py tests/test_saved_views.py tests/test_scheduler.py tests/test_service.py tests/test_shadow_matrix_corpus.py tests/test_subscriber_takeover_harness.py tests/test_ui_review.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_validation_bundle_continuation_scorecard.py tests/test_validation_policy.py tests/test_workflow.py tests/test_workspace_bootstrap.py; do test ! -e "$f"; done
```

Result:

```text
exit code 0
```

### Python file count

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1065 -name '*.py' | wc -l
```

Result:

```text
4
```

Scoped count note: this tranche did not remove additional `.py` files in the working
tree because the targeted Python tests were already gone in the baseline checkout.
The issue still closes a real migration gap by adding an explicit regression guard and
closeout evidence for the tranche inventory.

### Targeted Go regression

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1065/bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche29
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.364s
```

### Traceability scan

Command:

```bash
rg -n "test_(github_sync|governance|issue_archive|live_shadow_bundle|live_shadow_scorecard|mapping|memory|models|observability|operations|orchestration|parallel_refill|parallel_validation_bundle|pilot|planning|queue|repo_board|repo_collaboration|repo_gateway|repo_governance|repo_links|repo_registry|repo_rollout|repo_triage|reports|risk|roadmap|runtime|runtime_matrix|saved_views|scheduler|service|shadow_matrix_corpus|subscriber_takeover_harness|ui_review|validation_bundle_continuation_policy_gate|validation_bundle_continuation_scorecard|validation_policy|workflow|workspace_bootstrap)\.py" bigclaw-go/internal/regression/top_level_module_purge_tranche29_test.go reports/BIG-GO-1065-validation.md reports/BIG-GO-1065-closeout.md .symphony/workpad.md README.md
```

Result:

```text
matches only in the tranche-29 regression guard and the BIG-GO-1065 workpad/validation/closeout artifacts
```

## Python Count Impact

- before this turn: `4`
- after this turn: `4`
- delta: `0`

## Residual Risk

- the tranche scope here is evidence and regression coverage, not fresh deletion, because
  the named Python tests were already removed in the checked-out baseline
- the remaining Python footprint is limited to four live compatibility modules under
  `src/bigclaw/`; further count reduction requires core compatibility-surface migration,
  not another low-risk residual test sweep
