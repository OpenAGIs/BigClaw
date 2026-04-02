# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python risk test file with equivalent Go tests only.
- Keep the scope limited to existing Go risk and scheduler coverage.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_risk.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage remains explicit under `bigclaw-go/internal/risk` and `bigclaw-go/internal/scheduler`.
- Replacement coverage explicitly includes:
  - low-risk baseline scoring
  - medium-risk prod browser scoring
  - high-risk computed assessment and approval routing semantics
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/risk ./internal/scheduler`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/risk	(cached)`
    - `ok  	bigclaw-go/internal/scheduler	1.308s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `2 files changed, 8 insertions(+), 81 deletions(-)`
