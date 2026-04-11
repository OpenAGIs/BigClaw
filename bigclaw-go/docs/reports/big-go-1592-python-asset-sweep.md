# BIG-GO-1592 Python Asset Sweep

## Scope

Go-only refill lane `BIG-GO-1592` covers the remaining Python asset slice
called out in the issue description, with explicit focus on the missing
`src/bigclaw/*.py` service modules and `tests/*.py` execution, console IA, and
observability tests.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

Explicit assigned Python asset list now absent:

- `src/bigclaw/__main__.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/service.py`
- `tests/test_console_ia.py`
- `tests/test_execution_flow.py`
- `tests/test_observability.py`

This checkout is already at a zero-Python baseline, so the lane lands as
regression-prevention evidence rather than a live `.py` deletion batch.

## Go-Owned Replacement Paths

The active Go-owned surface covering the retired Python slice remains:

- `bigclaw-go/internal/events/bus.go`
- `bigclaw-go/internal/orchestrator/loop.go`
- `bigclaw-go/internal/api/server.go`
- `bigclaw-go/internal/consoleia/consoleia.go`
- `bigclaw-go/internal/consoleia/consoleia_test.go`
- `bigclaw-go/internal/contract/execution.go`
- `bigclaw-go/internal/contract/execution_test.go`
- `bigclaw-go/internal/observability/audit.go`
- `bigclaw-go/internal/observability/audit_test.go`

These Go-owned surfaces keep the event bus, orchestration, service boundary,
console IA, execution flow, and observability behaviors on repo-native paths.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority residual directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1592(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|AssignedPythonAssetsStayAbsent|GoOwnedReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.255s`
