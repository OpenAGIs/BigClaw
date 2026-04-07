# BIG-GO-1561 Workpad

## Plan

1. Confirm the live repository baseline for physical Python files, with explicit
   focus on the requested `src/bigclaw` deletion tranche.
2. Add lane-scoped evidence that the tranche is already absent on disk and that
   the active Go/native replacement paths remain available.
3. Run targeted validation, record the exact commands and results, then commit
   and push the lane branch.

## Acceptance

- Repository reality is captured exactly for `BIG-GO-1561`.
- The lane records that `src/bigclaw` and the repository root already contain
  `0` physical `.py` files in this checkout.
- Exact Go/native replacement evidence for the retired `src/bigclaw` tranche
  lands in git.
- Targeted validation commands and outcomes are recorded.
- Changes are committed and pushed on `BIG-GO-1561`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1561(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Current Findings

- The recovered branch baseline already has `0` physical `.py` files
  repository-wide.
- `src/bigclaw` is not present on disk in this checkout.
- Prior migration lanes already moved the repo to a Go-only posture, so this
  issue can only ship exact replacement evidence and refreshed validation for
  the requested tranche.
