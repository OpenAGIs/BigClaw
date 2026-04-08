# BIG-GO-151 Workpad

## Context
- Issue: `BIG-GO-151`
- Title: `Residual src/bigclaw Python sweep L`
- Goal: harden the already-zero `src/bigclaw` Python baseline with issue-specific regression and evidence for the retired workflow-definition tranche.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_151_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-151-src-bigclaw-tranche-l.md`
- `reports/BIG-GO-151-status.json`
- `reports/BIG-GO-151-validation.md`

## Plan
1. Record an issue-specific plan, acceptance criteria, and validation commands before code changes.
2. Add a focused regression guard for the retired `src/bigclaw` tranche-L Python surface and its Go replacement paths.
3. Add issue-scoped evidence artifacts that capture the zero-file inventory, exact focused ledger, and validation outcomes.
4. Run targeted inventory and regression commands, record exact commands and results, then commit and push `BIG-GO-151`.

## Acceptance
- `BIG-GO-151` has an issue-specific workpad, regression guard, report, status artifact, and validation report.
- The regression guard keeps the repository Python-free and proves the tranche-L `src/bigclaw` paths remain absent while their Go/native replacements remain available.
- Validation artifacts record exact commands and exact results for repository-wide inventory, focused `src/bigclaw` inventory, and the targeted Go regression run.
- Changes stay limited to issue-scoped regression and evidence files.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO151(RepositoryHasNoPythonFiles|WorkflowDefinitionTrancheLStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
