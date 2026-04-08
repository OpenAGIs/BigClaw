Issue: BIG-GO-114

Plan
- Audit the remaining root operator wrappers and helper references under `scripts/ops/`, repo docs, and regression tests.
- Remove the redundant helper wrappers that only proxy into `scripts/ops/bigclawctl`.
- Update docs and regression coverage so `bash scripts/ops/bigclawctl ...` is the only supported root operator path.
- Run targeted validation commands, capture exact commands and results, then commit and push.

Acceptance
- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and `scripts/ops/bigclaw-symphony` are removed.
- Supported docs no longer describe those wrappers as retained compatibility entrypoints.
- Targeted regression tests assert the wrapper removal and direct `bigclawctl` guidance.
- Targeted validation commands complete successfully and their exact commands/results are recorded.

Validation
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootOpsDirectoryStaysPythonFree|TestRootOpsMigrationDocsListOnlyGoEntrypoints'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRun(Issue|Panel|Symphony)'`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl panel --help`
- `bash scripts/ops/bigclawctl symphony --help`

Results
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootOpsDirectory(StaysPythonFree|RetiresRedundantHelperWrappers)|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestTopLevelModulePurgeTranche16|TestBIGGO(100|1353|1370|1371|1374|1376|1380|1382|1383|1385|1392|1393|1396|1406|1410|1419|1422|1424|1425|1426|1427|1430|1433|1436|1439|1454)(GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  - `ok  	bigclaw-go/internal/regression	0.187s`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRun(GitHubSyncHelpPrintsUsageAndExitsZero|WorkspaceHelpPrintsUsageAndExitsZero|CreateIssuesHelpPrintsUsageAndExitsZero|DevSmokeHelpPrintsUsageAndExitsZero|SymphonyHelpPrintsUsageAndExitsZero|IssueHelpPrintsUsageAndExitsZero|PanelHelpPrintsUsageAndExitsZero|IssueRoutesStateShortcutToLocalIssues|PanelUsesSymphonyFromPATH)$'`
  - `ok  	bigclaw-go/cmd/bigclawctl	1.090s`
- `bash scripts/ops/bigclawctl issue --help`
  - `usage: bigclawctl issue [flags] [args...]`
- `bash scripts/ops/bigclawctl panel --help`
  - `usage: bigclawctl panel [flags] [args...]`
- `bash scripts/ops/bigclawctl symphony --help`
  - `usage: bigclawctl symphony [flags] [args...]`
