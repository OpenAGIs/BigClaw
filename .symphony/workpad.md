# BIG-GO-1005 Workpad

## Plan
- Capture the exact `src/bigclaw/**` Python residual inventory and the repo-wide Python file baseline.
- Classify each residual module as delete, replace, or keep by tracing test imports, script entrypoints, and Go-mainline migration ownership already landed in `bigclaw-go`.
- Reduce the batch where safe by inlining migration-only compatibility helpers out of `src/bigclaw/**` into the direct callers, then delete the vacated module files.
- Add a batch report that records the full residual list plus the delete/replace/keep rationale and the repo-wide Python file count delta.
- Run targeted validation for the touched Python compatibility scripts and package imports, then commit and push the branch.

## Acceptance
- The final change set includes the explicit residual Python file list for `src/bigclaw/**`.
- The batch removes as many `src/bigclaw/**` Python files as safely possible in this workspace state.
- The final report explains the delete, replace, or keep basis for the batch modules.
- The final report states the impact on the total repository Python file count.

## Validation
- `rg --files src/bigclaw -g '*.py'`
- `rg --files -g '*.py' | wc -l`
- `python3 -m pytest tests/test_workspace_bootstrap.py tests/test_runtime_matrix.py`
- `python3 scripts/dev_smoke.py --help`
- `python3 scripts/ops/bigclaw_workspace_bootstrap.py --help`
- `python3 scripts/ops/symphony_workspace_validate.py --help`
- `python3 scripts/ops/symphony_workspace_bootstrap.py --help`
- `python3 scripts/ops/bigclaw_github_sync.py --help`
- `python3 scripts/ops/bigclaw_refill_queue.py --help`

## Validation Results
- `rg --files src/bigclaw -g '*.py'` -> `29` files after the sweep.
- `rg --files -g '*.py' | wc -l` -> `92`.
- `python3 -m pytest tests/test_workspace_bootstrap.py tests/test_runtime_matrix.py` -> `12 passed in 4.19s`.
- `python3 scripts/dev_smoke.py --help` -> exit `0`, printed `bigclawctl dev-smoke` usage.
- `python3 scripts/ops/bigclaw_workspace_bootstrap.py --help` -> exit `0`, printed `bigclawctl workspace` usage.
- `python3 scripts/ops/symphony_workspace_validate.py --help` -> exit `0`, printed `bigclawctl workspace validate` usage.
- `python3 scripts/ops/symphony_workspace_bootstrap.py --help` -> exit `0`, printed `bigclawctl workspace` usage.
- `python3 scripts/ops/bigclaw_github_sync.py --help` -> exit `0`, printed `bigclawctl github-sync` usage.
- `python3 scripts/ops/bigclaw_refill_queue.py --help` -> exit `0`, printed `bigclawctl refill` usage.
- `PYTHONPATH=src python3 -m bigclaw --help` -> exit `0`, printed the legacy `bigclaw` CLI help and deprecation warning.
