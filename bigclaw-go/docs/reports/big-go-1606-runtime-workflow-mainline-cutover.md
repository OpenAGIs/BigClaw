# BIG-GO-1606 Runtime/Workflow Mainline Cutover

## Scope

`BIG-GO-1606` removes the last dead compatibility artifacts that still modeled
retired Python workflow/runtime surfaces as if they were active migration
entrypoints.

The repository is already physically Python-free, so this lane cuts the final
compatibility layer by deleting the stale registry/manifest pair and relying on
direct Go-owned runtime, service, scheduler, and workflow entrypaths instead.

## Python Baseline

Repository-wide Python file count: `0`.

## Deleted Compatibility Artifacts

- `bigclaw-go/internal/migration/legacy_model_runtime_modules.go`
- `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`

## Active Go Mainline Entrypaths

- `bigclaw-go/internal/worker/runtime.go`
- `bigclaw-go/internal/worker/runtime_runonce.go`
- `bigclaw-go/internal/service/server.go`
- `bigclaw-go/internal/scheduler/scheduler.go`
- `bigclaw-go/internal/workflow/engine.go`
- `bigclaw-go/internal/workflow/orchestration.go`
- `bigclaw-go/cmd/bigclawd/main.go`

## Regression Guard

- `bigclaw-go/internal/regression/big_go_1606_runtime_workflow_mainline_test.go`

## Validation Commands

- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' \) -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1606(RepositoryHasNoPythonFiles|DeletedCompatibilityArtifactsStayAbsent|GoMainlineEntryPathsExist|LaneReportCapturesMainlineCutover)$'`

## Validation Results

- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' \) -print | sort` produced no output.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1606(RepositoryHasNoPythonFiles|DeletedCompatibilityArtifactsStayAbsent|GoMainlineEntryPathsExist|LaneReportCapturesMainlineCutover)$'` passed with `ok  	bigclaw-go/internal/regression`.
