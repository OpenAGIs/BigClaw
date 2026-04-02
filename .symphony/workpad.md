# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python scheduler test file with equivalent Go tests only.
- Keep the scope limited to existing Go scheduler coverage.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_scheduler.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage remains explicit under `bigclaw-go/internal/scheduler`.
- Replacement coverage explicitly includes:
  - high-risk routing
  - browser task routing
  - budget guardrail behavior
  - preemptive and policy-backed scheduler behavior
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/scheduler`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/scheduler	(cached)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `2 files changed, 9 insertions(+), 62 deletions(-)`
