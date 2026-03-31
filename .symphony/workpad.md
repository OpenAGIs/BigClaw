# BIG-GO-1036 Workpad

## Plan
- Replace a scoped tranche of Python tests that already have functional Go coverage.
- Add small Go parity tests so Go test coverage explicitly increases in this change.
- Delete only the matching `tests/*.py` files for this tranche.
- Run targeted Go tests and record exact commands and outcomes.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_dashboard_run_contract.py`
- `tests/test_github_sync.py`
- `tests/test_repo_board.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_triage.py`

## Acceptance
- Python test file count decreases by deleting the scoped files above.
- Go test coverage increases via new parity assertions under existing Go test files.
- Replacement coverage remains in Go under `bigclaw-go/internal/product`, `bigclaw-go/internal/githubsync`, and `bigclaw-go/internal/repo`.
- Changes stay scoped to this tranche only.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/product/dashboard_run_contract_test.go bigclaw-go/internal/githubsync/sync_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed
  - Output summary: `11 files changed, 136 insertions(+), 425 deletions(-)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/product ./internal/repo ./internal/githubsync`
  - First run failed in `internal/product` because a new round-trip assertion compared decoded `map[string]any` structures too strictly.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/product ./internal/repo ./internal/githubsync`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/product	0.470s`
    - `ok  	bigclaw-go/internal/repo	(cached)`
    - `ok  	bigclaw-go/internal/githubsync	(cached)`
