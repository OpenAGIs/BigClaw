# BIG-GO-973 Workpad

## Scope

Targeted legacy Python modules under `src/bigclaw` for this lane:

- `src/bigclaw/connectors.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/validation_policy.py`

Chosen consolidation destination:

- `src/bigclaw/models.py`

Current repository `src/bigclaw` Python file count before this lane: `45`

## Plan

1. Confirm the lane-owned modules, their import sites, and the targeted tests that exercise them.
2. Move the lightweight connector, issue-mapping, and validation-policy implementations into `src/bigclaw/models.py`.
3. Preserve the legacy import surface for `bigclaw.connectors`, `bigclaw.mapping`, and `bigclaw.validation_policy` via package compatibility modules.
4. Delete the superseded standalone module files once the compatibility surface is in place.
5. Run targeted validation for the touched surfaces and record the exact commands and results here.
6. Report the direct file disposition and the before/after Python file counts.
7. Commit and push the scoped change set.

## Acceptance

- Produce the exact file list owned by `BIG-GO-973`.
- Reduce the number of Python files in the targeted `src/bigclaw` surface.
- Preserve import compatibility for `bigclaw.connectors`, `bigclaw.mapping`, and `bigclaw.validation_policy`.
- Record delete/replace/retain reasoning for each targeted legacy file.
- Report before/after `src/bigclaw` Python file counts.

## Validation

- Targeted import smoke checks for the legacy module names after consolidation.
- Targeted tests for connectors, mapping, validation policy, and the shared models surface.
- `git status --short` to confirm the change set stays scoped to this lane.

## Notes

- The targeted modules are low-coupling leaf modules with direct, existing tests.
- Validation should use the repository Python environment already available in this checkout.

## Results

### File Disposition

- `src/bigclaw/models.py`
  - Retained and expanded.
  - Reason: became the implementation home for the lightweight connector, mapping, and validation-policy surfaces.
- `src/bigclaw/connectors.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/models.py`; `bigclaw.connectors` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/mapping.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/models.py`; `bigclaw.mapping` import compatibility is now provided from `src/bigclaw/__init__.py`.
- `src/bigclaw/validation_policy.py`
  - Deleted.
  - Reason: implementation moved into `src/bigclaw/models.py`; `bigclaw.validation_policy` import compatibility is now provided from `src/bigclaw/__init__.py`.

### Python File Count Impact

- Repository `src/bigclaw` Python files before: `45`
- Repository `src/bigclaw` Python files after: `42`
- Net reduction: `3`

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `PYTHONPATH=src python3 - <<'PY' ...`
  - Result: success; verified `bigclaw.connectors`, `bigclaw.mapping`, and `bigclaw.validation_policy` still import and resolve the consolidated exports.
- `PYTHONPATH=src python3 -m pytest tests/test_models.py tests/test_connectors.py tests/test_mapping.py tests/test_validation_policy.py`
  - Result: `9 passed in 0.07s`
- `git status --short`
  - Result: only `.symphony/workpad.md`, `src/bigclaw/__init__.py`, `src/bigclaw/models.py`, and the three deleted target modules are in the change set.
