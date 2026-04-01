# BIG-GO-1053 Validation

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1053`

Title: `Go-replacement W: remove bigclaw-go e2e Python helpers tranche 2`

This lane closes the tranche-2 `bigclaw-go/scripts/e2e/` entrypoint migration by
verifying that the e2e script surface remains Python-free, that operator-facing
references point to Go-native `bigclawctl automation e2e ...` commands or retained
shell wrappers, and that regression coverage enforces the contract.

The code migration itself is already present in the checked-out baseline at
`004de016252d6ca168a45dccda48fc9fa69e27f1`
(`BIG-GO-1053: remove stale e2e Python entrypoint refs`). This report captures the
fresh validation evidence and the missing in-repo closeout artifacts for the lane.

## Delivered

- `bigclaw-go/scripts/e2e/` is Python-free and contains only:
  - `broker_bootstrap_summary.go`
  - `kubernetes_smoke.sh`
  - `ray_smoke.sh`
  - `run_all.sh`
- `bigclaw-go/docs/go-cli-script-migration.md` now lists only active Go and shell e2e
  entrypoints, including:
  - `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`
  - `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`
  - `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...`
  - `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`
  - `./scripts/e2e/run_all.sh`
  - `./scripts/e2e/kubernetes_smoke.sh`
  - `./scripts/e2e/ray_smoke.sh`
- `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go` enforces:
  - no `.py` files can reappear under `bigclaw-go/scripts/e2e/`
  - the e2e migration doc does not advertise removed tranche-2 Python helpers as
    active entrypoints

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053 -name '*.py' | wc -l
```

Result:

```text
46
```

Note: the tranche-2 e2e Python helpers were already absent in this checkout before the
validation branch was created, so the measurable repo-wide `.py` reduction occurred in
the baseline migration commit on `main`. This lane branch records the missing validation
and closeout evidence for that already-landed cutover.

### Stale reference scan

Command:

```bash
dirs=(); for p in README.md bigclaw-go/docs docs .github .husky .git/hooks; do [ -e "$p" ] && dirs+=("$p"); done; rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/e2e/.*\.py" "${dirs[@]}"
```

Result:

```text
no matches
```

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	3.707s
ok  	bigclaw-go/internal/regression	0.471s
```

### E2E command help checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e run-task-smoke [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e export-validation-bundle [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e continuation-scorecard [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1053/bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e continuation-policy-gate [flags]`

## Commit And Push

- Baseline migration commit on `main`: `004de016`
- Evidence branch: `symphony/BIG-GO-1053-validation`
- Push target: `origin/symphony/BIG-GO-1053-validation`

## Residual Risk

- This evidence branch does not change the already-landed code migration on `main`; it adds
  the missing validation and closeout artifacts for the lane.
- The repo-wide Python file count remains `46`, so any further reduction depends on follow-on
  lanes outside the scoped tranche-2 e2e entrypoint migration.
