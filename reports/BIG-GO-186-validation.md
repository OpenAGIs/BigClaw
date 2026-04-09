# BIG-GO-186 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-186`

Title: `Residual support assets Python sweep N`

This lane audited the residual support-asset surfaces that still matter after
the Go-only migration work: `bigclaw-go/examples`,
`bigclaw-go/docs/reports/broker-failover-stub-artifacts`,
`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`, and
`scripts/ops`.

The checked-out workspace was already at a repository-wide Python file count of
`0`, so there was no physical `.py` asset left to delete or replace in-branch.
The delivered work hardens that zero-Python baseline with a lane-specific
regression guard and fresh validation evidence for the remaining examples,
fixture bundles, demo artifacts, and helper entrypoints.

## Remaining Python Asset Inventory

- Repository-wide physical `.py` files: `none`
- `bigclaw-go/examples/*.py`: `none`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/*.py`: `none`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/*.py`: `none`
- `scripts/ops/*.py`: `none`

## Native Replacement Paths

- Repository sweep verification: `bigclaw-go/internal/regression/big_go_186_zero_python_guard_test.go`
- Example payload: `bigclaw-go/examples/shadow-task.json`
- Example manifest: `bigclaw-go/examples/shadow-corpus-manifest.json`
- Broker failover fixture: `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`
- Broker failover fixture: `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`
- Multi-node takeover demo artifact: `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- Multi-node takeover demo artifact: `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl`
- Multi-node takeover demo artifact: `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl`
- Root operator helper: `scripts/ops/bigclawctl`
- Root issue helper alias: `scripts/ops/bigclaw-issue`
- Root panel helper alias: `scripts/ops/bigclaw-panel`
- Root symphony helper alias: `scripts/ops/bigclaw-symphony`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-186 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/examples /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/scripts/ops -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO186(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-186 -path '*/.git' -prune -o -type f -name '*.py' -print | sort
```

Result:

```text
none
```

### Support-asset directory inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/examples /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/scripts/ops -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text
none
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-186/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO186(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	3.199s
```

## Git

- Branch: `main`
- Baseline HEAD before lane commit: `4f4d46ad`
- Lane commit details: `git log --oneline --grep 'BIG-GO-186'`
- Final pushed lane commit: see `git log --oneline --grep 'BIG-GO-186'`
- Push target: `origin/main`

## Residual Risk

- The live branch baseline was already Python-free, so BIG-GO-186 can only
  lock in and document the Go-only state rather than numerically lower the
  repository `.py` count.
