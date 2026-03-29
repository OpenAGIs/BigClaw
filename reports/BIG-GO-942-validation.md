# BIG-GO-942 Validation Report

Date: 2026-03-29

## Scope

Issue: `BIG-GO-942`

Title: `Lane2 Root scripts to Go CLI`

Lane file list delivered in this slice:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

## Delivered

- Replaced the seven lane-scoped Python implementations above with shell wrappers that dispatch into
  `scripts/ops/bigclawctl`.
- Preserved wrapper compatibility behavior required by existing callers:
  - `scripts/ops/bigclaw_workspace_bootstrap.py` still injects default `--repo-url` and `--cache-key`.
  - `scripts/ops/symphony_workspace_validate.py` still translates `--report-file`,
    `--no-cleanup`, and positional `--issues ...`.
- Updated wrapper regression coverage in `tests/test_legacy_shim.py`.
- Updated operator guidance in `README.md`.
- Updated migration notes in `docs/go-cli-script-migration-plan.md`.
- Recorded execution plan, acceptance, validation, and risk in `.symphony/workpad.md`.

## Validation

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942/bigclaw-go && go test ./cmd/bigclawctl
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	2.946s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
```

Result:

```text
.................                                                        [100%]
17 passed in 1.36s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && bash scripts/dev_smoke.py
```

Result:

```text
stderr: scripts/dev_smoke.py is a legacy wrapper; use bash scripts/ops/bigclawctl dev-smoke.
stdout: smoke_ok local
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && bash scripts/create_issues.py --help
```

Result: usage for `bigclawctl create-issues`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && bash scripts/ops/bigclaw_refill_queue.py --help
```

Result: usage for `bigclawctl refill`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && bash scripts/ops/bigclaw_github_sync.py status --json
```

Result:

```json
{
  "ahead": 0,
  "behind": 0,
  "branch": "symphony/BIG-GO-942",
  "detached": false,
  "dirty": true,
  "diverged": false,
  "local_sha": "2bb918819564bb3580c1ed92b1b53dfc5feac5e3",
  "pushed": true,
  "relation_known": true,
  "remote_exists": true,
  "remote_sha": "2bb918819564bb3580c1ed92b1b53dfc5feac5e3",
  "status": "ok",
  "synced": true
}
```

Note: this `github-sync status` sample was captured while the validation artifact refresh was
unstaged on `symphony/BIG-GO-942`, so `dirty: true` is expected for that run.

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && BIGCLAW_BOOTSTRAP_REPO_URL=<tmp seeded bare repo with main> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py bootstrap --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
```

Result: passed with `workspace_mode: worktree_created`, `mirror_created: true`, and `seed_created: true`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && BIGCLAW_BOOTSTRAP_REPO_URL=<tmp seeded bare repo with main> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py cleanup --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
```

Result: passed with `workspace_mode: cleanup`, `removed: true`, and `cache_reused: true`

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-942 && bash scripts/ops/symphony_workspace_validate.py --repo-url <tmp seeded bare repo with main> --workspace-root <tmp>/validate --issues COMPAT-VAL-1 COMPAT-VAL-2 --report-file <tmp>/report.json --no-cleanup --json
```

Result: passed with `workspace_count: 2`; report file emitted successfully; summary confirmed one
shared cache root, one shared mirror, one shared seed, and worktree creation for both workspaces

## Branch State

- Branch: `symphony/BIG-GO-942`
- Last validated implementation commit: `2bb918819564bb3580c1ed92b1b53dfc5feac5e3`
- `git status --short --branch` is clean against `origin/symphony/BIG-GO-942`
- Later branch commits may refresh only report metadata and can advance the branch head without
  changing the validated wrapper behavior captured above.

## Risks

- The legacy wrapper paths still end in `.py`, but they now require shell execution semantics.
  Any caller still hardcoding `python3 <wrapper>.py` must switch to `bash <wrapper>.py` or
  `bash scripts/ops/bigclawctl ...`.
- `scripts/ops/bigclawctl` still uses `go run`, so wrapper latency and local Go toolchain
  availability remain operator dependencies.
