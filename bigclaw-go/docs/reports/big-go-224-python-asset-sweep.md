# BIG-GO-224 Residual Scripts Python Sweep R

## Scope

`BIG-GO-224` locks down the residual script, wrapper, and CLI-helper surface
left after the repository-wide Python retirements.

This workspace already carries no tracked `.py` files, so the issue lands as a
regression-prevention lane focused on the remaining native helper inventory
under `scripts` and `bigclaw-go/scripts`.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

## Retained Native Helper Inventory

The supported residual helper inventory for this lane is:

- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `scripts/ops/bigclawctl`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Replacement And Ownership Path

The residual wrappers stay intentionally thin:

- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and
  `scripts/ops/bigclaw-symphony` proxy directly into `scripts/ops/bigclawctl`.
- `scripts/ops/bigclawctl` resolves the repo root and dispatches into
  `go run ./cmd/bigclawctl`.
- `scripts/dev_bootstrap.sh` validates the Go environment by exercising
  `./cmd/bigclawctl` and bootstrap tests instead of reviving Python setup code.
- `bigclaw-go/scripts/benchmark/run_suite.sh` and
  `bigclaw-go/scripts/e2e/run_all.sh` remain shell entrypoints over Go-owned
  automation commands.
- `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go` remains a native Go
  helper consumed by the E2E shell flow.

## Validation Commands And Results

- `find scripts bigclaw-go/scripts -type f | sort`
  Result:
  `bigclaw-go/scripts/benchmark/run_suite.sh`
  `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
  `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
  `bigclaw-go/scripts/e2e/ray_smoke.sh`
  `bigclaw-go/scripts/e2e/run_all.sh`
  `scripts/dev_bootstrap.sh`
  `scripts/ops/bigclaw-issue`
  `scripts/ops/bigclaw-panel`
  `scripts/ops/bigclaw-symphony`
  `scripts/ops/bigclawctl`
- `find scripts bigclaw-go/scripts -type f -name '*.py' | sort`
  Result: no output; the residual script directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO224(ResidualScriptDirectoriesStayPythonFree|ResidualScriptInventoryRemainsNative|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.201s`
