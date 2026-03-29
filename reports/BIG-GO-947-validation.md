# BIG-GO-947 Validation

## Lane Inventory

Source lane reference requested by the issue was `reports/go-migration-lanes-2026-03-29.md`, but that file is not present in this workspace checkout. The lane scope below is derived from the issue contract and the repo test inventory.

| Lane area | Python test file | Go replacement or status | Action |
| --- | --- | --- | --- |
| governance | `tests/test_governance.py` | `bigclaw-go/internal/governance/freeze_test.go` | Deleted Python test |
| repo governance | `tests/test_repo_governance.py` | `bigclaw-go/internal/repo/governance_test.go` | Deleted Python test |
| reporting | `tests/test_reports.py` | Partial coverage exists in `bigclaw-go/internal/reporting/reporting_test.go`, `bigclaw-go/internal/api/server_test.go`, `bigclaw-go/internal/api/expansion_test.go` | Deferred deletion plan |
| risk | `tests/test_risk.py` | `bigclaw-go/internal/risk/risk_test.go` | Deleted Python test |
| planning | `tests/test_planning.py` | No direct Go package equivalent in this lane yet | Deferred deletion plan |
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
- Removed Python test functions now covered by Go while keeping partially unmigrated files in place:
  - `tests/test_reports.py::test_render_and_write_report`
  - `tests/test_reports.py::test_console_action_state_reflects_enabled_flag`
  - `tests/test_observability.py::test_render_task_run_report`
- Expanded Go reporting coverage for operations-only gaps:
  - Added `NormalizeDashboardLayout()` parity to `bigclaw-go/internal/reporting/reporting.go`
  - Added `BuildRepoCollaborationMetrics()` parity to `bigclaw-go/internal/reporting/reporting.go`
  - Added dashboard round-trip, layout normalization, and repo collaboration metric tests to `bigclaw-go/internal/reporting/reporting_test.go`
  - Added explicit `WriteReport()` and `ConsoleAction.State()` coverage to `bigclaw-go/internal/reporting/reporting_test.go`

## Deferred Deletion Plan

- `tests/test_planning.py`
  - Reason: the Python planning domain (`CandidateBacklog`, `EntryGate`, `FourWeekExecutionPlan`) does not have a direct Go package in `bigclaw-go` yet.
  - Deletion plan: port the planning model into a dedicated Go package before deleting this test file.
- `tests/test_reports.py`
  - Reason: the file mixes multiple report families. Generic report writing and console action state have been migrated, but `ReportStudio`, pilot portfolio/checklist flows, takeover queue, orchestration canvas, and billing entitlement report surfaces are not yet all consolidated into one Go-native replacement set.
  - Deletion plan: split by feature family and delete each Python slice once a direct Go suite exists.
- `tests/test_observability.py`
  - Reason: Go covers run detail, closeout, audit spec, recorder, and the run report surface, but there is not yet a single Go-native package mirroring the entire Python observability ledger/task-run API.
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

## Residual Risks

- `planning`, `reports`, and `observability` still retain Python test assets because the Go package boundaries are not yet one-to-one replacements.
- The missing local `reports/go-migration-lanes-2026-03-29.md` source artifact means the lane inventory had to be reconstructed from the issue scope and current repo contents.
