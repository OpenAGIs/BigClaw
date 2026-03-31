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
- `python3 -m pytest tests/test_legacy_shim.py`
- `bash scripts/ops/bigclaw_github_sync --help`
- `bash scripts/ops/bigclaw_refill_queue --help`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_validate --help`
- `bash scripts/ops/bigclaw_github_sync status --json`
- `git status --short && git diff --stat`
