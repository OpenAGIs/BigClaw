# BIG-GO-1011 Validation

## Scope

Refill sweep A for remaining repository-root/config/python packaging residuals.

## Branch

- branch: `big-go-1011-root-config-residuals`

## Repo impact

- `py files`: `101`
- `go files`: `267`
- `pyproject.toml`: `0`
- `setup.py`: `0`

## Committed cleanup

- `cd902ee` `BIG-GO-1011 remove root python config residuals`
- `63f6db5` `BIG-GO-1011 retire root python wrapper scripts`
- `fce74ce` `BIG-GO-1011 align cutover docs with wrapper retirement`
- `93f8495` `BIG-GO-1011 drop python hook env residue`
- `0ef27ab` `BIG-GO-1011 remove egg-info ignore residue`
- `57aefe0` `BIG-GO-1011 ignore root pytest cache residue`
- `49603e7` `BIG-GO-1011 record root residue scope boundary`
- `c8102e6` `BIG-GO-1011 add validation evidence report`
- `8de0ce9` `BIG-GO-1011 refresh validation report head`
- `48dc372` `BIG-GO-1011 finalize validation report metadata`

## Validation commands

```bash
find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'MANIFEST.in' -o -name '*.egg-info' -o -name '*.dist-info' -o -name '.pre-commit-config.yaml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name 'pytest.ini' -o -name '.python-version' -o -name '.coveragerc' -o -name 'Pipfile' -o -name 'poetry.lock' -o -name 'uv.lock' \) | sort
```

Result: no matches

```bash
make build
```

Result: passed

```bash
make test
```

Result: failed in existing Go regression:

```text
bigclaw-go/internal/regression
TestContinuationPolicyGateReviewerMetadata
runtime_report_followup_docs_test.go:144: unexpected policy gate reviewer digest issue: {ID: Slug:}
```

```bash
bash scripts/ops/bigclawctl github-sync --help
bash scripts/ops/bigclawctl refill --help
bash scripts/ops/bigclawctl workspace validate --help
```

Result: all exited `0`

```bash
git diff --stat origin/main...HEAD
```

Result:

```text
15 files changed, 66 insertions(+), 190 deletions(-)
```

## Scope boundary

Remaining root-level Python mentions are intentional migration-only validation surfaces:

- `scripts/dev_bootstrap.sh`
- source-level `PYTHONPATH=src python3 -m pytest ...` guidance
- root cache ignores for `__pycache__/`, `*.py[cod]`, and `.pytest_cache/`

No additional root `pyproject.toml`, `setup.py`, `*.egg-info`, repo-root Python wrapper scripts, or Python-specific CI/hook config residue remains.
