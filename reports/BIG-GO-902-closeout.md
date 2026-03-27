# BIG-GO-902 Closeout Index

Issue: `BIG-GO-902`

Title: `脚本层迁移到 Go CLI`

Date: `2026-03-27`

## Branch

`feat/BIG-GO-902-go-cli-script-migration`

## Latest Pushed Commit

`77ba309`

## Reviewer Links

- Compare URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...feat/BIG-GO-902-go-cli-script-migration?expand=1`
- PR seed URL:
  - `https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-902-go-cli-script-migration`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-902-validation.md`
- PR draft:
  - `reports/BIG-GO-902-pr.md`
- Machine-readable status:
  - `reports/BIG-GO-902-status.json`
- Migration plan:
  - `docs/go-cli-script-migration-plan.md`

## Outcome

- Root-level script entrypoints migrated to Go CLI subcommands for the first batch:
  - `create-issues`
  - `dev-smoke`
  - `symphony`
  - `issue`
  - `panel`
- Legacy Python/Bash entrypoint files retained as compatibility shims.
- Operator-facing docs shifted to prefer direct `scripts/ops/bigclawctl` commands.

## Validation Commands

```bash
cd bigclaw-go && go test ./cmd/bigclawctl
cd bigclaw-go && go test ./internal/refill
bash scripts/ops/bigclawctl dev-smoke
PYTHONPATH=src python3 scripts/dev_smoke.py
python3 scripts/create_issues.py --help
bash scripts/ops/bigclawctl issue --help
bash scripts/ops/bigclawctl panel --help
bash scripts/ops/bigclawctl symphony --help
bash scripts/ops/bigclaw-issue list
bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status
```

## Remaining Blocker

This workspace can push the branch but cannot create or verify the GitHub PR directly because no
GitHub CLI or API token is configured. The missing step is external authentication, not missing
repo content.

## Final Repo Check

- Current repo scan found no additional root-script migration gaps beyond the already-documented
  deferred items.
