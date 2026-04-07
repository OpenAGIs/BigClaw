# BIG-GO-1567 Scripts/Ops Deletion Tranche

## Summary

`BIG-GO-1567` audited the current `scripts/ops` surface after checkout from `origin/main`.
The repository baseline is already Python-free, so this lane lands exact Go/native replacement
evidence for the scripts/ops tranche instead of a fresh file deletion.

Repository-wide Python file count: `0`.

## Exact Replacement Map

- `scripts/ops/bigclawctl` is a Bash compatibility shim that dispatches to `go run ./cmd/bigclawctl`.
- `scripts/ops/bigclaw-issue` maps to `bash scripts/ops/bigclawctl issue`.
- `scripts/ops/bigclaw-panel` maps to `bash scripts/ops/bigclawctl panel`.
- `scripts/ops/bigclaw-symphony` maps to `bash scripts/ops/bigclawctl symphony`.
- retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`.
- retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`.
- retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`.
- retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`.
- retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`.

## Validation Commands

- `find . -name '*.py' | wc -l`
- `find scripts/ops -maxdepth 1 -type f | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps'`
