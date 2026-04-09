# BIG-GO-197 Workpad

## Plan

1. Audit the repository for remaining physical Python assets and identify a scoped set of high-impact Python-reference-heavy directories that are not yet pinned by a `BIG-GO-197` lane.
2. Add a focused Go regression test for `BIG-GO-197` that keeps the audited directories Python-free and verifies representative Go/native replacement assets still exist.
3. Add a matching lane report under `bigclaw-go/docs/reports/` that captures scope, baseline, retained replacement paths, and validation evidence.
4. Run targeted validation, record the exact commands and results here, then commit and push the branch.

## Acceptance

- `.symphony/workpad.md` exists before any code edits beyond this file.
- `bigclaw-go/internal/regression/big_go_197_zero_python_guard_test.go` exists and passes.
- `bigclaw-go/docs/reports/big-go-197-python-asset-sweep.md` exists and documents the audited directories, zero-Python baseline, retained replacement paths, and validation commands.
- Changes stay scoped to the `BIG-GO-197` regression/report lane.
- Targeted tests and repo scans are run and their exact commands/results are recorded.
- The branch is committed and pushed to the remote issue branch.

## Validation

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples bigclaw-go/internal/regression bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO197(RepositoryHasNoPythonFiles|HighImpactResidualDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Validation Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output.
- `find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples bigclaw-go/internal/regression bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`
  Result: no output.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO197(RepositoryHasNoPythonFiles|HighImpactResidualDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	5.270s`
