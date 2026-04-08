# BIG-GO-114 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-114`

Title: `Residual scripts Python sweep G`

This lane removed the residual root operator helper wrappers that only re-dispatched
into `scripts/ops/bigclawctl`, then refreshed the active docs and regression coverage
so the supported root command surface is the direct Go-backed `bigclawctl` entrypoint.

## Delivered

- deleted redundant root helper wrappers:
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
- updated root operator docs so the active path is:
  - `bash scripts/ops/bigclawctl issue ...`
  - `bash scripts/ops/bigclawctl panel ...`
  - `bash scripts/ops/bigclawctl symphony ...`
- added direct CLI help-path coverage in `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`
- updated root-ops and zero-python regression checks so the deleted helper wrappers stay absent

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootOpsDirectory(StaysPythonFree|RetiresRedundantHelperWrappers)|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestTopLevelModulePurgeTranche16|TestBIGGO(100|1353|1370|1371|1374|1376|1380|1382|1383|1385|1392|1393|1396|1406|1410|1419|1422|1424|1425|1426|1427|1430|1433|1436|1439|1454)(GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRun(GitHubSyncHelpPrintsUsageAndExitsZero|WorkspaceHelpPrintsUsageAndExitsZero|CreateIssuesHelpPrintsUsageAndExitsZero|DevSmokeHelpPrintsUsageAndExitsZero|SymphonyHelpPrintsUsageAndExitsZero|IssueHelpPrintsUsageAndExitsZero|PanelHelpPrintsUsageAndExitsZero|IssueRoutesStateShortcutToLocalIssues|PanelUsesSymphonyFromPATH)$'`
- `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl issue --help`
- `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl panel --help`
- `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl symphony --help`

## Validation Results

### Targeted regression suite

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/bigclaw-go && go test -count=1 ./internal/regression -run 'TestRootOpsDirectory(StaysPythonFree|RetiresRedundantHelperWrappers)|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestTopLevelModulePurgeTranche16|TestBIGGO(100|1353|1370|1371|1374|1376|1380|1382|1383|1385|1392|1393|1396|1406|1410|1419|1422|1424|1425|1426|1427|1430|1433|1436|1439|1454)(GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.187s
```

### Targeted CLI tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRun(GitHubSyncHelpPrintsUsageAndExitsZero|WorkspaceHelpPrintsUsageAndExitsZero|CreateIssuesHelpPrintsUsageAndExitsZero|DevSmokeHelpPrintsUsageAndExitsZero|SymphonyHelpPrintsUsageAndExitsZero|IssueHelpPrintsUsageAndExitsZero|PanelHelpPrintsUsageAndExitsZero|IssueRoutesStateShortcutToLocalIssues|PanelUsesSymphonyFromPATH)$'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	1.090s
```

### Root operator help checks

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl issue --help
```

Result: exit code `0`, first line `usage: bigclawctl issue [flags] [args...]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl panel --help
```

Result: exit code `0`, first line `usage: bigclawctl panel [flags] [args...]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-114/scripts/ops/bigclawctl symphony --help
```

Result: exit code `0`, first line `usage: bigclawctl symphony [flags] [args...]`

## Git

- Branch: `BIG-GO-114`
- Primary implementation commit: `e4271be6` (`BIG-GO-114: retire residual ops helper wrappers`)
- Push target: `origin/BIG-GO-114`

## Residual Risk

- Historical validation and status artifacts in `reports/` and `bigclaw-go/docs/reports/`
  still mention the retired wrapper names because they describe earlier lane baselines rather
  than the active operator contract.
