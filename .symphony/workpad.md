# BIG-GO-1606 Workpad

## Plan

1. Remove the dead Go-owned compatibility artifacts that still model retired
   Python workflow/runtime surfaces as active migration registries:
   - `bigclaw-go/internal/migration/legacy_model_runtime_modules.go`
   - `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
2. Replace the deleted compatibility layer with a direct regression contract
   that proves the live Go runtime, scheduler, service, and workflow entrypaths
   are the only remaining mainline surfaces for this lane.
3. Update any broad Python-removal regression/report references that still pin
   the deleted compatibility artifacts, then run the focused validation commands,
   record exact results, commit, and push.

## Acceptance

- The legacy compatibility registry/manifest files are deleted from the repo.
- Regression coverage proves the repository remains Python-free and that the
  live Go-owned runtime/workflow/service/scheduler entrypaths exist directly
  without the deleted compatibility layer.
- Lane-specific report artifacts capture the deleted surfaces, active Go
  replacements, exact validation commands, and exact observed results.
- Validation metadata records the exact commands, observed outputs, commit, and
  push target for this lane only.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1606 -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' \) -print | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1606/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1606(RepositoryHasNoPythonFiles|DeletedCompatibilityArtifactsStayAbsent|GoMainlineEntryPathsExist|LaneReportCapturesMainlineCutover)$|TestBIGGO167(RepositoryHasNoPythonFiles|ReferenceDenseGoOwnedDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestBIGGO248(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
