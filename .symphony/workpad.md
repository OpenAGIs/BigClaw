# BIG-GO-1063

## Plan
- Inventory the requested residual Python assets and confirm which ones can be removed without reopening wider report-generation migration work.
- Collapse `src/bigclaw/runtime.py` onto the existing compatibility shim path in `src/bigclaw/legacy_shim.py`, then delete the physical `runtime.py` file.
- Delete `src/bigclaw/ui_review.py` and clean Python exports/tests that still reference the removed module.
- Move `src/bigclaw/reports.py` to a non-`.py` legacy source asset loaded from package init, then lock all removed Python files out with focused regression coverage.

## Batch Asset List
- Purged and locked out from the requested scope: `src/bigclaw/repo_links.py`, `src/bigclaw/repo_plane.py`, `src/bigclaw/repo_registry.py`, `src/bigclaw/repo_triage.py`, `src/bigclaw/risk.py`, `src/bigclaw/roadmap.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/saved_views.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/service.py`, `src/bigclaw/validation_policy.py`, `src/bigclaw/workflow.py`, `src/bigclaw/workspace_bootstrap.py`, `src/bigclaw/workspace_bootstrap_cli.py`, `src/bigclaw/workspace_bootstrap_validation.py`.
- `src/bigclaw/runtime.py`: remove physical Python file; preserve legacy import compatibility through `src/bigclaw/legacy_shim.py`; Go replacement path remains `bigclaw-go/internal/worker/runtime.go`.
- `src/bigclaw/ui_review.py`: remove physical Python file and dependent Python tests/exports; no active Go runtime dependency.
- `src/bigclaw/reports.py`: remove physical Python file; preserve `bigclaw.reports` imports by preloading `src/bigclaw/_legacy/reports.legacy` from package init.

## Acceptance
- This batch reports the explicit residual asset list above.
- `src/bigclaw/runtime.py`, `src/bigclaw/ui_review.py`, and `src/bigclaw/reports.py` are removed from the repository.
- Legacy runtime imports still resolve through a non-`runtime.py` shim so existing scheduler/queue/service Python surfaces remain loadable.
- `bigclaw.reports` still resolves through a non-`.py` legacy source asset.
- Validation commands, exact results, residual risks, and Python file count delta are recorded.

## Validation
- Inventory: `find src -type f \( -name "*.py" -o -name "*.pyi" \) | wc -l`
- Python compatibility checks:
  - `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py tests/test_observability.py tests/test_repo_rollout.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_control_center.py tests/test_operations.py tests/test_evaluation.py -q`
- Go regression:
  - `cd bigclaw-go && go test ./internal/regression -run TestPythonResidualSweepKeepsTargetedModulesPurged -count=1`
- Reference sweep:
  - `rg -n "bigclaw\\.ui_review|src/bigclaw/ui_review.py|src/bigclaw/runtime.py|src/bigclaw/reports.py" src tests README.md docs bigclaw-go`

## Results
- `python3 - <<'PY'`
  - `import bigclaw.reports` resolved to `src/bigclaw/_legacy/reports.legacy`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py tests/test_observability.py tests/test_repo_rollout.py tests/test_runtime_matrix.py tests/test_risk.py tests/test_control_center.py tests/test_operations.py tests/test_evaluation.py -q`
  - `93 passed in 0.17s`
- `cd bigclaw-go && go test ./internal/regression -run TestPythonResidualSweepKeepsTargetedModulesPurged -count=1`
  - `ok  	bigclaw-go/internal/regression	0.750s`
- `find src -type f \( -name "*.py" -o -name "*.pyi" \) | wc -l`
  - `14`

## Residual Risk
- Code-path residual risk is now limited to compatibility shims and legacy source assets: `src/bigclaw/legacy_shim.py` and `src/bigclaw/_legacy/reports.legacy`.
- Historical cutover planning docs may still describe older migration states, but the explicit deleted-path references in `docs/go-mainline-cutover-issue-pack.md` were updated in this batch.
