# BIG-GO-1063

## Plan
- Inventory the requested residual Python assets and confirm which ones can be removed without reopening wider report-generation migration work.
- Collapse `src/bigclaw/runtime.py` onto the existing compatibility shim path in `src/bigclaw/legacy_shim.py`, then delete the physical `runtime.py` file.
- Delete `src/bigclaw/ui_review.py` and clean Python exports/tests that still reference the removed module.
- Add a targeted Go regression that locks the removed Python files out of the repo and run focused validation.

## Batch Asset List
- `src/bigclaw/runtime.py`: remove physical Python file; preserve legacy import compatibility through `src/bigclaw/legacy_shim.py`; Go replacement path remains `bigclaw-go/internal/worker/runtime.go`.
- `src/bigclaw/ui_review.py`: remove physical Python file and dependent Python tests/exports; no active Go runtime dependency.
- `src/bigclaw/reports.py`: inventoried but intentionally left for a later batch because it still backs `operations.py`, `evaluation.py`, `repo_rollout` tests, and `__main__.py`.

## Acceptance
- This batch reports the explicit residual asset list above.
- `src/bigclaw/runtime.py` and `src/bigclaw/ui_review.py` are removed from the repository.
- Legacy runtime imports still resolve through a non-`runtime.py` shim so existing scheduler/queue/service Python surfaces remain loadable.
- Validation commands, exact results, residual risks, and Python file count delta are recorded.

## Validation
- Inventory: `find src -type f \( -name "*.py" -o -name "*.pyi" \) | wc -l`
- Python compatibility checks:
  - `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_control_center.py tests/test_operations.py tests/test_evaluation.py -q`
- Go regression:
  - `cd bigclaw-go && go test ./internal/regression -run TestPythonResidualSweepRemovesRuntimeAndUIReview -count=1`
- Reference sweep:
  - `rg -n "bigclaw\\.ui_review|src/bigclaw/ui_review.py|src/bigclaw/runtime.py" src tests README.md docs bigclaw-go`

## Results
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_control_center.py tests/test_operations.py tests/test_evaluation.py -q`
  - `50 passed in 0.11s`
- `cd bigclaw-go && go test ./internal/regression -run TestPythonResidualSweepRemovesRuntimeAndUIReview -count=1`
  - `ok  	bigclaw-go/internal/regression	0.178s`
- `find src -type f \( -name "*.py" -o -name "*.pyi" \) | wc -l`
  - `15`
