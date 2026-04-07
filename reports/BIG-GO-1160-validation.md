# BIG-GO-1160 Validation

Date: 2026-04-04

## Scope

Issue: `BIG-GO-1160`

Title: `physical Python residual sweep 10`

This lane hardens the already-retired benchmark, e2e, migration, and repo-root
Python candidate set by locking the files to absent-on-disk and verifying that
the operator-facing Go replacements remain documented and test-covered.

The checked-out branch baseline was already at a repository-wide Python count of
`0`, so this lane could not reduce the physical `.py` count numerically. The
implemented work instead adds regression coverage and migration evidence so the
candidate paths cannot quietly reappear.

## Delivered

- Added `bigclaw-go/internal/regression/big_go_1160_script_migration_test.go`
  to enforce that the full BIG-GO-1160 candidate `.py` set remains deleted and
  that the migration docs still advertise the supported Go replacements.
- Extended `bigclaw-go/cmd/bigclawctl/automation_commands_test.go` to verify
  benchmark, e2e, and migration automation help surfaces for the replacement
  subcommands.
- Updated `bigclaw-go/docs/go-cli-script-migration.md` and
  `docs/go-cli-script-migration-plan.md` with BIG-GO-1160 sweep coverage and
  replacement guidance.
- Refreshed `.symphony/workpad.md` with the lane-scoped plan, acceptance, and
  exact validation results.

## Validation

### Python count baseline

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1160 -name '*.py' | wc -l
```

Result:

```text
0
```

### Targeted regression coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1160/bigclaw-go && go test ./internal/regression -run 'Test(E2EScriptDirectoryStaysPythonFree|E2EMigrationDocListsOnlyActiveEntrypoints|RootOpsDirectoryStaysPythonFree|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.429s
```

### Targeted Go command-surface coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1160/bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunAutomationRunTaskSmokeJSONOutput|AutomationSoakLocalWritesReport|AutomationShadowCompareDetectsMismatch|AutomationShadowMatrixBuildsCorpusCoverage|AutomationLiveShadowScorecardBuildsReport|AutomationExportLiveShadowBundleBuildsManifest|AutomationBenchmarkRunMatrixBuildsReport|AutomationBenchmarkCapacityCertificationBuildsReport|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	0.589s
```

## Git

- Commit: `b0f68824` (`BIG-GO-1160 harden python sweep regressions`)
- Push: `git push origin main` -> success

## Residual Risk

- This workspace already started at `find . -name '*.py' | wc -l = 0`, so the
  lane could only harden deletion enforcement and Go replacement coverage for
  the candidate set rather than numerically decrease an already-zero Python
  count.
