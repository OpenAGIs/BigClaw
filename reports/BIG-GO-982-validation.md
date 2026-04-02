# BIG-GO-982 Validation Report

Date: `2026-04-02`

## Scope

Issue: `BIG-GO-982`

Title: `Final sweep B: root scripts and ops scripts`

This lane removes the last root-level and `scripts/ops` Python shims so the supported operator
surface is the Go CLI wrapper at `bash scripts/ops/bigclawctl ...`.

## Batch File List

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

## Disposition

- deleted `scripts/create_issues.py`; replaced by `bash scripts/ops/bigclawctl create-issues`
- deleted `scripts/dev_smoke.py`; replaced by `bash scripts/ops/bigclawctl dev-smoke`
- deleted `scripts/ops/bigclaw_github_sync.py`; replaced by `bash scripts/ops/bigclawctl github-sync`
- deleted `scripts/ops/bigclaw_refill_queue.py`; replaced by `bash scripts/ops/bigclawctl refill`
- deleted `scripts/ops/bigclaw_workspace_bootstrap.py`; replaced by `bash scripts/ops/bigclawctl workspace`
- deleted `scripts/ops/symphony_workspace_bootstrap.py`; replaced by `bash scripts/ops/bigclawctl workspace`
- deleted `scripts/ops/symphony_workspace_validate.py`; replaced by `bash scripts/ops/bigclawctl workspace validate`
- updated `README.md`, `docs/go-cli-script-migration-plan.md`, and `docs/go-mainline-cutover-issue-pack.md` so active operator guidance points at the Go CLI path instead of the retired Python wrappers

## Python Count Impact

- repo-wide Python files before: `116`
- repo-wide Python files after: `109`
- in-scope root/ops Python files before: `7`
- in-scope root/ops Python files after: `0`
- net reduction: `7`

## Validation

### Go CLI tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/bigclaw-go && go test ./cmd/bigclawctl
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	2.767s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	2.415s
ok  	bigclaw-go/internal/legacyshim	0.393s
```

### Operator command checks

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl create-issues --help
```

Result: exit `0`; printed `usage: bigclawctl create-issues [flags]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl dev-smoke
```

Result:

```text
smoke_ok local
```

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl github-sync status --json
```

Result:

```json
{
  "ahead": 0,
  "behind": 0,
  "branch": "symphony/BIG-GO-982",
  "detached": false,
  "dirty": false,
  "diverged": false,
  "local_sha": "bccf69fafa95d287c384bdd792781c8867715357",
  "pushed": true,
  "relation_known": true,
  "remote_exists": true,
  "remote_sha": "bccf69fafa95d287c384bdd792781c8867715357",
  "status": "ok",
  "synced": true
}
```

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl refill --help
```

Result: exit `0`; printed `usage: bigclawctl refill [flags]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl workspace --help
```

Result: exit `0`; printed `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts/ops/bigclawctl workspace validate --help
```

Result: exit `0`; printed `usage: bigclawctl workspace validate [flags]`

### File inventory checks

Command:

```bash
rg --files /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/scripts -g '*.py'
```

Result: no output

Command:

```bash
rg --files /Users/openagi/code/bigclaw-workspaces/BIG-GO-982 -g '*.py' | wc -l
```

Result:

```text
109
```

Command:

```bash
rg -n "scripts/(create_issues|dev_smoke)\.py|scripts/ops/(bigclaw_github_sync|bigclaw_refill_queue|bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\.py" /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/README.md /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/.symphony /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/src /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-982/bigclaw-go
```

Result: only expected migration-history/workpad references plus one helper-path fixture in
`bigclaw-go/internal/legacyshim/wrappers_test.go`; no active operator docs still route through the
deleted Python wrappers.

## Artifacts

- `.symphony/workpad.md`
- `reports/BIG-GO-982-closeout.md`
- `reports/BIG-GO-982-validation.md`
