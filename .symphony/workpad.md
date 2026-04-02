# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python memory test file with Go coverage.
- Keep the scope limited to a small Go memory store surface.
- Add minimal persistence plus rule suggestion behavior that merges prior successful task patterns.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_memory.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/memory`.
- Replacement coverage explicitly includes:
  - remembering a successful task pattern
  - matching prior successful tasks by overlap
  - injecting matched acceptance criteria into the current suggestion
  - injecting matched validation plan steps into the current suggestion
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/memory/store.go bigclaw-go/internal/memory/store_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/memory`
  - Result: `ok  	bigclaw-go/internal/memory	0.513s`

## Completed
- Added `bigclaw-go/internal/memory/store.go` with a minimal JSON-backed task memory store for successful task patterns.
- Added `bigclaw-go/internal/memory/store_test.go` covering history reuse and rule suggestion merging.
- Deleted `tests/test_memory.py`.
