# BIG-GO-1361 Legacy Core Module Replacement Sweep

`BIG-GO-1361` closes out the remaining `src/bigclaw` core-module sweep by
recording the Go/native owners for retired intake, execution, planning, and
reporting-era Python modules.

## Baseline

- Repository-wide Python file count: `0`.
- The active branch cannot lower the `.py` count further, so acceptance for this
  lane is concrete Go/native replacement evidence landing in git.

## Go-Native Replacement Artifact

- Registry path: `bigclaw-go/internal/migration/legacy_core_modules.go`
- Purpose: record the retired Python core modules, the active Go owners that
  replaced them, and the checked-in evidence that keeps the sweep auditable.

## Replacement Mapping

### Intake and Definition Surface

- `src/bigclaw/connectors.py` -> `bigclaw-go/internal/intake/types.go`,
  `bigclaw-go/internal/intake/connector.go`,
  `bigclaw-go/internal/intake/connector_test.go`
- `src/bigclaw/mapping.py` -> `bigclaw-go/internal/intake/mapping.go`,
  `bigclaw-go/internal/intake/mapping_test.go`
- `src/bigclaw/dsl.py` -> `bigclaw-go/internal/workflow/definition.go`,
  `bigclaw-go/internal/workflow/definition_test.go`,
  `bigclaw-go/internal/workflow/engine.go`
- Evidence: `docs/go-domain-intake-parity-matrix.md`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche5_test.go`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche9_test.go`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go`

### Execution and Planning Surface

- `src/bigclaw/scheduler.py` -> `bigclaw-go/internal/scheduler/scheduler.go`,
  `bigclaw-go/internal/scheduler/scheduler_test.go`
- `src/bigclaw/workflow.py` -> `bigclaw-go/internal/workflow/engine.go`,
  `bigclaw-go/internal/workflow/model.go`,
  `bigclaw-go/internal/workflow/closeout.go`
- `src/bigclaw/queue.py` -> `bigclaw-go/internal/queue/queue.go`,
  `bigclaw-go/internal/queue/memory_queue.go`,
  `bigclaw-go/internal/queue/sqlite_queue.go`
- `src/bigclaw/planning.py` -> `bigclaw-go/internal/planning/planning.go`,
  `bigclaw-go/internal/planning/planning_test.go`
- `src/bigclaw/orchestration.py` ->
  `bigclaw-go/internal/workflow/orchestration.go`,
  `bigclaw-go/internal/orchestrator/loop.go`,
  `bigclaw-go/internal/control/controller.go`
- Evidence: `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`,
  `bigclaw-go/docs/reports/queue-reliability-report.md`,
  `docs/go-mainline-cutover-issue-pack.md`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`

### Reporting and Operations Surface

- `src/bigclaw/observability.py` ->
  `bigclaw-go/internal/observability/recorder.go`,
  `bigclaw-go/internal/observability/task_run.go`,
  `bigclaw-go/internal/observability/audit.go`
- `src/bigclaw/reports.py` -> `bigclaw-go/internal/reporting/reporting.go`,
  `bigclaw-go/internal/reportstudio/reportstudio.go`
- `src/bigclaw/evaluation.py` ->
  `bigclaw-go/internal/evaluation/evaluation.go`,
  `bigclaw-go/internal/evaluation/evaluation_test.go`
- `src/bigclaw/operations.py` ->
  `bigclaw-go/internal/product/dashboard_run_contract.go`,
  `bigclaw-go/internal/control/controller.go`,
  `bigclaw-go/internal/api/server.go`
- Evidence: `docs/go-mainline-cutover-issue-pack.md`,
  `bigclaw-go/docs/reports/go-control-plane-observability-report.md`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`,
  `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`

## Regression Guard

- `bigclaw-go/internal/regression/big_go_1361_legacy_core_module_replacement_test.go`
  verifies the replacement registry contents, the referenced Go paths, and this
  lane report.

## Validation Commands

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1361LegacyCoreModuleReplacement(ManifestMatchesRetiredModules|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
