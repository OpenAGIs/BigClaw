# BIG-GO-1001 Workpad

## Scope

Target the remaining repository-root packaging-exit residue after prior removal
of the Python build surface.

Batch file list:

- `.github/workflows/ci.yml`
- `README.md`
- `reports/OPE-66-validation.md`
- `reports/BIG-GO-941.md`
- `scripts/dev_bootstrap.sh`

Packaging artifact inventory at start of lane:

- Root `pyproject.toml`: absent
- Root `setup.py`: absent
- Root `setup.cfg`: absent
- Active workflow references still invoking root packaging:
  - `.github/workflows/ci.yml`: `pip install -e .[dev]`
  - `.github/workflows/ci.yml`: `python -m build`
- Bootstrap residue still installing packaging tooling:
  - `scripts/dev_bootstrap.sh`: installs Python `build`
- Historical packaging evidence still describing the removed build path:
  - `reports/OPE-66-validation.md`
  - `reports/BIG-GO-941.md`

Current repository Python file count before this lane: `108`
Current packaging-entry file count before this lane: `5`

## Plan

1. Remove active CI usage of deleted root Python packaging entrypoints.
2. Remove Python bootstrap packaging-tool installation that is no longer needed.
3. Update operator-facing documentation so it no longer points at obsolete
   packaging behavior.
4. Preserve only historical report content that still matches repository
   reality; delete or rewrite stale packaging-install claims.
5. Run targeted validation for the touched CI, shell, and documentation paths.
6. Record delete/replace/keep rationale and Python file count impact.
7. Commit and push the scoped changes for `BIG-GO-1001`.

## Acceptance

- Produce the exact `BIG-GO-1001` batch file list for packaging-exit residue.
- Reduce file count where a packaging-related artifact can be deleted safely in
  this lane.
- Document delete/replace/keep rationale for each targeted file.
- Report the repository-wide Python file count impact.

## Validation

- `find . -name '*.py' | sort | wc -l`
- `python3 - <<'PY' ... PY` to assert no active CI step still references
  `pip install -e .` or `python -m build`
- `python3 - <<'PY' ... PY` to assert `scripts/dev_bootstrap.sh` no longer
  installs Python `build`
- `python3 - <<'PY' ... PY` to assert README no longer suggests root packaging
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `.github/workflows/ci.yml`
  - Replaced.
  - Reason: removed the last active root packaging calls (`pip install -e .[dev]`
    and `python -m build`) and switched CI to source-based validation with
    `PYTHONPATH=src pytest`.
- `scripts/dev_bootstrap.sh`
  - Replaced.
  - Reason: removed Python `build` installation because the repository root is
    no longer a Python package build surface.
- `reports/OPE-66-validation.md`
  - Deleted.
  - Reason: it only documented the obsolete editable-install / `setup.py`
    packaging path and had no remaining repository references.
- `reports/BIG-GO-941.md`
  - Replaced.
  - Reason: kept as historical evidence for the original root build removal and
    appended follow-up status that closes the residual packaging references.
- `README.md`
  - Kept.
  - Reason: it already states that root Python packaging must not be used and
    already documents `PYTHONPATH=src` validation.

### Packaging Batch Summary

- Packaging-residue batch files: `.github/workflows/ci.yml`, `README.md`,
  `reports/OPE-66-validation.md`, `reports/BIG-GO-941.md`,
  `scripts/dev_bootstrap.sh`
- Packaging-related Python files in this batch: none
- Root packaging entry files currently present: none (`pyproject.toml`,
  `setup.py`, `setup.cfg` are absent)

### Python File Count Impact

- Repository Python files before: `108`
- Repository Python files after: `108`
- Net reduction: `0`
- Deleted packaging-related files in this lane: `1` markdown report

### Validation Record

- `find . -name '*.py' | sort | wc -l`
  - Result before: `108`
  - Result after: `108`
- `python3 - <<'PY' ... PY` on `.github/workflows/ci.yml`
  - Result: `ok`
- `python3 - <<'PY' ... PY` on `scripts/dev_bootstrap.sh`
  - Result: `ok`
- `python3 - <<'PY' ... PY` on `README.md`
  - Result: `ok`
- `bash -n scripts/dev_bootstrap.sh`
  - Result: exit `0`
- `PYTHONPATH=src python3 -m pytest --cov=bigclaw --cov-report=term-missing --cov-report=xml`
  - Result: failed under local Python `3.9.6`; one collection error from
    `bigclaw-go/scripts/e2e/export_validation_bundle.py` using `Path | None`
    syntax that requires Python `3.10+`
- `PYTHONPATH=src python3 -m pytest tests --cov=bigclaw --cov-report=term-missing --cov-report=xml`
  - Result: `226 passed`, `3 failed`
  - Known unrelated failures:
    - `tests/test_live_shadow_bundle.py::test_export_live_shadow_bundle_generates_index_and_rollup`
    - `tests/test_live_shadow_bundle.py::test_export_live_shadow_bundle_supports_documented_bigclaw_go_cwd`
    - `tests/test_parallel_validation_bundle.py::test_export_validation_bundle_generates_latest_reports_and_index`
  - Failure basis: two tests still point at migration scripts already removed by
    an earlier lane; one still executes the Python 3.11-only
    `export_validation_bundle.py` under local Python `3.9.6`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_reports.py`
  - Result: `43 passed in 3.11s`
- `git diff --check`
  - Result: clean
- `git status --short`
  - Result: scoped issue files only before commit
