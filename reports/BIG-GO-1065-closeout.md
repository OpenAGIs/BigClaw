# BIG-GO-1065 Closeout Index

Issue: `BIG-GO-1065`

Title: `tests residual sweep B`

Date: `2026-04-03`

## Branch

`symphony/BIG-GO-1065`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1065-validation.md`
- Regression guard:
  - `bigclaw-go/internal/regression/top_level_module_purge_tranche29_test.go`
- Workpad:
  - `.symphony/workpad.md`

## Outcome

- the 40 Python test assets named in the issue scope are explicitly inventoried and pinned as absent
- `bigclaw-go/internal/regression/top_level_module_purge_tranche29_test.go` now prevents those test files from silently reappearing
- active replacement coverage remains Go-owned across governance, issue archive, mapping, memory, observability, refill, repo, risk, runtime, scheduler, service, workflow, and automation entrypoints
- repo-wide Python file count remains `4`; this tranche adds regression/evidence coverage rather than a fresh `.py` deletion because the scoped test assets were already absent in the checked-out baseline

## Validation Commands

```bash
for f in tests/test_github_sync.py tests/test_governance.py tests/test_issue_archive.py tests/test_live_shadow_bundle.py tests/test_live_shadow_scorecard.py tests/test_mapping.py tests/test_memory.py tests/test_models.py tests/test_observability.py tests/test_operations.py tests/test_orchestration.py tests/test_parallel_refill.py tests/test_parallel_validation_bundle.py tests/test_pilot.py tests/test_planning.py tests/test_queue.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_rollout.py tests/test_repo_triage.py tests/test_reports.py tests/test_risk.py tests/test_roadmap.py tests/test_runtime.py tests/test_runtime_matrix.py tests/test_saved_views.py tests/test_scheduler.py tests/test_service.py tests/test_shadow_matrix_corpus.py tests/test_subscriber_takeover_harness.py tests/test_ui_review.py tests/test_validation_bundle_continuation_policy_gate.py tests/test_validation_bundle_continuation_scorecard.py tests/test_validation_policy.py tests/test_workflow.py tests/test_workspace_bootstrap.py; do test ! -e "$f"; done
find . -name '*.py' | wc -l
cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche29
rg -n "test_(github_sync|governance|issue_archive|live_shadow_bundle|live_shadow_scorecard|mapping|memory|models|observability|operations|orchestration|parallel_refill|parallel_validation_bundle|pilot|planning|queue|repo_board|repo_collaboration|repo_gateway|repo_governance|repo_links|repo_registry|repo_rollout|repo_triage|reports|risk|roadmap|runtime|runtime_matrix|saved_views|scheduler|service|shadow_matrix_corpus|subscriber_takeover_harness|ui_review|validation_bundle_continuation_policy_gate|validation_bundle_continuation_scorecard|validation_policy|workflow|workspace_bootstrap)\.py" bigclaw-go/internal/regression reports README.md .symphony/workpad.md
```

## Remaining Risk

No blocker remains for this tranche.

The only material caveat is scope: this lane formalizes and guards an already-removed
Python test batch. Additional repo-wide `.py` reduction now depends on migrating the
four remaining live compatibility modules under `src/bigclaw/`.
