# BIG-GO-946 Workpad

## Plan

1. Confirm the lane file list for `tests/conftest.py` plus the Python runtime/service/scheduler/workflow/orchestration/queue tests, then map each to the existing Go package and test surface under `bigclaw-go/internal/...`.
2. Add only the missing Go parity coverage needed to replace the Python lane cleanly, with focus on the service health/metrics surface if the current Go tests do not already cover it.
3. Remove the migrated Python lane assets once equivalent Go coverage or an explicit delete rationale exists for each file, while retaining shared Python bootstrap helpers that are still required by unrelated tests.
4. Run targeted Go tests for the touched packages, record the exact commands and outcomes, then commit and push the scoped branch.

## Acceptance

- Lane file list is explicit and limited to:
  - `tests/conftest.py`
  - `tests/test_runtime.py`
  - `tests/test_service.py`
  - `tests/test_scheduler.py`
  - `tests/test_workflow.py`
  - `tests/test_orchestration.py`
  - `tests/test_queue.py`
- Each Python test file has a Go replacement target or a clear delete plan.
- Validation commands and exact results are recorded after implementation.
- Remaining Python or non-Go assets in this lane are removed where replacement coverage exists.

## Go Replacement Targets

- `tests/conftest.py`
  - Retain for now. It has no Go analogue, but it is still required by unrelated surviving Python tests that import from `src/bigclaw`.
- `tests/test_runtime.py`
  - Replacement target: `bigclaw-go/internal/worker/runtime_test.go`
- `tests/test_service.py`
  - Replacement targets:
    - `bigclaw-go/internal/api/server_test.go`
    - `bigclaw-go/internal/repo/*.go` tests only if repo-governance parity is still missing
- `tests/test_scheduler.py`
  - Replacement target: `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_workflow.py`
  - Replacement target: `bigclaw-go/internal/workflow/engine_test.go`
- `tests/test_orchestration.py`
  - Replacement target: `bigclaw-go/internal/workflow/orchestration_test.go`
- `tests/test_queue.py`
  - Replacement targets:
    - `bigclaw-go/internal/queue/file_queue_test.go`
    - `bigclaw-go/internal/queue/memory_queue_test.go`
    - `bigclaw-go/internal/queue/sqlite_queue_test.go`

## Validation

- `cd bigclaw-go && go test ./internal/worker ./internal/api ./internal/scheduler ./internal/workflow ./internal/queue ./internal/repo`
- Additional focused `go test` commands for any specific package touched during parity work.
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/worker ./internal/api ./internal/scheduler ./internal/workflow ./internal/queue ./internal/repo`
  - Result: pass
  - Package results:
    - `ok  	bigclaw-go/internal/worker	(cached)`
    - `ok  	bigclaw-go/internal/api	3.619s`
    - `ok  	bigclaw-go/internal/scheduler	1.540s`
    - `ok  	bigclaw-go/internal/workflow	(cached)`
    - `ok  	bigclaw-go/internal/queue	(cached)`
    - `ok  	bigclaw-go/internal/repo	(cached)`

## Residual Risks

- The Python `test_service.py` governance helper may not map 1:1 to the current Go repo/service architecture; if no production Go equivalent exists, the lane must document that the Python helper is being retired rather than mechanically reimplemented.
- `tests/conftest.py` remains a cross-lane dependency until the rest of the Python test suite stops importing modules from `src/bigclaw`.
- The repository report `reports/go-migration-lanes-2026-03-29.md` was not present at the workspace root, so lane scoping is derived from the issue description and the repo contents in this branch.
