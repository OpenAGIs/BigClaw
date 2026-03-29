# BIG-GO-945 Validation

## Scope

Lane 5 from `docs/go-mainline-cutover-issue-pack.md`:

- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/ui_review.py`
- operator-facing parts of `src/bigclaw/service.py`

Go ownership for this lane:

- `bigclaw-go/internal/product/console.go`
- `bigclaw-go/internal/product/saved_views.go`
- `bigclaw-go/internal/api/v2.go`
- `bigclaw-go/internal/api/server.go`
- `bigclaw-go/internal/api/monitor.go`

## Completed Work

- Ported the remaining operator-facing `service.py` monitor compatibility surface into the Go daemon:
  - added `/health`
  - added `/metrics.json`
  - added `/alerts`
  - added `/monitor`
- Added request/error tracking middleware in the Go API server while preserving streaming and hijack behavior for existing SSE endpoints.
- Added Go regression coverage for the new monitor compatibility endpoints.
- Updated the Python legacy note in `src/bigclaw/service.py` so the monitor ownership is explicitly documented as Go-owned.

## Delete Plan

- `src/bigclaw/service.py`
  - keep as migration-only compatibility scaffolding for now
  - remove once callers are cut over from `python -m bigclaw serve` to `bigclaw-go/cmd/bigclawd`
- `src/bigclaw/console_ia.py`
  - retain temporarily because Python regression tests still assert its contract surface
  - remove after equivalent Go-owned contract fixtures or API-level coverage replaces the Python manifest tests
- `src/bigclaw/design_system.py`
  - retain temporarily for the same Python manifest/test coverage reason
  - remove after `internal/product/console.go` and `/v2/design-system` become the sole validated contract source
- `src/bigclaw/saved_views.py`
  - retain temporarily while Python tests cover the legacy report manifest
  - remove after the Go catalog/reporting path is the only tested export path
- `src/bigclaw/ui_review.py`
  - no Go replacement landed in this issue
  - remains a residual lane-5 Python asset and should move in a follow-up slice because its review-pack surface is larger than the operator monitor migration completed here

## Validation

- `cd bigclaw-go && gofmt -w internal/api/monitor.go internal/api/metrics.go internal/api/server.go internal/api/server_test.go`
- `cd bigclaw-go && go test ./internal/api ./cmd/bigclawd`
- `PYTHONPATH=src python3 -m pytest tests/test_service.py`

Results:

- `go test ./internal/api ./cmd/bigclawd` passed
- `python3 -m pytest tests/test_service.py` passed

## Remaining Risks

- Lane 5 is only partially retired: `console_ia.py`, `design_system.py`, `saved_views.py`, and especially `ui_review.py` still exist as Python-owned assets because the current repo still validates them through Python test fixtures.
- Full deletion of those Python modules needs a larger contract migration so coverage moves to Go without reducing regression confidence.
