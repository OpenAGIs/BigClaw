# BIG-GO-1166 Workpad

## Plan
- Confirm the repository baseline for physical Python files and verify whether the BIG-GO-1166 candidate paths are already absent on this branch.
- Add issue-scoped regression coverage that locks the BIG-GO-1166 candidate Python paths to the deleted state and pins the supported Go replacements in migration docs and CLI help coverage.
- Run targeted validation commands, capture exact commands and results, then commit and push `symphony/BIG-GO-1166`.

## Acceptance
- The BIG-GO-1166 real Python candidate assets are covered and verified absent from the repository.
- The Go replacement or compatibility paths for the BIG-GO-1166 candidate surfaces are explicitly documented and tested.
- `find . -name '*.py' | wc -l` is validated on the branch and the result is recorded, including the fact that the current baseline is already zero.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1166CandidatePythonFilesRemainDeleted|BIGGO1166MigrationDocsListGoReplacements|E2EScriptDirectoryStaysPythonFree|E2EMigrationDocListsOnlyActiveEntrypoints|RootOpsDirectoryStaysPythonFree|RootOpsMigrationDocsListOnlyGoEntrypoints)$'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunAutomationRunTaskSmokeJSONOutput|AutomationSoakLocalWritesReport|AutomationShadowCompareDetectsMismatch|AutomationShadowMatrixBuildsCorpusCoverage|AutomationLiveShadowScorecardBuildsReport|AutomationExportLiveShadowBundleBuildsManifest|AutomationBenchmarkRunMatrixBuildsReport|AutomationBenchmarkCapacityCertificationBuildsReport|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1166CandidatePythonFilesRemainDeleted|BIGGO1166MigrationDocsListGoReplacements|E2EScriptDirectoryStaysPythonFree|E2EMigrationDocListsOnlyActiveEntrypoints|RootOpsDirectoryStaysPythonFree|RootOpsMigrationDocsListOnlyGoEntrypoints)$'` -> `ok  	bigclaw-go/internal/regression	0.548s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|AutomationUsageListsBIGGO1166GoReplacements|RunAutomationRunTaskSmokeJSONOutput|AutomationSoakLocalWritesReport|AutomationShadowCompareDetectsMismatch|AutomationShadowMatrixBuildsCorpusCoverage|AutomationLiveShadowScorecardBuildsReport|AutomationExportLiveShadowBundleBuildsManifest|AutomationBenchmarkRunMatrixBuildsReport|AutomationBenchmarkCapacityCertificationBuildsReport|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.706s`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`, `bigclaw-go/docs/go-cli-script-migration.md`, `docs/go-cli-script-migration-plan.md`, `bigclaw-go/internal/regression/big_go_1166_script_migration_test.go`

## Residual Risk
- The branch baseline already has zero tracked `.py` files, so this issue can harden deletion coverage and Go replacement guidance but cannot reduce the Python file count below zero from the current starting point.
