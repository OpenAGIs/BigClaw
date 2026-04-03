# BIG-GO-1135 Validation

Date: 2026-04-04

## Scope

Issue: `BIG-GO-1135`

Title: `physical Python residual sweep 5`

This lane validates the candidate residual Python sweep list from the issue context against the
current worktree, records the already-materialized zero-`.py` repository baseline, and adds a
dedicated regression guard so the deleted benchmark, e2e, migration, and root-script Python assets
cannot silently reappear.

## Delivered

- `.symphony/workpad.md` now tracks `BIG-GO-1135` as the active workpad with plan, acceptance,
  validation commands, and the zero-baseline residual risk.
- `bigclaw-go/internal/regression/physical_python_residual_sweep5_test.go` enforces that the exact
  candidate list stays absent and that the documented Go or shell replacement surface still exists.
- `reports/BIG-GO-1135-closeout.md` and `reports/BIG-GO-1135-status.json` capture the issue-local
  validation evidence and branch metadata for unattended closeout.

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135 && git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l
```

Result:

```text
0
```

### Candidate reference scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135 && rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/dev_smoke\.py" bigclaw-go/docs/go-cli-script-migration.md
```

Result: exit code `1` with no matches

### Targeted regression tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go test ./internal/regression -run TestPhysicalPythonResidualSweep5
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.829s
```

### Go replacement help surfaces

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation benchmark capacity-certification [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation e2e run-task-smoke [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl automation migration shadow-compare [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go run ./cmd/bigclawctl create-issues --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl create-issues [flags]`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1135/bigclaw-go && go run ./cmd/bigclawctl dev-smoke --help | head -n 1
```

Result: exit code `0`, printed `usage: bigclawctl dev-smoke [flags]`

## Residual Risk

The issue acceptance asks for a lower repo-wide Python count, but this checkout already starts at
zero. This lane therefore closes by hardening and revalidating the migrated Go-only state rather
than by performing a fresh numeric reduction.
