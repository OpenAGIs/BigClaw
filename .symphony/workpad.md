# BIG-GO-1071

## Plan
- Confirm whether any tracked `src/bigclaw.egg-info` or repository packaging metadata remains and identify the residual execution paths that still assume Python bootstrap behavior.
- Replace workspace bootstrap/validate Python shims with non-Python wrappers that route directly to `scripts/ops/bigclawctl`, and remove stale packaging ignore residue tied to generated `.egg-info` artifacts if it is no longer needed.
- Update focused docs/tests only where required to reflect the Go-only workspace path, then run targeted validation for the affected wrappers and regression checks.
- Commit the scoped changes and push the branch.

## Acceptance
- No tracked `src/bigclaw.egg-info` or related packaging residue remains in the implementation path for this issue.
- Repository Python file count decreases because the legacy workspace bootstrap/validate Python shims are removed.
- The default workspace bootstrap/validate path is Go-first via `scripts/ops/bigclawctl`, with no Python import/bootstrap dependency.
- Targeted validation covers the new wrapper behavior and any regression checks touched by the change.

## Validation
- `find . -maxdepth 4 \( -name '*.egg-info' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'MANIFEST.in' \) | sort`
- `rg -n "\.egg-info|workspace_bootstrap|workspace_validate|symphony_workspace_bootstrap|symphony_workspace_validate" README.md scripts tests bigclaw-go .gitignore`
- `bash scripts/ops/bigclaw_workspace_bootstrap.py --help`
- `bash scripts/ops/symphony_workspace_bootstrap.py --help`
- `bash scripts/ops/symphony_workspace_validate.py --help`
- `go test ./bigclaw-go/cmd/bigclawctl/... ./bigclaw-go/internal/bootstrap/...`
