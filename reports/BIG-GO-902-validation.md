# BIG-GO-902 Validation Report

Date: 2026-03-28

## Scope

Issue: `BIG-GO-902`

Title: `脚本层迁移到 Go CLI`

This issue now covers two delivered migration batches:

- the repo-root script migration lane, which keeps behavior in the Go CLI while retaining the old
  script file names as compatibility shims
- the first `bigclaw-go/scripts/*` automation migration batch, which moves selected e2e,
  benchmark, and migration helpers into `bigclawctl automation ...`

## Delivered

- Go CLI remains the implementation owner for these migrated entrypoints:
  - `create-issues`
  - `dev-smoke`
  - `github-sync`
  - `refill`
  - `workspace`
  - `symphony`
  - `issue`
  - `panel`
- Compatibility shims now dispatch into `scripts/ops/bigclawctl` for:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
  - `scripts/ops/bigclaw-symphony`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
- Shared shim behavior and path resolution are centralized in:
  - `src/bigclaw/legacy_shim.py`
- Go CLI now also owns the first `bigclaw-go/scripts/*` automation batch:
  - `automation e2e run-task-smoke`
  - `automation benchmark soak-local`
  - `automation migration shadow-compare`
- Compatibility shims now dispatch directly into those Go automation commands for:
  - `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`
  - `bigclaw-go/scripts/migration/shadow_compare.py`
- Migration docs and operator guidance were refreshed in:
  - `docs/go-cli-script-migration-plan.md`
  - `bigclaw-go/docs/go-cli-script-migration.md`
  - `README.md`
  - `.symphony/workpad.md`

## Validation

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go test ./cmd/bigclawctl ./internal/refill
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	2.651s
ok  	bigclaw-go/internal/refill	(cached)
```

### Targeted Python tests

Command:

```bash
python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
```

Result:

```text
.................                                                        [100%]
17 passed in 1.76s
```

### Python syntax check

Command:

```bash
python3 -m py_compile src/bigclaw/legacy_shim.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/create_issues.py scripts/dev_smoke.py
```

Result: exit code `0`

### Command-level checks

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclawctl dev-smoke
```

Result:

```text
smoke_ok local
```

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/dev_smoke.py
```

Result:

```text
/Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/dev_smoke.py:17: DeprecationWarning: scripts/dev_smoke.py is frozen for migration-only use. bigclaw-go is the sole implementation mainline for active development; the legacy Python runtime surface remains migration-only. Use bash scripts/ops/bigclawctl dev-smoke instead.
  warn_legacy_runtime_surface("scripts/dev_smoke.py", "bash scripts/ops/bigclawctl dev-smoke")
smoke_ok local
```

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/create_issues.py --help
```

Result: usage for `bigclawctl create-issues`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclawctl issue --help
```

Result: usage for `bigclawctl issue`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw_github_sync.py --help
```

Result: usage for `bigclawctl github-sync`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw_workspace_bootstrap.py --help
```

Result: usage for `bigclawctl workspace <bootstrap|cleanup|validate>`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/symphony_workspace_bootstrap.py --help
```

Result: usage for `bigclawctl workspace <bootstrap|cleanup|validate>`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw_refill_queue.py --help
```

Result: usage for `bigclawctl refill`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/symphony_workspace_validate.py --help
```

Result: usage for `bigclawctl workspace validate`

Command:

```bash
python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw_github_sync.py status --json
```

Result:

```json
{
  "ahead": 0,
  "behind": 0,
  "branch": "feat/BIG-GO-902-go-cli-script-migration",
  "detached": false,
  "dirty": false,
  "diverged": false,
  "local_sha": "7bf0f59f3c8649328cabaca1e619136fbf114a30",
  "pushed": true,
  "relation_known": true,
  "remote_exists": true,
  "remote_sha": "7bf0f59f3c8649328cabaca1e619136fbf114a30",
  "status": "ok",
  "synced": true
}
```

Note: this check reflects the clean pushed branch state at the time of the last `github-sync status`
verification. Subsequent commits in this lane are metadata-only report syncs unless explicitly noted.

### Automation command checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go test ./cmd/bigclawctl/...
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	4.026s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go run ./cmd/bigclawctl automation --help
```

Result: exited `0`, printed automation category help for `e2e`, `benchmark`, and `migration`.

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
```

Result: exited `0`, printed `run-task-smoke` flag help.

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
```

Result: exited `0`, printed `soak-local` flag help.

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help
```

Result: exited `0`, printed `shadow-compare` flag help.

## Branch and PR

Branch:

```text
feat/BIG-GO-902-go-cli-script-migration
```

Latest code migration commit:

```text
3fe203ebcd99f0f054911c84cf6929a42af18f64
```

Last root-shim branch head verified via `github-sync status`:

```text
834f6441cd06fff89bb6b9305b27fa3ca0ddd21f
```

Note: later branch commits after `3fe203e...` only refreshed BIG-GO-902 tracker/report metadata,
opened PR `#215`, and recorded the final merge closeout; they did not change the migrated Go CLI
behavior validated above.

PR seed URL:

```text
https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-902-go-cli-script-migration
```

PR URL:

```text
https://github.com/OpenAGIs/BigClaw/pull/215
```

Compare URL:

```text
https://github.com/OpenAGIs/BigClaw/compare/main...feat/BIG-GO-902-go-cli-script-migration?expand=1
```

PR draft:

```text
reports/BIG-GO-902-pr.md
```

Closeout index:

```text
reports/BIG-GO-902-closeout.md
```

Machine-readable status:

```text
reports/BIG-GO-902-status.json
```

Public GitHub verification on 2026-03-28:

- Authenticated GitHub API creation succeeded with PR `#215`:
  `https://github.com/OpenAGIs/BigClaw/pull/215`.
- PR `#215` was later merged into `main` at `2026-03-27T17:59:20Z` as squash commit
  `56c8efbda59344f850890bfe2e8d835016ff1b3d`.
- The compare page had previously been stale and reported only `14 commits` / `26 files changed`;
  that rendering issue is no longer a delivery blocker because the merge completed successfully.

## Regression Surface

- Legacy workspace wrapper flag translation:
  `--issues`, `--report-file`, and `--no-cleanup` still need to map cleanly onto Go workspace
  validation flags.
- Root-level Python shim execution:
  direct Python entrypoints now self-bootstrap `src`, so regression checks need to keep that
  behavior intact without relying on environment variables.
- Operator invocation path:
  `scripts/ops/bigclawctl` is still the preferred human/operator entrypoint while the compatibility
  files remain in place.

## Risks and Deferred Follow-ups

- `scripts/dev_bootstrap.sh` remains a shell-owned bootstrap path and was intentionally left out of
  this migration slice.
- `scripts/ops/bigclawctl` still shells into `go run`, so local Go toolchain availability and
  startup latency remain operator dependencies.
- The remaining `bigclaw-go/scripts/*` backlog listed in `bigclaw-go/docs/go-cli-script-migration.md`
  is still deferred to later migration batches.
- No blocker remains for BIG-GO-902. The implementation is merged.
