# BIG-GO-1572 Go-only residual Python sweep 02

## Summary

- Sweep scope matched the issue candidate list exactly.
- Repository-wide physical Python file count at sweep time: `0`.
- Every candidate path was already physically absent on `main`, so this lane records the ledger explicitly and adds regression coverage for the exact sweep set.
- No Python compatibility shim was added in this sweep.

## Candidate ledger

- `src/bigclaw/__main__.py` -> `bigclaw-go/cmd/bigclawd/main.go`; `bigclaw-go/cmd/bigclawctl/main.go`
- `src/bigclaw/event_bus.py` -> `bigclaw-go/internal/events/transition_bus.go`
- `src/bigclaw/orchestration.py` -> `bigclaw-go/internal/orchestrator/loop.go`; `bigclaw-go/internal/workflow/orchestration.go`
- `src/bigclaw/repo_plane.py` -> `bigclaw-go/internal/repo/plane.go`
- `src/bigclaw/service.py` -> `bigclaw-go/cmd/bigclawd/main.go`; `bigclaw-go/internal/api/server.go`
- `tests/test_console_ia.py` -> `bigclaw-go/internal/consoleia/consoleia_test.go`
- `tests/test_execution_flow.py` -> `bigclaw-go/internal/contract/execution_test.go`; `bigclaw-go/internal/workflow/orchestration_test.go`
- `tests/test_observability.py` -> `bigclaw-go/internal/observability/recorder_test.go`; `bigclaw-go/internal/api/server_test.go`
- `tests/test_repo_gateway.py` -> `bigclaw-go/internal/repo/repo_surfaces_test.go`
- `tests/test_runtime_matrix.py` -> `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`; `bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command_test.go`
- `scripts/create_issues.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/scripts/benchmark/soak_local.py` -> `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`
- `bigclaw-go/scripts/e2e/run_task_smoke.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go`

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1572'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationUsageListsBIGGO1160GoReplacements|TestAutomationSubscriberTakeoverFaultMatrixBuildsReport|TestRunCreateIssuesCreatesOnlyMissing'`

## Residual risk

- This lane proves that the targeted Python assets remain deleted and that their Go replacements still exist; it does not introduce new runtime behavior.
- `run-task-smoke --autostart` and `soak-local --autostart` still depend on ephemeral port reservation before `bigclawd` binds, so local port races remain the operational caveat for the script-replacement surfaces.
- Runtime-matrix replacement evidence stays doc and harness driven through `parallel-validation-matrix.md` and the takeover-matrix command test rather than through a dedicated Python-era test file.
