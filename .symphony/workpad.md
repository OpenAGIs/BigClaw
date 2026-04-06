# BIG-GO-1553 Workpad

## Plan

1. Reconfirm the current physical `.py` inventory for the repository and the
   focused `bigclaw-go/scripts` surface, then identify the exact historical
   baseline commit and deleted-file ledger that produced the current zero-file
   state.
2. Add lane-scoped artifacts that record the before/after count delta, the
   exact `bigclaw-go/scripts` deleted-file evidence, and the replacement
   entrypoints that now own the migrated behavior.
3. Add focused regression coverage for repository-wide zero Python files and
   for the `bigclaw-go/scripts` surface so the documented count delta and
   replacement surface remain pinned.
4. Run the targeted validation commands, record exact commands plus results,
   then commit and push the `BIG-GO-1553` branch.

## Acceptance

- The lane records the current repository-wide physical `.py` count.
- The lane records the exact historical `bigclaw-go/scripts` `.py` baseline and
  the current `0`-file state.
- The lane records the exact `bigclaw-go/scripts` deleted-file ledger with
  commit evidence.
- The lane records the exact before/after count delta for
  `bigclaw-go/scripts`.
- The lane names the active replacement entrypoints for the retired Python
  scripts.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The branch is committed and pushed to `origin/BIG-GO-1553`.

## Validation

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find bigclaw-go/scripts -type f -name '*.py' | sort`
- `git ls-tree -r --name-only fdb20c43 bigclaw-go/scripts | rg '\.py$'`
- `git log --diff-filter=D --summary -- bigclaw-go/scripts`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1553(RepositoryHasNoPythonFiles|BigclawGoScriptsStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesExactDeltaAndLedger)$'`

## GitHub

- Branch target: `origin/BIG-GO-1553`
- Compare view: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-1553?expand=1`
