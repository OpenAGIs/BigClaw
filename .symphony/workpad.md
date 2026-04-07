# BIG-GO-1577 Workpad

## Context
- Issue: `BIG-GO-1577`
- Goal: perform a Go-only residual Python sweep over the specified candidate files, preferring deletion or Go replacements; if removal is not yet possible, reduce Python to a thin compatibility shim and document deletion conditions.
- Current repo state on entry: workspace contains only `.git` metadata and no checked-out tree yet; repository content must be fetched from `origin` before code changes.

## Scope
- `src/bigclaw/cost_control.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `tests/test_design_system.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_pilot.py`
- `tests/test_repo_triage.py`
- `tests/test_subscriber_takeover_harness.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

## Plan
1. Fetch and check out the actual repository contents from `origin`.
2. Inspect the candidate Python files and repo references to determine which can be deleted, replaced by Go commands, or reduced to shims.
3. Implement the smallest scoped changes that remove physical Python assets where feasible.
4. Run targeted validation commands covering touched Go commands/tests and any compatibility paths left behind.
5. Commit and push the issue branch.

## Acceptance
- Enumerate which candidate Python files were covered in this sweep.
- Remove, migrate, or replace Python files with Go implementations/commands wherever feasible.
- Any unavoidable residual Python must be reduced to a thin compatibility layer with explicit deletion conditions documented inline or nearby.
- Record exact validation commands and their outcomes.
- Note residual risks only if they remain after targeted validation.

## Validation
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1577(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestLiveShadowBundleSurface'`
  - Result: `ok  	bigclaw-go/internal/regression	0.179s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  - Result: `14 passed`
