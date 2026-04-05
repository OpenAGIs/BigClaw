# BIG-GO-1445 Workpad

## Plan
- Materialize the repository from `origin/main` into the current workspace and create the working branch for `BIG-GO-1445`.
- Inventory remaining Python assets with emphasis on `src/bigclaw/*.py`, `tests/*.py`, `scripts/*.py`, and `bigclaw-go/scripts/*.py`.
- Remove or shrink a bounded batch of residual Python files in scope, preferring deletion where a Go path already exists and thin wrappers only when compatibility requires them.
- Run targeted validation for the touched area, capture exact commands and results, then commit and push the branch.

## Acceptance
- Produce a concrete list of the remaining Python assets relevant to this lane after repo materialization.
- Reduce the physical Python file count in the repository as the primary outcome. Baseline was already `0`, so this lane closes by documenting and guarding the zero-Python state.
- Document the Go replacement path for each removed or shrunk Python asset.
- Record exact validation commands and outcomes.

## Validation
- `rg --files -g '*.py'`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` -> no output
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort` -> no output
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1445/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1445(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'` -> `ok  	bigclaw-go/internal/regression	3.183s`
- `git diff --stat`
