# BIG-GO-1078 PR Draft

## Suggested Title

`BIG-GO-1078: remove residual ops Python files tranche 2`

## Suggested Description

### Summary

- remove the remaining tranche-2 Python wrappers from `scripts/ops`
- keep refill and workspace operator flows on the Go-owned `scripts/ops/bigclawctl` path
- update migration docs so repo-default guidance no longer advertises deleted Python entrypoints
- add regression coverage that keeps both the deleted files absent and `scripts/ops` fully Python-free

### Delivered

- deleted:
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- retained Go or shell operator path:
  - `scripts/ops/bigclawctl`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
- updated docs:
  - `README.md`
  - `docs/go-cli-script-migration-plan.md`
  - `docs/go-mainline-cutover-issue-pack.md`
  - `reports/BIG-GO-1078-validation.md`
- added regression coverage:
  - `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`

### Validation

```bash
find . -name '*.py' | wc -l
find scripts/ops -maxdepth 1 -type f -name '*.py'
cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl
cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14
bash scripts/ops/bigclawctl refill --help
bash scripts/ops/bigclawctl workspace bootstrap --help
bash scripts/ops/bigclawctl workspace validate --help
```

### Risks / Deferred Follow-ups

- GitHub PR creation from this workspace is still blocked by missing GitHub CLI authentication.
- Historical reports under `reports/` and audit entries under `local-issues.json` still mention the deleted Python wrappers as part of prior migration history; this issue intentionally does not rewrite prior evidence artifacts.

## PR Seed URL

`https://github.com/OpenAGIs/BigClaw/pull/new/symphony/BIG-GO-1078`

## Compare URL

`https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-1078?expand=1`
