# BIG-GO-1063 Workpad

## Plan
1. Inventory the residual Python assets in `src/bigclaw` and confirm which files from the suggested scope still physically exist.
2. Trace imports, CLI entrypoints, and test references for the surviving files to determine whether they can be deleted outright or need a compatibility downgrade.
3. Apply the smallest scoped change that reduces Python file count while preserving a verifiable replacement or explicit non-support path.
4. Run targeted validation commands, capture exact results, and summarize file-count impact plus residual risks.
5. Commit the scoped change set and push the branch.

## Acceptance
- Identify the Python assets handled in this batch.
- Delete, replace, or downgrade them away from active Python implementation where feasible.
- Provide exact validation commands and outcomes.
- Report the impact on total `src/bigclaw` Python file count and any remaining risk.

## Validation
- `rg -n "(reports|risk|run_detail|runtime|ui_review)"`
- `find src/bigclaw -type f | sort | rg '\\.py$' | wc -l`
- Targeted tests or smoke checks based on any remaining references discovered during implementation.
- `git status --short`

## Archived Closeout

### BIG-GO-1053

- Baseline code migration landed on `main` at `004de016252d6ca168a45dccda48fc9fa69e27f1`.
- Closeout artifacts for the lane are tracked in:
  - `reports/BIG-GO-1053-validation.md`
  - `reports/BIG-GO-1053-closeout.md`
  - `reports/BIG-GO-1053-status.json`
- Additional stale Python entrypoint tests removed after closeout verification:
  - `tests/test_parallel_validation_bundle.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
- Validation recorded for `BIG-GO-1053`:
  - `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l` -> `0`
  - `find . -name '*.py' | wc -l` -> `43`
  - `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...` -> passed
- Historical branch handoff URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-1053-validation?expand=1`
- Historical evidence branch `symphony/BIG-GO-1053-validation` has been deleted after
  the closeout landed on `main`.
- Remote closeout comment posted on merged PR `#217`:
  - `https://github.com/OpenAGIs/BigClaw/pull/217#issuecomment-4167169146`
- No writable local tracker entry exists for `BIG-GO-1053` in `local-issues.json` or the
  Symphony local issue store, so any remaining active state is external to this workspace.
- Repo-side closeout for `BIG-GO-1053` is complete; the archived notes remain here to avoid losing lane evidence while `main` has moved on to later issues.

## Results
- Residual suggested-scope files physically present at start: `src/bigclaw/reports.py`, `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`, `src/bigclaw/runtime.py`, `src/bigclaw/ui_review.py`.
- Removed by this batch: `src/bigclaw/risk.py`, `src/bigclaw/run_detail.py`.
- Consolidation targets: `src/bigclaw/runtime.py` now owns risk scoring types; `src/bigclaw/reports.py` now owns run-detail rendering types and helpers.
- Validation command: `python3 -m compileall src/bigclaw` -> passed.
- Validation command: `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_runtime_matrix.py tests/test_reports.py tests/test_observability.py` -> `47 passed in 0.15s`.
- Python file count impact in `src/bigclaw`: `19 -> 17` (`-2`).
- Residual risk: `src/bigclaw/ui_review.py`, `src/bigclaw/runtime.py`, and `src/bigclaw/reports.py` remain because they still carry active package exports and broad test or runtime dependency surfaces that would require a wider refactor.
