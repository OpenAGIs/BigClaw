# BIG-GO-1483 Validation

Date: 2026-04-06

## Scope

Issue: `BIG-GO-1483`

Title: `Refill: remove remaining physical Python files under bigclaw-go/scripts by switching all checked-in callers to Go CLI`

This lane found `bigclaw-go/scripts` already physically Python-free in the
checked-out branch baseline, so the scoped implementation removes the last
checked-in migration-plan references to retired `bigclaw-go/scripts/*.py`
entrypoints and hardens that caller-cutover state with regression coverage.

## Before And After

- Repository-wide physical `.py` files before: `0`
- Repository-wide physical `.py` files after: `0`
- `bigclaw-go/scripts/*.py` before: `0`
- `bigclaw-go/scripts/*.py` after: `0`
- Checked-in caller references to retired `bigclaw-go/scripts` Python entrypoints before: `23`
- Checked-in caller references to retired `bigclaw-go/scripts` Python entrypoints after: `0`

## Delivered Changes

- Updated `docs/go-cli-script-migration-plan.md` so the `bigclaw-go/scripts`
  slice now lists only Go CLI or retained shell/Go entrypoints.
- Added `bigclaw-go/internal/regression/big_go_1483_checked_in_caller_cutover_test.go`
  to lock the caller-cutover state and lane evidence.
- Refreshed `bigclaw-go/internal/regression/big_go_1160_script_migration_test.go`
  so the older migration guard now rejects stale `bigclaw-go/scripts/*.py`
  references in the migration plan.
- Added lane evidence in `bigclaw-go/docs/reports/big-go-1483-python-asset-sweep.md`.

## Validation Commands

- `git show a63c8ec0f999d976a1af890c920a54ac2d6c693a:docs/go-cli-script-migration-plan.md | rg -n 'bigclaw-go/scripts/.*\.py' | wc -l | tr -d ' '`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go/scripts -type f -name '*.py' | sort`
- `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\\.py' /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/README.md /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/.github /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'`

## Validation Results

### Baseline stale-reference count

Command:

```bash
git show a63c8ec0f999d976a1af890c920a54ac2d6c693a:docs/go-cli-script-migration-plan.md | rg -n 'bigclaw-go/scripts/.*\.py' | wc -l | tr -d ' '
```

Result:

```text
23
```

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483 -path '*/.git' -prune -o -name '*.py' -type f -print | sort
```

Result:

```text

```

### `bigclaw-go/scripts` Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go/scripts -type f -name '*.py' | sort
```

Result:

```text

```

### Active caller references after update

Command:

```bash
rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\\.py' /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/README.md /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/.github /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go | sort
```

Result:

```text

```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1483/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.975s
```
