# BIG-GO-1004 Workpad

## Scope

Target the residual `src/bigclaw` Python surfaces in the repo/governance/reporting/risk/planning/mapping/memory/operations/observability batch.

Current batch file list:

- `src/bigclaw/governance.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/memory.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`

Repository Python file count before this lane: `108`
Targeted batch Python file count before this lane: `16`

## Plan

1. Verify importer and test coverage for each batch file to separate safe consolidation candidates from high-coupling keepers.
2. Consolidate low-coupling repo helper modules into existing kept surfaces and preserve old import paths through package compatibility modules.
3. Remove `mapping.py` by moving its tiny compatibility helpers into the package surface.
4. Keep high-coupling modules (`governance.py`, `reports.py`, `planning.py`, `operations.py`, `observability.py`, `memory.py`, `risk.py`, `repo_board.py`, `repo_plane.py`) when they still carry substantive runtime behavior or direct test coverage.
5. Run targeted tests covering the consolidated repo and governance/mapping paths.
6. Record delete/replace/keep rationale and report the repository Python file count delta, then commit and push.

## Acceptance

- Produce the exact residual batch file list for this lane.
- Reduce the number of Python files in this batch without breaking current imports or targeted tests.
- Document delete/replace/keep rationale for each file in the final report.
- Report the repository-wide Python file count delta.

## Validation

- `python3 -m pytest tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_observability.py tests/test_governance.py tests/test_planning.py -q`
- `find . -name '*.py' | wc -l`
- `git status --short`
- `git log -1 --stat`
