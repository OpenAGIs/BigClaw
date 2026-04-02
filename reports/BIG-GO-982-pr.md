# BIG-GO-982 PR Draft

## Title

`BIG-GO-982: retire root and ops Python shims`

## Summary

- remove the remaining Python entrypoints under `scripts/*.py` and `scripts/ops/*.py`
- keep the supported operator path on `bash scripts/ops/bigclawctl ...`
- refresh migration and cutover docs so they no longer describe the deleted Python wrappers as active
- publish reviewer artifacts for the final sweep outcome, validation evidence, and replacement map

## Scope

Deleted Python files:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

Replacement commands:

- `scripts/create_issues.py` -> `bash scripts/ops/bigclawctl create-issues`
- `scripts/dev_smoke.py` -> `bash scripts/ops/bigclawctl dev-smoke`
- `scripts/ops/bigclaw_github_sync.py` -> `bash scripts/ops/bigclawctl github-sync`
- `scripts/ops/bigclaw_refill_queue.py` -> `bash scripts/ops/bigclawctl refill`
- `scripts/ops/bigclaw_workspace_bootstrap.py` -> `bash scripts/ops/bigclawctl workspace`
- `scripts/ops/symphony_workspace_bootstrap.py` -> `bash scripts/ops/bigclawctl workspace`
- `scripts/ops/symphony_workspace_validate.py` -> `bash scripts/ops/bigclawctl workspace validate`

Python count impact:

- repo-wide Python files: `116 -> 109`
- in-scope root/ops Python files: `7 -> 0`

## Validation

```bash
cd bigclaw-go && go test ./cmd/bigclawctl
cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim
bash scripts/ops/bigclawctl create-issues --help
bash scripts/ops/bigclawctl dev-smoke
bash scripts/ops/bigclawctl github-sync status --json
bash scripts/ops/bigclawctl refill --help
bash scripts/ops/bigclawctl workspace --help
bash scripts/ops/bigclawctl workspace validate --help
rg --files scripts -g '*.py'
rg --files . -g '*.py' | wc -l
```

Results:

- `go test ./cmd/bigclawctl` -> passed
- `go test ./cmd/bigclawctl ./internal/legacyshim` -> passed
- `create-issues --help` -> passed
- `dev-smoke` -> `smoke_ok local`
- `github-sync status --json` -> clean synced branch state
- `refill --help` -> passed
- `workspace --help` -> passed
- `workspace validate --help` -> passed
- `rg --files scripts -g '*.py'` -> no output
- `rg --files . -g '*.py' | wc -l` -> `109`

## Reviewer Artifacts

- `.symphony/workpad.md`
- `reports/BIG-GO-982-closeout.md`
- `reports/BIG-GO-982-validation.md`
- `reports/BIG-GO-982-status.json`
- `reports/BIG-GO-982-pr.md`
