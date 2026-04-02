# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python evaluation test file with Go coverage.
- Keep the scope limited to a new self-contained Go evaluation package that reuses scheduler decisions.
- Add benchmark cases, replay comparisons, suite comparison/report rendering, and run-detail index generation needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_evaluation.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/evaluation`.
- Replacement coverage explicitly includes:
  - benchmark case scoring and pass/fail criteria
  - replay mismatch reporting
  - suite comparison and benchmark report rendering
  - run replay index/detail rendering
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/evaluation/evaluation.go bigclaw-go/internal/evaluation/evaluation_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/evaluation`
  - Result: `ok  	bigclaw-go/internal/evaluation	1.413s`

## Completed
- Added `bigclaw-go/internal/evaluation/evaluation.go` with benchmark cases, replay records/outcomes, suite comparison, and replay/report renderers.
- Added `bigclaw-go/internal/evaluation/evaluation_test.go` covering benchmark scoring, replay mismatches, suite reports, and run replay index/detail rendering.
- Deleted `tests/test_evaluation.py`.
