# BIG-GO-1176 Workpad

## Plan
- Confirm the current repository baseline for physical Python files and inspect the scoped residual script surfaces under `scripts/` and `bigclaw-go/scripts/`.
- Add issue-specific regression coverage that keeps the remaining shell/Go entrypoint directories free of `.py` assets and pins those directories to concrete Go-native replacements.
- Refresh the migration documentation so `BIG-GO-1176` records the already-zero Python baseline as explicit replacement evidence instead of implying a fresh in-branch deletion.
- Run targeted validation commands, capture exact commands and results, then commit and push the branch.

## Acceptance
- `find . -name '*.py' | wc -l` is revalidated from the current branch baseline and remains `0`.
- The live residual script surfaces for this lane are covered by regression tests that fail if Python assets reappear under `scripts/` or `bigclaw-go/scripts/`.
- Migration docs explicitly tie `BIG-GO-1176` to the supported Go/shell replacements for the audited script surfaces.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1176ResidualScriptSurfacesStayPythonFree|BIGGO1176MigrationDocsCaptureGoOnlyResidualSweep|E2EScriptDirectoryStaysPythonFree|RootOpsDirectoryStaysPythonFree|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1176ResidualScriptSurfacesStayPythonFree|BIGGO1176MigrationDocsCaptureGoOnlyResidualSweep|E2EScriptDirectoryStaysPythonFree|RootOpsDirectoryStaysPythonFree|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'` -> `ok  	bigclaw-go/internal/regression	0.496s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.679s`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/docs/go-cli-script-migration.md`, `docs/go-cli-script-migration-plan.md`, and `bigclaw-go/internal/regression/big_go_1176_script_residual_sweep_test.go` modified/added before commit
