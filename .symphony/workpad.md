# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python audit-events test file with Go coverage.
- Keep the scope limited to existing Go observability, scheduler, workflow, and reporting surfaces.
- Add only the missing report adapters needed to express canonical handoff and takeover audit events from ledger-like entries.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_audit_events.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/observability`, `bigclaw-go/internal/scheduler`, `bigclaw-go/internal/workflow`, and `bigclaw-go/internal/reporting`.
- Replacement coverage explicitly includes:
  - audit-event spec validation
  - required-field enforcement for canonical audit events
  - scheduler and approval-path audit payloads
  - reporting adapters that consume canonical handoff/takeover audit events
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/observability ./internal/scheduler ./internal/workflow ./internal/reporting ./internal/worker`
  - Result:
    - `ok  	bigclaw-go/internal/observability	0.506s`
    - `ok  	bigclaw-go/internal/scheduler	(cached)`
    - `ok  	bigclaw-go/internal/workflow	(cached)`
    - `ok  	bigclaw-go/internal/reporting	0.927s`
    - `ok  	bigclaw-go/internal/worker	(cached)`

## Completed
- Added ledger-entry reporting adapters in `bigclaw-go/internal/reporting/reporting.go` for orchestration canvas and takeover queue derivation from canonical audit events.
- Added Go test coverage in `bigclaw-go/internal/reporting/reporting_test.go` for canonical handoff/takeover event consumption.
- Reused existing Go coverage in `internal/observability`, `internal/scheduler`, `internal/workflow`, and `internal/worker` for audit-spec validation, required-field enforcement, handoff behavior, and acceptance annotation behavior.
- Deleted `tests/test_audit_events.py`.
