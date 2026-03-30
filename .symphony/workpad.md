# BIG-GO-981 Workpad

## Scope

Final sweep for repository-root Python build removal.

Batch file list for this lane:

- `pyproject.toml`
- `setup.py`
- `scripts/dev_bootstrap.sh`
- `README.md`

Current repository Python file count before this lane: `116`

Notes:

- `pyproject.toml` and `setup.py` are already absent from the working tree.
- This lane is limited to finalizing removal criteria, eliminating any remaining root Python build-tool bootstrap, and documenting the Go-only replacement entrypoints.

## Plan

1. Confirm the repository root no longer exposes Python packaging manifests or `python -m build` style entrypoints.
2. Remove any remaining root bootstrap behavior that still installs Python build tooling.
3. Document the exact delete/retain/replace decisions for the build-removal batch.
4. Run targeted validation for the Go-only root entrypoints and the updated bootstrap helper.
5. Record the exact commands, results, and repository Python file-count impact.
6. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact `BIG-GO-981` batch file list for the final sweep.
- State the deletion criteria for `pyproject.toml` / `setup.py` and any related Python build-tool entrypoints.
- Document the Go-only replacement entrypoints for root build and operator workflows.
- Keep changes scoped to Python build removal evidence and replacement guidance.
- Report before/after repository-wide Python file counts, even if the net reduction is `0`.

## Validation

- `make test`
- `bash scripts/dev_bootstrap.sh`
- `git status --short`

## Results

### File Disposition

- `pyproject.toml`
  - Already deleted before this lane.
  - Reason: root setuptools/build metadata is no longer valid for the Go-only
    repository contract.
- `setup.py`
  - Already deleted before this lane.
  - Reason: root editable-install packaging is intentionally unsupported.
- `scripts/dev_bootstrap.sh`
  - Updated.
  - Reason: removed the residual `build` package installation so the bootstrap
    helper no longer implies a root Python build workflow.
- `README.md`
  - Updated.
  - Reason: documents the delete/retain/replace decision and the Go-only root
    replacement entrypoints.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `116`
- Net reduction: `0`

### Validation Record

- `make test`
  - Result: `ok` across `bigclaw-go/cmd/*` and `bigclaw-go/internal/*`; `bigclaw-go/scripts/e2e` reported `[no test files]`.
- `bash scripts/dev_bootstrap.sh`
  - Result: `ok`; prints `BigClaw Go development environment is ready.` and `Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to validate the legacy Python migration surface with PYTHONPATH only.`
- `git status --short`
  - Result: only `.symphony/workpad.md`, `README.md`, `reports/BIG-GO-981.md`, and `scripts/dev_bootstrap.sh` changed before commit.
