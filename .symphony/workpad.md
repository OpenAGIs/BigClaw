# BIG-GO-1185 Workpad

## Plan
- Verify the live repository baseline for physical Python files, with focused
  checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard that fails if any `.py` asset
  reappears anywhere in the repository or in the issue's priority directories.
- Record the explicit remaining Python asset inventory and Go replacement path
  in lane-specific report artifacts.
- Run targeted validation, capture exact commands and results, then commit and
  push the branch.

## Acceptance
- The lane-specific remaining Python asset list is explicit and auditable.
- The repository keeps a physical Python file count of `0`, including the
  priority residual directories for this heartbeat refill lane.
- The Go replacement path and validation commands are documented in committed
  artifacts.

## Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1185 -name '*.py' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1185/bigclaw-go && go test ./internal/regression -run 'TestBIGGO1185(RemainingPythonAssetInventoryIsEmpty|PriorityResidualDirectoriesStayPythonFree)$'`
- `git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1185 status --short`
