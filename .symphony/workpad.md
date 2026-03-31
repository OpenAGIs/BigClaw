Issue: BIG-GO-1022

Plan
- Replace the remaining `scripts/ops/*.py` operator entrypoints with non-Python wrappers that execute `scripts/ops/bigclawctl` directly.
- Keep legacy command behavior for `github-sync`, `refill`, `workspace`, `workspace bootstrap`, and `workspace validate`, including default bootstrap flags and validate-argument translation.
- Update directly coupled docs/tests to reference the non-Python entrypoints and keep the migration inventory aligned with the repo state.
- Run targeted validation, capture exact commands/results, summarize `py`/`go` file-count impact plus `pyproject.toml`/`setup.py` impact, then commit and push.

Acceptance
- Changes stay scoped to the remaining root-level `scripts/ops` residual Python operator entrypoints and directly coupled documentation/tests.
- Repository `.py` file count decreases by removing the five `scripts/ops/*.py` wrappers from the physical tree.
- Replacement wrappers still allow the same operator flows to run through `scripts/ops/bigclawctl`.
- Final report includes the impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find scripts/ops -maxdepth 1 \\( -name '*.py' -o -type f \\) | sort`
- `find . -path '*/.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path '*/.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim`
- `bash scripts/ops/bigclaw_github_sync --help`
- `bash scripts/ops/bigclaw_refill_queue --help`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_validate --help`
- `bash scripts/ops/bigclaw_github_sync status --json`
- `repo_url="$(pwd)"; tmp_root=$(mktemp -d); tmp_cache=$(mktemp -d); tmp_ws="$tmp_root/ws"; BIGCLAW_BOOTSTRAP_REPO_URL="$repo_url" BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap bootstrap --workspace-root "$tmp_root" --workspace "$tmp_ws" --cache-root "$tmp_cache" --default-branch main; bash scripts/ops/bigclaw_workspace_bootstrap cleanup --workspace-root "$tmp_root" --workspace "$tmp_ws" --cache-root "$tmp_cache"`
- `repo_url="$(pwd)"; tmp_root=$(mktemp -d); tmp_report=$(mktemp); bash scripts/ops/symphony_workspace_validate --repo-url "$repo_url" --workspace-root "$tmp_root" --issues BIG-GO-1022-A BIG-GO-1022-B --report-file "$tmp_report" --json; test -s "$tmp_report"`
- `git status --short && git diff --stat`

Status
- Implemented and validated.
- Branch pushed: `symphony/BIG-GO-1022`
- Commit: `1ca6ff24c012370b0bda612073230356123d1c3b`
- File-count impact: `.py` `88 -> 83`, `.go` `282 -> 282`, `pyproject.toml` absent, `setup.py` absent.
- Blocker: `gh auth status` reports no logged-in GitHub host, so PR creation cannot be completed from this environment.
