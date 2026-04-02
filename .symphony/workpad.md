# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python control-center test file with equivalent Go tests.
- Keep the scope limited to `bigclaw-go/internal/reporting` and existing queue tests.
- Add the missing queue-control-center parity for blocked task actions and shared-view empty-state rendering.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_control_center.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/reporting`.
- Replacement coverage explicitly includes:
  - priority-ordered queue behavior through existing Go queue tests
  - queue control center queue/risk/media summaries
  - blocked task action rendering including reassign and pause-disabled semantics
  - shared-view empty-state rendering for queue control center
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/reporting ./internal/queue`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/reporting	1.624s`
    - `ok  	bigclaw-go/internal/queue	(cached)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `4 files changed, 85 insertions(+), 113 deletions(-)`
