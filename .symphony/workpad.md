# BIG-GO-1609 Workpad

## Plan

1. Confirm the repository state for the residual Python package bootstrap slice
   assigned to `BIG-GO-1609`, focusing on:
   - `src/bigclaw/__init__.py`
   - `src/bigclaw/__main__.py`
   - `src/bigclaw/legacy_shim.py`
   - `src/bigclaw/workspace_bootstrap_cli.py`
2. Add issue-scoped regression and evidence artifacts that freeze the removed
   package bootstrap glue as absent while proving the Go-native replacements
   remain available:
   - `bigclaw-go/internal/regression/big_go_1609_package_bootstrap_glue_test.go`
   - `bigclaw-go/docs/reports/big-go-1609-package-bootstrap-glue-sweep.md`
   - `reports/BIG-GO-1609-validation.md`
   - `reports/BIG-GO-1609-status.json`
3. Run the targeted inventory and regression validation, capture exact commands
   and results, then commit and push branch `symphony/BIG-GO-1609`.

## Acceptance

- The assigned residual Python package bootstrap files stay absent from the
  repository checkout.
- The current Go/native entrypoints and bootstrap surfaces that replaced the
  removed package glue remain present.
- Lane-specific regression coverage and reports capture the exact file list,
  validation commands, and observed results for `BIG-GO-1609`.
- The lane changes stay scoped to issue evidence and regression coverage, then
  land in git history and on the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609 -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`
- `for path in /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609/src/bigclaw/__init__.py /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609/src/bigclaw/__main__.py /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609/src/bigclaw/legacy_shim.py /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609/src/bigclaw/workspace_bootstrap_cli.py; do test ! -e "$path" || echo "present: $path"; done`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1609/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1609(RepositoryHasNoPythonFiles|PackageBootstrapGluePathsRemainAbsent|GoNativeBootstrapSurfacesRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche7$|TestTopLevelModulePurgeTranche17$'`
