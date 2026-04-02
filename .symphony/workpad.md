# BIG-GO-1071

## Status
- Completed on branch `symphony/BIG-GO-1071`.
- Python workspace bootstrap/validate shims were removed and replaced with shell wrappers that dispatch to `scripts/ops/bigclawctl`.
- Current implementation commit: `bb1d0b56f59eeb40779b09632b930201600042e5`.

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
- Result: no output
- `rg --files | rg '\.py$' | wc -l`
- Result: `38`
- `rg -n "\.egg-info|bigclaw_workspace_bootstrap\.py|symphony_workspace_bootstrap\.py|symphony_workspace_validate\.py" README.md docs bigclaw-go src tests scripts .gitignore`
- Result: no output
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- Result: passed; emitted `bigclawctl workspace bootstrap` usage
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- Result: passed; emitted `bigclawctl workspace <bootstrap|cleanup|validate>` usage
- `bash scripts/ops/symphony_workspace_validate --help`
- Result: passed; emitted `bigclawctl workspace validate` usage
- `go test ./bigclaw-go/cmd/bigclawctl/... ./bigclaw-go/internal/bootstrap/... ./bigclaw-go/internal/legacyshim/...`
- Result: failed from repo root with Go module path error
- `go test ./cmd/bigclawctl/... ./internal/bootstrap/... ./internal/legacyshim/...`
  run from `bigclaw-go/`
- Result: passed

## Notes
- Removed `.gitignore` entry `*.egg-info/` because the repository no longer carries packaging outputs for this path.
- Removed deleted-workspace-shim references from the legacy Python compile-check list so regression coverage now matches the reduced Python shim surface.
- Added automated Go regression coverage in `bigclaw-go/internal/legacyshim/shellwrappers_test.go` to execute the retained shell wrappers and verify their `bigclawctl` help surfaces.

## Follow-up Validation
- `go test ./internal/legacyshim/...`
- Result: passed
- `go test ./cmd/bigclawctl/... ./internal/bootstrap/... ./internal/legacyshim/...`
  run from `bigclaw-go/`
- Result: passed
