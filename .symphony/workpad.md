# BIG-GO-1002 Workpad

## Scope

Retire every Python file currently matched by `scripts/*.py` and `scripts/ops/*.py`:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

The intended outcome is zero Python files remaining in those two target globs.

## Plan

1. Confirm current references and behavior for each targeted script.
2. Delete each targeted `.py` entrypoint and remove helper code that only existed to support those
   wrappers.
3. Update docs and focused tests so the supported operator path is only `scripts/ops/bigclawctl`.
4. Run targeted validation and report the exact Python file count delta.

## Acceptance

- Produce the explicit before/after file inventory for `scripts/*.py` and `scripts/ops/*.py`.
- Reduce the Python file count in that batch as much as possible; target is full removal.
- Record delete/replace/keep rationale for each targeted path.
- Report impact on total repository Python file count.

## Validation

- `rg --files -g 'scripts/*.py' -g 'scripts/ops/*.py'`
- `python3 - <<'PY'` repo-wide Python file count snapshot
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim`

## Results

### File Disposition

- `scripts/create_issues.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl create-issues`.
- `scripts/dev_smoke.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl dev-smoke`.
- `scripts/ops/bigclaw_github_sync.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl github-sync`.
- `scripts/ops/bigclaw_refill_queue.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl refill`.
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl workspace ...`.
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl workspace ...`.
- `scripts/ops/symphony_workspace_validate.py`
  - Deleted.
  - Reason: wrapper-only entrypoint for `bash scripts/ops/bigclawctl workspace validate`.
- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Reason: orphaned helper module after removing the root/operator Python wrappers.
- `bigclaw-go/internal/legacyshim/wrappers.go`
  - Deleted.
  - Reason: orphaned Go helper layer that only modeled the removed wrappers.
- `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - Deleted.
  - Reason: coverage only for the removed wrapper helper layer.
- Remaining targeted Python files
  - None.
  - Reason: the full `scripts/*.py` and `scripts/ops/*.py` batch was removed.

### Python Count Impact

- Repository Python files before: `108`
- Repository Python files after: `100`
- Targeted batch Python files before: `7`
- Targeted batch Python files after: `0`
- Net reduction: `8` repository-wide and `7` in the targeted batch

### Validation Record

- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
  - Result: exit `0`; `ok   bigclaw-go/cmd/bigclawctl 2.492s`, `ok   bigclaw-go/internal/legacyshim (cached)`, and `ok   bigclaw-go/internal/regression (cached)`.
- `bash scripts/ops/bigclawctl create-issues --help`
  - Result: exit `0`; printed `usage: bigclawctl create-issues [flags]`.
- `bash scripts/ops/bigclawctl dev-smoke`
  - Result: exit `0`; printed `smoke_ok local`.
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: exit `0`; JSON reported `status: ok`, `branch: main`, `dirty: true`, `behind: 1`, and `synced: false` before committing this lane.
- `bash scripts/ops/bigclawctl refill --help`
  - Result: exit `0`; printed `usage: bigclawctl refill [flags]` with the `seed` subcommand.
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: exit `0`; printed `usage: bigclawctl workspace validate [flags]`.
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
  - Result: exit `0`; JSON reported `status: ok` for `src/bigclaw/runtime.py` and `src/bigclaw/__main__.py`.
- `find . -name '*.py' | wc -l`
  - Result: `100`
