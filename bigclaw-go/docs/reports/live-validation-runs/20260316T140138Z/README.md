# Live Validation Index

- Latest run: `20260316T140138Z`
- Generated at: `2026-03-16T15:54:25.340839+00:00`
- Status: `succeeded`
- Bundle: `docs/reports/live-validation-runs/20260316T140138Z`
- Summary JSON: `docs/reports/live-validation-runs/20260316T140138Z/summary.json`

## Latest bundle artifacts

### local
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json`
- Latest report: `docs/reports/sqlite-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/local.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260316T140138Z/local.service.log`
- Audit log: `docs/reports/live-validation-runs/20260316T140138Z/local.audit.jsonl`
- Task ID: `local-smoke-1773669703`

### kubernetes
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json`
- Latest report: `docs/reports/kubernetes-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.service.log`
- Audit log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.audit.jsonl`
- Task ID: `kubernetes-smoke-1773669703`

### ray
- Enabled: `True`
- Status: `succeeded`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Latest report: `docs/reports/ray-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/ray.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log`
- Service log: `docs/reports/live-validation-runs/20260316T140138Z/ray.service.log`
- Audit log: `docs/reports/live-validation-runs/20260316T140138Z/ray.audit.jsonl`
- Task ID: `ray-smoke-1773669703`

### shared-queue companion
- Available: `True`
- Status: `succeeded`
- Bundle summary: `docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json`
- Canonical summary: `docs/reports/shared-queue-companion-summary.json`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json`
- Canonical report: `docs/reports/multi-node-shared-queue-report.json`
- Cross-node completions: `99`
- Duplicate `task.started`: `0`
- Duplicate `task.completed`: `0`
- Missing terminal completions: `0`

## Workflow closeout commands

- `cd bigclaw-go && ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && go test ./...`
- `git push origin <branch> && git log -1 --stat`

## Recent bundles

- `20260316T140138Z` · `succeeded` · `2026-03-16T15:54:25.340839+00:00` · `docs/reports/live-validation-runs/20260316T140138Z`
- `20260314T164647Z` · `succeeded` · `2026-03-14T16:46:57.671520+00:00` · `docs/reports/live-validation-runs/20260314T164647Z`
- `20260314T163430Z` · `succeeded` · `2026-03-14T16:34:42.080370+00:00` · `docs/reports/live-validation-runs/20260314T163430Z`

## Continuation gate

- Status: `policy-go`
- Recommendation: `go`
- Report: `docs/reports/validation-bundle-continuation-policy-gate.json`
- Workflow mode: `review`
- Workflow outcome: `pass`
- Latest reviewed run: `20260316T140138Z`
- Failing checks: `0`
- Workflow exit code on current evidence: `0`
- Reviewer digest: `docs/reports/validation-bundle-continuation-digest.md`
- Reviewer index: `docs/reports/live-validation-index.md`
- Next action: `set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions`

## Continuation artifacts

- `docs/reports/validation-bundle-continuation-scorecard.json` summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.
- `docs/reports/validation-bundle-continuation-policy-gate.json` records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.

## Parallel follow-up digests

- `docs/reports/validation-bundle-continuation-digest.md` Validation bundle continuation caveats are consolidated here.
