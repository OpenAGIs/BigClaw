# BIG-GO-1359 Workpad

## Plan

1. Reconfirm the remaining physical Python asset inventory for the repository and the lane priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Replace the active Ray smoke validation default away from the inline Python entrypoint and update the corresponding repository-native documentation surfaces:
   - `bigclaw-go/scripts/e2e/ray_smoke.sh`
   - `bigclaw-go/docs/e2e-validation.md`
3. Add the lane-scoped artifacts that record the zero-Python baseline and the native replacement landed for this issue:
   - `bigclaw-go/docs/reports/big-go-1359-python-asset-sweep.md`
   - `bigclaw-go/internal/regression/big_go_1359_zero_python_guard_test.go`
4. Re-run targeted validation, record the exact commands and results, then commit and push the branch.

## Acceptance

- The repository-wide physical Python asset inventory remains explicit for the whole repository and the priority residual directories.
- The active Ray smoke default entrypoint no longer depends on inline Python and the checked-in validation docs reflect the shell-native replacement.
- The lane-scoped report and regression test capture the replacement path and exact validation commands.
- Exact validation commands and results are recorded.
- The change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1359(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|RaySmokeReplacementPathsRemainAvailable|LaneReportCapturesNativeReplacement)$'`

## Execution Notes

- 2026-04-05: `find . -name '*.py' | wc -l` already returned `0` in this checkout, so the issue must land via the concrete native-replacement acceptance path instead of a file-count reduction.
- 2026-04-05: The scoped replacement target is the active Ray smoke entrypoint and its checked-in validation guidance, which still referenced inline Python even though the repository no longer contains `.py` assets.
- 2026-04-05: Updated `bigclaw-go/scripts/e2e/ray_smoke.sh` to default `BIGCLAW_RAY_SMOKE_ENTRYPOINT` to `sh -c 'echo hello from ray'` and aligned `bigclaw-go/docs/e2e-validation.md` with the same shell-native path.
- 2026-04-05: Validation results:
  - `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` -> no output
  - `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort` -> no output
  - `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1359/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1359(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|RaySmokeReplacementPathsRemainAvailable|LaneReportCapturesNativeReplacement)$'` -> `ok  	bigclaw-go/internal/regression	0.583s`
