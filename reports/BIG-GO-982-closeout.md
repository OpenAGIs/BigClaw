# BIG-GO-982 Closeout

Issue: `BIG-GO-982`

Title: `Final sweep B: root scripts and ops scripts`

Date: `2026-04-02`

## Branch

`symphony/BIG-GO-982`

## Delivered

- retired the remaining root-level and `scripts/ops` Python shims:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- kept the supported operator path on `bash scripts/ops/bigclawctl ...`
- refreshed operator-facing docs so they no longer describe the deleted Python shims as active entrypoints
- recorded the plan, acceptance criteria, file disposition, and validation evidence in `.symphony/workpad.md`

## Replacement Map

- `scripts/create_issues.py` -> `bash scripts/ops/bigclawctl create-issues`
- `scripts/dev_smoke.py` -> `bash scripts/ops/bigclawctl dev-smoke`
- `scripts/ops/bigclaw_github_sync.py` -> `bash scripts/ops/bigclawctl github-sync`
- `scripts/ops/bigclaw_refill_queue.py` -> `bash scripts/ops/bigclawctl refill`
- `scripts/ops/bigclaw_workspace_bootstrap.py` -> `bash scripts/ops/bigclawctl workspace`
- `scripts/ops/symphony_workspace_bootstrap.py` -> `bash scripts/ops/bigclawctl workspace`
- `scripts/ops/symphony_workspace_validate.py` -> `bash scripts/ops/bigclawctl workspace validate`

## Python File Count Impact

- repository Python files before this sweep: `116`
- repository Python files after the sweep: `109`
- in-scope root/ops Python files before this sweep: `7`
- in-scope root/ops Python files after the sweep: `0`
- net reduction: `7`

## Validation

```bash
cd bigclaw-go && go test ./cmd/bigclawctl
bash scripts/ops/bigclawctl create-issues --help
bash scripts/ops/bigclawctl dev-smoke
bash scripts/ops/bigclawctl github-sync status --json
bash scripts/ops/bigclawctl refill --help
bash scripts/ops/bigclawctl workspace --help
bash scripts/ops/bigclawctl workspace validate --help
cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim
rg --files scripts -g '*.py'
rg --files . -g '*.py' | wc -l
```

Latest verified results on this branch:

- `cd bigclaw-go && go test ./cmd/bigclawctl` -> `ok   bigclaw-go/cmd/bigclawctl	2.767s`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`
- `bash scripts/ops/bigclawctl dev-smoke` -> `smoke_ok local`
- `bash scripts/ops/bigclawctl github-sync status --json` -> status `ok`, branch synced
- `bash scripts/ops/bigclawctl refill --help` -> exit `0`
- `bash scripts/ops/bigclawctl workspace --help` -> exit `0`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim` -> passed
- `rg --files scripts -g '*.py'` -> no output
- `rg --files . -g '*.py' | wc -l` -> `109`

## Implementation Commits

- `bccf69fafa95d287c384bdd792781c8867715357` `BIG-GO-982: retire root and ops Python shims`
- `6e77fee3f55f79d8638dfb02d2957adcaa46fe28` `BIG-GO-982: refresh final sweep docs`

Later branch commits add reviewer artifacts and do not change the retired-script outcome summarized here.

## Artifacts

- `.symphony/workpad.md`
- `docs/go-cli-script-migration-plan.md`
- `docs/go-mainline-cutover-issue-pack.md`
- `README.md`
- `reports/BIG-GO-982-closeout.md`
