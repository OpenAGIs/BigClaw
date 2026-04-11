# BIG-GO-235 Workpad

## Plan

1. Remove stale Python-first tooling guidance from the active bootstrap and
   cutover docs while preserving the historical migration context.
2. Add a lane-specific regression guard that locks the audited docs to Go-only
   tooling guidance and records the scoped sweep evidence.
3. Run the targeted repository and regression validations, then commit and push
   the `BIG-GO-235` branch tip.

## Acceptance

- `.symphony/workpad.md` reflects `BIG-GO-235` and this tooling-doc sweep.
- `docs/symphony-repo-bootstrap-template.md` and
  `docs/go-mainline-cutover-handoff.md` stop prescribing Python bootstrap or
  validation commands for the live Go-only workflow.
- A new `BIG-GO-235` regression guard passes and keeps the scoped docs aligned
  with the Go-only tooling posture.
- Exact validation commands and results are recorded, and the final lane tip is
  committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-235 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-235/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-235/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-235/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-235/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-235/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO235(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ToolingDocsStayGoOnly)$'`

## Execution Notes

- 2026-04-12: Repository-wide physical Python file count is already `0`; this
  lane is focused on removing residual Python tooling guidance from active docs
  and locking the Go-only baseline with regression coverage.
