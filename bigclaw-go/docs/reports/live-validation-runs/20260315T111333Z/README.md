# Live Validation Index

- Latest run: `20260315T111333Z`
- Generated at: `2026-03-15T11:13:51.358487+00:00`
- Status: `succeeded`
- Bundle: `docs/reports/live-validation-runs/20260315T111333Z`
- Summary JSON: `docs/reports/live-validation-runs/20260315T111333Z/summary.json`

## Latest bundle artifacts

### local
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260315T111333Z/sqlite-smoke-report.json`
- Latest report: `docs/reports/sqlite-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260315T111333Z/local.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260315T111333Z/local.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260315T111333Z/local.service.log`
- Audit log: `docs/reports/live-validation-runs/20260315T111333Z/local.audit.jsonl`
- Task ID: `local-smoke-1773573221`

### kubernetes
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260315T111333Z/kubernetes-live-smoke-report.json`
- Latest report: `docs/reports/kubernetes-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260315T111333Z/kubernetes.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260315T111333Z/kubernetes.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260315T111333Z/kubernetes.service.log`
- Audit log: `docs/reports/live-validation-runs/20260315T111333Z/kubernetes.audit.jsonl`
- Task ID: `kubernetes-smoke-1773573221`

### ray
- Enabled: `False`
- Status: `skipped`
- Bundle report: `docs/reports/live-validation-runs/20260315T111333Z/ray-live-smoke-report.json`
- Latest report: `docs/reports/ray-live-smoke-report.json`

### shared_queue
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260315T111333Z/multi-node-shared-queue-report.json`
- Latest report: `docs/reports/multi-node-shared-queue-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.stderr.log`
- node-a audit log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.node-1.audit.jsonl`
- node-a service log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.node-1.service.log`
- node-b audit log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.node-2.audit.jsonl`
- node-b service log: `docs/reports/live-validation-runs/20260315T111333Z/shared_queue.node-2.service.log`

## Supporting durability and takeover references

- Source: `docs/reports/replay-retention-semantics-report.md`
- Bundle copy: `docs/reports/live-validation-runs/20260315T111333Z/replay-retention-semantics-report.md`
- Source: `docs/reports/replicated-event-log-durability-rollout-contract.md`
- Bundle copy: `docs/reports/live-validation-runs/20260315T111333Z/replicated-event-log-durability-rollout-contract.md`
- Source: `docs/reports/multi-subscriber-takeover-validation-report.md`
- Bundle copy: `docs/reports/live-validation-runs/20260315T111333Z/multi-subscriber-takeover-validation-report.md`
- Source: `docs/reports/multi-subscriber-takeover-validation-report.json`
- Bundle copy: `docs/reports/live-validation-runs/20260315T111333Z/multi-subscriber-takeover-validation-report.json`

## Workflow closeout commands

- `cd bigclaw-go && ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && go test ./...`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260315T111333Z` · `succeeded` · `2026-03-15T11:13:51.358487+00:00` · `docs/reports/live-validation-runs/20260315T111333Z`
- `20260314T164647Z` · `succeeded` · `2026-03-14T16:46:57.671520+00:00` · `docs/reports/live-validation-runs/20260314T164647Z`
- `20260314T163430Z` · `succeeded` · `2026-03-14T16:34:42.080370+00:00` · `docs/reports/live-validation-runs/20260314T163430Z`
