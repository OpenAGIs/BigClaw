## Plan

1. Purge the first safe top-level Python tranche under `src/bigclaw` by deleting:
   - `src/bigclaw/cost_control.py`
   - `src/bigclaw/issue_archive.py`
   - `src/bigclaw/github_sync.py`
2. Remove any package exports that still point at those deleted Python modules so `src/bigclaw/__init__.py` no longer imports them.
3. Add a focused Go regression test that asserts the migration contract for this tranche:
   - the deleted Python files are absent
   - the corresponding Go replacement files exist
4. Run targeted validation for the touched Go packages and the new regression test.
5. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Python file count in the repository decreases from the pre-change baseline.
- `src/bigclaw/cost_control.py`, `src/bigclaw/issue_archive.py`, and `src/bigclaw/github_sync.py` are deleted.
- `src/bigclaw/__init__.py` no longer imports symbols from deleted modules.
- A Go test covers the tranche replacement contract against the repository tree.
- Targeted Go tests pass.
- Changes are committed and pushed to the remote branch for `BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/issuearchive ./internal/githubsync ./internal/regression -run 'TestTopLevelModulePurgeTranche1'`
- `git status --short`
- `git log -1 --stat`
# BIG-GO-1045

## Plan

1. Confirm which Python packaging and distribution residue still exists in this checkout and keep the change scoped to those paths.
2. Remove the legacy Python compatibility shims in `scripts/ops` that duplicate `bigclawctl` packaging/bootstrap/distribution entrypoints.
3. Update repository guidance and regression coverage so the deleted Python entrypoints are no longer referenced and the Go-owned path remains explicit.
4. Run targeted validation for the touched Go command and regression surfaces, plus a repository `*.py` count check.
5. Commit and push with a message body that enumerates deleted Python files and any added Go files or Go tests.

## Acceptance

- Python packaging/distribution residue covered by this issue is removed from the repo.
- The repository `find . -name "*.py" | wc -l` count is lower after the change.
- Remaining docs and tests reference the Go-owned `scripts/ops/bigclawctl` path instead of deleted Python shims.
- Targeted tests covering the affected migration/legacy-shim surfaces pass.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- `git status --short`

## Results

- `find . -name '*.py' | wc -l`
  Result: `78` before the purge, `73` after the purge.
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
  Result: `ok bigclaw-go/cmd/bigclawctl 5.215s`, `ok bigclaw-go/internal/legacyshim 2.623s`, `ok bigclaw-go/internal/regression 2.233s`.
- `bash scripts/ops/bigclawctl github-sync --help`
  Result: exited `0`; emitted `usage: bigclawctl github-sync <install|status|sync> [flags]`.
- `bash scripts/ops/bigclawctl workspace validate --help`
  Result: exited `0`; help output includes workspace validation flags such as `-report`, `-workspace`, and `-workspace-root`.
