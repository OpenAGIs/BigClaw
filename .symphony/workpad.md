Issue: BIG-GO-20

Plan
- Confirm the repository remains physically Python-free and identify the live documentation residue still in scope for this batch.
- Update `docs/symphony-repo-bootstrap-template.md` and `docs/go-mainline-cutover-handoff.md` so they reflect only Go-native bootstrap and validation guidance.
- Keep the lane-scoped evidence bundle aligned with that doc sweep:
  - `bigclaw-go/internal/regression/big_go_20_zero_python_guard_test.go`
  - `bigclaw-go/docs/reports/big-go-20-python-asset-sweep.md`
  - `reports/BIG-GO-20-status.json`
  - `reports/BIG-GO-20-validation.md`
- Run targeted validation, record exact commands and outcomes, then commit and push the branch.

Acceptance
- The repository-wide physical Python file count remains zero.
- The two live docs in scope no longer instruct users to rely on retired Python bootstrap assets or Python validation commands.
- The BIG-GO-20 regression/report artifacts capture the final Go-only sweep state and targeted validation evidence.
- The scoped changes are committed and pushed to `origin/BIG-GO-20`.

Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-20 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-20/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO20(RepositoryHasNoPythonFiles|LiveDocsRemainGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

Execution Notes
- 2026-04-11: Repository inspection confirmed the only BIG-GO-20 scoped work is the final live documentation sweep plus its lane-specific regression/report artifacts.
- 2026-04-11: `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-20 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` produced no output, confirming the repository-wide physical Python file count remained zero.
- 2026-04-11: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-20/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO20(RepositoryHasNoPythonFiles|LiveDocsRemainGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'` returned `ok  	bigclaw-go/internal/regression	3.223s`.
