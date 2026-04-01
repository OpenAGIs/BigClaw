## Plan

1. Delete the remaining Python migration-only queue and runtime scheduler smoke tests:
   - `tests/test_queue.py`
   - `tests/test_runtime_matrix.py`
   - `tests/test_scheduler.py`
2. Add a focused Go regression tranche that records those Python tests as retired and anchors the surviving Go-owned queue, scheduler, worker, and reporting test surfaces.
3. Keep validation scoped to the Go packages that now own those behaviors.
5. Add a focused Go regression test that asserts:
   - the deleted Python files are absent
   - the Go-owned replacement surfaces still exist
   - the tranche is recorded as `TestTopLevelModulePurgeTranche29`
6. Run targeted validation for the affected Go packages and the tranche regression, then measure the repository Python file count delta from the current baseline.
7. Commit with a message that explicitly lists deleted Python files and added Go files/Go tests, then push the branch.

## Acceptance

- Repository Python file count decreases from the continuation baseline of `21`.
- `tests/test_queue.py`, `tests/test_runtime_matrix.py`, and `tests/test_scheduler.py` are deleted.
- A Go regression test covers the deletion contract and the Go replacement files for the retired Python queue/runtime scheduler smoke tests.
- Targeted tests pass.
- Changes are committed and pushed on the working branch.

## Validation

- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/worker ./internal/reporting`
- `rg --files -g '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche29$'`
- `git status --short`
- `git log -1 --stat`

## Validation Results

- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/worker ./internal/reporting`
  - `ok  	bigclaw-go/internal/queue	27.732s`
  - `ok  	bigclaw-go/internal/scheduler	2.331s`
  - `ok  	bigclaw-go/internal/worker	2.163s`
  - `ok  	bigclaw-go/internal/reporting	0.935s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche29$'`
  - `ok  	bigclaw-go/internal/regression	1.701s`
- `rg --files -g '*.py' | wc -l`
  - `18`
