# BIG-GO-227 Workpad

## Plan

1. Audit the duplicated zero-Python regression coverage in
   `bigclaw-go/internal/regression` and identify the shared assertions that can
   be expressed once without reducing lane coverage.
2. Replace the per-lane duplicate zero-Python guard files with a single
   table-driven regression suite plus shared metadata that preserves the current
   assertions for repository scans, audited directories, replacement paths, and
   lane-report substrings.
3. Keep the change scoped to the regression sweep for this issue, refresh the
   lane evidence for `BIG-GO-227`, then run targeted validation on the new
   consolidated suite.
4. Record the exact validation commands and results, commit the change set, and
   push `BIG-GO-227` to the remote branch.

## Acceptance

- `bigclaw-go/internal/regression` no longer carries the duplicate
  `big_go_*_zero_python_guard_test.go` files that restate the same zero-Python
  checks lane by lane.
- A single consolidated zero-Python regression suite preserves the current
  branch behavior for repository-wide scans, directory audits, required
  replacement paths, deleted-path checks, and lane-report content validation.
- `BIG-GO-227` lane artifacts document the consolidation, the targeted test
  commands, and their exact results.
- The final change set is committed on `BIG-GO-227` and pushed to `origin`.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-227 && find bigclaw-go/internal/regression -type f -name 'big_go_*_zero_python_guard_test.go' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-227/bigclaw-go && go test -count=1 ./internal/regression -run 'TestZeroPythonGuardCatalog|TestZeroPythonGuardRepositoryAndAuditedDirectories|TestZeroPythonGuardReplacementAndDeletedPaths|TestZeroPythonGuardLaneReports'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-227 && git status --short`

## Execution Notes

- 2026-04-11: `BIG-GO-227` was provisioned as an empty directory, so the branch
  was re-materialized from the latest local BigClaw checkout (`BIG-GO-226`,
  commit `d50a1d12`) before issue work started.
- 2026-04-11: The live branch already had zero tracked `.py` files; the
  highest-density remaining Python surface was 191 duplicated
  `big_go_*_zero_python_guard_test.go` files inside
  `bigclaw-go/internal/regression`.
- 2026-04-11: Consolidated 186 standard zero-Python guard lanes into
  `bigclaw-go/internal/regression/big_go_227_zero_python_guard_catalog_test.go`
  and retained 5 specialized guard files with bespoke README or inventory
  assertions.
- 2026-04-11: Validation completed with:
  - `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-227/bigclaw-go && go test -count=1 ./internal/regression -run 'TestZeroPythonGuardCatalog|TestZeroPythonGuardRepositoryAndAuditedDirectories|TestZeroPythonGuardReplacementAndDeletedPaths|TestZeroPythonGuardLaneReports|TestBIGGO227ConsolidatedZeroPythonGuardFiles|TestBIGGO1235ReadmeStaysGoOnly|TestBIGGO124TargetResidualPythonPathsAbsent|TestBIGGO154|TestBIGGO176|TestBIGGO205'`
    Result: `ok  	bigclaw-go/internal/regression	0.504s`
  - `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-227 && find bigclaw-go/internal/regression -type f -name 'big_go_*_zero_python_guard_test.go' | wc -l`
    Result: `5`
