# BIG-GO-1356 Workpad

## Plan
- Inspect repo-native Ray smoke entrypoints and checked-in validation artifacts for residual `python -c` usage.
- Replace the default Ray smoke entrypoint with a shell-native command that preserves the smoke semantics without Python.
- Add or extend regression coverage to enforce the Go/shell-native Ray smoke path and verify checked-in evidence no longer records Python entrypoints for this flow.
- Run targeted tests and repository checks, then commit and push the issue branch.

## Acceptance
- Repository reality stays Python-free for physical files: `find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l` remains `0`.
- A concrete native replacement lands for Python entrypoint consolidation removal in the Ray smoke path.
- Checked-in docs/reports and regression coverage align with the native replacement.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l`
- `cd bigclaw-go && go test ./internal/regression ./scripts/e2e/... ./cmd/bigclawctl/...` if package layout permits; otherwise narrow to impacted packages.
- `git diff --stat`
