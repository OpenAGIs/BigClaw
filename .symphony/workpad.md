## BIG-GO-1048

### Plan
- Confirm the remaining governance/reporting/observability Python top-level modules targeted by this tranche and the existing Go replacements under `bigclaw-go/internal`.
- Delete the scoped Python module files for governance, reporting, and observability, plus their direct Python test files so the repository Python file count drops materially within this issue scope.
- Add or extend Go regression coverage to assert the deleted Python files are absent and the corresponding Go implementation and Go tests exist.
- Delete the now-dead legacy Python compatibility/runtime files that still import the purged governance/reporting/observability modules, plus their direct Python tests.
- Extend Go regression coverage for the second purge slice and keep exact validation evidence recorded here.
- Run targeted validation for the new regression coverage and record exact commands and results.
- Commit the scoped change with a message that lists deleted Python files and added Go regression coverage, then push the branch.

### Acceptance
- Python file count in the repository decreases versus the pre-change count.
- `src/bigclaw/governance.py`, `src/bigclaw/reports.py`, and `src/bigclaw/observability.py` are removed.
- Direct Python tests covering those modules are removed in the same tranche.
- Go regression coverage exists for this tranche and points at the Go replacements in `bigclaw-go/internal/governance`, `bigclaw-go/internal/reporting`, and `bigclaw-go/internal/observability`.
- Legacy Python compatibility/runtime files that import the removed governance/reporting/observability modules are also purged so the repo no longer carries dead-on-import Python entrypoints in this slice.

### Validation
- `find . -name "*.py" | wc -l`
- `go test ./internal/regression -run 'TestTopLevelModulePurgeTranche4|TestTopLevelModulePurgePythonCountDrops'`
- `git status --short`

### Notes
- Keep the change limited to the governance/reporting/observability purge tranche; do not broaden into unrelated Python cleanup.

### Results
- Pre-change Python file count: `find . -name "*.py" | wc -l` -> `66`
- Post-change Python file count: `find . -name "*.py" | wc -l` -> `61`
- Targeted Go regression: `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche4|TestTopLevelModulePurgePythonCountDrops'` -> `ok  	bigclaw-go/internal/regression	0.517s`
- Second-slice Python file count: `find . -name "*.py" | wc -l` -> `39`
- Source/test import sweep: `rg -n "bigclaw\.(governance|observability|reports)|from \.?(governance|observability|reports)|from bigclaw\.(governance|observability|reports)|import bigclaw\.(governance|observability|reports)" src tests` -> no matches
- Expanded Go regression first run: `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche4|TestTopLevelModulePurgePythonCountDrops|TestTopLevelModulePurgeTranche5|TestTopLevelModulePurgePythonCountDropsAfterTranche5'` -> `FAIL` because tranche 4 still pinned the intermediate count `61`
- Expanded Go regression final run: `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche4|TestTopLevelModulePurgePythonCountDrops|TestTopLevelModulePurgeTranche5|TestTopLevelModulePurgePythonCountDropsAfterTranche5'` -> `ok  	bigclaw-go/internal/regression	0.474s`
