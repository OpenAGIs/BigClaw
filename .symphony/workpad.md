# BIG-GO-1036 Workpad

## Plan
- Replace a scoped tranche of remaining Python tests with equivalent Go tests only.
- Keep the scope limited to queue/control-center and workflow/DSL test surfaces that already exist in `bigclaw-go`.
- Add any missing Go assertions needed to cover the deleted Python behavior.
- Delete the matched Python test files once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_dsl.py`

## Acceptance
- Python test file count decreases by deleting the scoped files above.
- Go test coverage increases under `bigclaw-go/internal/workflow`.
- Replacement coverage explicitly includes:
  - workflow definition parsing/rendering
  - workflow acceptance/manual-approval behavior
  - invalid workflow step rejection
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/workflow/definition.go bigclaw-go/internal/workflow/definition_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/workflow`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/workflow	0.996s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `4 files changed, 39 insertions(+), 241 deletions(-)`
