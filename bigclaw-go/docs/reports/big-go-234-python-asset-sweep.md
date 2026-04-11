# BIG-GO-234 Python Asset Sweep

## Scope

`BIG-GO-234` (`Residual scripts Python sweep S`) records the already-zero
Python baseline for the residual scripts, wrappers, and CLI-helper surfaces
that now resolve through retained shell entrypoints and Go-native commands.

This lane focuses on the repo-level script helpers, the `bigclaw-go/scripts/*`
automation wrappers, and the checked-in Go CLI migration docs that describe the
supported replacement paths.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files
- `bigclaw-go/cmd`: `0` Python files
- `bigclaw-go/docs`: `0` Python files

This lane therefore lands as a regression-prevention sweep rather than a direct
Python-file deletion batch in this checkout.

## Go Or Native Replacement Paths

The retained Go/native wrapper and CLI-helper surface covering this sweep is:

- `docs/go-cli-script-migration-plan.md`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/docs/go-cli-script-migration.md`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find scripts bigclaw-go/scripts bigclaw-go/cmd bigclaw-go/docs -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the residual scripts, wrappers, and CLI-helper directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO234(RepositoryHasNoPythonFiles|ScriptAndCLIHelperSurfacesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	3.325s`

## Residual Risk

- This lane records and hardens an already Python-free baseline; it does not
  retire additional in-branch `.py` files by itself.
- The supported operator path still relies on retained shell wrappers around
  `bigclawctl`, so future drift between the wrappers, the docs, and the Go
  subcommands remains a maintenance risk outside this zero-Python guard.
