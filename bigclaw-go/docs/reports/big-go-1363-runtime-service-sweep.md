# BIG-GO-1363 Runtime/Service Sweep

## Scope

- Focus area: `src/bigclaw` runtime/service removal sweep.
- Repository-wide Python file count: `0`.
- Because the branch was already physically Python-free, this lane lands concrete Go/native replacement evidence for the retired runtime/service surfaces instead of relying on a `.py` count drop.

## Retired Python Surfaces

- `src/bigclaw/service.py`
  - Historical Python service entry for static serving plus `/health`, `/metrics`, `/metrics.json`, and `/monitor`.
- `tests/test_service.py`
  - Historical Python regression coverage for the service entry and monitoring endpoints.

## Go/Native Replacement

- `bigclaw-go/internal/migration/runtime_service_surfaces.go`
  - Lane-scoped replacement manifest that records the retired Python runtime/service paths and their Go owners.
- `bigclaw-go/internal/service/server.go`
  - Go-native service entry surface for static serving, health, metrics, alerts, and monitor endpoints.
- `bigclaw-go/internal/service/server_test.go`
  - Go-native regression coverage for quota enforcement, service endpoints, and static index bootstrap.

## Historical Evidence Anchoring The Replacement

- `reports/OPE-148-150-validation.md`
  - Records that `src/bigclaw/service.py` originally owned the Python service entry, `/health`, and `/metrics`.
- `reports/OPE-151-153-validation.md`
  - Records that the Python service surface later owned `/monitor` and `/metrics.json`.
- `reports/BIG-GO-948-validation.md`
  - Records the Go replacement package and tests in `bigclaw-go/internal/service`.

## Validation Commands

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics|TestEnsureStaticIndex'`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1363RuntimeServiceReplacement(ManifestMatchesRetiredPythonSurfaces|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`

## Validation Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  - Result: no output, confirming repository-wide Python file count remains `0`.
- `cd bigclaw-go && go test -count=1 ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics|TestEnsureStaticIndex'`
  - Result: `ok  	bigclaw-go/internal/service`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1363RuntimeServiceReplacement(ManifestMatchesRetiredPythonSurfaces|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  - Result: `ok  	bigclaw-go/internal/regression`

## Acceptance

- The repository remains at `0` physical `.py` files.
- Concrete Go/native runtime/service replacement evidence now maps the retired Python service entry and service test surfaces to the active Go package.
- Regression coverage protects the replacement manifest, service package ownership, and lane report evidence.
