# BIG-GO-1160 Workpad

## Plan
- Verify the current repository baseline for physical Python files and identify whether the candidate lane assets were already removed.
- Inspect existing Go automation commands, regression tests, and migration docs for the benchmark, e2e, migration, and root script candidates listed in BIG-GO-1160.
- Add scoped regression and migration-document coverage that pins the retired candidate `.py` entrypoints to concrete Go replacements and keeps the relevant script surfaces Python-free.
- Run targeted validation commands, capture exact command lines and results, then commit and push the branch.

## Acceptance
- Real Python candidate assets covered by this lane remain absent from the repository.
- Go replacement or compatibility paths are explicitly validated for the covered benchmark, e2e, migration, and root helper surfaces.
- `find . -name '*.py' | wc -l` is validated and documented from the current branch baseline.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(E2EScriptDirectoryStaysPythonFree|E2EMigrationDocListsOnlyActiveEntrypoints|RootOpsDirectoryStaysPythonFree|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunAutomationRunTaskSmokeJSONOutput|AutomationSoakLocalWritesReport|AutomationShadowCompareDetectsMismatch|AutomationShadowMatrixBuildsCorpusCoverage|AutomationLiveShadowScorecardBuildsReport|AutomationExportLiveShadowBundleBuildsManifest|AutomationBenchmarkRunMatrixBuildsReport|AutomationBenchmarkCapacityCertificationBuildsReport|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(E2EScriptDirectoryStaysPythonFree|E2EMigrationDocListsOnlyActiveEntrypoints|RootOpsDirectoryStaysPythonFree|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'` -> `ok  	bigclaw-go/internal/regression	0.429s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunAutomationRunTaskSmokeJSONOutput|AutomationSoakLocalWritesReport|AutomationShadowCompareDetectsMismatch|AutomationShadowMatrixBuildsCorpusCoverage|AutomationLiveShadowScorecardBuildsReport|AutomationExportLiveShadowBundleBuildsManifest|AutomationBenchmarkRunMatrixBuildsReport|AutomationBenchmarkCapacityCertificationBuildsReport|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.589s`
- `git status --short` -> clean after commit

## Residual Risk
- The repository already starts from a zero-`.py` baseline in this workspace, so this issue can only harden deletion enforcement and Go replacement coverage for the lane; it cannot make the Python file count numerically lower from the current baseline.
