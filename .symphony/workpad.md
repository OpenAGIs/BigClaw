# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python bootstrap test file with equivalent Go tests only.
- Keep the scope limited to `bigclaw-go/internal/bootstrap`.
- Add missing Go assertions for cache reuse, stale seed recovery, cleanup preservation, and validation report summary coverage.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_workspace_bootstrap.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/bootstrap`.
- Replacement coverage explicitly includes:
  - repo cache key derivation
  - cache root selection
  - first bootstrap worktree creation
  - second workspace warm-cache reuse
  - same workspace reuse
  - cleanup preserving shared cache
  - stale seed recovery without remote reclone
  - cleanup pruning bootstrap branch/worktree
  - validation report summary for three workspaces sharing one cache
- Changes remain scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/bootstrap`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/bootstrap	4.312s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `3 files changed, 163 insertions(+), 223 deletions(-)`
