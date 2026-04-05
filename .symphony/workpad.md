# BIG-GO-1457 Workpad

## Plan

1. Confirm the repository-wide and priority-directory Python asset inventory for
   `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Add a lane report that records the remaining inventory, the Go replacement
   paths, and the exact validation commands/results for this refill sweep.
3. Add a lane-specific Go regression test that asserts the repository remains
   free of physical Python files and that the lane report keeps the expected
   replacement references and validation commands.
4. Run targeted validation, then commit and push `BIG-GO-1457`.

## Acceptance

- The lane has an explicit remaining Python asset inventory.
- The lane records the Go replacement paths for the retired Python surfaces.
- The lane keeps or strengthens zero-Python regression coverage without
  widening scope beyond this issue.
- Targeted validation commands and exact results are captured in the report and
  final handoff.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1457(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
