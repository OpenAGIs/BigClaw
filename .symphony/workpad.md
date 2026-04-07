# BIG-GO-1571 Workpad

## Plan

1. Confirm the repository-wide Python baseline and the BIG-GO-1571 candidate
   residual-path list.
2. Record exact Go/native replacement evidence for the removed source, test,
   and script assets covered by this sweep.
3. Add issue-scoped regression coverage that keeps the candidate Python paths
   absent and asserts their replacement files and commands still exist.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1571`.

## Acceptance

- The BIG-GO-1571 candidate Python file list is explicitly covered in repo
  artifacts.
- Candidate paths are either already deleted or represented by exact
  Go/native replacement evidence; no new Python assets are introduced.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1571`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1571(RepositoryHasNoPythonFiles|CandidatePythonSweepPathsStayAbsent|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Outcome

- Repository baseline was already zero-`*.py`, so this issue landed as
  BIG-GO-1571-specific regression/report hardening instead of in-branch file
  deletion.
