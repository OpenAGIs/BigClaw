# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python planning test file with Go coverage.
- Keep the scope limited to a new pure-data Go planning package plus the existing governance audit type.
- Add candidate backlog evaluation, entry-gate decisions, four-week execution planning, static builders, and report rendering needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_planning.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/planning`.
- Replacement coverage explicitly includes:
  - candidate backlog ranking and gate evaluation
  - baseline-audit handling through `governance.ScopeFreezeAudit`
  - four-week execution-plan rollups and validation
  - static candidate/gate/plan builders and report rendering
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/planning`
  - Result: `ok  	bigclaw-go/internal/planning	1.089s`

## Completed
- Added `bigclaw-go/internal/planning/planning.go` with candidate backlog models, entry-gate evaluation, four-week execution planning, static builders, and report rendering.
- Added `bigclaw-go/internal/planning/planning_test.go` covering ranking, baseline handling, report output, execution-plan rollups, and built backlog traceability.
- Deleted `tests/test_planning.py`.
