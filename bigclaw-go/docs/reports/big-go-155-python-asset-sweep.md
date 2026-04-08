# BIG-GO-155 Python Asset Sweep

## Scope

`BIG-GO-155` tightens the remaining repo tooling/build-helper/dev-utility
surface. This lane removes the dead Python-only `ruff-pre-commit` dependency
from the root pre-commit config and records the active Go/native replacements
that now carry the operator workflow.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused tooling/build-helper/dev-utility physical Python file count before lane changes: `0`
- Focused tooling/build-helper/dev-utility physical Python file count after lane changes: `0`

This checkout was already Python-free before the lane started, so the shipped
work lands as residual tooling reduction plus regression hardening rather than a
checked-in Python file deletion batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Removed tooling-only Python hooks in this lane: `["ruff-pre-commit", "ruff-check", "ruff-format"]`

Focused tooling ledger: `[]`

## Residual Scan Detail

- `.pre-commit-config.yaml`: removed residual Python-only `ruff-pre-commit` hooks
- `scripts`: `0` Python files
- `.githooks`: `0` Python files
- `bigclaw-go/cmd/bigclawctl`: `0` Python files
- `bigclaw-go/internal/githubsync`: `0` Python files
- `bigclaw-go/internal/refill`: `0` Python files
- `bigclaw-go/internal/bootstrap`: `0` Python files

## Go Or Native Replacement Paths

The active Go/native replacement surface for this residual area remains:

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

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find scripts .githooks bigclaw-go/cmd/bigclawctl bigclaw-go/internal/githubsync bigclaw-go/internal/refill bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the focused tooling/build-helper/dev-utility surface remained Python-free.
- `rg -n 'ruff-pre-commit|ruff-check|ruff-format' .pre-commit-config.yaml`
  Result: no output; the residual Python-only pre-commit hooks were removed.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO155(RepositoryHasNoPythonFiles|ResidualToolingSurfaceStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.195s`
