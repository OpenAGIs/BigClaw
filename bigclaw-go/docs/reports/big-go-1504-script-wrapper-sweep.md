# BIG-GO-1504 Script Wrapper Sweep

## Scope

`BIG-GO-1504` audits the physical Python wrapper inventory under `scripts/` and
`scripts/ops/`, including compatibility-only operator launchers.

The checked-out `origin/main` baseline is already on the zero-Python side of the
migration, so this lane records repository reality and adds a targeted
regression guard instead of manufacturing a fake deletion.

## Before And After Counts

- `scripts/*.py` before count: `0`
- `scripts/ops/*.py` before count: `0`
- `scripts/*.py` after count: `0`
- `scripts/ops/*.py` after count: `0`

Deleted physical `.py` files: `none`

## Replacement Surface

The compatibility launchers that remain in place are shell entrypoints, not
Python wrappers:

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

The immediate repository evidence for that state is:

- `git show --stat --summary a63c8ec -- scripts scripts/ops`
- Commit `a63c8ec0f999d976a1af890c920a54ac2d6c693a` introduced the current
  extensionless shell launchers on `2026-04-06`.

## Validation Commands And Results

- `rg --files | rg '^(scripts|scripts/ops)/.*\.py$' | wc -l`
  Result: `0`
- `rg --files | rg '\.py$' | wc -l`
  Result: `0`
- `find scripts -maxdepth 3 -type f | sort`
  Result:
  - `scripts/dev_bootstrap.sh`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
  - `scripts/ops/bigclawctl`
- `sed -n '1,120p' scripts/ops/bigclawctl`
  Result: bash launcher that resolves the repo root and executes `go run ./cmd/bigclawctl`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1504(ScriptsAndOpsStayPythonFree|CompatibilityLaunchersRemainNonPython|LaneReportCapturesRepoReality)$'`
  Result: `ok  	bigclaw-go/internal/regression	1.067s`

## Outcome

`BIG-GO-1504` is blocked from reducing the physical `.py` file count because the
checked-out repository already has no `scripts/*.py` or `scripts/ops/*.py`
files left to delete. The delivered change is therefore a scoped regression
guard plus issue-specific validation evidence tied to the current checkout.
