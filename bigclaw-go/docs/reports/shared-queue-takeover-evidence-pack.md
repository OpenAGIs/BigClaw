# Shared Queue Takeover Evidence Pack

## Scope

This pack is the stable review entry point for shared-queue coordination, takeover planning, lease recovery, and dead-letter replay evidence.

## Canonical Artifacts

- Multi-node proof report: `docs/reports/multi-node-shared-queue-report.json`
- Stable shared-queue bundle: `docs/reports/shared-queue-takeover-evidence/latest`
- Takeover matrix: `docs/reports/multi-subscriber-takeover-validation-report.json`
- Takeover narrative: `docs/reports/multi-subscriber-takeover-validation-report.md`
- Lease recovery evidence: `docs/reports/lease-recovery-report.md`
- Queue and dead-letter evidence: `docs/reports/queue-reliability-report.md`
- Validation workflow: `docs/e2e-validation.md`

## Current Result

- The shared-queue proof now exports a repo-native bundle with the queue database, per-node audit logs, and per-node service logs, so review attachments do not depend on transient temp paths.
- The takeover matrix remains planning-only, but it now sits beside the concrete shared-queue, lease-recovery, and dead-letter evidence needed to judge implementation readiness.
- Dead-letter handling and replay are already covered by automated queue and API evidence, so takeover follow-up work can focus on subscriber-group ownership and checkpoint fencing instead of basic recovery plumbing.

## Review Commands

```bash
cd bigclaw-go
python3 scripts/e2e/multi_node_shared_queue.py \
  --count 200 \
  --submit-workers 8 \
  --state-dir tmp/shared-queue-proof \
  --bundle-dir docs/reports/shared-queue-takeover-evidence/latest \
  --report-path docs/reports/multi-node-shared-queue-report.json
python3 scripts/e2e/subscriber_takeover_fault_matrix.py --pretty
```

## What Reviewers Can Verify Today

- Cross-node queue ownership can shift between two `bigclawd` processes without duplicate terminal task completion.
- Queue evidence includes the exact per-node audit and service logs used by the generated shared-queue report.
- Lease expiry and reacquisition are already covered by automated queue/runtime tests.
- Dead-letter listing and replay are already covered by API and queue-level automated tests.

## Remaining Gap

- Subscriber-group checkpoint leases, takeover-owner audit events, and stale-writer rejection counters are still missing, so the takeover matrix is an executable contract for follow-up implementation rather than a completed end-to-end run.
