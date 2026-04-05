# BIG-GO-1363 Workpad

## Plan

1. Reconfirm the repository Python baseline and the historical runtime/service Python surfaces retired in this lane.
2. Land a Go-native runtime/service replacement manifest that maps the retired Python service entry and service tests to the current Go package ownership.
3. Add lane-scoped regression coverage and a validation report that prove the replacement stays aligned.
4. Run targeted validation, record exact commands and results, then commit and push the scoped branch changes.

## Acceptance

- The repository remains at zero physical `.py` files or lower than the starting count for this branch.
- Concrete Go/native replacement evidence lands for the runtime/service removal sweep, specifically for the retired Python service entry and monitoring test surfaces.
- Regression coverage verifies the replacement manifest, referenced Go paths, and lane report contents.
- Exact validation commands and results are recorded in-repo.
- The change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363/bigclaw-go && go test -count=1 ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics|TestEnsureStaticIndex'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1363RuntimeServiceReplacement(ManifestMatchesRetiredPythonSurfaces|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`

## Execution Notes

- 2026-04-05: Starting branch already reports zero physical `.py` files, so this lane must land concrete Go/native runtime/service replacement evidence rather than rely on a file-count drop.
- 2026-04-05: Historical validation reports identify `src/bigclaw/service.py` and `tests/test_service.py` as the retired Python runtime/service surfaces now owned by `bigclaw-go/internal/service`.
- 2026-04-05: Added `bigclaw-go/internal/migration/runtime_service_surfaces.go` as the lane-scoped Go-native replacement manifest for the retired Python runtime/service surfaces.
- 2026-04-05: Added `bigclaw-go/internal/regression/big_go_1363_runtime_service_replacement_test.go` and `bigclaw-go/docs/reports/big-go-1363-runtime-service-sweep.md` to keep the replacement evidence and report aligned.
- 2026-04-05: Extended `bigclaw-go/internal/service/server_test.go` with `TestEnsureStaticIndex` to cover the static index bootstrap retained in the Go-native service entry.
- 2026-04-05: Ran `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` and observed no output, confirming the repository remains physically Python-free.
- 2026-04-05: Ran `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363/bigclaw-go && go test -count=1 ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics|TestEnsureStaticIndex'` and observed `ok  	bigclaw-go/internal/service	0.588s`.
- 2026-04-05: Ran `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1363/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1363RuntimeServiceReplacement(ManifestMatchesRetiredPythonSurfaces|ReplacementPathsExist|LaneReportCapturesReplacementState)$'` and observed `ok  	bigclaw-go/internal/regression	0.268s`.
