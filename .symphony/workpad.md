# BIG-GO-1003 Workpad

## Scope

Target the remaining Python compatibility surfaces for the `runtime/service/scheduler/workflow/orchestration/queue` batch under `src/bigclaw`.

Initial batch inventory:

- Physical Python files:
  - `src/bigclaw/runtime.py`
- Logical compatibility surfaces still backed by that file:
  - `bigclaw.runtime`
  - `bigclaw.queue`
  - `bigclaw.orchestration`
  - `bigclaw.scheduler`
  - `bigclaw.workflow`
  - `bigclaw.service`

Repository-wide Python file count before this lane: `108`
Targeted physical Python file count before this lane: `1`

## Plan

1. Verify the exact residual Python inventory for this batch and confirm how the logical compatibility surfaces are wired.
2. Add a repository-local inventory map for the legacy runtime batch so the kept/replaced rationale and Go replacement targets are explicit in code.
3. Add targeted tests that pin the single-file residual state and the compatibility-export wiring.
4. Run targeted validation with local `PYTHONPATH` so this branch is tested instead of any globally installed `bigclaw` package.
5. Record keep/delete/replace rationale and Python count impact, then commit and push the scoped change set.

## Acceptance

- Produce the exact residual Python file list for the `runtime/service/scheduler/workflow/orchestration/queue` batch.
- Reduce the targeted Python file count if a safe deletion is possible in this lane; otherwise document why the residual file must stay.
- Record delete/replace/keep rationale for each targeted logical surface.
- Report repository-wide and targeted Python file count impact.

## Validation

- `find src/bigclaw -maxdepth 1 -type f \( -name 'runtime.py' -o -name 'service.py' -o -name 'scheduler.py' -o -name 'workflow.py' -o -name 'orchestration.py' -o -name 'queue.py' \) | sort`
- `find . -name '*.py' | wc -l`
- `PYTHONPATH=$PWD/src python3 -m pytest -q tests/test_runtime_matrix.py tests/test_queue.py tests/test_orchestration.py tests/test_scheduler.py`
- `git status --short`
- `git log -1 --stat`

## Results

### Residual Inventory

- Physical Python files in this batch:
  - `src/bigclaw/runtime.py`
- Logical compatibility surfaces still backed by that file:
  - `bigclaw.runtime`
  - `bigclaw.queue`
  - `bigclaw.orchestration`
  - `bigclaw.scheduler`
  - `bigclaw.workflow`
  - `bigclaw.service`

### Disposition

- `src/bigclaw/runtime.py`
  - Kept.
  - Reason: it is the final physical Python compatibility file for this batch and still backs the exported `queue`, `orchestration`, `scheduler`, `workflow`, and `service` surfaces through `bigclaw.__init__`.
- `bigclaw.runtime`
  - Replaced on the Go mainline by `bigclaw-go/internal/worker/runtime.go`, but kept as the local compatibility implementation.
- `bigclaw.queue`
  - Replaced on the Go mainline by `bigclaw-go/internal/queue/queue.go`, but currently re-exported from `runtime.py`.
- `bigclaw.orchestration`
  - Replaced on the Go mainline by `bigclaw-go/internal/workflow/orchestration.go`, but currently re-exported from `runtime.py`.
- `bigclaw.scheduler`
  - Replaced on the Go mainline by `bigclaw-go/internal/scheduler/scheduler.go`, but currently re-exported from `runtime.py`.
- `bigclaw.workflow`
  - Replaced on the Go mainline by `bigclaw-go/internal/workflow/engine.go`, but currently re-exported from `runtime.py`.
- `bigclaw.service`
  - Replaced on the Go mainline by `bigclaw-go/cmd/bigclawd/main.go`, but currently re-exported from `runtime.py`.

### Count Impact

- Repository Python files before: `108`
- Repository Python files after: `108`
- Targeted physical Python files before: `1`
- Targeted physical Python files after: `1`
- Net reduction: `0`

### Validation Record

- `find src/bigclaw -maxdepth 1 -type f \( -name 'runtime.py' -o -name 'service.py' -o -name 'scheduler.py' -o -name 'workflow.py' -o -name 'orchestration.py' -o -name 'queue.py' \) | sort`
  - Result: only `src/bigclaw/runtime.py`
- `find . -name '*.py' | wc -l`
  - Result: `108`
- `PYTHONPATH=$PWD/src python3 -m pytest -q tests/test_runtime_matrix.py tests/test_queue.py tests/test_orchestration.py tests/test_scheduler.py`
  - Result: `18 passed in 0.10s`
- `git status --short`
  - Result: only `.symphony/workpad.md`, `src/bigclaw/__init__.py`, `src/bigclaw/runtime.py`, and `tests/test_runtime_matrix.py` modified for this lane
