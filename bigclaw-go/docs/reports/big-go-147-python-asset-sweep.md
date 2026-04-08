# BIG-GO-147 Python Asset Sweep

## Scope

Refill lane `BIG-GO-147` records the repository-state outcome for a broad
high-impact sweep across the retired `src/bigclaw` contract and governance
surface that still appears densely in the cutover docs.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused retired contract/governance tranche physical Python file count before lane changes: `0`
- Focused retired contract/governance tranche physical Python file count after lane changes: `0`

This checkout was already Python-free before the lane started, so the shipped
work lands as exact Go/native replacement evidence and regression hardening
rather than an in-branch deletion batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Focused tranche ledger: `[]`

## Retired Python Surface

- `src/bigclaw/models.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/dsl.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/audit_events.py`

## Go Or Native Replacement Paths

The active Go/native replacement surface for this contract and governance
tranche remains:

- `bigclaw-go/internal/domain/task.go`
- `bigclaw-go/internal/intake/connector.go`
- `bigclaw-go/internal/intake/types.go`
- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/risk/assessment.go`
- `bigclaw-go/internal/governance/freeze.go`
- `bigclaw-go/internal/contract/execution.go`
- `bigclaw-go/internal/observability/audit_spec.go`
- `docs/go-domain-intake-parity-matrix.md`
- `docs/go-mainline-cutover-issue-pack.md`
- `docs/go-mainline-cutover-handoff.md`

These paths capture the current Go ownership and cutover evidence for the
retired contract/governance tranche that used to live under `src/bigclaw`.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw -type f \( -name 'models.py' -o -name 'connectors.py' -o -name 'dsl.py' -o -name 'risk.py' -o -name 'governance.py' -o -name 'execution_contract.py' -o -name 'audit_events.py' \) 2>/dev/null | sort`
  Result: no output; the focused retired contract/governance tranche remained absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO147(RepositoryHasNoPythonFiles|RetiredContractAndGovernanceTrancheStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.159s`
