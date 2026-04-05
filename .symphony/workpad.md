# BIG-GO-1474 Workpad

## Plan

1. Reconfirm the repository-wide physical Python asset inventory, with explicit
   checks for `scripts/ops` and the broader priority directories
   `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Update the current source-of-truth migration docs so the deleted
   `scripts/ops/*_*.py` wrappers are treated as historical removals, not active
   Python assets, and pin the Go ownership for each replacement path.
3. Add a lane-scoped regression/report pair for `BIG-GO-1474` that proves the
   `scripts/ops` directory stays Python-free, the deleted wrapper paths stay
   absent, and the live Go or shell replacements remain available.
4. Run targeted validation, record the exact commands and results, then commit
   and push the branch.

## Acceptance

- The repository and `scripts/ops` inventories explicitly show no physical
  Python files in the checked-out branch.
- The deleted `scripts/ops/*_*.py` wrapper set is documented with replacement
  Go ownership or explicit delete conditions.
- The lane adds regression coverage that fails if Python-suffixed ops wrappers
  reappear or if the replacement paths disappear.
- Exact validation commands and outcomes are recorded in lane artifacts.
- The change is committed and pushed to the remote branch.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/scripts/ops -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1474(OpsDirectoryHasNoPythonFiles|DeletedPythonWrapperPathsStayAbsent|ActiveGoReplacementPathsRemainAvailable|DocsAndLaneReportPinDeletedWrapperState)$'`

## Execution Notes

- 2026-04-06: Baseline inventory on branch start shows the working tree already
  contains no physical `.py` files, including under `scripts/ops`.
- 2026-04-06: This lane is therefore scoped to current-doc cleanup and
  regression hardening around the deleted root `scripts/ops` Python wrappers.
- 2026-04-06: Updated `docs/go-cli-script-migration-plan.md` and
  `docs/go-mainline-cutover-issue-pack.md` so deleted `scripts/ops/*_*.py`
  wrappers are explicitly historical removals with pinned Go ownership.
- 2026-04-06: Added `bigclaw-go/internal/regression/big_go_1474_ops_wrapper_guard_test.go`,
  `bigclaw-go/docs/reports/big-go-1474-ops-wrapper-sweep.md`,
  `reports/BIG-GO-1474-validation.md`, and `reports/BIG-GO-1474-status.json`.
- 2026-04-06: Ran `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474 -path '*/.git' -prune -o -name '*.py' -type f -print | sort` and observed no output.
- 2026-04-06: Ran `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/scripts/ops -type f -name '*.py' -print | sort` and observed no output.
- 2026-04-06: Ran `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/src/bigclaw /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/tests /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort` and observed no output.
- 2026-04-06: Ran `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1474/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1474(OpsDirectoryHasNoPythonFiles|DeletedPythonWrapperPathsStayAbsent|ActiveGoReplacementPathsRemainAvailable|DocsAndLaneReportPinDeletedWrapperState)$'` and observed `ok  	bigclaw-go/internal/regression	1.549s`.
