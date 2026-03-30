Issue: BIG-GO-1011

Plan
- Inspect repository-root and near-root Python packaging/config residue: `pyproject.toml`, `setup.py`, `setup.cfg`, `MANIFEST.in`, `*.egg-info`, and bootstrap/docs references that still imply root packaging workflows.
- Remove or tighten only the remaining root/config/bootstrap residue that still treats the repo root as a Python-managed environment surface.
- Run targeted validation for the touched bootstrap/config/docs paths and capture exact commands and results.
- Commit and push the scoped change set to the current remote branch.

Acceptance
- Changes are limited to repository root/config/python packaging residuals and directly related docs or bootstrap scripts.
- Repository root remains free of `pyproject.toml`, `setup.py`, and `*.egg-info` assets.
- Any remaining root bootstrap flow avoids unnecessary Python packaging behavior and reflects the Go-mainline posture.
- Final report includes the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find . -maxdepth 1 \\( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'MANIFEST.in' -o -name '*.egg-info' -o -name '*.py' \\) | sort`
- `bash scripts/dev_bootstrap.sh`
- `BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh`
- `git diff --stat`
- `git status --short`
