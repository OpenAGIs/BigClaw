# BIG-GO-1599 Workpad

## Plan

1. Confirm the repository-wide Python inventory baseline and verify that the
   assigned tranche assets are already absent from this checkout.
2. Add `BIG-GO-1599`-scoped regression evidence:
   `bigclaw-go/internal/regression/big_go_1599_zero_python_guard_test.go`,
   `bigclaw-go/docs/reports/big-go-1599-python-asset-sweep.md`,
   `reports/BIG-GO-1599-validation.md`, and `reports/BIG-GO-1599-status.json`.
3. Run the targeted inventory and regression commands, record the exact command
   lines and results, then commit and push the lane changes.

## Acceptance

- `BIG-GO-1599` adds lane-specific regression coverage for the already-removed
  Python tranche anchored on `src/bigclaw/design_system.py`,
  `src/bigclaw/models.py`, `src/bigclaw/repo_gateway.py`,
  `src/bigclaw/runtime.py`, `tests/conftest.py`,
  `tests/test_evaluation.py`, `tests/test_mapping.py`, and
  `tests/test_queue.py`.
- The lane guard enforces that the repository remains Python-free and that the
  priority legacy directories `src/bigclaw`, `tests`, `scripts`, and
  `bigclaw-go/scripts` stay Python-free.
- The lane report and validation artifacts capture the zero-Python baseline,
  name the Go/native replacement paths for the assigned tranche, and record the
  exact validation commands and results.
- The resulting change set is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1599/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1599(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|AssignedTrancheAssetsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- The tarball snapshot baseline is already repository-wide Python-free, so this
  lane hardens the Go-only state and refreshes tranche-specific evidence rather
  than deleting in-branch `.py` files.
