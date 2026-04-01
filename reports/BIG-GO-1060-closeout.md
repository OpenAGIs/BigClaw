# BIG-GO-1060 Closeout Index

Issue: `BIG-GO-1060`

Title: `Go-replacement AD: remove residual Python entrypoints tracked in README/workflow`

Date: `2026-04-01`

## Branch

`symphony/BIG-GO-1060`

## Latest Code Migration Commit

`c8f270b1fa5e58ae413561d29aae983a8d7ab55e`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1060-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-1060-status.json`
- Migration plan:
  - `docs/go-cli-script-migration-plan.md`
- Workpad:
  - `.symphony/workpad.md`

## Outcome

- The repo no longer ships Python operator entrypoints for refill or workspace flows under
  `scripts/ops/`.
- README, workflow-adjacent docs, hooks, and CI now track the Go entrypoints instead of treating
  the deleted Python wrappers as supported defaults.
- `bigclaw-go/internal/regression/operator_entrypoint_cutover_test.go` now enforces both wrapper
  deletion and Go-only operator references across the tracked repo surfaces.
- The repository `.py` count drops from `45` in `HEAD^` to `41` in `HEAD`.

## Validation Commands

```bash
git ls-tree -r --name-only HEAD^ | rg '\.py$' | wc -l
git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l
cd bigclaw-go && go test ./internal/regression -run 'TestResidualPythonOperatorEntrypointsStayDeleted|TestTrackedOperatorSurfacesStayGoOnly|TestE2EMigrationDocListsOnlyActiveEntrypoints'
cd bigclaw-go && go test ./internal/legacyshim
bash scripts/ops/bigclawctl refill --help | sed -n '1,20p'
bash scripts/ops/bigclawctl workspace validate --help | sed -n '1,20p'
rg -n "python3 scripts/ops/bigclaw_refill_queue\.py|scripts/ops/\*workspace\*\.py|python3 scripts/ops/symphony_workspace_validate\.py|python3 scripts/ops/bigclaw_workspace_bootstrap\.py|python3 scripts/ops/symphony_workspace_bootstrap\.py" README.md .github/workflows/ci.yml workflow.md .githooks docs/go-cli-script-migration-plan.md
git status --short --branch && git rev-parse HEAD && git rev-parse origin/symphony/BIG-GO-1060 && git log -1 --stat --oneline
```

## Remaining Risk

No blocking repo action remains for `BIG-GO-1060`.

The only caveat is tracking-related: `BIG-GO-1060` is not mirrored in `local-issues.json`, so
this branch records closeout evidence in `reports/` rather than a repo-local tracker state change.

## Final Repo Check

- `git status --short --branch` was clean against `origin/symphony/BIG-GO-1060` when the closeout
  was recorded.
- `git rev-parse HEAD` matched `git rev-parse origin/symphony/BIG-GO-1060` before this
  report-only follow-up commit was created.
