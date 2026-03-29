# BIG-GO-944 Lane 4 Migration Status

## Scope

This lane covers the `repo/governance/reporting/risk/planning/mapping/memory/operations/observability` slice from `src/bigclaw` for the March 29, 2026 Go-only materialization pass.

## File Inventory

| Python file | Go owner | Status | Action |
| --- | --- | --- | --- |
| `src/bigclaw/repo_governance.py` | `bigclaw-go/internal/repo/governance.go` | migrated | keep Python only as compatibility surface until Python entrypoints are removed |
| `src/bigclaw/governance.py` | `bigclaw-go/internal/governance/freeze.go` | migrated | keep Python only as compatibility surface while Python imports remain |
| `src/bigclaw/risk.py` | `bigclaw-go/internal/risk/risk.go` | migrated | keep Python only as compatibility surface while Python scheduler/workflow tests remain |
| `src/bigclaw/mapping.py` | `bigclaw-go/internal/intake/mapping.go` | migrated | keep Python only as compatibility surface while Python ingestion tests remain |
| `src/bigclaw/planning.py` | `bigclaw-go/internal/planning/planning.go` | migrated in `BIG-GO-944` | new Go package added |
| `src/bigclaw/memory.py` | `bigclaw-go/internal/memory/store.go` | migrated in `BIG-GO-944` | new Go package added |
| `src/bigclaw/observability.py` | `bigclaw-go/internal/observability/audit.go`, `recorder.go`, `runtime.go` | partially migrated before, completed for lane runtime closeout shapes in `BIG-GO-944` | new Go runtime closeout and ledger types added |
| `src/bigclaw/reports.py` | `bigclaw-go/internal/reporting/reporting.go` | lane-owned reporting surfaces migrated | Python file still contains broader HTML/report helpers not yet cut over |
| `src/bigclaw/operations.py` | `bigclaw-go/internal/reporting/reporting.go`, `bigclaw-go/internal/api/*`, `bigclaw-go/internal/triage/*` | lane-owned operations/reporting surfaces migrated | Python file still contains broader compatibility helpers not yet cut over |

## Go Deliverables Added In This Issue

- `bigclaw-go/internal/planning/planning.go`
- `bigclaw-go/internal/planning/planning_test.go`
- `bigclaw-go/internal/memory/store.go`
- `bigclaw-go/internal/memory/store_test.go`
- `bigclaw-go/internal/observability/runtime.go`
- `bigclaw-go/internal/observability/runtime_test.go`

## Delete Plan

Immediate deletion is blocked by active Python imports and Python-only tests. The delete sequence for the remaining lane files is:

1. Switch Python callers and tests that still import `src/bigclaw/planning.py`, `memory.py`, `observability.py`, `reports.py`, and `operations.py` to Go-backed APIs, fixtures, or generated artifacts.
2. Remove re-exports from `src/bigclaw/__init__.py` once no Python runtime consumer depends on the lane symbols.
3. Delete compatibility Python files in this order:
   - `src/bigclaw/planning.py`
   - `src/bigclaw/memory.py`
   - `src/bigclaw/observability.py`
   - `src/bigclaw/reports.py`
   - `src/bigclaw/operations.py`
4. Re-run repo-wide validation after each deletion batch to confirm no Python entrypoint still depends on those modules.

## Residual Risks

- Python tests still import lane files directly, especially `tests/test_planning.py`, `tests/test_memory.py`, `tests/test_observability.py`, `tests/test_reports.py`, and `tests/test_operations.py`.
- `src/bigclaw/reports.py` and `src/bigclaw/operations.py` still contain broader compatibility/report rendering logic that is not isolated to a single Go package yet.
- The lane is materially Go-owned now, but the repository is not yet Python-free for these surfaces.
