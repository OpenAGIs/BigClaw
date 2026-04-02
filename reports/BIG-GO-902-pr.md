# BIG-GO-902 PR Draft

## Suggested Title

`BIG-GO-902: migrate script entrypoints to Go CLI`

## Suggested Description

### Summary

- move the remaining repo-root script compatibility layer behind Go-owned `bigclawctl` commands
- migrate the first `bigclaw-go/scripts/*` automation batch into `bigclawctl automation ...`
- keep the old script file names as thin shims during cutover so automation call sites do not
  break before the later retirement cleanup
- refresh the migration plan, validation report, closeout index, and status artifact for reviewers

### Delivered

- Go CLI continues to own the behavior for:
  - `create-issues`
  - `dev-smoke`
  - `github-sync`
  - `refill`
  - `workspace`
  - `symphony`
  - `issue`
  - `panel`
- converted or retained these entrypoints as compatibility shims:
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
- centralized common shim behavior in `src/bigclaw/legacy_shim.py`
- added Go-owned automation commands for:
  - `bigclawctl automation e2e run-task-smoke`
  - `bigclawctl automation benchmark soak-local`
  - `bigclawctl automation migration shadow-compare`
- converted these `bigclaw-go/scripts/*` entrypoints into compatibility shims:
  - `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`
  - `bigclaw-go/scripts/migration/shadow_compare.py`
- updated reviewer/operator docs:
  - `README.md`
  - `docs/go-cli-script-migration-plan.md`
  - `bigclaw-go/docs/go-cli-script-migration.md`
  - `reports/BIG-GO-902-validation.md`
  - `reports/BIG-GO-902-closeout.md`
  - `reports/BIG-GO-902-status.json`
  - `local-issues.json`

### Validation

```bash
cd bigclaw-go && go test ./cmd/bigclawctl ./internal/refill
python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
python3 -m py_compile src/bigclaw/legacy_shim.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/create_issues.py scripts/dev_smoke.py
bash scripts/ops/bigclawctl dev-smoke
python3 scripts/dev_smoke.py
python3 scripts/create_issues.py --help
bash scripts/ops/bigclawctl issue --help
python3 scripts/ops/bigclaw_github_sync.py --help
python3 scripts/ops/bigclaw_workspace_bootstrap.py --help
python3 scripts/ops/symphony_workspace_bootstrap.py --help
python3 scripts/ops/bigclaw_refill_queue.py --help
python3 scripts/ops/symphony_workspace_validate.py --help
python3 scripts/ops/bigclaw_github_sync.py status --json
cd bigclaw-go && go test ./cmd/bigclawctl/...
cd bigclaw-go && go run ./cmd/bigclawctl automation --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help
```

### Risks / Deferred Follow-ups

- `scripts/dev_bootstrap.sh` remains shell-owned and was not migrated in this slice
- `scripts/ops/bigclawctl` still shells into `go run`, so local Go toolchain availability remains required
- remaining `bigclaw-go/scripts/*` helpers beyond the first migrated batch are still deferred

## Open PR URL

`https://github.com/OpenAGIs/BigClaw/pull/215`

## Compare URL

`https://github.com/OpenAGIs/BigClaw/compare/main...feat/BIG-GO-902-go-cli-script-migration?expand=1`

## Public Verification Note

As of 2026-03-28, authenticated GitHub API creation succeeded and opened PR `#215` for this
branch. The compare URL still appears stale and continues to report only `14 commits` /
`26 files changed` plus repeated `Uh oh!` load failures, so the remaining concern is GitHub-side
compare rendering rather than PR creation or branch publication.
