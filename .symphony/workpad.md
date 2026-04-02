# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python repo-rollout test file with Go coverage.
- Keep the scope limited to `bigclaw-go/internal/reporting`.
- Add a small pilot-rollout and repo-narrative helper surface in Go.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_repo_rollout.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/reporting`.
- Replacement coverage explicitly includes:
  - pilot rollout scorecard recommendation
  - candidate gate evaluation and report rendering
  - weekly repo evidence narrative rendering
  - markdown/text/html repo narrative exports
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/reporting`
  - Result: `ok  	bigclaw-go/internal/reporting	0.869s`

## Completed
- Added Go test coverage in `bigclaw-go/internal/reporting/reporting_test.go` for pilot rollout scorecard, candidate gate evaluation, rollout gate report rendering, weekly repo evidence rendering, and repo narrative exports.
- Kept the helper implementation scoped to `bigclaw-go/internal/reporting/reporting.go`.
- Deleted `tests/test_repo_rollout.py`.
