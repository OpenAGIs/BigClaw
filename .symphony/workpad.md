## Plan

1. Delete `src/bigclaw/run_detail.py` by moving its retained dataclasses and HTML rendering helpers into `src/bigclaw/reports.py`.
2. Repoint retained Python callers in `src/bigclaw/evaluation.py` and package compatibility wiring in `src/bigclaw/__init__.py` so the deleted module path is no longer required.
3. Add a tranche-specific Go regression test proving `src/bigclaw/run_detail.py` is absent and its Go replacement files remain present.
4. Run focused Python and Go validation plus the repository Python file count check.
5. Commit the scoped changes with the deleted Python file and added Go test file called out explicitly, then push to `origin/BIG-GO-1041`.

## Acceptance

- `src/bigclaw/run_detail.py` is deleted.
- `src/bigclaw/reports.py` owns the `RunDetail*` dataclasses plus `render_run_detail_console`, `render_resource_grid`, and `render_timeline_panel`.
- `src/bigclaw/evaluation.py` and `src/bigclaw/__init__.py` no longer import from `src/bigclaw/run_detail.py`.
- `bigclaw.run_detail` continues to resolve through package compatibility wiring if imported.
- `bigclaw-go/internal/regression/top_level_module_purge_tranche18_test.go` asserts the deleted Python file and replacement Go files.
- `find . -name '*.py' | wc -l` drops below the current baseline of `41`.
- Focused Python and Go tests pass.
- The branch is committed and pushed to `origin/BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_evaluation.py -q`
- `cd bigclaw-go && go test ./internal/product ./internal/regression -run 'TestDashboardRunContractAuditFlagsMissingRequiredFields|TestTopLevelModulePurgeTranche(1|2|3|4|5|6|7|8|9|10|11|12|13|14|15|16|17|18)'`
- `git status --short`
- `git log -1 --stat`
