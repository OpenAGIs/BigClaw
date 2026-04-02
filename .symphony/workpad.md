# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python reports test file with Go coverage.
- Keep the scope limited to `bigclaw-go/internal/reporting` and adjacent existing Go packages only if needed for type reuse.
- Add Go report models, file writers, closure-gate helpers, portfolio rollups, and renderers needed to remove the Python file.
- Reuse the existing Go orchestration/takeover reporting adapters where they already match the Python coverage.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_reports.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/reporting`.
- Replacement coverage explicitly includes:
  - issue validation report writing and closure gates
  - report studio, pilot scorecard, pilot portfolio, launch/final-delivery checklist rendering
  - shared-view collaboration rendering
  - auto-triage center, takeover queue, orchestration portfolio, and billing/entitlements reporting
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/report_surfaces.go bigclaw-go/internal/reporting/report_surfaces_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/reporting`
  - Result: `ok  	bigclaw-go/internal/reporting	0.855s`

## Completed
- Extended `bigclaw-go/internal/reporting/reporting.go` to carry the richer orchestration/takeover/shared-view report state required by the Python tests.
- Added `bigclaw-go/internal/reporting/report_surfaces.go` with validation report writing, closure gates, report studio, pilot scorecards, launch/final-delivery checklists, shared-view rendering, auto-triage center, orchestration portfolio, and billing/entitlements report helpers.
- Added `bigclaw-go/internal/reporting/report_surfaces_test.go` covering the replaced Python report behaviors with targeted Go tests.
- Deleted `tests/test_reports.py`.
