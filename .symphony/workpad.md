# BIG-GO-974 Workpad

## Scope

Batch 3 targets the remaining small Python test modules under `tests/` that can be consolidated without changing production code:

- `tests/test_connectors.py`
- `tests/test_mapping.py`
- `tests/test_memory.py`
- `tests/test_validation_policy.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_triage.py`

Repository Python file count before this lane: `118`
Targeted `tests/*.py` file count before this lane: `43`

## Plan

1. Read each batch-owned test module and group them into coherent replacement bundles.
2. Create one foundation bundle for connectors, mapping, memory, and validation-policy coverage.
3. Create one repo-surface bundle for repo board, collaboration, gateway, governance, links, registry, rollout, and triage coverage.
4. Delete the superseded per-file test modules after the bundled replacements are in place.
5. Run targeted pytest commands against the new bundle modules and record exact results.
6. Record before/after Python file counts, commit the scoped changes, and push the branch.

## Acceptance

- Produce the exact `BIG-GO-974` batch file list under `tests/`.
- Replace the batch-owned Python test files with fewer bundled test modules.
- Preserve existing assertion coverage for the migrated test cases.
- Report the before/after total repository Python file count and targeted `tests/` Python file count.
- Record the exact validation commands and results used for this lane.

## Validation

- `python3 -m pytest tests/test_foundation_bundle.py tests/test_repo_surface_bundle.py`
- `git diff --stat`
- `find tests -maxdepth 1 -name '*.py' | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
- `git status --short`

## Results

### Replacement Files

- `tests/test_foundation_bundle.py`
  - Replaces:
    - `tests/test_connectors.py`
    - `tests/test_mapping.py`
    - `tests/test_memory.py`
    - `tests/test_validation_policy.py`
- `tests/test_repo_surface_bundle.py`
  - Replaces:
    - `tests/test_repo_board.py`
    - `tests/test_repo_collaboration.py`
    - `tests/test_repo_gateway.py`
    - `tests/test_repo_governance.py`
    - `tests/test_repo_links.py`
    - `tests/test_repo_registry.py`
    - `tests/test_repo_rollout.py`
    - `tests/test_repo_triage.py`

### Python File Count Impact

- Targeted `tests/*.py` files before: `43`
- Targeted `tests/*.py` files after: `33`
- Net targeted reduction: `10`
- Repository Python files before: `118`
- Repository Python files after: `108`
- Net repository reduction: `10`

### Validation Record

- `PYTHONPATH=src python3 -m pytest tests/test_foundation_bundle.py tests/test_repo_surface_bundle.py`
  - Result: `19 passed in 0.15s`
- `find tests -maxdepth 1 -name '*.py' | wc -l`
  - Result: `33`
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
  - Result: `108`
- `git diff --stat`
  - Result: deleted the 12 batch-owned files and updated the workpad; new bundle files are present as untracked additions prior to commit.
