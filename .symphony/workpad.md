# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python orchestration test file with Go coverage.
- Keep the scope limited to `bigclaw-go/internal/workflow` plus existing scheduler/worker coverage.
- Add only the missing orchestration plan renderer and parity tests needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_orchestration.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/workflow`, `bigclaw-go/internal/scheduler`, and `bigclaw-go/internal/worker`.
- Replacement coverage explicitly includes:
  - cross-department orchestration planning
  - premium policy cost and blocked-department handling
  - orchestration plan rendering
  - scheduler/runtime orchestration handoff behavior
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/workflow/orchestration.go bigclaw-go/internal/workflow/orchestration_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/workflow ./internal/scheduler ./internal/worker`
  - Result:
    - `ok  	bigclaw-go/internal/workflow	1.083s`
    - `ok  	bigclaw-go/internal/scheduler	1.254s`
    - `ok  	bigclaw-go/internal/worker	2.352s`

## Completed
- Added `RenderOrchestrationPlan` to `bigclaw-go/internal/workflow/orchestration.go`.
- Added Go test coverage in `bigclaw-go/internal/workflow/orchestration_test.go` for expanded blocked-department cost handling and orchestration-plan rendering.
- Reused existing Go coverage in `internal/workflow`, `internal/scheduler`, and `internal/worker` for orchestration planning, policy application, and handoff behavior.
- Deleted `tests/test_orchestration.py`.
