# BIG-GO-1070 Workpad

## Plan
- Confirm the remaining packaging-adjacent Python assets in scope for this slice and baseline the current `.py` file count.
- Delete the operator-facing Python compatibility wrappers under `scripts/ops/` that already defer to `scripts/ops/bigclawctl`.
- Remove the shared Python wrapper helper `src/bigclaw/legacy_shim.py` and shrink Go-side compile-check/regression coverage to the Python assets that still exist.
- Update repository docs so the supported operator path is `bash scripts/ops/bigclawctl ...` and stale Python wrapper references are removed.
- Run targeted validation, record exact commands and outcomes, then commit and push the branch.

## Acceptance
- This batch explicitly covers the remaining physical Python wrapper assets:
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
  - `src/bigclaw/legacy_shim.py`
- The wrapper files above are deleted rather than retained as Python compatibility shims.
- Go-first operator entrypoints remain verifiable through `scripts/ops/bigclawctl`.
- Regression coverage pins the deleted Python paths so they do not silently return.
- Closeout reports the pre/post repository `.py` file counts and any residual Python risk left outside this issue's scope.

## Validation
- `rg --files . | rg '\.py$' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl refill --help`
- `BIGCLAW_BOOTSTRAP_REPO_URL=git@github.com:OpenAGIs/BigClaw.git BIGCLAW_BOOTSTRAP_CACHE_KEY=openagis-bigclaw bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
