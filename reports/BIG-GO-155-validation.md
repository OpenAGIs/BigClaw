# BIG-GO-155 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-155`

Title: `Residual tooling Python sweep K`

This lane removed the unused Python-only `ruff-pre-commit` dependency from the
root pre-commit config and hardened the remaining tooling/build-helper/dev-utility
surface with a focused regression guard and sweep report.

## Before And After Counts

- Repository-wide physical `.py` files before lane changes: `0`
- Repository-wide physical `.py` files after lane changes: `0`
- Focused tooling/build-helper/dev-utility physical `.py` files before lane changes: `0`
- Focused tooling/build-helper/dev-utility physical `.py` files after lane changes: `0`

## Exact Deleted-File Ledger

- Lane deletions: `[]`
- Removed tooling-only Python hooks: `["ruff-pre-commit", "ruff-check", "ruff-format"]`
- Focused tooling deletions: `[]`

## Go Replacement Paths

- `.pre-commit-config.yaml`
- `Makefile`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `.githooks/post-commit`
- `.githooks/post-rewrite`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/internal/githubsync/sync.go`
- `bigclaw-go/internal/refill/queue.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/cmd/bigclawctl /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/githubsync /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/refill /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`
- `rg -n 'ruff-pre-commit|ruff-check|ruff-format' /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.pre-commit-config.yaml`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO155(RepositoryHasNoPythonFiles|ResidualToolingSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155 -path '*/.git' -prune -o -name '*.py' -type f -print | sort
```

Result:

```text

```

### Focused tooling inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/cmd/bigclawctl /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/githubsync /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/refill /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text

```

### Pre-commit hook sweep

Command:

```bash
rg -n 'ruff-pre-commit|ruff-check|ruff-format' /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/.pre-commit-config.yaml
```

Result:

```text

```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-155/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO155(RepositoryHasNoPythonFiles|ResidualToolingSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.195s
```

## Git

- Branch: `BIG-GO-155`
- Baseline HEAD before lane commit: `52a22a80`
- Push target: `origin/BIG-GO-155`
- Compare URL: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-155?expand=1`
