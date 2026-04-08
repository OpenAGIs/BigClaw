# BIG-GO-17 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-17`

Title: `Sweep auxiliary nested Python modules batch B`

This lane audited nested auxiliary report and evidence directories outside the
primary source and script paths:
`docs/reports`,
`bigclaw-go/docs/reports/broker-failover-stub-artifacts`,
`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`,
`bigclaw-go/docs/reports/live-shadow-runs`, and
`bigclaw-go/docs/reports/live-validation-runs`.

The checked-out workspace was already at a repository-wide Python file count of
`0`, so the delivered work hardens and documents the zero-Python baseline
rather than deleting in-branch Python assets.

## Remaining Python Asset Inventory

- Repository-wide physical `.py` files: `none`
- `docs/reports/*.py`: `none`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/*.py`: `none`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/*.py`: `none`
- `bigclaw-go/docs/reports/live-shadow-runs/*.py`: `none`
- `bigclaw-go/docs/reports/live-validation-runs/*.py`: `none`

## Retained Native Evidence Assets

- Markdown evidence: `docs/reports/bootstrap-cache-validation.md`
- Broker failover artifact: `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-05/publish-attempt-ledger.json`
- Broker failover fault trace: `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-07/fault-timeline.json`
- Multi-node takeover audit trail: `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- Live shadow bundle summary: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- Live validation bundle summary: `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-17 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO17(NestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidenceAssetsRemainAvailable|LaneReportCapturesSweepState)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-17 -path '*/.git' -prune -o -type f -name '*.py' -print | sort
```

Result:

```text
no output
```

### Nested auxiliary directory inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text
no output
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-17/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO17(NestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidenceAssetsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.197s
```

## Git

- Branch: `main`
- Commit: read the pushed tip from `git rev-parse --short HEAD`
- Push target: `origin/main`

## Residual Risk

- The live branch baseline was already Python-free, so BIG-GO-17 documents and
  locks in nested auxiliary report and evidence surfaces rather than reducing
  the repository `.py` count in this checkout.
