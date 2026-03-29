# BIG-GO-963 Validation

## Scope

Directly covered Python files in `src/bigclaw` for this material pass:

- `governance.py`
- `reports.py`
- `risk.py`
- `planning.py`
- `mapping.py`
- `memory.py`
- `operations.py`
- `observability.py`
- `repo_board.py`
- `repo_commits.py`
- `repo_gateway.py`
- `repo_governance.py`
- `repo_links.py`
- `repo_plane.py`
- `repo_registry.py`
- `repo_triage.py`

## Outcome

Retained:

- `governance.py`: active governance models and audits with dedicated tests.
- `reports.py`: large reporting surface with broad downstream use.
- `risk.py`: small but single-purpose domain logic with direct tests.
- `planning.py`: planning models and rendering helpers used by repo rollout/report tests.
- `mapping.py`: compact mapping helper still exercised by dedicated tests.
- `memory.py`: active task-memory abstraction with dedicated tests.
- `operations.py`: large operations surface with substantial test coverage.
- `observability.py`: active observability models and report integration points.

Deleted and replaced:

- `repo_board.py`
- `repo_commits.py`
- `repo_gateway.py`
- `repo_governance.py`
- `repo_links.py`
- `repo_plane.py`
- `repo_registry.py`
- `repo_triage.py`

Replacement:

- Added `repo.py` to hold the repo/governance/reporting support types previously split across the eight tiny `repo_*` files.
- Updated `__init__.py` to register import-compatible aliases so existing imports such as `bigclaw.repo_board` continue to work.
- Updated `observability.py` to depend on the consolidated `repo.py` module directly.

Rationale:

- The deleted `repo_*` files were small, tightly related, and mostly dataclass/helper containers.
- Consolidation removes physical Python assets without changing the public import surface.
- Larger modules in the batch were retained because they still carry distinct domain logic and broad test coverage, so collapsing them would create unnecessary migration risk in this pass.

## Python File Count Impact

- Targeted batch before: 16 Python files.
- Targeted batch after: 9 Python files.
- Net targeted reduction: 7 Python files.
- Total `src/bigclaw` Python files before: 50.
- Total `src/bigclaw` Python files after: 43.
- Net total reduction: 7 Python files.

## Validation

Commands and results:

- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: `repo-import-aliases-ok`; legacy imports resolve to `bigclaw.repo`.
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_links.py tests/test_repo_registry.py tests/test_repo_rollout.py tests/test_repo_triage.py`
  - Result: `13 passed in 0.08s`.
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_reports.py`
  - Result: `41 passed in 0.15s`.
- `find src/bigclaw -type f -name '*.py' | wc -l`
  - Result: `43`.
