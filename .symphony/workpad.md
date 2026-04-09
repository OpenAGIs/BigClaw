# BIG-GO-185 Workpad

## Plan

1. Confirm the current residual Python tooling/build-helper/dev-utility bucket for this lane and identify the repo-native replacement surface already checked in.
2. Add a scoped regression test for `BIG-GO-185` that pins the targeted retired Python helper paths as absent and the replacement Go/shell entrypoints as present.
3. Add a matching sweep report under `bigclaw-go/docs/reports/` documenting scope, replacement paths, validation commands, and exact results.
4. Run targeted validation for the new regression coverage and record the exact commands and outcomes.
5. Commit the scoped changes and push the issue branch to the remote.

## Acceptance

- `BIG-GO-185` only touches the Python tooling/dev-helper residual bucket.
- A new regression test file covers the targeted retired Python paths plus their active replacements.
- A new report file captures the sweep state and validation evidence for `BIG-GO-185`.
- Targeted tests pass and the exact commands/results are recorded in the report and final handoff.
- Changes are committed and pushed to a remote branch for this issue.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find scripts bigclaw-go -type f \\( -name '*.py' -o -path 'bigclaw-go/go.mod' -o -path 'bigclaw-go/cmd/bigclawctl/main.go' -o -path 'bigclaw-go/cmd/bigclawctl/migration_commands.go' -o -path 'bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go' -o -path 'bigclaw-go/internal/githubsync/sync.go' -o -path 'bigclaw-go/internal/refill/queue.go' -o -path 'bigclaw-go/internal/bootstrap/bootstrap.go' -o -path 'scripts/ops/bigclawctl' -o -path 'scripts/dev_bootstrap.sh' \\) 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO185(ResidualPythonToolingPathsStayAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
