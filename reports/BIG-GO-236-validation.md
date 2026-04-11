# BIG-GO-236 Validation

Date: 2026-04-12

## Scope

Issue: `BIG-GO-236`

Title: `Residual support assets Python sweep S`

This lane audited the residual support-asset surfaces that still matter after
the Go-only migration work: `bigclaw-go/examples`,
`bigclaw-go/docs/reports/broker-failover-stub-artifacts`,
`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`,
`bigclaw-go/docs/reports/live-shadow-runs`,
`bigclaw-go/docs/reports/live-validation-runs`, and `scripts/ops`.

The checked-out workspace was already at a repository-wide Python file count of
`0`, so there was no physical `.py` asset left to delete or replace in-branch.
The delivered work hardens that zero-Python baseline with a lane-specific
regression guard and fresh validation evidence for the remaining examples,
fixtures, demos, and support helpers.

## Remaining Python Asset Inventory

- Repository-wide physical `.py` files: `none`
- `bigclaw-go/examples/*.py`: `none`
- `bigclaw-go/docs/reports/broker-failover-stub-artifacts/*.py`: `none`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/*.py`: `none`
- `bigclaw-go/docs/reports/live-shadow-runs/*.py`: `none`
- `bigclaw-go/docs/reports/live-validation-runs/*.py`: `none`
- `scripts/ops/*.py`: `none`

## Native Replacement Paths

- Repository sweep verification: `bigclaw-go/internal/regression/big_go_236_zero_python_guard_test.go`
- Example payload: `bigclaw-go/examples/shadow-task.json`
- Example payload: `bigclaw-go/examples/shadow-task-budget.json`
- Example payload: `bigclaw-go/examples/shadow-task-validation.json`
- Example manifest: `bigclaw-go/examples/shadow-corpus-manifest.json`
- Broker failover fixture: `bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`
- Multi-node takeover demo artifact: `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`
- Live-shadow bundle report: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json`
- Live-validation bundle summary: `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json`
- Root operator helper: `scripts/ops/bigclawctl`
- Root issue helper alias: `scripts/ops/bigclaw-issue`
- Root panel helper alias: `scripts/ops/bigclaw-panel`
- Root symphony helper alias: `scripts/ops/bigclaw-symphony`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-236 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/examples /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-validation-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/scripts/ops -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO236(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-236 -path '*/.git' -prune -o -type f -name '*.py' -print | sort
```

Result:

```text
none
```

### Support-asset directory inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/examples /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go/docs/reports/live-validation-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/scripts/ops -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text
none
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-236/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO236(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.149s
```

## Git

- Branch: `main`
- Baseline HEAD before lane commit: `7872e4fa`
- Lane commit details: `98676441 BIG-GO-236: add residual support asset python sweep guard`
- Final pushed lane commit: `98676441 BIG-GO-236: add residual support asset python sweep guard`
- Push target: `origin/main`
- Remote verification: `98676441b233b92b3c677183f0df55a5498b141d refs/heads/main`

## Residual Risk

- The live branch baseline was already Python-free, so BIG-GO-236 can only
  lock in and document the Go-only state rather than numerically lower the
  repository `.py` count.
- `origin/main` advanced twice during the first push attempts, so the lane was
  rebased onto the moving remote before the final successful push.
