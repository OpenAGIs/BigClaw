Issue: BIG-GO-1030

Plan
- Fold the residual Python `audit_events` compatibility surface into `src/bigclaw/observability.py` and retire `src/bigclaw/audit_events.py`.
- Update internal imports to use the new owner module so package initialization does not depend on a deleted physical `audit_events.py` file.
- Keep `bigclaw.audit_events` import compatibility by installing a package-level legacy surface module from `src/bigclaw/__init__.py`.
- Merge the dedicated audit-event regression coverage into `tests/test_observability.py`, then delete `tests/test_audit_events.py`.
- Refresh directly coupled docs that still point at `src/bigclaw/audit_events.py` as a standalone residual Python asset.
- Re-run targeted observability/runtime/report validation, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/audit_events.py` and `tests/test_audit_events.py` are removed from the tree.
- `bigclaw.audit_events` imports still resolve through package compatibility shims.
- Canonical audit-event constants/specs and required-field validation still work from the migrated owner module.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_runtime_matrix.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nimport bigclaw.audit_events\nfrom bigclaw.observability import missing_required_fields\nprint(bigclaw.audit_events.AuditEventSpec.__name__)\nprint(missing_required_fields(\"execution.scheduler_decision\", {\"task_id\": \"t\", \"run_id\": \"r\", \"medium\": \"docker\"}))\nPY`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
