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
3. Purge the isolated bootstrap validation surface:
   - `src/bigclaw/workspace_bootstrap_validation.py`
4. Purge the next low-coupling top-level product/reporting tranche:
   - `src/bigclaw/pilot.py`
   - `src/bigclaw/dashboard_run_contract.py`
   - `src/bigclaw/saved_views.py`
5. Purge the next low-coupling top-level contract/intake tranche:
   - `src/bigclaw/mapping.py`
   - `src/bigclaw/execution_contract.py`
6. Purge the isolated refill queue surface:
   - `src/bigclaw/parallel_refill.py`
7. Purge the isolated workspace bootstrap CLI surface:
   - `src/bigclaw/workspace_bootstrap_cli.py`
8. Purge the isolated workspace bootstrap implementation surface:
   - `src/bigclaw/workspace_bootstrap.py`
9. Purge the next low-coupling top-level intake/roadmap surface:
   - `src/bigclaw/connectors.py`
   - `src/bigclaw/roadmap.py`
10. Remove any package exports or Python tests that still point at deleted Python modules.
11. Add focused Go regression tests that assert the migration contract for each tranche:
   - the deleted Python files are absent
   - the corresponding Go replacement files exist
12. Run targeted validation for the touched Go packages and the new regression tests.
13. Commit with a message that explicitly lists deleted Python files and added Go test files, then push the branch.

## Acceptance

- Python file count in the repository decreases from the pre-change baseline.
- `src/bigclaw/cost_control.py`, `src/bigclaw/issue_archive.py`, and `src/bigclaw/github_sync.py` are deleted.
- `src/bigclaw/repo_board.py`, `src/bigclaw/repo_commits.py`, `src/bigclaw/repo_gateway.py`, `src/bigclaw/repo_governance.py`, `src/bigclaw/repo_registry.py`, and `src/bigclaw/repo_triage.py` are deleted.
- `src/bigclaw/workspace_bootstrap_validation.py` is deleted.
- `src/bigclaw/pilot.py`, `src/bigclaw/dashboard_run_contract.py`, and `src/bigclaw/saved_views.py` are deleted.
- `src/bigclaw/mapping.py` and `src/bigclaw/execution_contract.py` are deleted.
- `src/bigclaw/parallel_refill.py` is deleted.
- `src/bigclaw/workspace_bootstrap_cli.py` is deleted.
- `src/bigclaw/workspace_bootstrap.py` is deleted.
- `src/bigclaw/connectors.py` and `src/bigclaw/roadmap.py` are deleted.
- `src/bigclaw/__init__.py` and retained Python tests no longer import deleted modules.
- Go regression tests cover the tranche replacement contracts against the repository tree.
- Targeted Go tests pass.
- Changes are committed and pushed to the remote branch for `BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/intake ./internal/regression -run 'TestConnectorByNameReturnsKnownConnectors|TestConnectorStubsReturnSeededIssues|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract|TestTopLevelModulePurgeTranche(1|2|3|4|5|6|7|8|9)'`
- `git status --short`
- `git log -1 --stat`
