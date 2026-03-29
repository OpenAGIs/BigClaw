# BIG-GO-947 Validation

## Lane Inventory

Source lane reference requested by the issue was `reports/go-migration-lanes-2026-03-29.md`, but that file is not present in this workspace checkout. The lane scope below is derived from the issue contract and the repo test inventory.

| Lane area | Python test file | Go replacement or status | Action |
| --- | --- | --- | --- |
| governance | `tests/test_governance.py` | `bigclaw-go/internal/governance/freeze_test.go` | Deleted Python test |
| repo governance | `tests/test_repo_governance.py` | `bigclaw-go/internal/repo/governance_test.go` | Deleted Python test |
| reporting | `tests/test_reports.py` | Partial coverage exists in `bigclaw-go/internal/reporting/reporting_test.go`, `bigclaw-go/internal/api/server_test.go`, `bigclaw-go/internal/api/expansion_test.go` | Deferred deletion plan |
| risk | `tests/test_risk.py` | `bigclaw-go/internal/risk/risk_test.go` | Deleted Python test |
| planning | `tests/test_planning.py` | `bigclaw-go/internal/planning/planning_test.go` | Added Go replacement and deleted Python test |
| mapping | `tests/test_mapping.py` | `bigclaw-go/internal/intake/mapping_test.go` | Deleted Python test |
| memory | `tests/test_memory.py` | `bigclaw-go/internal/memory/store_test.go` | Added Go replacement and deleted Python test |
| operations | `tests/test_operations.py` | `bigclaw-go/internal/reporting/reporting_test.go`, `bigclaw-go/internal/api/expansion_test.go` | Added missing Go replacements and deleted Python test |
| observability | `tests/test_observability.py` | Partial coverage in `bigclaw-go/internal/observability/*.go`, `bigclaw-go/internal/api/server_test.go`, `bigclaw-go/internal/workflow/closeout_test.go` | Deferred deletion plan |

## Implemented In This Change

- Added `bigclaw-go/internal/memory/store.go` as a Go-native replacement for the Python task memory store behavior used by the lane.
- Added `bigclaw-go/internal/memory/store_test.go` to preserve the prior `test_big501_memory_store_reuses_history_and_injects_rules` behavior under `go test`.
- Removed Python tests already covered by repo-native Go suites:
  - `tests/test_governance.py`
  - `tests/test_repo_governance.py`
  - `tests/test_mapping.py`
  - `tests/test_risk.py`
  - `tests/test_memory.py`
  - `tests/test_operations.py`
  - `tests/test_planning.py`
- Removed Python test functions now covered by Go while keeping partially unmigrated files in place:
  - `tests/test_reports.py::test_render_and_write_report`
  - `tests/test_reports.py::test_console_action_state_reflects_enabled_flag`
  - `tests/test_reports.py::test_render_pilot_scorecard_includes_roi_and_recommendation`
  - `tests/test_reports.py::test_pilot_scorecard_returns_hold_when_value_is_negative`
  - `tests/test_reports.py::test_issue_closure_requires_non_empty_validation_report`
  - `tests/test_reports.py::test_issue_closure_blocks_failed_validation_report`
  - `tests/test_reports.py::test_issue_closure_allows_completed_validation_report`
  - `tests/test_reports.py::test_launch_checklist_auto_links_documentation_status`
  - `tests/test_reports.py::test_final_delivery_checklist_tracks_required_outputs_and_recommended_docs`
  - `tests/test_reports.py::test_issue_closure_blocks_incomplete_linked_launch_checklist`
  - `tests/test_reports.py::test_issue_closure_blocks_missing_required_final_delivery_outputs`
  - `tests/test_reports.py::test_issue_closure_allows_when_required_final_delivery_outputs_exist`
  - `tests/test_reports.py::test_issue_closure_allows_when_linked_launch_checklist_is_ready`
  - `tests/test_reports.py::test_render_pilot_portfolio_report_summarizes_commercial_readiness`
  - `tests/test_observability.py::test_render_task_run_report`
  - `tests/test_observability.py::test_render_repo_sync_audit_report`
- Expanded Go reporting coverage for operations-only gaps:
  - Added `NormalizeDashboardLayout()` parity to `bigclaw-go/internal/reporting/reporting.go`
  - Added `BuildRepoCollaborationMetrics()` parity to `bigclaw-go/internal/reporting/reporting.go`
  - Added dashboard round-trip, layout normalization, and repo collaboration metric tests to `bigclaw-go/internal/reporting/reporting_test.go`
  - Added explicit `WriteReport()` and `ConsoleAction.State()` coverage to `bigclaw-go/internal/reporting/reporting_test.go`
- Added `bigclaw-go/internal/reporting/closeout_pilot.go` to replace Python pilot scorecard, pilot portfolio, validation report, checklist, and issue-closure helper coverage.
- Added `bigclaw-go/internal/planning/planning.go` and `bigclaw-go/internal/planning/planning_test.go` to replace the Python candidate backlog, entry gate, and four-week execution-plan test coverage.
- Added `bigclaw-go/internal/observability/repo_sync.go` and `bigclaw-go/internal/observability/repo_sync_test.go` to replace Python repo-sync audit report rendering coverage.

## Deferred Deletion Plan

- `tests/test_reports.py`
  - Reason: generic report writing, console action state, pilot scorecards, pilot portfolio, validation report, launch checklist, final delivery checklist, and issue-closure helpers have been migrated, but `ReportStudio`, shared-view context, auto triage, takeover queue, orchestration canvas, and billing entitlement report surfaces are not yet all consolidated into one Go-native replacement set.
  - Deletion plan: split by feature family and delete each Python slice once a direct Go suite exists.
- `tests/test_observability.py`
  - Reason: Go covers repo-sync audit report rendering, run detail, closeout, audit spec, recorder, and the run report surface, but there is not yet a single Go-native package mirroring the entire Python observability ledger/task-run API and HTML detail renderer.
  - Deletion plan: continue converging on the Go run-detail/closeout surface and remove the Python file after full behavior parity is represented in Go tests.

## Validation

Command run:

```sh
cd bigclaw-go && go test ./internal/memory ./internal/governance ./internal/repo ./internal/risk ./internal/intake ./internal/reporting ./internal/observability ./internal/events ./internal/api
```

Result:

```text
ok  	bigclaw-go/internal/memory	1.045s
ok  	bigclaw-go/internal/governance	1.473s
ok  	bigclaw-go/internal/repo	1.836s
ok  	bigclaw-go/internal/risk	2.658s
ok  	bigclaw-go/internal/intake	2.244s
ok  	bigclaw-go/internal/reporting	2.934s
ok  	bigclaw-go/internal/observability	3.334s
ok  	bigclaw-go/internal/events	3.834s
ok  	bigclaw-go/internal/api	4.884s
```

Additional command run after expanding operations parity:

```sh
cd bigclaw-go && go test ./internal/reporting ./internal/api
```

Result:

```text
ok  	bigclaw-go/internal/reporting	0.488s
ok  	bigclaw-go/internal/api	2.114s
```

Additional command run after migrating generic report helpers:

```sh
cd bigclaw-go && go test ./internal/reporting
```

Result:

```text
ok  	bigclaw-go/internal/reporting	0.820s
```

Additional command run after migrating planning parity:

```sh
cd bigclaw-go && go test ./internal/planning ./internal/governance
```

Result:

```text
ok  	bigclaw-go/internal/planning	0.144s
ok  	bigclaw-go/internal/governance	(cached)
```

Additional command run after migrating repo-sync audit rendering parity:

```sh
cd bigclaw-go && go test ./internal/observability
```

Result:

```text
ok  	bigclaw-go/internal/observability	0.853s
```

Additional command run after migrating pilot/checklist/issue-closure reporting parity:

```sh
cd bigclaw-go && go test ./internal/reporting
```

Result:

```text
ok  	bigclaw-go/internal/reporting	1.127s
```

## Residual Risks

- `reports` and `observability` still retain Python test assets because the Go package boundaries are not yet one-to-one replacements.
- The missing local `reports/go-migration-lanes-2026-03-29.md` source artifact means the lane inventory had to be reconstructed from the issue scope and current repo contents.
