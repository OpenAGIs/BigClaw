# BIG-GO-1065

## Plan
- inventory the Python test files from the issue batch and map each surviving file to an existing Go replacement or removal candidate
- remove the residual Python files that are already covered by Go tests or regression guards
- run targeted validation for the affected Go packages and regression coverage
- record exact commands, results, Python file count impact, and residual risks
- commit and push the scoped change set to the issue branch

## Acceptance
- identify the Python assets handled in this tranche
- reduce the repository Python file count by deleting or replacing covered Python artifacts
- keep a verifiable Go replacement path for each removed asset
- provide exact validation commands and results
- report remaining risks and the Python file count delta

## Validation
- `rg --files tests | sort`
- `git diff --stat`
- `cd bigclaw-go && go test ./...`
- narrower `go test` package/regression commands for touched replacement paths if full sweep is unnecessary

## Completed
- confirmed the issue's suggested `tests/test_*.py` tranche was already absent from the repo before this turn
- removed `src/bigclaw/governance.py`, `src/bigclaw/planning.py`, and `src/bigclaw/ui_review.py`
- removed the corresponding legacy exports from `src/bigclaw/__init__.py`
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go` to pin the deletions against Go replacement paths
- updated `docs/go-mainline-cutover-issue-pack.md` so the migration inventory reflects the deleted Python assets

## Validation Results
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/operations.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/runtime.py src/bigclaw/observability.py tests/test_control_center.py tests/test_evaluation.py tests/test_console_ia.py tests/test_design_system.py` -> passed
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15|TestFollowUpLaneDocsStayAligned|TestExecutionPackRoadmapDocsStayAligned|TestExecutionPackRoadmapUniqueOwnersContract'` -> `ok  	bigclaw-go/internal/regression	0.766s`
- `cd bigclaw-go && go test ./internal/governance ./internal/product` -> `ok  	bigclaw-go/internal/governance	0.454s`; `ok  	bigclaw-go/internal/product	(cached)`
- `PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_evaluation.py tests/test_console_ia.py tests/test_design_system.py -q` -> `36 passed in 0.13s`

## Python Count Impact
- before: `28`
- after: `25`
- delta: `-3`

## Residual Risks
- `src/bigclaw/risk.py`, `src/bigclaw/runtime.py`, `src/bigclaw/reports.py`, `src/bigclaw/operations.py`, and related modules still participate in the surviving Python test surface, so they remain out of scope for this tranche
- legacy Python CLI shim files under `scripts/ops/*.py` and `src/bigclaw/legacy_shim.py` remain active compatibility wrappers and were not touched
