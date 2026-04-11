# BIG-GO-251 Workpad

Date: 2026-04-12

## Scope

Issue: `BIG-GO-251`

Title: `Residual src/bigclaw Python sweep V`

This lane is scoped to regression hardening for one uncovered retired
`src/bigclaw` tranche in a checkout that is already repository-wide
Python-free.

Assigned tranche: tranche 12 (`src/bigclaw/dsl.py`).

## Plan

1. Identify one uncovered `src/bigclaw` tranche and confirm the matching
   Go/native replacement surface already present in the repo.
2. Add a lane-specific Go regression guard under
   `bigclaw-go/internal/regression` for the assigned retired Python paths and
   replacement paths.
3. Add the matching lane sweep report under `bigclaw-go/docs/reports` and a
   validation record under `reports` with exact commands and observed results.
4. Run the targeted validation commands, capture exact outputs, then commit and
   push the lane to `origin/main`.

## Acceptance

- `.symphony/workpad.md` records plan, acceptance, and validation for this
  lane before code changes land.
- Changes remain scoped to `BIG-GO-251` lane artifacts and targeted regression
  coverage.
- The assigned retired `src/bigclaw` Python paths remain absent.
- The matching Go/native replacement paths remain present.
- The targeted regression guard passes.
- Validation artifacts record exact commands and exact results.
- The final lane commit is pushed to `origin/main`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-251 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `for path in /Users/openagi/code/bigclaw-workspaces/BIG-GO-251/src/bigclaw/dsl.py; do test ! -e "$path" && printf 'absent %s\n' "$path"; done`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-251/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO251(RepositoryHasNoPythonFiles|SrcBigclawTranche12PathRemainsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche12$'`
