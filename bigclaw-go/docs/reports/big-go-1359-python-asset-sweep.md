# BIG-GO-1359 Python Asset Sweep

## Scope

Heartbeat refill lane `BIG-GO-1359` records the remaining physical Python asset inventory for the repository and replaces the active Ray smoke validation default with a shell-native entrypoint so the checked-in governance/reporting surface stays practical for a Go-only repository.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

This checkout therefore lands as a native-replacement sweep rather than a direct Python-file deletion batch because there are no physical `.py` assets left to remove in-branch.

## Native Replacement Landed

The active Ray smoke validation path now stays shell-native instead of defaulting to inline Python:

- `bigclaw-go/scripts/e2e/ray_smoke.sh` now defaults `BIGCLAW_RAY_SMOKE_ENTRYPOINT` to `sh -c 'echo hello from ray'`
- `bigclaw-go/docs/e2e-validation.md` no longer lists `python3` as a prerequisite for the active smoke path
- `bigclaw-go/docs/e2e-validation.md` now documents `export BIGCLAW_RAY_SMOKE_ENTRYPOINT="sh -c 'echo 123'"` as the override example

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority residual directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1359(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|RaySmokeReplacementPathsRemainAvailable|LaneReportCapturesNativeReplacement)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.583s`
