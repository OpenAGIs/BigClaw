## Plan

1. Purge `src/bigclaw/governance.py` by moving the retained Python governance dataclasses and report helpers into `src/bigclaw/planning.py`.
2. Keep the `bigclaw.governance` import path working by exporting the moved governance surface from `src/bigclaw/__init__.py` and installing a synthetic compatibility module there.
3. Add tranche 15 Go regression coverage proving the deleted Python file is gone and the Go governance replacement files exist.
4. Run focused Python and Go validation plus the repository Python file count check.
5. Commit with the deleted Python file and added Go test file explicitly listed, then push to `origin/BIG-GO-1041`.

## Acceptance

- `src/bigclaw/governance.py` is deleted.
- `src/bigclaw/planning.py` provides the retained Python governance surface previously owned by `governance.py`.
- `src/bigclaw/__init__.py` no longer imports from `src/bigclaw/governance.py`, and `import bigclaw.governance` still resolves through package compatibility wiring.
- `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go` asserts the Python deletion and Go governance replacement files.
- `find . -name '*.py' | wc -l` decreases from the current baseline of `44`.
- Focused Python and Go tests pass.
- Changes are committed and pushed to `origin/BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_repo_rollout.py -q`
- `cd bigclaw-go && go test ./internal/governance ./internal/regression -run 'TestScopeFreezeBoardRoundTripPreservesManifestShape|TestScopeFreezeAuditFlagsBacklogGovernanceAndCloseoutGaps|TestScopeFreezeAuditRoundTripAndReadyState|TestRenderScopeFreezeReportSummarizesBoardAndRunCloseoutRequirements|TestTopLevelModulePurgeTranche(1|2|3|4|5|6|7|8|9|10|11|12|13|14|15)'`
- `git status --short`
- `git log -1 --stat`
