# BIG-GO-1515 Workpad

## Plan
1. Confirm the checked-out repository baseline and repair the local branch workspace if needed.
2. Measure the repository-wide physical `.py` inventory and a reporting/observability-focused inventory for `reports`, `bigclaw-go/docs/reports`, `bigclaw-go/internal/reporting`, and `bigclaw-go/internal/observability`.
3. If residual reporting/observability Python files exist, delete them and capture an exact deleted-file ledger.
4. If the baseline is already Python-free, record the exact blocker state with before/after counts and a deleted-file ledger of `none`, without widening scope beyond this issue.
5. Add a targeted regression guard and issue-specific validation artifacts for the verified reporting/observability Python-free baseline.
6. Run targeted validation, record exact commands and results, then commit and push `BIG-GO-1515`.

## Acceptance
- The issue artifacts record the exact repository-wide before and after physical `.py` file counts.
- The reporting/observability scope is explicitly inventoried.
- The deleted-file ledger is exact, even if the result is `none` because the baseline was already `0`.
- Targeted validation commands and their exact results are recorded in-repo.
- Changes stay scoped to `BIG-GO-1515`, and the branch is committed and pushed.

## Validation
- `find /tmp/BIG-GO-1515-clone -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /tmp/BIG-GO-1515-clone/reports /tmp/BIG-GO-1515-clone/bigclaw-go/docs/reports /tmp/BIG-GO-1515-clone/bigclaw-go/internal/reporting /tmp/BIG-GO-1515-clone/bigclaw-go/internal/observability -type f -name '*.py' 2>/dev/null | sort`
- `git diff --name-status --diff-filter=D`
- `cd /tmp/BIG-GO-1515-clone/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1515(RepositoryPythonInventoryStaysZero|ReportingAndObservabilityPathsStayPythonFree|LedgerCapturesBlockedDeletionState)$'`
