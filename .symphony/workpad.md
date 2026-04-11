# BIG-GO-201 Workpad

## Plan

1. Replace the stale lane metadata with `BIG-GO-201` issue-scoped planning and
   validation targets tied to the retired `src/bigclaw` Python tree.
2. Add a lane-specific regression guard and evidence bundle for the zero-Python
   `src/bigclaw` state:
   - `bigclaw-go/internal/regression/big_go_201_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-201-python-asset-sweep.md`
   - `reports/BIG-GO-201-status.json`
   - `reports/BIG-GO-201-validation.md`
3. Run the target inventory checks and regression test, record exact commands
   and results, then commit and push the lane branch.

## Acceptance

- `BIG-GO-201` has lane-specific regression coverage for the repository-wide
  zero-Python baseline.
- The retired `src/bigclaw` tree stays absent and Python-free.
- The lane report and status artifacts record the active replacement paths and
  the exact validation commands/results.
- The resulting change set is committed and pushed to `origin/BIG-GO-201`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-201 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `if test ! -d /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/src/bigclaw; then echo absent; else echo present; fi`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO201(RepositoryHasNoPythonFiles|SrcBigclawTreeStaysAbsentAndPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- Baseline HEAD before lane changes: `36121df8`.
- Current inspection shows the repository-wide physical Python file inventory
  is already `0`.
- Current inspection also shows `src/bigclaw` is absent in this checkout, so
  this lane is a regression-hardening pass rather than an in-branch Python file
  deletion batch.
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-201 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  produced no output.
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/src/bigclaw -type f
  -name '*.py' 2>/dev/null | sort` produced no output.
- `if test ! -d /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/src/bigclaw;
  then echo absent; else echo present; fi` returned `absent`.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-201/bigclaw-go && go test
  -count=1 ./internal/regression -run
  'TestBIGGO201(RepositoryHasNoPythonFiles|SrcBigclawTreeStaysAbsentAndPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  returned `ok   bigclaw-go/internal/regression 0.195s`.
