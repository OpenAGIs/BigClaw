# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python event-bus test file with Go coverage.
- Keep the scope limited to a new self-contained Go event-bus package.
- Add event publication, subscriber callbacks, JSON-backed run persistence, and the three status-transition behaviors needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_event_bus.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/eventbus`.
- Replacement coverage explicitly includes:
  - pull-request comment approval transitions
  - CI-completed transitions
  - task-failed transitions
  - subscriber callbacks and persisted audit trails
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/eventbus/eventbus.go bigclaw-go/internal/eventbus/eventbus_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/eventbus`
  - Result: `ok  	bigclaw-go/internal/eventbus	1.718s`

## Completed
- Added `bigclaw-go/internal/eventbus/eventbus.go` with a self-contained event bus, subscriber callbacks, and JSON-backed run persistence.
- Added `bigclaw-go/internal/eventbus/eventbus_test.go` covering pull-request comment approval, CI completion, and task failure transitions plus persisted audit trails.
- Deleted `tests/test_event_bus.py`.
