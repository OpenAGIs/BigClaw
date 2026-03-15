# Live Validation Index

- Schema: `live_validation_bundle/v1`
- Latest run: `20260314T164647Z`
- Generated at: `2026-03-15T16:14:13.522935+00:00`
- Status: `succeeded`
- Bundle: `docs/reports/live-validation-runs/20260314T164647Z`
- Bundle summary: `docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- Bundle README: `docs/reports/live-validation-runs/20260314T164647Z/README.md`

## Component bundles

### local
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260314T164647Z/sqlite-smoke-report.json`
- Canonical report: `docs/reports/sqlite-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260314T164647Z/local.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260314T164647Z/local.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260314T164647Z/local.service.log`
- Audit log: `docs/reports/live-validation-runs/20260314T164647Z/local.audit.jsonl`
- Task ID: `local-smoke-1773506812`
- Task state: `succeeded`
- Latest event: `task.completed`
- Execution artifacts: `stdout.log`

### kubernetes
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260314T164647Z/kubernetes-live-smoke-report.json`
- Canonical report: `docs/reports/kubernetes-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260314T164647Z/kubernetes.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260314T164647Z/kubernetes.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260314T164647Z/kubernetes.service.log`
- Audit log: `docs/reports/live-validation-runs/20260314T164647Z/kubernetes.audit.jsonl`
- Task ID: `kubernetes-smoke-1773506812`
- Task state: `succeeded`
- Latest event: `task.completed`
- Execution artifacts: `k8s://jobs/ray/bigclaw-kubernetes-smoke-1773506812-1773506812, k8s://pods/ray/bigclaw-kubernetes-smoke-1773506812-1773506812-krbvv`

### ray
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json`
- Canonical report: `docs/reports/ray-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260314T164647Z/ray.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260314T164647Z/ray.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260314T164647Z/ray.service.log`
- Audit log: `docs/reports/live-validation-runs/20260314T164647Z/ray.audit.jsonl`
- Task ID: `ray-smoke-1773506812`
- Task state: `succeeded`
- Latest event: `task.completed`
- Execution artifacts: `ray://jobs/bigclaw-ray-smoke-1773506812`

## Workflow closeout commands

- `cd bigclaw-go && ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && go test ./...`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260314T164647Z` · `succeeded` · `2026-03-15T16:14:13.522935+00:00` · `docs/reports/live-validation-runs/20260314T164647Z`
- `20260314T163430Z` · `succeeded` · `2026-03-14T16:34:42.080370+00:00` · `docs/reports/live-validation-runs/20260314T163430Z`
