# BIG-GO-1474 Ops Wrapper Sweep

## Scope

`BIG-GO-1474` audits the root `scripts/ops` surface to make sure deleted
Python-suffixed wrappers stay deleted and are not treated as live physical
Python assets in the repository inventory.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `scripts/ops`: `0` Python files
- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

This lane therefore lands as doc cleanup plus regression hardening rather than
an in-branch Python-file deletion batch.

## Deleted Wrapper Set And Go Ownership

The following root `scripts/ops` Python wrappers remain deleted and should stay
out of the physical Python asset inventory:

- `scripts/ops/bigclaw_github_sync.py`
  Replacement: `scripts/ops/bigclawctl github-sync`
  Go ownership: `bigclaw-go/internal/githubsync/sync.go`
- `scripts/ops/bigclaw_refill_queue.py`
  Replacement: `scripts/ops/bigclawctl refill`
  Go ownership: `bigclaw-go/internal/refill/queue.go`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  Replacement: `scripts/ops/bigclawctl workspace bootstrap`
  Go ownership: `bigclaw-go/internal/bootstrap/bootstrap.go`
- `scripts/ops/symphony_workspace_bootstrap.py`
  Replacement: `scripts/ops/bigclawctl workspace bootstrap`
  Go ownership: `bigclaw-go/internal/bootstrap/bootstrap.go`
- `scripts/ops/symphony_workspace_validate.py`
  Replacement: `scripts/ops/bigclawctl workspace validate`
  Go ownership: `bigclaw-go/internal/bootstrap/bootstrap.go`

Active operator entrypoints still present in the repo:

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find scripts/ops -type f -name '*.py' -print | sort`
  Result: no output; `scripts/ops` remained Python-free.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority residual directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1474(OpsDirectoryHasNoPythonFiles|DeletedPythonWrapperPathsStayAbsent|ActiveGoReplacementPathsRemainAvailable|DocsAndLaneReportPinDeletedWrapperState)$'`
  Result: `ok  	bigclaw-go/internal/regression	1.549s`
