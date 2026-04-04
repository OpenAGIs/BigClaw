# BIG-GO-1230 Python Asset Sweep

Date: `2026-04-05`

## Remaining Python asset inventory

- `src/bigclaw/*.py`: `0`
- `tests/*.py`: `0`
- `scripts/*.py`: `0`
- `bigclaw-go/scripts/*.py`: `0`
- repository-wide tracked `.py` files: `0`

## Go-owned replacement paths

- `bash scripts/ops/bigclawctl ...`
- `bash scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl`
- `bigclaw-go/internal/regression/big_go_1220_zero_python_guard_test.go`
- `bigclaw-go/internal/regression/big_go_1230_python_asset_sweep_test.go`

## Validation commands

- `find . -type f -name '*.py' | sort`
- `git ls-files '*.py'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO(1220|1230)|TestPythonTestTranche17Removed'`
