# BIG-GO-205 Workpad

## Plan

1. Audit the residual repo-root tooling surfaces that still reference Python-based
   developer helpers and confirm the live checkout baseline.
2. Remove the checked-in Python tooling config from the root helper surface and
   update the documented repository hygiene path to use Go/shell-native commands
   already supported by this repo.
3. Add lane-scoped regression evidence for the deleted tooling config and the
   retained Go-only helper entrypoints:
   - `bigclaw-go/internal/regression/big_go_205_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-205-python-asset-sweep.md`
   - `reports/BIG-GO-205-validation.md`
   - `reports/BIG-GO-205-status.json`
4. Run the targeted validation commands, archive this workpad, then commit and
   push the issue-scoped change set to `origin/main`.

## Acceptance

- `.pre-commit-config.yaml` is no longer present in the repository root.
- `README.md` no longer directs operators to use `pre-commit`; the repository
  hygiene guidance points at retained Go/shell-native verification commands.
- `BIG-GO-205` adds regression coverage and lane reports that verify the Python
  tooling config stays absent while the retained root helper surface remains
  available.
- The exact validation commands and observed results are recorded in the lane
  artifacts, and the final change set is committed and pushed.

## Validation

- `test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-205/.pre-commit-config.yaml`
- `rg -n "pre-commit|ruff" /Users/openagi/code/bigclaw-workspaces/BIG-GO-205/README.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-205/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO205(ResidualPythonToolingConfigStaysAbsent|RootGoHelperSurfaceRemainsAvailable|LaneReportCapturesToolingSweep)$'`

## Execution Notes

- 2026-04-11: Initial inspection found no tracked `.py` files in the repo, but
  the root `.pre-commit-config.yaml` still referenced Python-based developer
  tooling and `README.md` still documented `pre-commit run --all-files` as the
  repository hygiene path.
