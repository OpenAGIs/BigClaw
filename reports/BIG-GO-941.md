# BIG-GO-941 Lane1 Root build/config removal

## Lane file list

- `Makefile`
- `README.md`
- `scripts/dev_bootstrap.sh`
- `pyproject.toml` (deleted)
- `setup.py` (deleted)

## Go replacement and deletion plan

- Added a root `Makefile` so repository-level build, test, and run entrypoints resolve into `bigclaw-go`.
- Updated the root README to point operators at the Go-only root entrypoints instead of Python packaging from the repository root.
- Updated `scripts/dev_bootstrap.sh` to validate Go first and to run any migration-only Python checks from source via `PYTHONPATH`, not via editable install.
- Deleted `pyproject.toml` and `setup.py` because the repository root is no longer a Python package build surface.

## Validation plan

- `cd bigclaw-go && go test ./...`
- `make test`
- `make build`
- `git status --short`

## Residual risks

- Legacy workflows that still assume `pip install -e .` from the repository root will now fail and must switch to direct source execution or Go CLI entrypoints.
- Python migration-only checks still depend on external Python tooling availability when `BIGCLAW_ENABLE_LEGACY_PYTHON=1` is used.
