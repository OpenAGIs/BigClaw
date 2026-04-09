# BIG-GO-172 Workpad

## Plan

1. Audit the remaining retired Python test-contract surfaces that were not yet
called out by the earlier residual-test sweeps and identify the corresponding
Go-owned replacement directories.
2. Add lane-specific regression coverage for `BIG-GO-172` that locks those
remaining test-heavy replacement directories at zero Python files while
asserting the representative Go/native replacement paths still exist.
3. Add the matching lane report plus `reports/BIG-GO-172-{validation,status}`
artifacts, run targeted validation, record exact commands and results, then
commit and push the scoped change set.

## Acceptance

- `BIG-GO-172` adds lane-specific regression coverage for the remaining retired
  Python test-contract slice not already singled out by prior sweeps.
- The guard enforces that the targeted Go-owned replacement directories remain
  Python-free.
- The lane report and `reports/BIG-GO-172-{validation,status}` artifacts
  document the audited directories, representative retired Python test paths,
  retained Go/native replacements, and exact validation commands/results.
- The resulting change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-172 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/api /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/contract /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/events /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/githubsync /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/governance /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/observability /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/orchestrator /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/planning /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/policy /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/product /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/queue /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/repo /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-172/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO172(RepositoryHasNoPythonFiles|RemainingTestHeavyReplacementDirectoriesStayPythonFree|RepresentativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
