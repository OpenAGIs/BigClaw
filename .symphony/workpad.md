# BIG-GO-131 Workpad

## Context
- Issue: `BIG-GO-131`
- Title: `Residual src/bigclaw Python sweep J`
- Goal: record and harden a broad residual `src/bigclaw` control-center and reporting tranche after the repo reached a zero-physical-Python baseline.
- Current repo state on entry: repository-wide physical `.py` inventory is already `0`, and `src/bigclaw` is not present in this checkout.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_131_src_bigclaw_sweep_j_test.go`
- `bigclaw-go/docs/reports/big-go-131-src-bigclaw-sweep-j.md`
- `reports/BIG-GO-131-validation.md`
- `reports/BIG-GO-131-status.json`

## Plan
1. Replace the stale workpad with issue-specific scope, acceptance, and validation targets before code edits.
2. Add an issue-local regression guard that keeps the selected residual `src/bigclaw` tranche absent and pins the active Go replacement paths.
3. Add an issue-local lane report plus validation and status artifacts that document the before/after zero-Python baseline, the retired module ledger, and the exact test commands/results.
4. Run the targeted inventory and regression commands, capture exact outputs, then commit and push `BIG-GO-131`.

## Acceptance
- `BIG-GO-131` has an issue-specific workpad, regression guard, lane report, validation report, and status artifact.
- The regression guard proves the selected residual `src/bigclaw` tranche remains absent and that the mapped Go/native replacement paths still exist.
- Validation records exact commands and exact results for repository Python inventory, focused `src/bigclaw` inventory, and the issue-local regression run.
- Changes remain scoped to `BIG-GO-131` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO131(RepositoryHasNoPythonFiles|ResidualSrcBigClawSweepJStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
