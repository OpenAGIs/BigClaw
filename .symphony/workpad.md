# BIG-GO-1481 Workpad

## Plan

1. Confirm the repository-wide Python-file baseline, with explicit focus on `src/bigclaw`.
2. Add a `BIG-GO-1481` zero-baseline sweep report under `bigclaw-go/docs/reports/` capturing exact before/after counts and command evidence for the current checkout.
3. Add targeted regression coverage under `bigclaw-go/internal/regression/` to keep the repository and priority residual directories Python-free and to assert the lane report records the sweep state.
4. Run the targeted inventory commands and focused Go regression tests, then record exact commands and results.
5. Commit the scoped changes and push branch `BIG-GO-1481` to `origin`.

## Acceptance Criteria

- `.symphony/workpad.md` exists before tracked code edits and records plan, acceptance, and validation.
- `src/bigclaw` remains free of physical Python files in the current checkout.
- A new `BIG-GO-1481` sweep report records exact repository-wide and priority-path before/after counts for this lane.
- A targeted regression test enforces the Python-free repository baseline and the presence of the `BIG-GO-1481` sweep report.
- Validation commands and results are recorded exactly in the final output and reflected in the lane report.
- Changes stay scoped to this issue.

## Validation

- `find . -path '*/.git' -prune -o \( -iname '*.py' -o -iname '*.pyi' -o -iname '*.pyw' \) -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f \( -iname '*.py' -o -iname '*.pyi' -o -iname '*.pyw' \) 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1481(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
