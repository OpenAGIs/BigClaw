# BIG-GO-135

## Context
- Issue: `BIG-GO-135`
- Goal: close the remaining root-level Python tooling/build-helper gap by locking down the retired Python build metadata alongside the already-retired root script shims.
- Current repo state on entry: `main` is already Go-only at the repository root, but the residual-sweep regression tests do not explicitly assert that `setup.py` and `pyproject.toml` stay deleted.

## Scope
- `.symphony/workpad.md`
- `README.md`
- `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
- `local-issues.json`
- `reports/BIG-GO-135-validation.md`
- `reports/BIG-GO-135-status.json`

## Plan
1. Replace the stale carried-over workpad with issue-specific plan, acceptance, and validation targets before editing tracked files.
2. Extend the root residual-sweep regression so it explicitly treats retired Python build helpers as part of the deleted root tooling surface.
3. Refresh README root-posture guidance so it states the root no longer carries Python build metadata as well as `.py` assets.
4. Record repo-native closeout state in the local tracker and lane artifacts, then push the refreshed branch.
5. Rebase or rebuild the lane onto current `origin/main` as needed so the issue stays scoped and lands cleanly.

## Acceptance
- The workpad is specific to `BIG-GO-135`.
- The root residual-sweep regression explicitly fails if `setup.py` or `pyproject.toml` reappear.
- README root-posture guidance matches the enforced Go-only build-helper posture.
- Validation records exact commands and exact results for the regression and root inventory checks.
- The local tracker and lane artifacts capture the pushed branch and current landing state.
- Changes remain scoped to this issue.

## Validation
- `git ls-files 'scripts/*.py' 'scripts/ops/*.py' 'setup.py' 'pyproject.toml' | sort`
- `find . -path './.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sed 's#^./##' | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootScriptResidualSweep|TestRootScriptResidualSweepDocs'`
- `python3 -m json.tool local-issues.json >/dev/null`
- `python3 -m json.tool reports/BIG-GO-135-status.json >/dev/null`
