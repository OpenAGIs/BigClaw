## BIG-GO-1011

### Plan
1. Inspect root-level Python packaging and config residue, keeping scope limited to repository-root assets.
2. Remove obsolete root Python-specific config that is no longer required for the Go-first repository entrypoints.
3. Update root documentation to match the remaining migration-only Python posture.
4. Run targeted validation for edited root surfaces, then commit and push the change set.

### Acceptance
- Reduce actual repository-root Python packaging/config residue.
- Keep the change scoped to root/config cleanup rather than legacy Python source migration.
- Report impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.
- Record exact validation commands and results.

### Validation
- Re-scan root inventory for `pyproject.toml`, `setup.py`, `*.egg-info`, and related root config.
- Run targeted validation for the edited root automation/docs surfaces.
- Verify git diff is limited to this issue's changes, then commit and push to the remote branch.

### Continuation Notes
- Retired the remaining repo-root `scripts/ops/*.py` compatibility wrappers now that `scripts/ops/bigclawctl` is the supported operator path.
- Updated migration/cutover docs so they describe those Python wrappers as retired rather than active shims.

### Validation Results
- `find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'MANIFEST.in' -o -name '.pre-commit-config.yaml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name 'pytest.ini' -o -name '.python-version' -o -name '.coveragerc' -o -name '*.egg-info' -o -name '*.dist-info' \) | sort` -> no matches
- `bash scripts/ops/bigclawctl github-sync --help` -> exit 0
- `bash scripts/ops/bigclawctl refill --help` -> exit 0
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit 0
- `make build` -> exit 0

### Final Consistency Pass
- Updated remaining cutover/handoff docs that still implied the retired repo-root Python wrappers were active compatibility shims.
- Removed stale `PYTHONDONTWRITEBYTECODE` exports from `.githooks/post-commit` and `.githooks/post-rewrite` because those hooks now call the Go-first `scripts/ops/bigclawctl` path only.
