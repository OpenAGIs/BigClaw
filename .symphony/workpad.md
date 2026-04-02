# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python repo-collaboration test file with Go coverage.
- Keep the scope limited to a small Go collaboration surface plus a repo discussion board helper.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_repo_collaboration.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/collaboration` and `bigclaw-go/internal/repo`.
- Replacement coverage explicitly includes:
  - collaboration thread construction
  - repo discussion post conversion into collaboration comments
  - merged native and repo collaboration thread behavior
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/collaboration/thread.go bigclaw-go/internal/collaboration/thread_test.go bigclaw-go/internal/repo/discussion_board.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/collaboration ./internal/repo`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/collaboration	0.535s`
    - `ok  	bigclaw-go/internal/repo	(cached)`
