# BIG-GO-1580 Workpad

## Plan

1. Confirm the repository-wide physical Python baseline and verify the exact
   candidate paths from `BIG-GO-1580` are already absent in the current
   checkout.
2. Record the Go/native replacements for the targeted retired Python surface in
   a lane-specific report, including the candidate modules, legacy tests, and
   retired automation scripts listed in the issue.
3. Add a regression guard that keeps the repository Python-free, keeps the
   `BIG-GO-1580` candidate paths absent, asserts the replacement paths remain
   available, and verifies the lane report plus Go-only handoff documentation.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1580`.

## Acceptance

- The `BIG-GO-1580` candidate Python file list is explicitly recorded in the
  repo.
- Targeted Python modules, tests, and scripts remain deleted or replaced by
  Go/native surfaces, with no new compatibility shim introduced.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1580`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src tests bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1580(RepositoryHasNoPythonFiles|CandidatePathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState|GoMainlineCutoverHandoffStaysGoOnly)$'`

## Outcome

- Repository-wide and focused candidate-path Python scans both returned no
  output, confirming the checkout remains physically Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1580(RepositoryHasNoPythonFiles|CandidatePathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState|GoMainlineCutoverHandoffStaysGoOnly)$'`
  passed with `ok  	bigclaw-go/internal/regression	1.193s`.
- Commit/push remains pending while git metadata is re-established in this
  tarball-materialized workspace.
