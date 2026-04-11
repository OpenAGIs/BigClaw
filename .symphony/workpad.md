# BIG-GO-203 Workpad

## Plan

1. Confirm the residual retired Python test paths that are still not pinned by
   prior residual-test sweep lanes, and keep this issue scoped to that gap
   only.
2. Add `BIG-GO-203` regression coverage for the remaining retired test slice:
   `tests/test_cost_control.py`, `tests/test_event_bus.py`,
   `tests/test_execution_flow.py`, `tests/test_github_sync.py`,
   `tests/test_governance.py`, `tests/test_issue_archive.py`,
   `tests/test_mapping.py`, `tests/test_memory.py`,
   `tests/test_models.py`, `tests/test_observability.py`, and
   `tests/test_pilot.py`.
3. Record the matching lane report plus `reports/BIG-GO-203-validation.md` and
   `reports/BIG-GO-203-status.json`, run targeted validation, then commit and
   push the scoped change set.

## Acceptance

- `BIG-GO-203` adds lane-specific regression coverage for the remaining
  residual retired Python tests not already covered by prior sweeps.
- The guard proves the repository remains Python-free, the retired test paths
  stay absent, and the mapped Go/native replacement paths remain available.
- `bigclaw-go/docs/reports/big-go-203-python-asset-sweep.md`,
  `reports/BIG-GO-203-validation.md`, and `reports/BIG-GO-203-status.json`
  capture the exact residual inventory, validation commands, and outcomes.
- The resulting change is committed and pushed to `origin/main`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-203 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/costcontrol /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/events /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/executor /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/githubsync /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/governance /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/intake /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/issuearchive /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/observability /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/pilot /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/policy /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-203/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO203(RepositoryHasNoPythonFiles|ResidualPythonTestGapSliceStaysAbsent|GapSliceReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
