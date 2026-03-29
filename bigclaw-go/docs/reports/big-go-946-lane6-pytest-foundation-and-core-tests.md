# BIG-GO-946 Lane6 Pytest Foundation And Core Tests

## Lane File List

- `tests/conftest.py`
- `tests/test_runtime.py`
- `tests/test_service.py`
- `tests/test_scheduler.py`
- `tests/test_workflow.py`
- `tests/test_orchestration.py`
- `tests/test_queue.py`

## Go Replacement Or Delete Plan

- `tests/conftest.py`
  - Status: retained for now
  - Plan: delete after the remaining Python test lanes stop depending on the shared `src/` path bootstrap.
  - Reason: deleting it in this lane would break unrelated Python tests that still exist in the repository.
- `tests/test_runtime.py`
  - Replaced by `bigclaw-go/internal/worker/runtime_test.go`
- `tests/test_service.py`
  - Replaced in part by `bigclaw-go/internal/api/server_test.go`
  - Retired in part where the old Python static-file monitor helper has no direct production Go analogue.
- `tests/test_scheduler.py`
  - Replaced by `bigclaw-go/internal/scheduler/scheduler_test.go`
- `tests/test_workflow.py`
  - Replaced by `bigclaw-go/internal/workflow/engine_test.go`
  - Additional workflow closeout/orchestration behavior is covered by `bigclaw-go/internal/workflow/closeout_test.go`
- `tests/test_orchestration.py`
  - Replaced by `bigclaw-go/internal/workflow/orchestration_test.go`
- `tests/test_queue.py`
  - Replaced by:
    - `bigclaw-go/internal/queue/file_queue_test.go`
    - `bigclaw-go/internal/queue/memory_queue_test.go`
    - `bigclaw-go/internal/queue/sqlite_queue_test.go`

## Scope Applied In This Lane

- Added Go service endpoint coverage for `/healthz` and JSON `/metrics`.
- Removed lane-specific Python tests once matching Go coverage already existed.
- Left `tests/conftest.py` in place with an explicit delete plan because it is still shared by unrelated Python assets outside this lane.

## Validation Commands

```bash
cd bigclaw-go && go test ./internal/worker ./internal/api ./internal/scheduler ./internal/workflow ./internal/queue ./internal/repo
```

## Residual Risks

- The repository still contains other Python tests, so `tests/conftest.py` cannot be safely deleted in this lane without cross-lane coordination.
- The Python `test_service.py` monitor helper exercised an older Python-specific service surface; the Go control-plane service validates health and metrics endpoints instead of reproducing that helper verbatim.
