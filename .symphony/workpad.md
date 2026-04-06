# BIG-GO-1511

## Plan
- Capture baseline Python file counts for the repository and `src/bigclaw`.
- Remove physically deletable Python modules from `src/bigclaw` that are no longer consumed by the active compatibility surface.
- Trim `src/bigclaw/__init__.py` so package imports do not reference deleted modules.
- Run targeted validation on the retained Python entrypoints and record exact commands plus results.
- Commit and push the scoped deletion-first change set.

## Acceptance
- The repository-wide `*.py` count decreases from the baseline.
- The `src/bigclaw` `*.py` count decreases from the baseline.
- Deleted-file evidence shows actual removals under `src/bigclaw`.
- Remaining supported Python entrypoints still import/execute for the targeted checks.
- Changes stay scoped to this issue.

## Validation
- `rg --files -g '*.py' | wc -l`
- `rg --files src/bigclaw -g '*.py' | wc -l`
- `git diff --name-status --diff-filter=D`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/audit_events.py src/bigclaw/collaboration.py src/bigclaw/deprecation.py src/bigclaw/models.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/risk.py src/bigclaw/run_detail.py src/bigclaw/runtime.py`
- `PYTHONPATH=src python3 -m bigclaw --help`
- `PYTHONPATH=src python3 - <<'PY' ... import bigclaw ... PY`

## Validation Results
- `rg --files -g '*.py' | wc -l` before -> `23`
- `rg --files src/bigclaw -g '*.py' | wc -l` before -> `19`
- `rg --files -g '*.py' | wc -l` after -> `16`
- `rg --files src/bigclaw -g '*.py' | wc -l` after -> `12`
- `git diff --name-status --diff-filter=D` -> deleted `src/bigclaw/console_ia.py`, `src/bigclaw/design_system.py`, `src/bigclaw/evaluation.py`, `src/bigclaw/governance.py`, `src/bigclaw/operations.py`, `src/bigclaw/planning.py`, and `src/bigclaw/ui_review.py`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/audit_events.py src/bigclaw/collaboration.py src/bigclaw/deprecation.py src/bigclaw/models.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/risk.py src/bigclaw/run_detail.py src/bigclaw/runtime.py` -> success
- `PYTHONPATH=src python3 -m bigclaw --help` -> success; emitted the expected migration-only deprecation warning and printed CLI help
- `PYTHONPATH=src python3 - <<'PY' ... import bigclaw ... PY` -> `import_ok True True`
