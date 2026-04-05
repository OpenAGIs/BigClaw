# BIG-GO-1315 Workpad

## Plan

1. Confirm the remaining physical Python asset inventory for the full repository and the priority residual directories: `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Add lane-scoped zero-Python sweep artifacts for `BIG-GO-1315`:
   - `bigclaw-go/docs/reports/big-go-1315-python-asset-sweep.md`
   - `bigclaw-go/internal/regression/big_go_1315_zero_python_guard_test.go`
   - `reports/BIG-GO-1315-status.json`
   - `reports/BIG-GO-1315-validation.md`
3. Run targeted validation commands, record exact commands and results in the validation artifacts, then commit and push the lane changes.

## Acceptance

- Produce an explicit remaining Python asset inventory for the repository and the priority residual directories.
- Reduce physical Python files if any remain, or document and harden the zero-Python baseline if the checkout is already Python-free.
- Record the Go replacement paths that cover the retired Python surface.
- Provide exact validation commands and results.
- Publish the change as a committed and pushed lane update.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1315(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-05: Confirmed repository-wide physical Python inventory is already `0`, including the lane priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- 2026-04-05: BIG-GO-1315 therefore lands as a zero-Python regression-prevention sweep that refreshes issue-scoped metadata and Go validation evidence.
