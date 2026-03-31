# BIG-GO-1036 Workpad

## Plan
- Replace a scoped tranche of Python tests that already have functional Go coverage.
- Add small Go parity tests so Go test coverage explicitly increases in this change.
- Delete only the matching `tests/*.py` files for this tranche.
- Run targeted Go tests and record exact commands and outcomes.
- Commit and push the branch.
- Continue with a second scoped tranche for Python model/governance contract tests already covered by Go.

## Scoped Tranche
- `tests/test_dashboard_run_contract.py`
- `tests/test_github_sync.py`
- `tests/test_repo_board.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_triage.py`

## Scoped Tranche 2
- `tests/test_models.py`
- `tests/test_governance.py`

## Scoped Tranche 3
- `tests/test_queue.py`

## Scoped Tranche 4
- `tests/test_saved_views.py`

## Acceptance
- Python test file count decreases by deleting the scoped files above.
- Go test coverage increases via new parity assertions under existing Go test files.
- Replacement coverage remains in Go under `bigclaw-go/internal/product`, `bigclaw-go/internal/githubsync`, and `bigclaw-go/internal/repo`.
- Tranche 2 replacement coverage remains in Go under `bigclaw-go/internal/governance`, `bigclaw-go/internal/risk`, `bigclaw-go/internal/triage`, `bigclaw-go/internal/workflow`, `bigclaw-go/internal/billing`, and `bigclaw-go/internal/domain`.
- Tranche 3 replacement coverage remains in Go under `bigclaw-go/internal/queue`.
- Tranche 4 replacement coverage remains in Go under `bigclaw-go/internal/product`.
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
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed for tranche 2
  - Output summary: `3 files changed, 6 insertions(+), 332 deletions(-)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/governance ./internal/risk ./internal/triage ./internal/workflow ./internal/billing ./internal/domain`
  - Passed
  - Exact result:
    - `ok  	bigclaw-go/internal/governance	0.855s`
    - `ok  	bigclaw-go/internal/risk	0.441s`
    - `ok  	bigclaw-go/internal/triage	1.263s`
    - `ok  	bigclaw-go/internal/workflow	2.521s`
    - `ok  	bigclaw-go/internal/billing	2.094s`
    - `ok  	bigclaw-go/internal/domain	1.674s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/queue/file_queue.go bigclaw-go/internal/queue/file_queue_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/queue`
  - Passed before Python deletion
  - Exact result:
    - `ok  	bigclaw-go/internal/queue	25.918s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed for tranche 3
  - Output summary: `4 files changed, 124 insertions(+), 110 deletions(-)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/queue`
  - Passed after Python deletion
  - Exact result:
    - `ok  	bigclaw-go/internal/queue	(cached)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && gofmt -w bigclaw-go/internal/product/saved_views_test.go`
  - Passed
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/product`
  - Passed before Python deletion
  - Exact result:
    - `ok  	bigclaw-go/internal/product	0.463s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036 && git diff --stat`
  - Passed for tranche 4
  - Output summary: `3 files changed, 161 insertions(+), 159 deletions(-)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1036/bigclaw-go && go test ./internal/product`
  - Passed after Python deletion
  - Exact result:
    - `ok  	bigclaw-go/internal/product	(cached)`
