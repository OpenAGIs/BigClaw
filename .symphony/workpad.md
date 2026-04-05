# BIG-GO-1354

## Plan

- Inspect the current `scripts/ops` entrypoints and confirm whether any Python-backed paths remain.
- Replace redundant compatibility wrappers in `scripts/ops` with a single native dispatcher path that still routes to the Go `bigclawctl` subcommands.
- Add targeted validation for the replacement path and verify the repo remains free of `.py` assets.
- Commit the scoped change and push the branch to the configured remote.

## Acceptance

- `scripts/ops/*.py` replacement work lands as a concrete repo change in the ops entrypoint layer.
- Operator compatibility entrypoints still resolve to the correct Go `bigclawctl` subcommands.
- Targeted tests pass.
- `find . -name '*.py' | wc -l` remains at `0` or lower than baseline.

## Validation

- `go test ./cmd/bigclawctl`
- `find . -name '*.py' | wc -l`
- Manual wrapper checks via `scripts/ops/bigclawctl`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354/bigclaw-go && go test ./cmd/bigclawctl`
  - `ok  	bigclaw-go/cmd/bigclawctl	3.744s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354/bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche16`
  - `ok  	bigclaw-go/internal/regression	0.487s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354/bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'TestBIGGO1354|TestTopLevelModulePurgeTranche16'`
  - `ok  	bigclaw-go/cmd/bigclawctl	0.775s [no tests to run]`
  - `ok  	bigclaw-go/internal/regression	0.438s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  - no output
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  - no output
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && bash scripts/ops/bigclaw-issue --help`
  - exit `0`
  - output included `usage: bigclawctl issue [flags] [args...]`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && bash scripts/ops/bigclaw-panel --help`
  - exit `0`
  - output included `usage: bigclawctl panel [flags] [args...]`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && bash scripts/ops/bigclaw-symphony --help`
  - exit `0`
  - output included `usage: bigclawctl symphony [flags] [args...]`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1354 && find . -name '*.py' | wc -l`
  - `0`
