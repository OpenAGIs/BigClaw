# BIG-GO-961 Workpad

## Plan

1. Inventory the root-config and `scripts/**` Python assets that are directly in scope for this lane, and confirm whether `pyproject.toml` / `setup.py` still exist.
2. Replace or remove the remaining root-level Python shim entrypoints where the Go-first shell entrypoints already cover the same operator path.
3. Delete any root Python-only helper modules that become dead code after the shim removal.
4. Update the active repo docs to point at the canonical shell/Go entrypoints instead of deleted Python wrappers.
5. Record the exact in-scope file list, delete/replace/keep rationale, and Python file-count impact in a lane report.
6. Run targeted validation commands, record exact commands and results, then commit and push the branch.

## Acceptance

- Produce an explicit in-scope file list for `pyproject.toml`, `setup.py`, and the root `scripts/**` Python assets covered by this lane.
- Reduce the number of Python files in the targeted root/script surface as much as possible without widening scope.
- Document why each in-scope asset was deleted, replaced, or kept.
- Report the overall repository Python file-count delta caused by this lane.

## Validation

- `rg --files -g '*.py' | wc -l` before and after the change.
- Targeted shell help/status invocations for the canonical replacement entrypoints under `scripts/ops/bigclawctl`.
- `python3 -m pytest` for the touched Python tests only if active tests cover the changed surface.
- `git status --short` before commit to confirm the scoped diff.

## Results

- Confirmed `pyproject.toml` and `setup.py` were already absent at the repository root before this lane.
- Deleted 7 root/script Python compatibility shims:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- Deleted the dead helper module `src/bigclaw/legacy_shim.py` and the now-unused Go wrapper helper file pair under `bigclaw-go/internal/legacyshim/`.
- Updated `README.md` and `docs/go-cli-script-migration-plan.md` to remove Python-wrapper guidance and point operators to `bash scripts/ops/bigclawctl ...`.
- Added `reports/BIG-GO-961.md` with the explicit asset list, delete/replace/keep rationale, and Python file-count impact.
- Repository Python file count moved from `123` to `115`, for a net reduction of `8`.
- Validation results:
  - `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim` -> passed
  - `bash scripts/ops/bigclawctl dev-smoke` -> `smoke_ok local`
  - `bash scripts/ops/bigclawctl create-issues --help` -> passed
  - `bash scripts/ops/bigclawctl refill --help` -> passed
  - `bash scripts/ops/bigclawctl workspace bootstrap --help` -> passed
  - `bash scripts/ops/bigclawctl workspace validate --help` -> passed
  - `bash scripts/ops/bigclawctl github-sync status --json` -> `status: ok`, `synced: true`, `dirty: true`
