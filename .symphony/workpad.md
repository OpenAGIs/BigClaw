# BIG-GO-147 Workpad

## Context
- Issue: `BIG-GO-147`
- Goal: harden the repo-wide zero-Python baseline with a broad high-impact sweep over retired `src/bigclaw` contract and governance surfaces that are still densely referenced in cutover docs.
- Current repo state on entry: repository-wide physical Python inventory is already `0`, so this lane should land as regression hardening and evidence capture rather than in-branch `.py` deletion.

## Scope
- `.symphony/workpad.md`
- `bigclaw-go/internal/regression/big_go_147_zero_python_guard_test.go`
- `bigclaw-go/docs/reports/big-go-147-python-asset-sweep.md`
- `reports/BIG-GO-147-status.json`
- `reports/BIG-GO-147-validation.md`

## Plan
1. Record an issue-specific plan, acceptance criteria, and validation targets before any source edits.
2. Add a lane-specific regression guard for repository-wide zero Python plus the high-impact retired `src/bigclaw` contract/governance tranche called out in the cutover docs.
3. Add the matching lane report and issue artifacts that document the retired Python paths, the retained Go replacement surface, and the exact validation commands.
4. Run targeted inventory and regression commands, capture exact commands and results, then commit and push `BIG-GO-147`.

## Acceptance
- `BIG-GO-147` has an issue-specific workpad, regression guard, lane report, validation report, and status artifact.
- The regression guard verifies repository-wide Python file count `0`, keeps the targeted retired `src/bigclaw` tranche absent, and confirms the mapped Go replacement paths still exist.
- The lane report and validation report record exact commands and exact results for the repository inventory, the focused tranche inventory, and the targeted Go regression run.
- Changes remain scoped to `BIG-GO-147` artifacts only.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f \\( -name 'models.py' -o -name 'connectors.py' -o -name 'dsl.py' -o -name 'risk.py' -o -name 'governance.py' -o -name 'execution_contract.py' -o -name 'audit_events.py' \\) 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO147(RepositoryHasNoPythonFiles|RetiredContractAndGovernanceTrancheStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
