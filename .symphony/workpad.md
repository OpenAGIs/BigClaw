# BIG-GO-972 Workpad

## Scope

Targeted legacy Python modules under `src/bigclaw` for this lane:

- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_triage.py`

Retained implementation home:

- `src/bigclaw/repo_plane.py`

Current repository Python file count before this lane: `45`

## Plan

1. Confirm the exact lane-owned `repo_*` files and their import/test dependencies.
2. Consolidate the seven targeted modules into `src/bigclaw/repo_plane.py` while preserving the legacy import surface for `bigclaw.repo_commits`, `bigclaw.repo_board`, `bigclaw.repo_gateway`, `bigclaw.repo_links`, `bigclaw.repo_registry`, `bigclaw.repo_governance`, and `bigclaw.repo_triage`.
3. Delete the superseded module files after package-level compatibility modules are installed from `src/bigclaw/__init__.py`.
4. Run targeted validation for the repo-plane compatibility surface and record exact commands and results here.
5. Report the exact target file list, the delete/replace/retain rationale, and the net impact on total `src/bigclaw` Python file count.
6. Commit and push the scoped lane changes.

## Acceptance

- Produce the exact Python file list directly owned by `BIG-GO-972`.
- Reduce the number of Python files in the targeted `repo_*` surface.
- Preserve import-compatible behavior for `bigclaw.repo_commits`, `bigclaw.repo_board`, `bigclaw.repo_gateway`, `bigclaw.repo_links`, `bigclaw.repo_registry`, `bigclaw.repo_governance`, and `bigclaw.repo_triage`.
- Record delete/replace/retain reasoning for each targeted file.
- Report before/after total Python file counts for `src/bigclaw`.

## Validation

- `python3 -m compileall src/bigclaw`
- `PYTHONPATH=src python3 - <<'PY' ...` import smoke check for the retained and compatibility repo surfaces
- `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_repo_governance.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_observability.py`
- `git status --short`

## Results

### File Disposition

- `src/bigclaw/repo_plane.py`
  - Retained and expanded.
  - Reason: now serves as the single implementation home for the repo-support data model, gateway helpers, discussion board helpers, governance helpers, link binding helpers, registry helpers, and triage helpers.
- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_board` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_commits.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_commits` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_gateway.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_gateway` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_governance` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_links.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_links` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_registry` import compatibility is now installed from `src/bigclaw/__init__.py`.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/repo_plane.py`; `bigclaw.repo_triage` import compatibility is now installed from `src/bigclaw/__init__.py`.

### Python File Count Impact

- Repository `src/bigclaw` Python files before: `45`
- Repository `src/bigclaw` Python files after: `38`
- Net reduction: `7`

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY' ...`
  - Result: success; verified `bigclaw.repo_board`, `bigclaw.repo_commits`, `bigclaw.repo_gateway`, `bigclaw.repo_governance`, `bigclaw.repo_links`, `bigclaw.repo_registry`, and `bigclaw.repo_triage` all import cleanly after file deletion.
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_repo_collaboration.py tests/test_observability.py`
  - Result: `18 passed in 0.09s`
- `git status --short`
  - Result: only `.symphony/workpad.md`, `src/bigclaw/__init__.py`, `src/bigclaw/repo_plane.py`, and the seven deleted batch-owned `src/bigclaw/repo_*` files are present in the scoped diff.
