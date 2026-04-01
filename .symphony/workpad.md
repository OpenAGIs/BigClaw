# BIG-GO-1060 Workpad

## Plan
- Confirm the residual Python operator entrypoints still shipped in `scripts/ops/` and every live README/workflow/doc/hook/CI reference that tracks them.
- Delete the four residual Python wrappers: `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py`.
- Update README and migration-tracking docs so the supported operator paths are Go-only via `bash scripts/ops/bigclawctl ...`.
- Replace wrapper-focused regression coverage with checks that the deleted Python entrypoints stay absent and that the tracked docs no longer advertise them as supported defaults.
- Run targeted validation, capture exact commands and results, then commit and push the branch.

## Acceptance
- The four residual Python entrypoints under `scripts/ops/` are deleted from the repo.
- README, workflow-adjacent docs, hooks, and CI references for refill/workspace operator entrypoints use `bash scripts/ops/bigclawctl ...` and do not present the deleted Python wrappers as supported defaults.
- Regression tests fail if those deleted Python wrappers reappear or if the tracked migration docs regress back to Python-first guidance.
- The repository `.py` file count decreases from the pre-change baseline.

## Validation
- Capture pre/post `.py` counts with `rg --files -g '*.py' | wc -l`.
- Run targeted Go regression tests for the entrypoint migration and legacy-shim wrapper coverage.
- Run direct `bash scripts/ops/bigclawctl` help commands for `refill` and `workspace validate` to confirm the supported Go entrypoints remain usable.
- Record every command and result exactly for closeout.
