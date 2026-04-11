# BIG-GO-244 Python Asset Sweep

## Scope

`BIG-GO-244` records the residual root script, wrapper, and CLI-helper surface
that still matters operationally after the earlier Python-entrypoint removals.

This sweep focuses on the active root helper inventory under `scripts` and
`scripts/ops`, plus the Go-owned CLI entrypoints under `bigclaw-go/scripts`
and `bigclaw-go/cmd`.

## Residual Scan Detail

- Repository-wide Python file count: `0`.
- `scripts`: `0` Python files
- `scripts/ops`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files
- `bigclaw-go/cmd`: `0` Python files

This checkout was already physically Python-free before the lane started, so
the shipped work hardens the shell-wrapper and Go-CLI contract rather than
deleting in-branch `.py` assets.

## Supported Wrapper And CLI Surface

- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `docs/local-tracker-automation.md`
- `docs/go-cli-script-migration-plan.md`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`
- `bigclaw-go/cmd/bigclawd/main.go`

## Wrapper Contract

- The root helper inventory remains limited to `scripts/dev_bootstrap.sh` and
  the flat `scripts/ops` wrapper set.
- `scripts/ops/bigclawctl` stays a shell wrapper over `go run ./cmd/bigclawctl`.
- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and
  `scripts/ops/bigclaw-symphony` stay thin aliases into `scripts/ops/bigclawctl`.
- Operator docs keep pointing at `bash scripts/ops/bigclawctl ...` rather than
  retired Python entrypoints.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find scripts scripts/ops bigclaw-go/scripts bigclaw-go/cmd -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the residual script, wrapper, and CLI-helper surface remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO244(RepositoryHasNoPythonFiles|ResidualScriptAndCLIHelperSurfacesStayPythonFree|SupportedWrapperAndCLIPathsRemainAvailable|WrapperInventoryMatchesContract|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.207s`
