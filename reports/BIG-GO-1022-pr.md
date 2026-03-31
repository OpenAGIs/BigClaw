# BIG-GO-1022 PR Draft

## Suggested Title

`BIG-GO-1022: remove residual root ops python wrappers`

## Suggested Description

### Summary

- remove the remaining root `scripts/ops/*.py` operator entrypoints from the physical repo tree
- replace them with non-Python executable wrappers that dispatch into `scripts/ops/bigclawctl`
- preserve legacy bootstrap defaults and workspace-validate flag translation
- refresh directly coupled docs and legacy-shim tests so the repo inventory matches the new wrapper paths

### Delivered

- removed:
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- added replacement wrappers:
  - `scripts/ops/bigclaw_github_sync`
  - `scripts/ops/bigclaw_refill_queue`
  - `scripts/ops/bigclaw_workspace_bootstrap`
  - `scripts/ops/symphony_workspace_bootstrap`
  - `scripts/ops/symphony_workspace_validate`
- updated:
  - `README.md`
  - `docs/go-cli-script-migration-plan.md`
  - `docs/go-mainline-cutover-issue-pack.md`
  - `bigclaw-go/internal/legacyshim/wrappers.go`
  - `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - `.symphony/workpad.md`

### File Count Impact

- `.py`: `88 -> 83`
- `.go`: `282 -> 282`
- `pyproject.toml`: not present, unchanged
- `setup.py`: not present, unchanged

### Validation

```bash
cd bigclaw-go && go test ./internal/legacyshim
bash scripts/ops/bigclaw_github_sync --help
bash scripts/ops/bigclaw_refill_queue --help
bash scripts/ops/bigclaw_workspace_bootstrap --help
bash scripts/ops/symphony_workspace_bootstrap --help
bash scripts/ops/symphony_workspace_validate --help
bash scripts/ops/bigclaw_github_sync status --json
repo_url="$(pwd)"; tmp_root=$(mktemp -d); tmp_cache=$(mktemp -d); tmp_ws="$tmp_root/ws"; BIGCLAW_BOOTSTRAP_REPO_URL="$repo_url" BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap bootstrap --workspace-root "$tmp_root" --workspace "$tmp_ws" --cache-root "$tmp_cache" --default-branch main; bash scripts/ops/bigclaw_workspace_bootstrap cleanup --workspace-root "$tmp_root" --workspace "$tmp_ws" --cache-root "$tmp_cache"
repo_url="$(pwd)"; tmp_root=$(mktemp -d); tmp_report=$(mktemp); bash scripts/ops/symphony_workspace_validate --repo-url "$repo_url" --workspace-root "$tmp_root" --issues BIG-GO-1022-A BIG-GO-1022-B --report-file "$tmp_report" --json; test -s "$tmp_report"
find scripts/ops -maxdepth 1 \( -name '*.py' -o -type f \) | sort
find . -path '*/.git' -prune -o -name '*.py' -print | sort | wc -l
find . -path '*/.git' -prune -o -name '*.go' -print | sort | wc -l
```

### Branch

- branch: `symphony/BIG-GO-1022`
- head: `9fb84a01136c3e1bb4f90af0ac838acb89001846`

### PR Status

- push completed to `origin/symphony/BIG-GO-1022`
- `gh auth status` reports no logged-in GitHub host in this environment
- PR creation is therefore blocked on unavailable GitHub credentials rather than repository state

## Compare URL

`https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-1022?expand=1`

## PR Seed URL

`https://github.com/OpenAGIs/BigClaw/pull/new/symphony/BIG-GO-1022`
