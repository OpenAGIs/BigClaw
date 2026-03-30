# BIG-GO-981 Final Sweep A: pyproject/setup/build removal

## Batch file list

Primary build-removal surfaces covered by this lane:

- `pyproject.toml`
- `setup.py`
- `scripts/dev_bootstrap.sh`
- `README.md`

## Current repository state

- Repository Python file count before lane: `116`
- Repository Python file count after lane: `116`
- Net Python file reduction from this lane: `0`

This lane does not reduce the `*.py` count because the root build manifests
were `pyproject.toml` and `setup.py`, not Python source files. The remaining
Python footprint is migration/reference code and compatibility shims, not root
package-build infrastructure.

## Disposition

### Deleted

- `pyproject.toml`
  - Status: already absent in the working tree before this lane.
  - Delete basis: the repository root is no longer a Python package build
    surface, and no active workflow should depend on setuptools metadata,
    editable install, or `python -m build`.
  - Historical evidence: removed by commit `79c26c3` (`chore: remove root python build config`).

- `setup.py`
  - Status: already absent in the working tree before this lane.
  - Delete basis: duplicated the old setuptools packaging path for `src/bigclaw`
    and is incompatible with the Go-only root build contract.
  - Historical evidence: removed by commit `79c26c3` (`chore: remove root python build config`).

### Replaced

- Root Python build entrypoints
  - Previous behavior: `pip install -e .`, setuptools metadata resolution, and
    `python -m build` were implied by the root manifests.
  - Replacement basis: active repository build/run entrypoints live behind the
    root `Makefile`.
  - Go-only replacements:
    - `make test`
    - `make build`
    - `make run`

- Python build-tool bootstrap in `scripts/dev_bootstrap.sh`
  - Previous behavior: installed `build` into the temporary virtualenv even
    though the repository no longer exposes root packaging metadata.
  - Replacement basis: legacy Python validation only needs lint/test tools and
    direct source execution.
  - Action in this lane: removed `build` from the bootstrap install set.

### Retained

- `scripts/dev_bootstrap.sh`
  - Retained because it is still the repo helper for "Go first, optional legacy
    Python validation" during migration.
  - Retention condition: it must not recreate or imply a root Python packaging
    workflow.

- `src/bigclaw/**`, `tests/**`, `scripts/**/*.py`, `bigclaw-go/scripts/**/*.py`
  - Retained because they are migration-reference code, compatibility shims, or
    automation utilities. They are not repository-root build manifests.
  - Removal from this lane was intentionally out of scope.

## Validation

- `make test`
  - Result: pass
- `bash scripts/dev_bootstrap.sh`
  - Result: pass; prints Go-only readiness message without installing Python build tooling
- `git status --short`
  - Result: only `.symphony/workpad.md`, `README.md`, `reports/BIG-GO-981.md`, and `scripts/dev_bootstrap.sh` changed before commit

## Summary

The root Python build manifests remain removed, the last residual bootstrap
installation of the Python `build` tool was eliminated, and the README now
states the Go-only root replacement entrypoints explicitly. Repository Python
file count is unchanged at `116` because this sweep targeted build manifests and
bootstrap behavior rather than deleting migration-reference `*.py` files.
