## Plan

1. Purge `src/bigclaw/audit_events.py` by moving the retained Python audit event constants, specs, and validation helpers into `src/bigclaw/observability.py`.
2. Keep the `bigclaw.audit_events` import path working by exporting that moved surface from `src/bigclaw/__init__.py` and installing a synthetic compatibility module there.
3. Repoint retained Python callers in `runtime.py` and `reports.py` to the observability-owned surface.
4. Add tranche 16 Go regression coverage proving the deleted Python file is gone and the Go observability replacement files exist.
5. Run focused Python and Go validation plus the repository Python file count check.
6. Commit with the deleted Python file and added Go test file explicitly listed, then push to `origin/BIG-GO-1041`.

## Acceptance

- `src/bigclaw/audit_events.py` is deleted.
- `src/bigclaw/observability.py` provides the retained Python audit-event surface previously owned by `audit_events.py`.
- `src/bigclaw/__init__.py` no longer imports from `src/bigclaw/audit_events.py`, and `import bigclaw.audit_events` still resolves through package compatibility wiring.
- Retained Python callers use the observability-owned audit-event surface.
- `bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go` asserts the Python deletion and Go observability replacement files.
- `find . -name '*.py' | wc -l` decreases from the current baseline of `43`.
- Focused Python and Go tests pass.
- Changes are committed and pushed to `origin/BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py -q`
- `cd bigclaw-go && go test ./internal/observability ./internal/regression -run 'TestP0AuditEventSpecsDefineRequiredOperationalEvents|TestTopLevelModulePurgeTranche(1|2|3|4|5|6|7|8|9|10|11|12|13|14|15|16)'`
- `git status --short`
- `git log -1 --stat`
