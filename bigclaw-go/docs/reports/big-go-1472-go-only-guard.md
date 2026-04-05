# BIG-GO-1472 Go-only guard sweep

## Scope

This refill slice verified the repository had already reached zero physical
`.py` files and then converted the remaining live bootstrap guidance into an
enforceable Go-only contract.

## Migrated or deleted assets

- Deleted in this slice: none. The repository already contained `0` physical
  `.py` files at the start of work.
- Rewritten guidance: `docs/symphony-repo-bootstrap-template.md` no longer
  instructs downstream repos to add `workspace_bootstrap.py`,
  `workspace_bootstrap_cli.py`, `conftest.py`, `pytest.ini`, or similar Python
  bootstrap/test manifests.

## Go ownership

- Workspace bootstrap and cleanup entrypoint: `scripts/ops/bigclawctl`
- Bootstrap implementation ownership: `bigclaw-go/internal/bootstrap/*`
- Optional repo-root validation wrapper: `scripts/dev_bootstrap.sh`

## Explicit delete conditions

The repository should continue to reject:

- any tracked `*.py` file anywhere in the repository tree
- tracked `conftest.py`, `pytest.ini`, `pyproject.toml`, `tox.ini`,
  `requirements*.txt`, `Pipfile`, `poetry.lock`, or `.python-version` files
- template guidance that tells operators to restore Python bootstrap
  compatibility files

## Validation

- `find . -type f -name '*.py' | sort` -> no output
- `find . -maxdepth 3 \( -name 'pytest.ini' -o -name 'conftest.py' -o -name 'pyproject.toml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name '.python-version' \) | sort` -> no output
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1472'` -> pass
- `cd bigclaw-go && go test -count=1 ./internal/regression` -> pass
