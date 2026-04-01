## Plan

1. Purge the first safe top-level Python tranche under `src/bigclaw` by deleting:
   - `src/bigclaw/cost_control.py`
   - `src/bigclaw/issue_archive.py`
   - `src/bigclaw/github_sync.py`
2. Purge the next safe top-level Python repo-surface tranche under `src/bigclaw` by deleting:
   - `src/bigclaw/repo_board.py`
   - `src/bigclaw/repo_commits.py`
   - `src/bigclaw/repo_gateway.py`
   - `src/bigclaw/repo_governance.py`
   - `src/bigclaw/repo_registry.py`
   - `src/bigclaw/repo_triage.py`
3. Remove any package exports or Python tests that still point at deleted Python modules.
4. Add focused Go regression tests that assert the migration contract for each tranche:
   - the deleted Python files are absent
   - the corresponding Go replacement files exist
5. Run targeted validation for the touched Go packages and the new regression tests.
6. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Python file count in the repository decreases from the pre-change baseline.
- `src/bigclaw/cost_control.py`, `src/bigclaw/issue_archive.py`, and `src/bigclaw/github_sync.py` are deleted.
- `src/bigclaw/repo_board.py`, `src/bigclaw/repo_commits.py`, `src/bigclaw/repo_gateway.py`, `src/bigclaw/repo_governance.py`, `src/bigclaw/repo_registry.py`, and `src/bigclaw/repo_triage.py` are deleted.
- `src/bigclaw/__init__.py` and retained Python tests no longer import deleted modules.
- Go regression tests cover the tranche replacement contracts against the repository tree.
- Targeted Go tests pass.
- Changes are committed and pushed to the remote branch for `BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/issuearchive ./internal/githubsync ./internal/repo ./internal/regression -run 'TestTopLevelModulePurgeTranche(1|2)|TestBindRunCommitsAndAcceptedHash|TestRepoRegistryResolvesSpaceChannelAndAgent|TestNormalizeGatewayPayloadsAndErrors|TestRecommendTriageAction'`
- `git status --short`
- `git log -1 --stat`
