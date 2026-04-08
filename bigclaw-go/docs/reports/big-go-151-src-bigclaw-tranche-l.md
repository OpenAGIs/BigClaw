# BIG-GO-151 `src/bigclaw` Tranche-L Sweep

## Scope

Refill lane `BIG-GO-151` records the repository-state outcome for the residual
`src/bigclaw` tranche-L sweep covering the retired workflow-definition and
intake-model Python surface.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused `src/bigclaw` tranche-L physical Python file count before lane changes: `0`
- Focused `src/bigclaw` tranche-L physical Python file count after lane changes: `0`

This checkout was already Python-free before the lane started, so the shipped
work lands as exact Go/native replacement evidence and regression hardening
rather than an in-branch deletion batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Focused tranche-L ledger: `[]`

## Retired Python Surface

- `src/bigclaw`: directory not present, so residual Python files = `0`
- `src/bigclaw/models.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/dsl.py`

## Go Or Native Replacement Paths

The active Go/native replacement surface for tranche L remains:

- `bigclaw-go/internal/domain/task.go`
- `bigclaw-go/internal/domain/priority.go`
- `bigclaw-go/internal/intake/connector.go`
- `bigclaw-go/internal/intake/mapping.go`
- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/workflow/model.go`
- `bigclaw-go/internal/prd/intake.go`
- `docs/go-mainline-cutover-issue-pack.md`

These paths match the workflow-definition and intake ownership mapping in
`docs/go-mainline-cutover-issue-pack.md` for the retired tranche-L
`src/bigclaw` modules.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the focused `src/bigclaw` tranche-L surface remained
  absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO151(RepositoryHasNoPythonFiles|WorkflowDefinitionTrancheLStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
  Result: `ok  	bigclaw-go/internal/regression	4.838s`
