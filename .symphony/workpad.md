Issue: BIG-GO-1019

Plan
- Inspect the remaining `bigclaw-go/scripts/e2e/**` Python residue and pick a low-coupling live-validation slice with a stable checked-in report surface.
- Replace `mixed_workload_matrix.py` with a Go-native `bigclawctl automation e2e mixed-workload-matrix` command while preserving report shape, autostart behavior, and routing assertions.
- Update directly coupled docs to call the Go entrypoint and remove the migrated Python script.
- Run targeted validation for the mixed-workload migration slice, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- Mixed workload validation remains invokable through a Go-native CLI path that writes the same canonical report surface.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `go test ./cmd/bigclawctl -run 'TestAutomationMixedWorkloadMatrixBuildsReport'`
- `go run ./cmd/bigclawctl automation e2e mixed-workload-matrix --help`
- `go test ./internal/regression -run 'TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8CrossProcessCoordinationSurfaceStaysAligned'`
- `git diff --stat && git status --short`
