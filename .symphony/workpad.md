# BIG-GO-1128

## Plan
- Inventory candidate Python files in scope and confirm current repository Python count.
- Identify existing Go replacements or reusable Go entrypoints for the benchmark and e2e script surfaces covered by this lane.
- Replace selected real Python scripts/tests with Go implementations or remove obsolete Python assets once equivalent Go coverage exists.
- Run targeted validation for changed Go packages and record exact commands and outcomes.
- Verify repository Python file count decreases, then commit and push the branch.

## Acceptance
- Real Python files in this lane are removed or replaced with Go-backed equivalents.
- The affected benchmark/e2e flows still have a Go execution path or compatible test coverage.
- The issue candidate file list is explicitly locked by regression coverage so those Python paths cannot silently return.
- `find . -name '*.py' | wc -l` decreases relative to the starting count.

## Validation
- Baseline: `find . -name '*.py' | wc -l`
- Targeted discovery: inspect candidate files and adjacent Go packages/tests.
- Targeted tests: run `go test` for each changed package or command surface.
- Final verification: rerun `find . -name '*.py' | wc -l`, inspect git diff, commit, and push.

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLane8CandidatePythonFilesStayRemoved|TestBenchmarkScriptDirectoryStaysPythonFree|TestMigrationScriptDirectoryStaysPythonFree|TestRepoRootScriptDirectoryStaysPythonFree|TestScriptMigrationPlanListsOnlyActiveGoEntrypoints|TestE2EScriptDirectoryStaysPythonFree|TestRootOpsDirectoryStaysPythonFree'` -> `ok  	bigclaw-go/internal/regression	0.481s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBenchmarkScriptDirectoryStaysPythonFree|TestMigrationScriptDirectoryStaysPythonFree|TestRepoRootScriptDirectoryStaysPythonFree|TestScriptMigrationPlanListsOnlyActiveGoEntrypoints|TestE2EScriptDirectoryStaysPythonFree|TestRootOpsDirectoryStaysPythonFree'` -> `ok  	bigclaw-go/internal/regression	0.448s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestBenchmarkScriptsStayGoOnly|TestRunDevSmokeHelp|TestRunCreateIssuesHelp|TestRunDevSmoke|TestRunCreateIssuesSkipsExistingAndCreatesMissing'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.643s`
- final `find . -name '*.py' | wc -l` -> `0`

## Blockers
- The repository already materialized to a zero-`.py` baseline before this issue branch started, so the acceptance target to make the raw Python-file count numerically decrease is not achievable from the current state. This change hardens the migrated benchmark, migration, and repo-root script surfaces against Python reintroduction instead.
