# BIG-GO-929 Control/Service/Queue Migration

## Python Asset Inventory

- `tests/test_control_center.py`
  - legacy source surfaces: `src/bigclaw/operations.py`, `src/bigclaw/reports.py`, `src/bigclaw/queue.py`
- `tests/test_service.py`
  - legacy source surface: `src/bigclaw/service.py`
- `tests/test_queue.py`
  - legacy source surface: `src/bigclaw/queue.py`

## Go Replacements

- `tests/test_control_center.py`
  - queue ordering and persistence are covered in `bigclaw-go/internal/queue/*_test.go`
  - queue control center rendering now lives in `bigclaw-go/internal/reporting/reporting.go`
  - shared-view empty state is covered by `bigclaw-go/internal/reporting/reporting_test.go`
- `tests/test_service.py`
  - repo governance enforcement now lives in `bigclaw-go/internal/repo/governance.go`
  - static monitoring service now lives in `bigclaw-go/internal/service/service.go`
- `tests/test_queue.py`
  - file-backed persistence, dead-letter replay, cancel/reload, and priority ordering already live in `bigclaw-go/internal/queue/file_queue_test.go`

## Legacy Python Deletion Conditions

- Remove `tests/test_control_center.py`, `tests/test_service.py`, and `tests/test_queue.py` after the Go tests below stay green on CI and no remaining Python callers are the source of truth for these surfaces.
- Remove or freeze the corresponding Python implementations only after repo workflows and documentation point to the Go replacements above as the maintained path.

## Regression Validation

- `cd bigclaw-go && go test ./internal/queue ./internal/reporting ./internal/control ./internal/repo ./internal/service`
- `cd bigclaw-go && go test ./...`
