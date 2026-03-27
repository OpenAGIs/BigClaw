# BIG-GO-902 PR Draft

## Suggested Title

`BIG-GO-902: migrate repo script entrypoints to Go CLI`

## Suggested Description

### Summary

- migrate the first batch of root-level script automation entrypoints into `bigclaw-go/cmd/bigclawctl`
- keep legacy Python/Bash entrypoint files as compatibility shims that now dispatch into the Go CLI
- update operator-facing docs and generated refill queue guidance to prefer direct `scripts/ops/bigclawctl` usage
- add issue-scoped migration/validation evidence and repo-local tracker closeout for `BIG-GO-902`

### Delivered

- added Go CLI subcommands:
  - `create-issues`
  - `dev-smoke`
  - `symphony`
  - `issue`
  - `panel`
- converted these entrypoints into thin shims:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw-symphony`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
- updated docs:
  - `README.md`
  - `docs/go-cli-script-migration-plan.md`
  - `docs/parallel-refill-queue.md`
  - `reports/BIG-GO-902-validation.md`

### Validation

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

### Risks / Deferred Follow-ups

- `scripts/dev_bootstrap.sh` remains shell-owned and was not migrated in this slice
- `scripts/ops/bigclawctl` still shells into `go run`, so local Go toolchain availability remains required
- `bigclaw-go/scripts/*` helper scripts were intentionally left out of this root-level script migration

## Open PR URL

`https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-902-go-cli-script-migration`
