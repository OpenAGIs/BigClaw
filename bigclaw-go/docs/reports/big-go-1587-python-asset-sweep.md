# BIG-GO-1587 Python Asset Sweep

## Scope

Strict bucket lane `BIG-GO-1587` hardens the `bigclaw-go/scripts/migration`
Python bucket. In this checkout the directory is already absent, so the lane
lands as regression-prevention evidence rather than an in-branch deletion of a
live `.py` file.

## Bucket State

Repository-wide Python file count: `0`.

- `bigclaw-go`: `0` Python files
- `bigclaw-go/scripts/migration`: `0` Python files

Directory absent on disk: `yes`.

This keeps the physical `.py` count for the target bucket at `0` while locking
the missing directory in place.

## Go Migration Replacement Paths

The active repo-native migration replacement surface remains:

- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `bigclaw-go/docs/go-cli-script-migration.md`
- `docs/go-cli-script-migration-plan.md`

These paths cover the `bigclawctl automation migration
shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`
replacement workflow for the retired Python bucket.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the target migration bucket contained `0` Python files.
- `test ! -d bigclaw-go/scripts/migration`
  Result: `absent`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1587(RepositoryHasNoPythonFiles|MigrationBucketStaysAbsentAndPythonFree|GoMigrationReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.183s`
