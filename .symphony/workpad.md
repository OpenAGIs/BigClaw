# BIG-GO-138 Workpad

## Plan

1. Inspect active repo docs, reports, and regression guards for stale Python-era operator guidance that still appears in the current mainline surface.
2. Update the scoped active documentation/reporting surface to describe Go/Bash entrypoints as the only supported path while preserving historical references only where they are explicitly marked retired.
3. Add or extend regression coverage for the sweep so the targeted stale Python guidance does not reappear unnoticed.
4. Run targeted inventory and regression validation, then commit and push the branch.

## Acceptance

- Active repo documentation touched by this issue does not present removed Python operator entrypoints as runnable current workflow.
- `BIG-GO-138` has a dedicated scoped report/guard artifact consistent with the existing Python-reduction lane pattern.
- Targeted regression coverage passes and validates the intended Go-first guidance.
- Validation commands and exact results are captured for closeout.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO138MigrationGuidancePrefersGoAutomation|BIGGO138LaneReportCapturesSweepState|LiveShadowBundleSummaryAndIndexStayAligned)$'`
