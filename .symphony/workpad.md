Issue: BIG-GO-1019

Plan
- Verify the new `cross-process-coordination-surface` Go command compiles, matches the removed Python report shape, and writes the canonical report artifact.
- Replace `cross_process_coordination_surface.py` with the Go-native `bigclawctl automation e2e cross-process-coordination-surface` entrypoint.
- Update directly coupled docs and regression coverage to point at the Go command and reflect the removed Python asset.
- Run targeted validation for the coordination-surface migration slice, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- Coordination-surface validation remains invokable through a Go-native CLI path that writes the same canonical report surface.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `gofmt -w bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command_test.go bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `go test ./cmd/bigclawctl -run 'TestAutomationCrossProcessCoordinationSurfaceBuildsReport'`
- `go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface --help`
- `go test ./internal/regression -run 'TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8CrossProcessCoordinationSurfaceStaysAligned'`
- `git diff --stat && git status --short`
