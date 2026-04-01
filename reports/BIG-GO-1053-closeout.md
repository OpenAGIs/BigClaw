# BIG-GO-1053 Closeout Index

Issue: `BIG-GO-1053`

Title: `Go-replacement W: remove bigclaw-go e2e Python helpers tranche 2`

Date: `2026-04-01`

## Branch

`symphony/BIG-GO-1053-validation`

## Latest Code Migration Commit

`004de016`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1053-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-1053-status.json`
- Automation migration matrix:
  - `bigclaw-go/docs/go-cli-script-migration.md`
- Regression guard:
  - `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`
- Workpad:
  - `.symphony/workpad.md`

## Outcome

- `bigclaw-go/scripts/e2e/` remains Go-only with no tracked Python files.
- Active tranche-2 e2e entrypoints resolve through:
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e run-task-smoke ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e export-validation-bundle ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e continuation-scorecard ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e continuation-policy-gate ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e mixed-workload-matrix ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e cross-process-coordination-surface ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e external-store-validation ...`
  - `go run ./bigclaw-go/cmd/bigclawctl automation e2e multi-node-shared-queue ...`
  - `bigclaw-go/scripts/e2e/run_all.sh`
  - `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
  - `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go` prevents deleted
  tranche-2 Python helper filenames from reappearing in the e2e script directory or
  the active migration doc.

## Validation Commands

```bash
find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
find . -name '*.py' | wc -l
dirs=(); for p in README.md bigclaw-go/docs docs .github .husky .git/hooks; do [ -e "$p" ] && dirs+=("$p"); done; rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/e2e/.*\.py" "${dirs[@]}"
cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
```

## Remaining Risk

No blocking repo action remains for `BIG-GO-1053`.

The only caveat is historical: the tranche-2 e2e Python helpers had already been removed
in the baseline `main` commit before this evidence branch was created, so this branch adds
the missing validation and closeout artifacts rather than another fresh `.py` deletion.

## Final Repo Check

- `git status --short --branch` is clean on `symphony/BIG-GO-1053-validation` after the
  closeout artifacts are committed.
- `git rev-parse origin/main` points to the landed migration commit `004de016252d6ca168a45dccda48fc9fa69e27f1`.
