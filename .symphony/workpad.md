# BIG-GO-1042 Workpad

## Plan

1. Verify which `src/bigclaw/*.py` top-level modules already have canonical Go owners and still have only limited Python compatibility references.
2. Replace the package-level compatibility for the tranche in `src/bigclaw/__init__.py` so the standalone Python files are no longer required, including early-installed compatibility modules needed by sibling imports during package load.
3. Delete the retired Python tranche files from `src/bigclaw/` and add focused Go regression tests for the canonical owners under `bigclaw-go/internal/...`.
4. Run targeted validation for the touched Python and Go surfaces, plus before/after Python file counts, and record exact commands and results.
5. Commit the scoped change set with messages that list the deleted Python files and the added Go files/tests, then keep the branch pushed.

## Acceptance

- The repository-wide `*.py` file count decreases from the starting count for this issue.
- The deleted tranche is limited to top-level `src/bigclaw/*.py` modules already assigned to canonical Go owners, specifically the intake/DSL compatibility slice, the risk/governance/audit/execution-contract compatibility slice, the repo-governance compatibility slice, the small repo-collaboration lineage slice (`repo_links`, `repo_commits`, `repo_gateway`), the repo-platform slice (`repo_plane`, `repo_registry`, `repo_board`), the repo-triage slice (`repo_triage`), and the product-planning slice (`collaboration`, `issue_archive`, `roadmap`).
- No new Python files are added.
- The final commit message names the deleted Python files and the added Go file(s) and Go test file(s).
- Targeted Python compatibility checks and targeted Go tests pass for the touched surfaces.

## Validation

- `find . -name "*.py" | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py -q`
- `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py -q`
- `cd bigclaw-go && go test ./internal/intake ./internal/workflow ./internal/risk ./internal/governance ./internal/observability ./internal/regression`
- `cd bigclaw-go && go test ./internal/contract ./internal/regression`
- `git status --short`

## Validation Results

- `find . -name "*.py" | wc -l`
  - `63`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py -q`
  - `8 passed in 0.21s`
- `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py -q`
  - superseded by broader focused sweep below
- `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py -q`
  - `2 passed in 0.17s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_audit_events.py -q`
  - failed: `ERROR: file or directory not found: tests/test_audit_events.py`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py -q`
  - `66 passed in 0.18s`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py -q`
  - `2 passed in 0.10s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_repo_governance.py -q`
  - `68 passed in 0.16s`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_links.py tests/test_repo_gateway.py tests/test_observability.py -q`
  - `10 passed in 0.09s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_gateway.py -q`
  - `71 passed in 0.14s`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_registry.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_observability.py -q`
  - failed initially: `TypeError: unsupported operand type(s) for |: '_GenericAlias' and 'NoneType'`
  - rerun after shim fix: `11 passed in 0.08s`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_triage.py -q`
  - `2 passed in 0.07s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_observability.py tests/test_repo_collaboration.py -q`
  - `42 passed in 0.10s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_board.py tests/test_repo_collaboration.py -q`
  - `75 passed in 0.12s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_triage.py -q`
  - `77 passed in 0.13s`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_memory.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_planning.py tests/test_observability.py tests/test_reports.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_triage.py -q`
  - `77 passed in 0.12s`
- `cd bigclaw-go && go test ./internal/intake ./internal/workflow ./internal/risk ./internal/governance ./internal/observability ./internal/regression`
  - `ok  	bigclaw-go/internal/intake	(cached)`
  - `ok  	bigclaw-go/internal/workflow	(cached)`
  - `ok  	bigclaw-go/internal/risk	1.631s`
  - `ok  	bigclaw-go/internal/governance	2.167s`
  - `ok  	bigclaw-go/internal/observability	2.550s`
  - `ok  	bigclaw-go/internal/regression	2.895s`
- `cd bigclaw-go && go test ./internal/contract ./internal/regression`
  - `ok  	bigclaw-go/internal/contract	1.242s`
  - `ok  	bigclaw-go/internal/regression	2.082s`
- `cd bigclaw-go && go test ./internal/contract ./internal/repo ./internal/regression`
  - `ok  	bigclaw-go/internal/contract	(cached)`
  - `ok  	bigclaw-go/internal/repo	1.067s`
  - `ok  	bigclaw-go/internal/regression	1.368s`
- `cd bigclaw-go && go test ./internal/repo ./internal/regression`
  - `ok  	bigclaw-go/internal/repo	(cached)`
  - `ok  	bigclaw-go/internal/regression	0.514s`
- `cd bigclaw-go && go test ./internal/repo ./internal/regression`
  - `ok  	bigclaw-go/internal/repo	(cached)`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `cd bigclaw-go && go test ./internal/repo ./internal/regression`
  - `ok  	bigclaw-go/internal/repo	(cached)`
  - `ok  	bigclaw-go/internal/regression	1.127s`
- `cd bigclaw-go && go test ./internal/regression`
  - `ok  	bigclaw-go/internal/regression	0.914s`
- `git status --short`
  - pending final stage/commit for second tranche
