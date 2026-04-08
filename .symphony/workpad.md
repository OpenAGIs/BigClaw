# BIG-GO-155 Workpad

## Plan

1. Confirm the current repository baseline for checked-in Python assets and inspect the existing root-tooling migration surfaces.
2. Remove the unused Python-only `ruff-pre-commit` hook from the root `.pre-commit-config.yaml`.
3. Add a focused `BIG-GO-155` residual tooling/build-helper/dev-utility sweep report under `bigclaw-go/docs/reports/`.
4. Add a focused regression guard under `bigclaw-go/internal/regression/` that:
   - keeps the repository Python-free,
   - keeps the selected tooling/dev-utility residual paths absent,
   - pins the active Go/native replacement paths, and
   - verifies the lane report records the exact sweep ledger and validation commands.
5. Record targeted validation evidence in `reports/BIG-GO-155-validation.md` and `reports/BIG-GO-155-status.json`.
6. Commit and push branch `BIG-GO-155`.

## Acceptance

- `BIG-GO-155` remains scoped to residual tooling, build-helper, and dev-utility cleanup only.
- The unused Python-only `ruff-pre-commit` dependency is removed from the root pre-commit configuration.
- The repository still has no checked-in `.py` files after the change.
- A new focused regression test protects the selected residual tooling/dev-utility surface from Python reintroduction.
- A new lane report documents exact before/after counts, focused ledger entries, replacement paths, and validation commands.
- Validation artifacts capture the exact commands run and their results.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/cmd/bigclawctl /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/githubsync /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/refill /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`
- `rg -n 'ruff-pre-commit|ruff-check|ruff-format' /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.pre-commit-config.yaml`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO155(RepositoryHasNoPythonFiles|ResidualToolingSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
