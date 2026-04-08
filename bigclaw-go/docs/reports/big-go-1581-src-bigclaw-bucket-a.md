# BIG-GO-1581 `src/bigclaw` Bucket-A Sweep

## Scope

Strict bucket lane `BIG-GO-1581` records the repository-state outcome for
bucket A of the retired `src/bigclaw/*.py` top-level modules and the matching
Go/native ownership paths.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused bucket-A physical Python file count before lane changes: `0`
- Focused bucket-A physical Python file count after lane changes: `0`

This checkout was already Python-free before the lane started, so the shipped
work lands as bucket-specific regression and replacement evidence rather than an
in-branch deletion batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Focused bucket-A ledger: `[]`

## Retired Python Surface

- `src/bigclaw`: directory not present, so residual Python files = `0`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/github_sync.py`
- `scripts/ops/bigclaw_github_sync.py`

## Go Or Native Replacement Paths

The active Go/native replacement surface for bucket A remains:

- `bigclaw-go/internal/costcontrol/controller.go`
- `bigclaw-go/internal/issuearchive/archive.go`
- `bigclaw-go/internal/githubsync/sync.go`
- `docs/go-mainline-cutover-issue-pack.md`

These paths match the retired cost-control, issue-archive, and GitHub-sync
ownership mapping that the Go-mainline cutover pack preserves for the legacy
Python surface.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw scripts/ops -type f \( -name 'cost_control.py' -o -name 'issue_archive.py' -o -name 'github_sync.py' -o -name 'bigclaw_github_sync.py' \) 2>/dev/null | sort`
  Result: no output; the focused bucket-A surface remained absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1581(RepositoryHasNoPythonFiles|BucketARetiredPythonPathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.182s`
