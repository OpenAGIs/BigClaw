# BIG-GO-1562 `src/bigclaw` Tranche-B Sweep

## Scope

Refill lane `BIG-GO-1562` records the repository-state outcome for the new
unblocked `src/bigclaw` deletion tranche B covering the retired workflow and
orchestration Python surface.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused `src/bigclaw` tranche-B physical Python file count before lane changes: `0`
- Focused `src/bigclaw` tranche-B physical Python file count after lane changes: `0`

This checkout was already Python-free before the lane started, so the shipped
work lands as exact Go/native replacement evidence and regression hardening
rather than an in-branch deletion batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Focused tranche-B ledger: `[]`

## Retired Python Surface

- `src/bigclaw`: directory not present, so residual Python files = `0`
- `src/bigclaw/runtime.py`
- `src/bigclaw/scheduler.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/workflow.py`
- `src/bigclaw/queue.py`

## Go Or Native Replacement Paths

The active Go/native replacement surface for tranche B remains:

- `bigclaw-go/internal/scheduler/scheduler.go`
- `bigclaw-go/internal/worker/runtime.go`
- `bigclaw-go/internal/orchestrator/loop.go`
- `bigclaw-go/internal/queue/queue.go`
- `bigclaw-go/internal/control/controller.go`
- `docs/go-mainline-cutover-issue-pack.md`

These paths match the workflow-orchestration ownership mapping in
`docs/go-mainline-cutover-issue-pack.md` for the retired tranche-B
`src/bigclaw` modules.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the focused `src/bigclaw` tranche-B surface remained
  absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1562(RepositoryHasNoPythonFiles|WorkflowOrchestrationTrancheBStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
  Result: `ok  	bigclaw-go/internal/regression	5.913s`
