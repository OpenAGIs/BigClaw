# BIGCLAW-197 Workpad

## Plan
- Verify the current shared snapshot and prewarm implementation, then close the remaining delivery gap in the operator-facing CLI.
- Add explicit CLI support for preparing a reusable shared snapshot and prewarming multiple issue worktrees from one snapshot.
- Allow single-workspace bootstrap to consume a snapshot file so the shared configuration can be reused across unattended runs.
- Add focused regression tests for snapshot file emission, snapshot-backed bootstrap, and parallel prewarm invocation.

## Acceptance
- Operators can generate a shared bootstrap snapshot from `workspace_bootstrap_cli`.
- Operators can prewarm multiple issue worktrees from one shared snapshot through `workspace_bootstrap_cli`.
- Single-workspace bootstrap continues to work and can optionally consume a snapshot file.
- Focused tests cover the new CLI surfaces and pass.

## Validation
- `python3 -m pytest tests/test_workspace_bootstrap.py`
- `python3 -m py_compile src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_cli.py src/bigclaw/workspace_bootstrap_validation.py tests/test_workspace_bootstrap.py`

## Validation Results
- `2026-03-26`: `python3 -m pytest tests/test_workspace_bootstrap.py` -> `15 passed in 4.51s`
- `2026-03-26`: `python3 -m py_compile src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_cli.py src/bigclaw/workspace_bootstrap_validation.py tests/test_workspace_bootstrap.py` -> `ok`
