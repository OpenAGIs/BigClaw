# BIG-GO-968 Workpad

## Plan

1. Lock the batch-2 scope to the remaining `tests/**` Python files that already have clear Go package ownership and mostly matching Go-native behavior.
2. Compare each selected Python test with existing Go coverage and add only the missing Go assertions needed to preserve the contract before deleting Python assets.
3. Delete the migrated Python tests, leaving unrelated Python-heavy suites untouched.
4. Run targeted `go test` commands for the touched Go packages, then record exact commands and results here.
5. Commit the scoped change set and push the branch to `origin`.

## Batch 2 File List

- `tests/test_connectors.py`
- `tests/test_mapping.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_board.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_triage.py`
- `tests/test_saved_views.py`
- `tests/test_dashboard_run_contract.py`

## Acceptance

- The batch-2 file list is explicit and limited to the files above.
- Corresponding Go tests cover the deleted Python test behavior, including any parity gaps found during review.
- The selected Python files are removed from `tests/`.
- The workpad records exact validation commands and results.
- The final report includes delete/replace/keep rationale and the impact on total Python file count.

## Validation

- `go test ./internal/intake ./internal/repo ./internal/product`
- Focused `go test` runs for any newly added test names in touched packages.
- `rg --files tests | rg '\.py$'`
- `rg --files | rg '\.py$' | wc -l`
- `git status --short`

## Notes

- Delete: Python tests whose behavior is already represented by Go-owned packages after parity review.
- Replace: missing Python assertions should move into nearby Go `_test.go` files instead of keeping dual-language coverage.
- Keep: Python tests that still exercise Python-only implementations or broad report/rendering surfaces outside this issue's scope.

## Results

- Deleted and replaced with Go-owned coverage:
  - `tests/test_connectors.py`
    - Reason: covered by `bigclaw-go/internal/intake/connector_test.go`.
  - `tests/test_mapping.py`
    - Reason: covered by `bigclaw-go/internal/intake/mapping_test.go`.
  - `tests/test_repo_governance.py`
    - Reason: covered by `bigclaw-go/internal/repo/governance_test.go`.
  - `tests/test_repo_registry.py`
    - Reason: covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`, plus added JSON round-trip parity coverage.
  - `tests/test_repo_gateway.py`
    - Reason: covered by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - `tests/test_repo_triage.py`
    - Reason: covered by `bigclaw-go/internal/triage/repo_test.go` and `bigclaw-go/internal/repo/repo_surfaces_test.go`.
  - `tests/test_saved_views.py`
    - Reason: covered by `bigclaw-go/internal/product/saved_views_test.go`, including existing JSON round-trip coverage.
  - `tests/test_dashboard_run_contract.py`
    - Reason: covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`, plus added JSON round-trip and deterministic sample-gap assertions.

- Kept for later lanes:
  - `tests/test_repo_board.py`
    - Reason: current Go repo board tests cover create/reply/filter behavior, but do not yet replace the Python collaboration-comment projection contract.
  - Other remaining `tests/**` Python files
    - Reason: they still exercise Python-owned runtime, reporting, UI review, operations, or larger end-to-end surfaces outside this scoped batch.

- Added Go parity tests:
  - `bigclaw-go/internal/repo/repo_surfaces_test.go`
    - Added `TestRepoRegistryJSONRoundTrip`.
  - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
    - Added deterministic dashboard/run-detail sample-gap assertions.
    - Added `TestDashboardRunContractJSONRoundTrip`.

- Python file count impact:
  - `tests/**` Python files: `43 -> 35` (`-8`)
  - Repository-wide Python files: `123 -> 115` (`-8`)

## Validation Results

- `cd bigclaw-go && go test ./internal/repo -run 'TestRepoRegistryJSONRoundTrip|TestRepoRegistryResolvesSpaceChannelAndAgent|TestNormalizeGatewayPayloadsAndErrors|TestBuildApprovalEvidencePacket'`
  - `ok  	bigclaw-go/internal/repo	0.961s`
- `cd bigclaw-go && go test ./internal/product -run 'TestDashboardRunContractAuditDetectsMissingPaths|TestDashboardRunContractJSONRoundTrip|TestBuildDefaultDashboardRunContractIsReleaseReady|TestSavedViewCatalogJSONRoundTrip|TestAuditSavedViewCatalogAndRenderReport'`
  - `ok  	bigclaw-go/internal/product	1.830s`
- `cd bigclaw-go && go test ./internal/intake ./internal/triage`
  - `ok  	bigclaw-go/internal/intake	0.493s`
  - `ok  	bigclaw-go/internal/triage	1.472s`
- `cd bigclaw-go && go test ./internal/intake ./internal/repo ./internal/product ./internal/triage`
  - `ok  	bigclaw-go/internal/intake	(cached)`
  - `ok  	bigclaw-go/internal/repo	0.246s`
  - `ok  	bigclaw-go/internal/product	0.455s`
  - `ok  	bigclaw-go/internal/triage	(cached)`
- `rg --files tests | rg '\.py$' | wc -l`
  - `35`
- `rg --files | rg '\.py$' | wc -l`
  - `115`
- `git status --short`
  - scoped changes only in `.symphony/workpad.md`, the two Go test files, and the eight deleted Python tests
