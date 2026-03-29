# BIG-GO-942 Closeout Index

Issue: `BIG-GO-942`

Title: `Lane2 Root scripts to Go CLI`

Date: `2026-03-29`

## Branch

`symphony/BIG-GO-942`

## Latest Pushed Commit

`87fd42c3043163e86f937a7fc8a4524802e5e9eb`

## Reviewer Links

- Compare URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-942?expand=1`
- PR seed URL:
  - `https://github.com/OpenAGIs/BigClaw/pull/new/symphony/BIG-GO-942`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-942-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-942-status.json`
- Workpad:
  - `.symphony/workpad.md`
- Migration note updated by this issue:
  - `docs/go-cli-script-migration-plan.md`

## Outcome

- The remaining lane-scoped root script Python implementations were removed.
- The retained legacy file paths now execute as shell wrappers over Go-owned `bigclawctl`
  behavior.
- Wrapper-only compatibility logic is kept narrow and explicit for workspace bootstrap and
  workspace validate callers.
- Validation evidence for tests, wrapper smoke checks, and temp-repo workspace flows now lives
  in-repo for reviewers.

## Validation Commands

```bash
cd bigclaw-go && go test ./cmd/bigclawctl
python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
bash scripts/dev_smoke.py
bash scripts/create_issues.py --help
bash scripts/ops/bigclaw_refill_queue.py --help
bash scripts/ops/bigclaw_github_sync.py status --json
BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py bootstrap --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py cleanup --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
bash scripts/ops/symphony_workspace_validate.py --repo-url <tmp bare repo> --workspace-root <tmp>/validate --issues COMPAT-VAL-1 COMPAT-VAL-2 --report-file <tmp>/report.json --no-cleanup --json
```

## Remaining Blocker

No in-repo implementation blocker remains.

External PR creation is still not automated from this workspace because `gh` is not installed and
both `GITHUB_TOKEN` and `GH_TOKEN` are unset.
