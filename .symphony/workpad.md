# BIGCLAW-197 Workpad

## Plan
- Inspect the shared mirror/seed/worktree bootstrap flow and isolate the minimal extension point for reusable configuration state.
- Add a serializable shared bootstrap snapshot that captures cache root, mirror path, seed path, default branch, and repo identity for reuse across repeated workspace bootstrap calls.
- Add a multi-worktree prewarm path that prepares several issue worktrees from one shared snapshot without changing single-workspace semantics.
- Update validation/reporting helpers to surface snapshot reuse and prewarm summary details.
- Add focused regression tests for snapshot creation, snapshot-backed bootstrap reuse, and prewarm result reporting.

## Acceptance
- `workspace_bootstrap` exposes a reusable shared configuration snapshot for the shared cache bootstrap flow.
- A prewarm entrypoint can prepare multiple issue worktrees while reusing one cache root, mirror, and seed checkout.
- `bootstrap_workspace()` still works for single workspace callers without requiring snapshot input.
- Validation/reporting can describe whether the shared snapshot and cache were reused across the warmed worktrees.
- Focused tests cover the new snapshot and prewarm behavior and pass.

## Validation
- `python3 -m pytest tests/test_workspace_bootstrap.py`
- `python3 -m py_compile src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py tests/test_workspace_bootstrap.py`

## Validation Results
- `2026-03-25`: `python3 -m pytest tests/test_workspace_bootstrap.py` -> `12 passed in 3.57s`
- `2026-03-25`: `python3 -m py_compile src/bigclaw/workspace_bootstrap.py src/bigclaw/workspace_bootstrap_validation.py tests/test_workspace_bootstrap.py` -> `ok`
