# BIG-GO-1591 Workpad

## Plan

1. Confirm that the issue's named Python assets are already absent from the
   checkout and identify the current Go-owned replacement surfaces for
   evaluation, scheduler, and repo operations behavior.
2. Add a `BIG-GO-1591` regression guard plus lane evidence so this slice
   records the zero-Python state instead of reintroducing tracker-only churn:
   - `bigclaw-go/internal/regression/big_go_1591_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-1591-python-asset-sweep.md`
   - `reports/BIG-GO-1591-validation.md`
   - `reports/BIG-GO-1591-status.json`
3. Run the exact inventory and targeted regression commands, capture their
   results in the lane evidence, then commit and push the branch.

## Acceptance

- The named Python assets from the issue focus remain absent from the
  repository checkout.
- A lane-specific Go regression test locks the repository-wide zero-Python
  baseline and verifies the exact focus paths stay absent.
- The lane report and validation report record the exact commands run, their
  results, and the residual risk that the branch started Python-free.
- The scoped `BIG-GO-1591` change set is committed and pushed to the remote
  branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1591 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1591/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1591/tests -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1591 && for path in src/bigclaw/__init__.py src/bigclaw/evaluation.py src/bigclaw/operations.py src/bigclaw/repo_links.py src/bigclaw/scheduler.py tests/test_connectors.py tests/test_execution_contract.py tests/test_models.py; do if test -e \"$path\"; then echo \"present:$path\"; else echo \"absent:$path\"; fi; done`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1591/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1591(RepositoryHasNoPythonFiles|FocusAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
