# BIG-GO-984 Workpad

## Plan
1. Identify the exact `src/bigclaw/**` Python files that belong to this batch and measure current Python file counts.
2. Inspect references, Go replacements, and packaging entry points to decide whether each target can be removed, replaced, or must be retained.
3. Apply the minimal scoped changes for this batch to reduce Python files where safe.
4. Run targeted validation and count Python files before/after.
5. Commit and push changes to the remote branch.

## Acceptance
- Produce a clear list of Python files covered by this batch under `src/bigclaw/**`.
- Reduce Python file count in the targeted area where safe.
- Record rationale for each file as removed, replaced, or retained.
- Report impact on total repository Python file count.

## Validation
- Capture exact discovery commands for the batch file list and total Python file counts.
- Run targeted tests or checks covering touched packaging/runtime paths.
- Record exact commands and outcomes in the final report.

## Batch Files
- Removed: `src/bigclaw/repo_board.py`
- Removed: `src/bigclaw/repo_commits.py`
- Removed: `src/bigclaw/repo_gateway.py`
- Removed: `src/bigclaw/repo_governance.py`
- Removed: `src/bigclaw/repo_registry.py`
- Removed: `src/bigclaw/repo_triage.py`
- Removed: `src/bigclaw/issue_archive.py`
- Removed: `src/bigclaw/roadmap.py`
- Retained: `src/bigclaw/collaboration.py`
- Retained: `src/bigclaw/repo_links.py`
- Retained: `src/bigclaw/repo_plane.py`

## Rationale
- The eight removed files had no active `src/bigclaw` or script consumers; remaining references were limited to legacy Python tests and stale package exports.
- Go-owned replacements already exist for the removed repo-oriented surfaces under `bigclaw-go/internal/repo/*` and related `bigclaw-go/internal/api/*` wiring.
- `issue_archive.py` and `roadmap.py` had no active consumers beyond `src/bigclaw/__init__.py`, so keeping them only preserved dead Python surface area.
- `collaboration.py`, `repo_links.py`, and `repo_plane.py` remain because `src/bigclaw/observability.py` and `src/bigclaw/reports.py` still import them directly.

## Results
- Total Python files before: `116`
- Total Python files after: `102`
- `src/bigclaw` Python files before: `45`
- `src/bigclaw` Python files after: `37`

## Validation Results
- `PYTHONPATH=src python3 -m pytest -q tests/test_observability.py tests/test_reports.py` -> `41 passed in 0.22s`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/collaboration.py src/bigclaw/repo_links.py src/bigclaw/repo_plane.py` -> passed
