# BIG-GO-1583 Tests Bucket-A Sweep

Date: 2026-04-09

## Scope

Issue: `BIG-GO-1583`

Title: `Strict bucket lane 1583: tests/*.py bucket A`

This lane records the already-zero repository baseline for the retired root
`tests` surface and locks down a focused bucket-A ledger of retired
`tests/*.py` paths with issue-specific regression coverage.

## Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused `tests/*.py` bucket-A physical Python file count before lane changes: `0`
- Focused `tests/*.py` bucket-A physical Python file count after lane changes: `0`

## Deleted-File Ledger

- Deleted files in this lane: `[]`
- Focused bucket-A ledger: `[]`

## Retired Bucket-A Paths

- `tests/conftest.py`
- `tests/test_audit_events.py`
- `tests/test_connectors.py`
- `tests/test_console_ia.py`
- `tests/test_control_center.py`
- `tests/test_cost_control.py`

## Residual Directory State

- `tests`: directory not present, so residual Python files = `0`

## Go And Native Replacement Evidence

- `tests/conftest.py`
  Replacement surface: `bigclaw-go/internal/regression/regression.go`,
  `bigclaw-go/internal/regression/regression_test.go`
- `tests/test_audit_events.py`
  Replacement surface: `bigclaw-go/internal/observability/audit_test.go`
- `tests/test_connectors.py`
  Replacement surface: `bigclaw-go/internal/intake/connector_test.go`
- `tests/test_console_ia.py`
  Replacement surface: `bigclaw-go/internal/consoleia/consoleia_test.go`
- `tests/test_control_center.py`
  Replacement surface: `bigclaw-go/internal/control/controller.go`,
  `bigclaw-go/internal/control/controller_test.go`,
  `bigclaw-go/internal/api/server.go`
- `tests/test_cost_control.py`
  Replacement surface: `bigclaw-go/internal/costcontrol/controller.go`,
  `bigclaw-go/internal/costcontrol/controller_test.go`

## Validation Commands

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find tests -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1583(RepositoryHasNoPythonFiles|RootTestsDirectoryStaysAbsent|BucketATestPathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesBucketASweep)$'`
