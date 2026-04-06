# BIG-GO-1491 Validation

Date: 2026-04-06

## Scope

Issue: `BIG-GO-1491`

Title: `Refill: largest residual directory sweep for src/bigclaw Python files with before-after repo counts`

This lane audited the checked-out repository for physical Python files, with
explicit sweep focus on `src/bigclaw`, `tests`, `scripts`, and
`bigclaw-go/scripts`.

The baseline branch was already at a repository-wide Python file count of `0`,
so no physical `.py` file could be removed in-branch. The delivered change
documents the exact `0 -> 0` result and adds a targeted Go regression guard for
that baseline.

## Exact Counts

- Repository-wide Python files before sweep: `0`
- Repository-wide Python files after sweep: `0`
- `src/bigclaw` Python files before sweep: `0`
- `src/bigclaw` Python files after sweep: `0`
- `tests` Python files before sweep: `0`
- `tests` Python files after sweep: `0`
- `scripts` Python files before sweep: `0`
- `scripts` Python files after sweep: `0`
- `bigclaw-go/scripts` Python files before sweep: `0`
- `bigclaw-go/scripts` Python files after sweep: `0`

## Deleted File List

- None. No physical `.py` file remained in the checked-out branch baseline.

## Go Ownership Or Delete Conditions

- `src/bigclaw`: delete condition satisfied because the directory is absent.
- `tests`: delete condition satisfied because the directory is absent.
- `scripts`: owned by `scripts/dev_bootstrap.sh`, `scripts/ops/bigclawctl`,
  `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and
  `scripts/ops/bigclaw-symphony`.
- `bigclaw-go/scripts`: owned by `bigclaw-go/scripts/e2e/run_all.sh`,
  `bigclaw-go/cmd/bigclawctl/main.go`, and
  `bigclaw-go/cmd/bigclawd/main.go`.

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491 -path '*/.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1491(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoOwnershipAndDeleteConditionsRemainDocumented|LaneReportCapturesBeforeAfterCounts)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491 -path '*/.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort
```

Result:

```text

```

### Residual directory inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text

```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1491/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1491(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoOwnershipAndDeleteConditionsRemainDocumented|LaneReportCapturesBeforeAfterCounts)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.439s
```

## Git

- Branch: `BIG-GO-1491`
- Baseline HEAD before lane commit: `a63c8ec0`
- Push target: `origin/BIG-GO-1491`

## Blocker

- The checked-out baseline already had a repository-wide physical Python file
  count of `0`, so this lane could not numerically reduce the live `.py` count.
