# BIG-GO-1354 Python Asset Sweep

## Scope

Heartbeat refill lane `BIG-GO-1354` records the `scripts/ops/*.py` replacement state and keeps the repository's residual Python inventory at zero while consolidating the remaining compatibility entrypoints onto the Go-native `bigclawctl` dispatcher.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

This lane therefore lands as a concrete replacement batch rather than a direct Python-file deletion in this checkout: the duplicate `scripts/ops` compatibility wrappers now collapse into symlinks that dispatch through the single Go-owned `scripts/ops/bigclawctl` path.

## Go Replacement Paths

The Go-only replacement surface for the retired `scripts/ops/*.py` helpers now includes:

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority residual directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'TestBIGGO1354|TestTopLevelModulePurgeTranche16'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl` and `ok  	bigclaw-go/internal/regression`; the Go dispatcher and regression guards both passed.
