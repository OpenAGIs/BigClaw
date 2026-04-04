## BIG-GO-1171

### Plan
- Confirm the current repository Python footprint, with emphasis on `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add commit-ready, Go-native regression evidence that the repository does not regain Python assets, since the live `.py` count is already zero.
- Run targeted validation for the new regression guard plus the repository-wide `.py` count check.
- Commit and push the scoped lane changes.

### Acceptance
- `find . -name '*.py' | wc -l` remains at `0`, which satisfies the lane objective because there are no residual Python assets left to delete in the prioritized areas.
- The branch contains concrete replacement evidence in Go-native regression coverage that enforces the zero-Python repository state.
- Validation commands and exact results are recorded for this lane.

### Validation
- `find . -name '*.py' | wc -l`
- `go test ./bigclaw-go/internal/regression -run TestRepositoryPythonAssetCountIsZero -count=1`
- `git status --short`

### Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run TestRepositoryPythonAssetCountIsZero -count=1` -> `ok  	bigclaw-go/internal/regression	0.464s`
- `git status --short` -> `M .symphony/workpad.md` and `?? bigclaw-go/internal/regression/repository_python_asset_count_test.go`
