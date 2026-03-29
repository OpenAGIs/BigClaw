# BIG-GO-976 Workpad

## Scope

Targeted remaining Python tests for this batch:

- `tests/test_dashboard_run_contract.py`
- `tests/test_saved_views.py`
- `tests/test_dsl.py`

Go-native replacement surfaces for this lane:

- `bigclaw-go/internal/product/dashboard_run_contract.go`
- `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `bigclaw-go/internal/product/saved_views.go`
- `bigclaw-go/internal/product/saved_views_test.go`
- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/workflow/definition_test.go`
- `bigclaw-go/internal/workflow/closeout.go`
- `bigclaw-go/internal/workflow/closeout_test.go`

Repository Python file count before this lane: `118`

## Plan

1. Confirm the exact Python test file list owned by this batch and the Go-native replacement coverage for each file.
2. Remove only the Python tests with clear Go replacements in product contract, saved-view, and workflow-definition surfaces.
3. Update any in-repo planning references that still point at the deleted Python tests.
4. Run targeted Go and Python tests for the touched planning and replacement surfaces, and record the exact commands and results here.
5. Report the batch file list, replacement mapping, and repository Python file count delta.
6. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact batch-owned Python test file list for `BIG-GO-976`.
- Delete the batch-owned Python test files only where a clear Go-native replacement already exists.
- Update planning metadata/tests so the repository no longer points at deleted Python tests.
- Record the exact replacement path for each deleted Python test.
- Report before/after total repository Python file counts and the net change.

## Validation

- `cd bigclaw-go && go test ./internal/product -run 'TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestBuildSavedViewCatalog'`
- `cd bigclaw-go && go test ./internal/workflow -run 'TestDefinition|TestBuildCloseout'`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `git status --short`

## Notes

- `tests/test_console_ia.py` and `tests/test_workflow.py` stay out of this lane because their remaining Python behavior does not yet have a complete Go-native replacement surface in this checkout.
- Historical validation reports that mention deleted Python tests are treated as historical artifacts, not live planning metadata.

## Results

### File Disposition

- `tests/test_dashboard_run_contract.py`
  - Deleted.
  - Replacement path: `bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - Reason: the release-ready contract, missing-path audit, and report rendering assertions now live in the Go-native dashboard contract surface.
- `tests/test_saved_views.py`
  - Deleted.
  - Replacement path: `bigclaw-go/internal/product/saved_views_test.go`
  - Reason: saved-view catalog shape, scoped route generation, digest coverage, and baseline field contracts are already asserted in the Go-native product surface.
- `tests/test_dsl.py`
  - Deleted.
  - Replacement paths: `bigclaw-go/internal/workflow/definition_test.go`, `bigclaw-go/internal/workflow/closeout_test.go`
  - Reason: workflow-definition parsing/rendering and closeout approval/evidence path rendering now live in the Go-native workflow package.
- `src/bigclaw/planning.py`
  - Updated.
  - Reason: removed the deleted saved-views Python test from the candidate validation command and pointed the saved-views evidence link at the Go-native test file.
- `tests/test_planning.py`
  - Updated.
  - Reason: aligned the planning test expectations with the new saved-views Go-native replacement path.

### Python File Count Impact

- Repository Python files before: `118`
- Repository Python files after: `115`
- Net reduction: `3`

### Validation Record

- `cd bigclaw-go && go test ./internal/product -run 'TestBuildDefaultDashboardRunContractIsReleaseReady|TestDashboardRunContractAuditDetectsMissingPaths|TestRenderDashboardRunContractReport|TestBuildSavedViewCatalog'`
  - Result: `ok  	bigclaw-go/internal/product	0.878s`
- `cd bigclaw-go && go test ./internal/workflow -run 'TestDefinition|TestBuildCloseout'`
  - Result: `ok  	bigclaw-go/internal/workflow	0.449s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  - Result: `14 passed in 0.16s`
- `git status --short`
  - Result: scoped changes in `.symphony/workpad.md`, `src/bigclaw/planning.py`, `tests/test_planning.py`, and the three deleted Python test files.
