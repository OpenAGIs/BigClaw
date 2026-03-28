# Go-owned Python Build Surface Cutover

Issue: `BIG-GO-921`

## Current inventory

Active migration-only Python assets:

- `src/bigclaw`: frozen legacy runtime/reference implementation
- `tests`: frozen Python regression coverage
- `pytest.ini`: standalone pytest configuration
- `.ruff.toml`: standalone Ruff configuration
- `scripts/dev_bootstrap.sh`: optional migration bootstrap without editable install

Retired packaging/build assets:

- `pyproject.toml`
- `setup.py`

## Go replacement path

The repository-owned source of truth for this cutover is:

```bash
bash scripts/ops/bigclawctl legacy-python build-surface --json
```

That Go command reports the active frozen Python assets, the retired packaging
assets, the conditions for deleting the remaining legacy surface, and the
regression commands that must stay green during the cutover.

## Deletion conditions

- `src/bigclaw` no longer blocks any active operator or runtime workflow from running entirely through Go.
- Python regression coverage is either migrated to Go or explicitly retired for inactive legacy surfaces.
- `bigclawctl legacy-python compile-check` is no longer needed because the frozen compatibility shims have been removed.
- README and bootstrap flows no longer require any Python migration command for standard development or release validation.

## Regression commands

```bash
cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl
PYTHONPATH=src python3 -m pytest
python3 -m ruff check src tests scripts
bash scripts/ops/bigclawctl legacy-python build-surface --json
bash scripts/ops/bigclawctl legacy-python compile-check --json
```
