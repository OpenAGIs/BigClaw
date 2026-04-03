# BIG-GO-1106

## Plan
- confirm the lane-covered residual Python files that still exist in this workspace and identify which ones can be removed without breaking surviving Python imports
- migrate active planning evidence links away from deleted Python paths and onto the Go-native planning/uireview implementations
- delete the removable residual Python assets in this lane: `src/bigclaw/planning.py` and `src/bigclaw/ui_review.py`
- inline the residual `risk.py` logic into `runtime.py` so the standalone Python risk module can be deleted without changing runtime behaviour
- inline the residual `run_detail.py` rendering helpers into `reports.py` and repoint `evaluation.py` to the surviving module so the standalone Python run-detail module can be deleted
- refresh migration documentation that still lists those files as active residual Python assets
- run targeted validation, record exact commands and results, then commit and push the scoped change set

## Acceptance
- lane coverage is explicit: from the provided candidate list, the sweep started from `src/bigclaw/planning.py`, `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, and `src/bigclaw/ui_review.py`; after this lane only `src/bigclaw/reports.py` remains
- this change removes real Python assets rather than only editing tracker/docs cosmetics
- `find . -name '*.py' | wc -l` decreases from the pre-change baseline
- Go-native planning evidence and migration docs no longer point at the deleted Python files
- surviving Python imports no longer depend on standalone `risk.py` or `run_detail.py`
- validation commands and residual risks are captured with exact results

## Validation
- `find . -name '*.py' | sort`
- `rg -n "src/bigclaw/(planning|ui_review)\\.py|src/bigclaw/planning\\.py|src/bigclaw/ui_review\\.py" bigclaw-go docs src .symphony`
- `cd bigclaw-go && go test ./internal/planning ./internal/uireview ./internal/regression`
- `python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/operations.py`
- `rg -n "from \\.risk import|src/bigclaw/risk\\.py|from \\.run_detail import|src/bigclaw/run_detail\\.py" src bigclaw-go docs`
- `find . -name '*.py' | wc -l`

## Validation Results
- `find . -name '*.py' | sort` -> `./src/bigclaw/audit_events.py`, `./src/bigclaw/collaboration.py`, `./src/bigclaw/console_ia.py`, `./src/bigclaw/deprecation.py`, `./src/bigclaw/design_system.py`, `./src/bigclaw/evaluation.py`, `./src/bigclaw/governance.py`, `./src/bigclaw/legacy_shim.py`, `./src/bigclaw/models.py`, `./src/bigclaw/observability.py`, `./src/bigclaw/operations.py`, `./src/bigclaw/reports.py`, `./src/bigclaw/risk.py`, `./src/bigclaw/run_detail.py`, `./src/bigclaw/runtime.py`
- `rg -n "src/bigclaw/(planning|ui_review)\\.py|src/bigclaw/planning\\.py|src/bigclaw/ui_review\\.py" bigclaw-go docs src` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/planning ./internal/uireview ./internal/regression` -> `ok   bigclaw-go/internal/planning 1.056s`; `ok   bigclaw-go/internal/uireview 1.470s`; `ok   bigclaw-go/internal/regression 1.682s`
- `find . -name '*.py' | wc -l` -> `15` after the sweep, down from the pre-change baseline `17`
- `python3 -m py_compile src/bigclaw/runtime.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/operations.py` -> exit `0`
- `PYTHONPATH=src python3 -c "import bigclaw.runtime, bigclaw.reports, bigclaw.evaluation, bigclaw.operations"` -> exit `0`
- `rg -n "from \\.risk import|src/bigclaw/risk\\.py|from \\.run_detail import|src/bigclaw/run_detail\\.py" src bigclaw-go docs` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/risk ./internal/evaluation ./internal/regression` -> `ok   bigclaw-go/internal/risk 0.776s`; `ok   bigclaw-go/internal/evaluation 0.842s`; `ok   bigclaw-go/internal/regression (cached)`
- follow-up sweep: `find . -name '*.py' | wc -l` -> `13` after deleting `src/bigclaw/risk.py` and `src/bigclaw/run_detail.py`
