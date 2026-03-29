# BIG-GO-977 Workpad

## Scope

Targeted legacy Python files under `scripts/ops` for this batch:

- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

Current repository Python file count before this batch: `118`

## Plan

1. Confirm the exact `scripts/ops/**` Python file list and the current replacement paths already provided by `scripts/ops/bigclawctl`.
2. Remove the five compatibility-only Python entrypoints from `scripts/ops`.
3. Delete or reduce shim helper code that only existed to support those removed Python scripts.
4. Update active operator documentation so the replacement commands are explicit and the deleted Python entrypoints are no longer presented as live shims.
5. Run targeted validation for the touched Go and Python surfaces and record exact commands plus results here.
6. Report the direct file list, replacement mapping, and before/after total Python file counts.
7. Commit and push the scoped batch changes.

## Acceptance

- Produce the exact `scripts/ops/**` Python file list owned by `BIG-GO-977`.
- Reduce the number of Python files in `scripts/ops`.
- Record the direct replacement path for each removed Python entrypoint.
- Report before/after total Python file counts for the repository.
- Keep the changes scoped to the `scripts/ops` batch and its supporting docs/tests/helpers.

## Validation

- `go test` for the touched legacy-shim package and the `bigclawctl` command package.
- `python3 -m py_compile` for the touched Python helper module.
- `bash scripts/ops/bigclawctl ... --help` smoke checks for the replacement commands.
- `rg --files scripts/ops -g '*.py'` and `rg --files . -g '*.py' | wc -l` to confirm the Python file reduction.
- `git status --short` to confirm the change set stays scoped to this batch.

## Results

### File Disposition

- `scripts/ops/bigclaw_github_sync.py`
  - Deleted.
  - Replacement: `bash scripts/ops/bigclawctl github-sync ...`
- `scripts/ops/bigclaw_refill_queue.py`
  - Deleted.
  - Replacement: `bash scripts/ops/bigclawctl refill ...`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Deleted.
  - Replacement: `bash scripts/ops/bigclawctl workspace bootstrap ...`
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Deleted.
  - Replacement: `bash scripts/ops/bigclawctl workspace bootstrap ...`
- `scripts/ops/symphony_workspace_validate.py`
  - Deleted.
  - Replacement: `bash scripts/ops/bigclawctl workspace validate ...`
- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Reason: all remaining imports were from the removed `scripts/ops/*.py` wrappers.
- `bigclaw-go/internal/legacyshim/wrappers.go`
  - Deleted.
  - Reason: mirrored helper logic only existed for the removed Python wrapper scripts.
- `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - Deleted.
  - Reason: covered the removed wrapper-helper API only.

### Python File Count Impact

- Repository Python files before: `118`
- Repository Python files after: `112`
- Net reduction: `6`
- `scripts/ops` Python files before: `5`
- `scripts/ops` Python files after: `0`
- Net reduction in `scripts/ops`: `5`

### Validation Record

- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
  - Result: success
- `python3 -m py_compile src/bigclaw/__main__.py`
  - Result: success
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: success; reported `status: "ok"`, `synced: true`, `dirty: true`
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: success
- `bash scripts/ops/bigclawctl refill --help`
  - Result: success
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
  - Result: success; frozen file list now contains only `src/bigclaw/__main__.py`
- `rg --files scripts/ops -g '*.py'`
  - Result: no matches
- `rg --files . -g '*.py' | wc -l`
  - Result: `112`
