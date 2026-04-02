# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python runtime-matrix test file with Go coverage.
- Keep the scope limited to `bigclaw-go/internal/worker` with scheduler parity relying on existing Go scheduler tests.
- Add a minimal tool-runtime surface in Go for multi-tool lifecycle and policy/audit assertions.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_runtime_matrix.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/worker`.
- Replacement coverage explicitly includes:
  - multi-tool worker lifecycle completion
  - blocked/allowed tool policy behavior with audit outcomes
  - scheduler medium routing already covered by existing Go scheduler tests
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/worker/tool_runtime.go bigclaw-go/internal/worker/tool_runtime_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/worker ./internal/scheduler`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/worker	1.820s`
    - `ok  	bigclaw-go/internal/scheduler	(cached)`
