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
