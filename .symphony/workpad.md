# BIG-GO-249 Workpad

## Plan

1. Confirm the repository-wide Python inventory and inspect hidden, nested, and
   overlooked auxiliary directories that fit `BIG-GO-249`:
   - `.githooks`
   - `.github`
   - `.symphony`
   - `docs/reports`
   - `bigclaw-go/docs/reports/live-shadow-runs`
   - `bigclaw-go/docs/reports/live-validation-runs`
2. Add issue-scoped evidence for `BIG-GO-249`:
   - `bigclaw-go/internal/regression/big_go_249_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-249-python-asset-sweep.md`
   - `reports/BIG-GO-249-validation.md`
   - `reports/BIG-GO-249-status.json`
3. Run the targeted inventory checks and regression test, then commit and push
   the issue-scoped changes to the current remote branch.

## Acceptance

- `BIG-GO-249` records a repository-wide Python file count of `0` and verifies
  that the assigned hidden and nested auxiliary directories remain Python-free.
- A lane-specific Go regression guard covers the repo-wide baseline, the chosen
  overlooked directories, the retained native evidence paths, and the lane
  report contents.
- The validation report and status artifact capture the exact commands, exact
  observed results, git commit metadata, and the zero-Python baseline caveat.
- The change set stays scoped to this issue and is committed and pushed.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-249 -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/.github /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/.symphony /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/bigclaw-go/docs/reports/live-validation-runs -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-249/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO249(RepositoryHasNoPythonFiles|HiddenNestedAuxiliaryDirectoriesStayPythonFree|NativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-12: Initial inspection found no tracked or untracked Python files in
  the checkout, so `BIG-GO-249` will harden the zero-Python baseline for
  hidden, nested, and overlooked auxiliary surfaces rather than deleting
  in-branch `.py` files.
