# BIG-GO-1012 Workpad

## Scope

Reduce the remaining root-level Python script residuals under `scripts/` and
`scripts/ops/`, with priority on deleting `.py` entrypoints that only proxy to
the Go mainline.

Targeted residual files at start of lane:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `src/bigclaw/legacy_shim.py`

Repository inventory at start of lane:

- Root `pyproject.toml`: absent
- Root `setup.py`: absent
- Repository `.py` file count before lane: `108`

## Plan

1. Delete the remaining root script Python wrappers that only dispatch into
   `scripts/ops/bigclawctl`.
2. Remove now-unused Python shim helpers from `src/bigclaw/legacy_shim.py`.
3. Update repository docs and tests to point at the Go-first shell entrypoints.
4. Update Go-side shim coverage to reflect the retained shell-wrapper contract
   instead of deleted Python entrypoints.
5. Run targeted validation covering changed Go tests, shell entrypoints, and the
   repository `.py` inventory.
6. Commit and push the scoped branch for `BIG-GO-1012`.

## Acceptance

- Directly reduce repository Python residuals in the root script layer.
- Decrease the repository `.py` file count by deleting safe wrapper entrypoints.
- Keep scope limited to the root script residual cleanup.
- Report exact impact on `py files`, `go files`, `pyproject.toml`, and
  `setup.py`.
- Record exact validation commands and results.

## Validation

- `find . -name '*.py' | sort | wc -l`
- `find bigclaw-go -name '*.go' | sort | wc -l`
- `go test ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl github-sync --help`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `scripts/create_issues.py`
  - Deleted.
  - Reason: root Python shim only proxied to `bigclawctl create-issues`.
- `scripts/dev_smoke.py`
  - Deleted.
  - Reason: root Python shim only proxied to `bigclawctl dev-smoke`.
- `scripts/ops/bigclaw_github_sync.py`
  - Deleted.
  - Reason: Python operator shim replaced by shell alias.
- `scripts/ops/bigclaw_refill_queue.py`
  - Deleted.
  - Reason: Python operator shim replaced by shell alias.
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Deleted.
  - Reason: Python wrapper logic moved into a shell alias with the same default injection.
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Deleted.
  - Reason: Python operator shim replaced by shell alias.
- `scripts/ops/symphony_workspace_validate.py`
  - Deleted.
  - Reason: Python wrapper logic moved into a shell alias with the same legacy flag translation.
- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Reason: no remaining root-script Python wrappers import it after this lane.
- `scripts/create-issues`
  - Added.
  - Reason: keeps a root script-shaped operator alias without reintroducing a `.py` entrypoint.
- `scripts/dev-smoke`
  - Added.
  - Reason: keeps a root script-shaped operator alias without reintroducing a `.py` entrypoint.
- `scripts/ops/bigclaw-github-sync`
  - Added.
  - Reason: preserves a mnemonic operator alias as shell only.
- `scripts/ops/bigclaw-refill-queue`
  - Added.
  - Reason: preserves a mnemonic operator alias as shell only.
- `scripts/ops/bigclaw-workspace-bootstrap`
  - Added.
  - Reason: preserves repo-url/cache-key default injection without Python.
- `scripts/ops/symphony-workspace-bootstrap`
  - Added.
  - Reason: preserves a script-shaped workspace alias without Python.
- `scripts/ops/symphony-workspace-validate`
  - Added.
  - Reason: preserves `--report-file`, `--no-cleanup`, and `--issues` translation without Python.

### Inventory Impact

- Repository `.py` files before: `108`
- Repository `.py` files after: `100`
- Net `.py` reduction: `8`
- `bigclaw-go` `.go` files before: `267`
- `bigclaw-go` `.go` files after: `267`
- Net `.go` reduction: `0`
- Root `pyproject.toml`: absent before, absent after
- Root `setup.py`: absent before, absent after

### Validation Record

- `find . -name '*.py' | sort | wc -l`
  - Result: `100`
- `find bigclaw-go -name '*.go' | sort | wc -l`
  - Result: `267`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
  - Result: passed
- `bash -n scripts/create-issues`
  - Result: exit `0`
- `bash -n scripts/dev-smoke`
  - Result: exit `0`
- `bash -n scripts/ops/bigclaw-github-sync`
  - Result: exit `0`
- `bash -n scripts/ops/bigclaw-refill-queue`
  - Result: exit `0`
- `bash -n scripts/ops/bigclaw-workspace-bootstrap`
  - Result: exit `0`
- `bash -n scripts/ops/symphony-workspace-bootstrap`
  - Result: exit `0`
- `bash -n scripts/ops/symphony-workspace-validate`
  - Result: exit `0`
- `bash scripts/dev-smoke`
  - Result: `smoke_ok local`
- `bash scripts/create-issues --help`
  - Result: passed; printed `bigclawctl create-issues` usage
- `bash scripts/ops/bigclaw-github-sync --help`
  - Result: passed; printed `bigclawctl github-sync` usage
- `bash scripts/ops/bigclaw-refill-queue --help`
  - Result: passed; printed `bigclawctl refill` usage
- `bash scripts/ops/bigclaw-workspace-bootstrap --help`
  - Result: passed; printed `bigclawctl workspace bootstrap` usage
- `bash scripts/ops/symphony-workspace-bootstrap --help`
  - Result: passed; printed `bigclawctl workspace` usage
- `bash scripts/ops/symphony-workspace-validate --help`
  - Result: passed; printed `bigclawctl workspace validate` usage
- `bash scripts/ops/symphony-workspace-validate --report-file <tmp> --no-cleanup --issues BIG-1 BIG-2 --help`
  - Result: passed; printed `bigclawctl workspace validate` usage after shell translation
- `git diff --check`
  - Result: clean
- `git status --short`
  - Result: scoped issue files only before commit
