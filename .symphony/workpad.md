# BIG-GO-986

## Plan
- Audit `tests/conftest.py` and the core Python runtime-chain tests in scope for this batch.
- Map each in-scope Python test to existing Go coverage under `bigclaw-go/internal/...`, then add any missing Go assertions needed to preserve behavior.
- Delete the redundant Python test files once equivalent Go coverage exists.
- Run targeted Go tests plus repository-level Python file counts, then record exact commands and results.
- Commit the scoped change set and push the branch to `origin`.

## Acceptance
- Enumerate the Python files handled by this batch, centered on `tests/conftest.py` and core runtime-chain tests.
- Reduce Python file count in the targeted area by deleting files that are now replaced by Go tests.
- Document keep/replace/delete rationale in the final report.
- Report the repository-wide Python file count delta caused by this batch.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/workflow`
- `git status --short`
- `git log -1 --stat`

## Results
- Batch file list:
  - `tests/conftest.py`
  - `tests/test_runtime.py`
  - `tests/test_execution_flow.py`
  - `tests/test_workflow.py`
- Disposition:
  - `tests/conftest.py` kept because the remaining Python tests still rely on its `src/` path bootstrap.
  - `tests/test_runtime.py` deleted and replaced by Go coverage in `bigclaw-go/internal/worker/runtime_test.go` plus `bigclaw-go/internal/scheduler/scheduler_test.go`.
  - `tests/test_execution_flow.py` deleted and replaced by Go runtime-chain coverage in `bigclaw-go/internal/worker/runtime_test.go`.
  - `tests/test_workflow.py` deleted and replaced by Go workflow coverage in `bigclaw-go/internal/workflow/engine_test.go`, `bigclaw-go/internal/workflow/closeout_test.go`, and `bigclaw-go/internal/workflow/orchestration_test.go`.
- Supporting rewires:
  - updated `src/bigclaw/planning.py` and `tests/test_planning.py` so validation/evidence references now point at Go-native replacements instead of deleted Python tests
  - added Go-native workpad journal reload/replay helpers in `bigclaw-go/internal/workflow/engine.go` with coverage in `bigclaw-go/internal/workflow/engine_test.go`
- Python count impact:
  - repo-wide `*.py`: `116 -> 113` (`-3`)
  - `tests/*.py`: `41 -> 38` (`-3`)
- Validation results:
  - `find . -name '*.py' | wc -l` -> `113`
  - `cd bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/workflow` -> `ok`
  - `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q` -> `14 passed in 0.19s`
  - `git status --short` -> clean after commit/push
  - `git log -1 --stat` -> commit `b1fdd413a94e624055f78805e82e0efea41a3610`
- Git:
  - commit: `b1fdd413a94e624055f78805e82e0efea41a3610`
  - push: `git push origin main` -> `d295f07..b1fdd41  main -> main`
