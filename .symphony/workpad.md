## BIG-GO-1048

### Plan
- Confirm the remaining governance/reporting/observability Python top-level modules targeted by this tranche and the existing Go replacements under `bigclaw-go/internal`.
- Delete the scoped Python module files for governance, reporting, and observability, plus their direct Python test files so the repository Python file count drops materially within this issue scope.
- Add or extend Go regression coverage to assert the deleted Python files are absent and the corresponding Go implementation and Go tests exist.
- Run targeted validation for the new regression coverage and record exact commands and results.
- Commit the scoped change with a message that lists deleted Python files and added Go regression coverage, then push the branch.

### Acceptance
- Python file count in the repository decreases versus the pre-change count.
- `src/bigclaw/governance.py`, `src/bigclaw/reports.py`, and `src/bigclaw/observability.py` are removed.
- Direct Python tests covering those modules are removed in the same tranche.
- Go regression coverage exists for this tranche and points at the Go replacements in `bigclaw-go/internal/governance`, `bigclaw-go/internal/reporting`, and `bigclaw-go/internal/observability`.

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
