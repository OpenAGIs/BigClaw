# Live Validation Index

- Latest run: `20260314T163430Z`
- Generated at: `2026-03-14T16:34:42.080370+00:00`
- Status: `succeeded`
- Bundle: `docs/reports/live-validation-runs/20260314T163430Z`
- Summary JSON: `docs/reports/live-validation-runs/20260314T163430Z/summary.json`

## Latest bundle artifacts

### local
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260314T163430Z/sqlite-smoke-report.json`
- Latest report: `docs/reports/sqlite-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260314T163430Z/local.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260314T163430Z/local.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260314T163430Z/local.service.log`
- Audit log: `docs/reports/live-validation-runs/20260314T163430Z/local.audit.jsonl`
- Task ID: `local-smoke-1773506074`

### kubernetes
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260314T163430Z/kubernetes-live-smoke-report.json`
- Latest report: `docs/reports/kubernetes-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260314T163430Z/kubernetes.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260314T163430Z/kubernetes.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260314T163430Z/kubernetes.service.log`
- Audit log: `docs/reports/live-validation-runs/20260314T163430Z/kubernetes.audit.jsonl`
- Task ID: `kubernetes-smoke-1773506074`

### ray
- Enabled: `False`
- Status: `skipped`
- Reason: `executor disabled; no Ray smoke report was produced for this bundle`

## Workflow closeout commands

- `cd bigclaw-go && ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && go test ./...`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260314T163430Z` · `succeeded` · `2026-03-14T16:34:42.080370+00:00` · `docs/reports/live-validation-runs/20260314T163430Z`
- `20260314T163139Z` · `succeeded` · `2026-03-14T16:31:45.133913+00:00` · `docs/reports/live-validation-runs/20260314T163139Z`
