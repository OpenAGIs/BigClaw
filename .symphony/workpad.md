# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python observability test file with Go coverage.
- Keep the scope limited to the existing Go observability package.
- Add run-level observability records, JSON persistence, repo-sync closeout types, and report/detail renderers needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_observability.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/observability`.
- Replacement coverage explicitly includes:
  - task-run logs, traces, artifacts, audits, and closeout persistence
  - repo-sync audit serialization
  - task-run markdown/html renderers
  - repo-sync audit report rendering
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/observability/taskrun.go bigclaw-go/internal/observability/taskrun_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/observability`
  - Result: `ok  	bigclaw-go/internal/observability	0.825s`

## Completed
- Added `bigclaw-go/internal/observability/taskrun.go` with task-run records, JSON persistence, repo-sync audit types, and task-run/repo-sync renderers.
- Added `bigclaw-go/internal/observability/taskrun_test.go` covering run persistence, repo-sync audit serialization, and task-run markdown/html rendering.
- Deleted `tests/test_observability.py`.
