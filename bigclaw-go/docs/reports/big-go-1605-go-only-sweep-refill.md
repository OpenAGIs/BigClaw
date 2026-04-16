# BIG-GO-1605 Go-Only Sweep Refill

## Scope

Issue `BIG-GO-1605` covers the remaining reporting and observability Python
surface named in the migration brief:

- `src/bigclaw/observability.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/operations.py`
- `tests/test_observability.py`
- `tests/test_reports.py`
- `tests/test_evaluation.py`
- `tests/test_operations.py`

## Python Inventory

Repository-wide Python file count before lane changes: `0`.

Repository-wide Python file count after lane changes: `0`.

Explicit remaining Python asset list: none.

This lane therefore lands as regression-prevention evidence. The assigned
Python assets are already absent in this checkout, so the repo-visible work is
the added guardrail, the Go CLI completion, and refreshed operator guidance
for the Go-owned reporting and observability paths.

## Go-Owned Replacement Surfaces

- `src/bigclaw/observability.py` -> `bigclaw-go/internal/observability/recorder.go`
- `src/bigclaw/reports.py` -> `bigclaw-go/internal/reporting/reporting.go`
- `src/bigclaw/evaluation.py` -> `bigclaw-go/internal/evaluation/evaluation.go`
- `src/bigclaw/operations.py` -> `bigclaw-go/cmd/bigclawctl/reporting_commands.go`
- `tests/test_observability.py` -> `bigclaw-go/internal/observability/recorder_test.go`
- `tests/test_reports.py` -> `bigclaw-go/internal/reporting/reporting_test.go`
- `tests/test_evaluation.py` -> `bigclaw-go/internal/evaluation/evaluation_test.go`
- `tests/test_operations.py` -> `bigclaw-go/cmd/bigclawctl/reporting_commands_test.go`
- Weekly report API: `GET /v2/reports/weekly`
- Weekly report export API: `GET /v2/reports/weekly/export`
- Weekly report CLI: `bigclawctl reporting weekly`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `for path in src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/operations.py tests/test_observability.py tests/test_reports.py tests/test_evaluation.py tests/test_operations.py; do test ! -e "$path" && printf 'absent %s\n' "$path"; done`
  Result: printed `absent ...` for all eight assigned stale Python paths.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/reporting ./internal/api`
  Result: see `reports/BIG-GO-1605-validation.md` for the exact package-level outputs recorded for this lane.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run TestBIGGO1605`
  Result: see `reports/BIG-GO-1605-validation.md` for the exact regression output recorded for this lane.

Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1605 hardens that Go-only reporting/observability baseline rather than lowering the numeric file count further.
