# BIG-GO-172 Python Asset Sweep

## Scope

`BIG-GO-172` is a wide residual pass over the remaining retired Python test
contracts that now map into Go test-heavy replacement directories. In this
checkout, the uncovered slice after the earlier residual-test sweeps is the
replacement surface for these retired tests:

- `tests/test_cross_process_coordination_surface.py`
- `tests/test_event_bus.py`
- `tests/test_execution_contract.py`
- `tests/test_execution_flow.py`
- `tests/test_github_sync.py`
- `tests/test_governance.py`
- `tests/test_memory.py`
- `tests/test_models.py`
- `tests/test_observability.py`
- `tests/test_orchestration.py`
- `tests/test_planning.py`
- `tests/test_queue.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`

This branch already has no physical `.py` assets left to delete, so the lane
hardens the remaining Go-owned replacement directories rather than claiming a
fresh Python deletion batch.

## Python Baseline

Repository-wide Python file count: `0`.

Audited remaining test-heavy replacement directory state:

- `bigclaw-go/internal/api`: `0` Python files
- `bigclaw-go/internal/contract`: `0` Python files
- `bigclaw-go/internal/events`: `0` Python files
- `bigclaw-go/internal/githubsync`: `0` Python files
- `bigclaw-go/internal/governance`: `0` Python files
- `bigclaw-go/internal/observability`: `0` Python files
- `bigclaw-go/internal/orchestrator`: `0` Python files
- `bigclaw-go/internal/planning`: `0` Python files
- `bigclaw-go/internal/policy`: `0` Python files
- `bigclaw-go/internal/product`: `0` Python files
- `bigclaw-go/internal/queue`: `0` Python files
- `bigclaw-go/internal/repo`: `0` Python files
- `bigclaw-go/internal/workflow`: `0` Python files

Explicit remaining Python asset list: none.

## Representative Go Or Native Replacement Paths

- `reports/BIG-GO-948-validation.md`
- `bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`
- `bigclaw-go/internal/api/coordination_surface.go`
- `bigclaw-go/internal/events/bus_test.go`
- `bigclaw-go/internal/contract/execution_test.go`
- `bigclaw-go/internal/workflow/orchestration_test.go`
- `bigclaw-go/internal/githubsync/sync_test.go`
- `bigclaw-go/internal/governance/freeze_test.go`
- `bigclaw-go/internal/policy/memory_test.go`
- `bigclaw-go/internal/workflow/model_test.go`
- `bigclaw-go/internal/observability/recorder_test.go`
- `bigclaw-go/internal/orchestrator/loop_test.go`
- `bigclaw-go/internal/planning/planning_test.go`
- `bigclaw-go/internal/queue/sqlite_queue_test.go`
- `bigclaw-go/internal/repo/gateway.go`
- `bigclaw-go/internal/repo/governance.go`
- `bigclaw-go/internal/repo/links.go`
- `bigclaw-go/internal/repo/registry.go`
- `bigclaw-go/internal/product/clawhost_rollout_test.go`

## Why This Sweep Is Safe

The retired Python tests in scope no longer exist in the branch, and their
behavioral coverage now lives in Go-owned packages that are dense with test and
contract surfaces. This lane records those directories as already Python-free
and pins representative replacement paths so a future Python reintroduction in
the remaining test-heavy areas trips a focused regression guard.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/internal/api bigclaw-go/internal/contract bigclaw-go/internal/events bigclaw-go/internal/githubsync bigclaw-go/internal/governance bigclaw-go/internal/observability bigclaw-go/internal/orchestrator bigclaw-go/internal/planning bigclaw-go/internal/policy bigclaw-go/internal/product bigclaw-go/internal/queue bigclaw-go/internal/repo bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the remaining test-heavy replacement directories stayed
  Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO172(RepositoryHasNoPythonFiles|RemainingTestHeavyReplacementDirectoriesStayPythonFree|RepresentativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.193s`
