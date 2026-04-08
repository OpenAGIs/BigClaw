# BIG-GO-1581 Workpad

## Context
- Issue: `BIG-GO-1581`
- Title: `Strict bucket lane 1581: src/bigclaw/*.py bucket A`
- Goal: record and harden the bucket-A retirement state for the first `src/bigclaw`
  top-level Python tranche and its Go/native replacements.
- Current repo state on entry: repository-wide physical Python inventory is
  already `0`, and `src/bigclaw` is absent in this checkout.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_1581_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-1581-src-bigclaw-bucket-a.md`
- `reports/BIG-GO-1581-status.json`
- `reports/BIG-GO-1581-validation.md`

## Bucket A
- Retired Python paths:
  `src/bigclaw/cost_control.py`, `src/bigclaw/issue_archive.py`,
  `src/bigclaw/github_sync.py`, `scripts/ops/bigclaw_github_sync.py`
- Active Go/native owners:
  `bigclaw-go/internal/costcontrol/controller.go`,
  `bigclaw-go/internal/issuearchive/archive.go`,
  `bigclaw-go/internal/githubsync/sync.go`,
  `docs/go-mainline-cutover-issue-pack.md`

## Plan
1. Replace the stale shared workpad with this issue-specific plan before editing
   code or evidence files.
2. Add a lane-specific regression guard for the bucket-A retired Python paths,
   repository-wide zero-Python baseline, and the mapped Go/native replacement
   paths.
3. Add lane evidence artifacts that record exact before/after counts, the empty
   deletion ledger for this already-clean checkout, and the exact validation
   commands/results.
4. Run targeted inventory and regression commands, capture exact results in the
   validation artifacts, then commit and push the lane branch.

## Acceptance
- The workpad, regression guard, lane report, status artifact, and validation
  report are present and scoped to `BIG-GO-1581`.
- The regression guard proves bucket-A retired Python paths remain absent and
  mapped Go/native replacement paths remain available.
- Validation records exact commands and exact results for repository-wide Python
  inventory, bucket-A inventory, and the targeted Go regression run.
- The lane acknowledges that this checkout was already Python-free on entry, so
  before/after physical `.py` counts remain `0` while the lane lands regression
  and evidence hardening.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw scripts/ops -type f \\( -name 'cost_control.py' -o -name 'issue_archive.py' -o -name 'github_sync.py' -o -name 'bigclaw_github_sync.py' \\) 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1581(RepositoryHasNoPythonFiles|BucketARetiredPythonPathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
