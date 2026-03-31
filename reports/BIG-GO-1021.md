# BIG-GO-1021 Autorefill sweep A: remaining root/config/python residuals

## Scope

- `README.md`
- `.github/workflows/ci.yml`
- `scripts/dev_bootstrap.sh`
- `tests/conftest.py` (deleted)

## Changes

- Rewrote the root migration guidance to state explicitly that repository-root
  Python packaging artifacts remain retired and that legacy validation must use
  `PYTHONPATH=src`.
- Split CI into a Go-mainline job and a legacy-Python migration job so the root
  workflow no longer reads as a Python-first repository entrypoint.
- Tightened the bootstrap helper messaging to call out explicit `PYTHONPATH`
  usage for any migration-only Python validation.
- Deleted `tests/conftest.py`, removing implicit `src` path injection and
  reducing repository `.py` file count by one.

## File-count impact

- `.py`: `50 -> 49`
- `.go`: `282 -> 282`
- `pyproject.toml`: absent before, absent after
- `setup.py`: absent before, absent after
- `setup.cfg`: absent before, absent after
- `*.egg-info`: absent before, absent after

## Validation

- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l`
- `find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' \) -print`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_planning.py -q`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./cmd/bigclawd`
- `rg -n "pyproject|setup.py|egg-info|pip install -e|python -m build|setuptools" -S README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh reports/BIG-GO-1021.md`

## Residual risk

- The repository still contains legacy Python source and tests under `src/` and
  `tests/`; this lane only removes remaining root/config entrypoint residue and
  one implicit Python test bootstrap file.
