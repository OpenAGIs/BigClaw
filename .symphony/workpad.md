# BIG-GO-971 Workpad

## Scope

Targeted legacy Python modules under `src/bigclaw` for this batch:

- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`

Planned consolidation host:

- `src/bigclaw/repo_plane.py`

Current repository Python file count before this lane: `45`

## Plan

1. Confirm the exact import and test dependency surface for the seven targeted repo modules.
2. Consolidate the data models and helper functions from the targeted modules into `src/bigclaw/repo_plane.py`.
3. Preserve the legacy import surface for `bigclaw.repo_board`, `bigclaw.repo_commits`, `bigclaw.repo_gateway`, `bigclaw.repo_governance`, `bigclaw.repo_links`, `bigclaw.repo_registry`, and `bigclaw.repo_triage` through package-level compatibility shims.
4. Delete the superseded module files after the compatibility surface is in place.
5. Run targeted validation for the touched repo-plane and observability surfaces and record exact commands and results here.
6. Report the batch file list, delete/replace/retain rationale, and the before/after Python file counts.
7. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact Python file list directly owned by `BIG-GO-971`.
- Reduce the number of Python files in the targeted repo metadata / gateway surface.
- Preserve import compatibility for `bigclaw.repo_board`, `bigclaw.repo_commits`, `bigclaw.repo_gateway`, `bigclaw.repo_governance`, `bigclaw.repo_links`, `bigclaw.repo_registry`, and `bigclaw.repo_triage`.
- Record delete/replace/retain reasoning for each targeted legacy file.
- Report before/after total Python file counts for `src/bigclaw`.

## Validation

- Import smoke checks for the legacy module names after consolidation.
- Targeted tests for repo-board, repo-gateway, repo-governance, repo-links, repo-registry, repo-triage, repo-collaboration, and observability closeout paths.
- `python3 -m compileall src/bigclaw` for syntax coverage across the migrated package.
- `git status --short` to confirm the change set stays scoped to this lane.

## Notes

- This batch is intentionally limited to short, tightly related `repo_*` modules to keep the blast radius low while still removing multiple files.
- `repo_plane.py` is already the shared domain home for repo-space and run-link entities, making it the least surprising consolidation target.

## Results

### File Disposition

- `src/bigclaw/repo_plane.py`
  - Retained and expanded.
  - Reason: became the consolidated implementation home for repo board state, repo commit models, repo gateway normalization helpers, repo governance policy, run commit binding helpers, repo registry state, and repo triage helpers.
- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_board` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_commits.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_commits` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_gateway.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_gateway` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_governance` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_links.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_links` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_registry` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_triage` import compatibility is now provided from `src/bigclaw/__init__.py`.

### Python File Count Impact

- Repository `src/bigclaw` Python files before: `45`
- Repository `src/bigclaw` Python files after: `38`
- Net reduction: `7`

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY' ...`
  - Result: success; verified `bigclaw.repo_plane`, `bigclaw.repo_board`, `bigclaw.repo_governance`, `bigclaw.repo_triage`, `bigclaw.repo_commits`, `bigclaw.repo_gateway`, `bigclaw.repo_links`, and `bigclaw.repo_registry` all import cleanly through the consolidated implementation.
- `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_repo_board.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_collaboration.py tests/test_observability.py`
  - Result: `18 passed in 0.09s`
- `git status --short`
  - Result: scoped to `.symphony/workpad.md`, `src/bigclaw/__init__.py`, `src/bigclaw/repo_plane.py`, and deletion of the seven targeted legacy module files.
