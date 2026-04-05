# BIG-GO-1369 Python Asset Sweep

## Scope

`BIG-GO-1369` records concrete Go-native replacement evidence for the `scripts/ops` python replacement sweep.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

This checkout therefore lands as a replacement-evidence sweep rather than a direct Python-file deletion batch.

## Go Or Native Replacement Paths

The retained replacement surface for the old `scripts/ops` Python lane is:

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/internal/regression/big_go_1369_zero_python_guard_test.go`

## Validation Commands And Results

- `find . -name '*.py' | wc -l`
  Result: `0`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1369(RepositoryHasNoPythonFiles|OpsReplacementPathsRemainAvailable|OpsWrappersStayGoNative|BigclawctlWrapperResolvesRelativeRepoFromInvocationDir|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	1.324s`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	3.138s`
- `bash scripts/ops/bigclawctl --help`
  Result: printed the Go-native root usage for `bigclawctl`.
- `bash scripts/ops/bigclaw-issue --help`
  Result: printed `usage: bigclawctl issue [flags] [args...]`.
- `bash scripts/ops/bigclaw-panel --help`
  Result: printed `usage: bigclawctl panel [flags] [args...]`.
- `bash scripts/ops/bigclaw-symphony --help`
  Result: printed `usage: bigclawctl symphony [flags] [args...]`.
- `bash scripts/ops/bigclawctl local-issues list --repo .. --local-issues local-issues.json --json`
  Result: covered by `TestBIGGO1369BigclawctlWrapperResolvesRelativeRepoFromInvocationDir`, which verified the wrapper resolves invocation-relative `--repo` values to the repository root and emits the absolute `local_issues` path.
