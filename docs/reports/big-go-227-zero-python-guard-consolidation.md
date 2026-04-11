# BIG-GO-227 Zero-Python Guard Consolidation

`BIG-GO-227` replaces the standard zero-Python regression fanout in
`bigclaw-go/internal/regression` with one catalog-driven suite.

## Scope

- Previous standard guard files: `191`
- Consolidated standard cases moved into
  `bigclaw-go/internal/regression/big_go_227_zero_python_guard_catalog_test.go`:
  `186`
- Retained specialized guard files: `5`
  - `big_go_1235_zero_python_guard_test.go`
  - `big_go_124_zero_python_guard_test.go`
  - `big_go_154_zero_python_guard_test.go`
  - `big_go_176_zero_python_guard_test.go`
  - `big_go_205_zero_python_guard_test.go`

## Outcome

- The duplicated repository scan, audited-directory, replacement-path, deleted-path,
  and lane-report assertions now run through one table-driven catalog.
- Specialized lanes that still carry README or inventory-contract assertions remain
  separate to avoid collapsing distinct behavior.
- The branch keeps the same zero-Python coverage intent while sharply reducing the
  Python-heavy literal footprint inside `bigclaw-go/internal/regression`.

## Targeted Validation

- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestZeroPythonGuardCatalog|TestZeroPythonGuardRepositoryAndAuditedDirectories|TestZeroPythonGuardReplacementAndDeletedPaths|TestZeroPythonGuardLaneReports|TestBIGGO227ConsolidatedZeroPythonGuardFiles|TestBIGGO1235ReadmeStaysGoOnly|TestBIGGO124TargetResidualPythonPathsAbsent|TestBIGGO154|TestBIGGO176|TestBIGGO205'`
