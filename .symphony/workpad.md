# BIG-GO-1168 Workpad

## Plan
- Confirm the current repository baseline for physical Python files and the listed lane candidates.
- Inspect existing Go migration coverage so this issue adds only the missing `BIG-GO-1168` enforcement instead of duplicating older sweeps.
- Update migration docs and regression tests so the lane candidate paths stay deleted and their Go replacements remain the documented operator path.
- Run targeted validation, capture exact commands and results, then commit and push the branch.

## Acceptance
- The `BIG-GO-1168` candidate Python assets remain absent from the repository.
- The benchmark, e2e, migration, and root helper replacements remain explicitly mapped to Go-compatible entrypoints.
- `find . -name '*.py' | wc -l` is revalidated on this branch and recorded exactly, with the current zero-file baseline called out as a constraint.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1168CandidatePythonFilesRemainDeleted|BIGGO1168MigrationDocsListGoReplacements|BIGGO1168RepositoryPythonCountRemainsZero)$'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(AutomationUsageListsBIGGO1168GoReplacements|BenchmarkScriptsStayGoOnly|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1168CandidatePythonFilesRemainDeleted|BIGGO1168MigrationDocsListGoReplacements|BIGGO1168RepositoryPythonCountRemainsZero)$'` -> `ok  	bigclaw-go/internal/regression	0.484s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(AutomationUsageListsBIGGO1168GoReplacements|BenchmarkScriptsStayGoOnly|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.856s`
- `git status --short` -> pending until commit is created

## Constraint
- The workspace already starts from a zero-`.py` baseline, so this issue can harden and document the residual sweep but cannot make the Python count numerically lower from the observed starting state.
