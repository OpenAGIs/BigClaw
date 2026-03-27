# BIG-GO-902 PR Draft

## Suggested Title

`BIG-GO-902: migrate repo script entrypoints to Go CLI`

## Suggested Description

### Summary

- move the remaining repo-root script compatibility layer behind Go-owned `bigclawctl` commands
- keep the old script file names as thin shims so automation call sites do not break during cutover
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
- updated reviewer/operator docs:
  - `README.md`
  - `docs/go-cli-script-migration-plan.md`
  - `reports/BIG-GO-902-validation.md`
  - `reports/BIG-GO-902-closeout.md`
  - `reports/BIG-GO-902-status.json`
  - `local-issues.json`

### Validation

```bash
cd bigclaw-go && go test ./cmd/bigclawctl ./internal/refill
python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
python3 -m py_compile src/bigclaw/legacy_shim.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py
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
```

### Risks / Deferred Follow-ups

- `scripts/dev_bootstrap.sh` remains shell-owned and was not migrated in this slice
- `scripts/ops/bigclawctl` still shells into `go run`, so local Go toolchain availability remains required
- `bigclaw-go/scripts/*` helper scripts remain outside this root-level script migration scope

## Open PR URL

`https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-902-go-cli-script-migration`

## Compare URL

`https://github.com/OpenAGIs/BigClaw/compare/main...feat/BIG-GO-902-go-cli-script-migration?expand=1`

## Public Verification Note

As of 2026-03-28, public GitHub search still showed no PR result for this branch/title, while the
compare URL remained reachable and displayed GitHub's `Open a pull request` flow. The remaining
gap is PR creation/authentication, not branch publication.
