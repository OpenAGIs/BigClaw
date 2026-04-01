# BIG-GO-1037

## Plan
- Verify the reporting tranche already migrated on this branch in `bigclaw-go/internal/reporting`.
- Current scoped tranche:
  reporting/report studio: `tests/test_reports.py`
  Go replacements: `bigclaw-go/internal/reporting/reporting_test.go`, `bigclaw-go/internal/reporting/report_studio_test.go`, `bigclaw-go/internal/reporting/report_studio.go`
- Add only the missing Go reporting implementation/tests required for the migrated report-studio coverage.
- Delete the scoped Python reporting test file after the Go tests pass.
- Run targeted reporting Go tests, capture exact commands/results, then commit and push `BIG-GO-1037`.

## Acceptance
- Python file count decreases within the scoped reporting tranche.
- Go test coverage remains present in `bigclaw-go/internal/reporting` for the removed Python test behavior.
- No unrelated Python files are added or expanded.
- The final change can state exactly which Python file was removed and which Go files replace it.

## Validation
- `go test ./internal/reporting`
- `git diff --stat`
- `git status --short`
- `git push origin BIG-GO-1037`
