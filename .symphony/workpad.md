# BIG-GO-1595 Workpad

## Plan

1. Reconfirm the repository-wide `*.py` inventory and the issue-specific
   retired Python source/test paths:
   - `src/bigclaw/connectors.py`
   - `src/bigclaw/governance.py`
   - `src/bigclaw/planning.py`
   - `src/bigclaw/reports.py`
   - `src/bigclaw/workflow.py`
   - `tests/test_cross_process_coordination_surface.py`
   - `tests/test_governance.py`
   - `tests/test_parallel_refill.py`
2. Add `BIG-GO-1595` regression coverage and issue artifacts that pin the
   retired Python paths and the Go-owned replacement surfaces now carrying the
   same contracts.
3. Run the targeted inventory and regression commands, then record exact
   results in the issue reports.
4. Commit the scoped lane changes on `BIG-GO-1595` and push to `origin`.

## Acceptance

- The repository still contains no physical `.py` files.
- The assigned `src/bigclaw` and `tests` Python assets remain absent.
- The Go-owned replacement paths for connectors, governance, planning,
  reporting, workflow, coordination, and refill remain available.
- Exact validation commands, exact results, and residual risk are recorded in
  repo-visible artifacts.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1595 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1595/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1595/tests -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1595/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1595(RepositoryHasNoPythonFiles|AssignedPythonSourceAndTestsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-11: Repository hydration required a direct no-proxy Git path because
  Git over the configured local proxy stalled and partial-fetches failed.
- 2026-04-11: The checked-out branch baseline is already repository-wide
  Python-free, so this lane is a regression-and-evidence refresh rather than an
  in-branch Python-file deletion batch.
