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
- Removed the redundant Python saved-views implementation in `src/bigclaw/saved_views.py`.
- Removed the redundant Python regression file `tests/test_saved_views.py`.
- Repointed the planning evidence for the saved-views capability to the Go owner:
  - `bigclaw-go/internal/product/saved_views.go`
  - `bigclaw-go/internal/product/saved_views_test.go`
- Removed the redundant Python console IA implementation in `src/bigclaw/console_ia.py`.
- Removed the redundant Python regression file `tests/test_console_ia.py`.
- Repointed the planning evidence for the console-shell capability to the Go owner:
  - `bigclaw-go/internal/product/console.go`
  - `bigclaw-go/internal/product/console_test.go`
- Removed the redundant Python design-system implementation in `src/bigclaw/design_system.py`.
- Removed the redundant Python regression file `tests/test_design_system.py`.
- Repointed the planning evidence for the design-system capability to the Go owner:
  - `bigclaw-go/internal/product/console.go`
  - `bigclaw-go/internal/api/expansion_test.go`
- Removed the redundant Python UI-review implementation in `src/bigclaw/ui_review.py`.
- Removed the redundant Python regression file `tests/test_ui_review.py`.
- Repointed the planning evidence for review/readiness coverage to Go-owned artifacts:
  - `bigclaw-go/docs/reports/review-readiness.md`
  - `bigclaw-go/internal/api/server_test.go`

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
  - removed in this issue
  - canonical owner is now `bigclaw-go/internal/product/saved_views.go`
- `src/bigclaw/console_ia.py`
  - removed in this issue
  - canonical owner is now `bigclaw-go/internal/product/console.go`
- `src/bigclaw/design_system.py`
  - removed in this issue
  - canonical owner is now `bigclaw-go/internal/product/console.go`
- `src/bigclaw/ui_review.py`
  - removed in this issue
  - retired in favor of Go-owned review readiness artifacts and API reviewer-bundle coverage
- `src/bigclaw/ui_review.py`
  - no Go replacement landed in this issue
  - remains a residual lane-5 Python asset and should move in a follow-up slice because its review-pack surface is larger than the operator monitor migration completed here

## Validation

- `cd bigclaw-go && gofmt -w internal/api/monitor.go internal/api/metrics.go internal/api/server.go internal/api/server_test.go`
- `cd bigclaw-go && go test ./internal/api ./cmd/bigclawd`
- `PYTHONPATH=src python3 -m pytest tests/test_service.py`
- `PYTHONPATH=src python3 -c "import bigclaw; print('ok')"`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py`
- `cd bigclaw-go && go test ./internal/product ./internal/api`

Results:

- `go test ./internal/api ./cmd/bigclawd` passed
- `python3 -m pytest tests/test_service.py` passed
- `python3 -c "import bigclaw; print('ok')"` passed
- `python3 -m pytest tests/test_planning.py` passed
- `go test ./internal/product ./internal/api` passed

## Remaining Risks

- Lane 5 Python application modules are fully retired, but the Go replacements are not like-for-like data-model ports in every case. Some surfaces were retired by moving planning and validation evidence to existing Go APIs and reviewer-readiness artifacts instead of creating a dedicated new Go package.
- Residual Python assets still exist elsewhere in the repo outside lane 5; this issue only covers the lane-5 module inventory from the cutover plan.
