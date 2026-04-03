# Live Validation Index

- Latest run: `20260316T140138Z`
- Generated at: `2026-04-03T07:57:34.810349Z`
- Status: `succeeded`
- Bundle: `docs/reports/live-validation-runs/20260316T140138Z`
- Summary JSON: `docs/reports/live-validation-runs/20260316T140138Z/summary.json`

## Latest bundle artifacts

### local
- Enabled: `true`
- Status: `succeeded`
- Validation lane: `local`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json`
- Latest report: `docs/reports/sqlite-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/local.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log`
- Task ID: `local-smoke-1773669703`
- Latest event: `task.completed`
- Routing reason: `required executor=local`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log` kind=`stderr_log`

### kubernetes
- Enabled: `true`
- Status: `succeeded`
- Validation lane: `k8s`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json`
- Latest report: `docs/reports/kubernetes-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log`
- Task ID: `kubernetes-smoke-1773669703`
- Latest event: `task.completed`
- Routing reason: `required executor=kubernetes`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log` kind=`stderr_log`

### ray
- Enabled: `true`
- Status: `succeeded`
- Validation lane: `ray`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Latest report: `docs/reports/ray-live-smoke-report.json`
- Stdout log: `docs/reports/live-validation-runs/20260316T140138Z/ray.stdout.log`
- Stderr log: `docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log`
- Task ID: `ray-smoke-1773669703`
- Latest event: `task.completed`
- Routing reason: `required executor=ray`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log` kind=`stderr_log`

## Validation matrix

- Lane `local` executor=`local` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json`
- Lane `local` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log` kind=`stderr_log` message=``
- Lane `k8s` executor=`kubernetes` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json`
- Lane `k8s` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log` kind=`stderr_log` message=``
- Lane `ray` executor=`ray` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Lane `ray` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log` kind=`stderr_log` message=``

## Parallel evidence bundle

- Status: `succeeded`
- Canonical JSON: `docs/reports/parallel-validation-evidence-bundle.json`
- Canonical Markdown: `docs/reports/parallel-validation-evidence-bundle.md`
- Bundle JSON: `docs/reports/live-validation-runs/20260316T140138Z/parallel-validation-evidence-bundle.json`
- Bundle Markdown: `docs/reports/live-validation-runs/20260316T140138Z/parallel-validation-evidence-bundle.md`

### broker
- Enabled: `false`
- Status: `skipped`
- Configuration state: `not_configured`
- Bundle summary: `docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`
- Canonical summary: `docs/reports/broker-validation-summary.json`
- Bundle bootstrap summary: `docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json`
- Canonical bootstrap summary: `docs/reports/broker-bootstrap-review-summary.json`
- Validation pack: `docs/reports/broker-failover-fault-injection-validation-pack.md`
- Bootstrap ready: `false`
- Runtime posture: `contract_only`
- Live adapter implemented: `false`
- Config completeness: driver=`false` urls=`false` topic=`false` consumer_group=`false`
- Proof boundary: `broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof`
- Validation error: `broker event log config missing driver, urls, topic`
- Reason: `not_configured`

### shared-queue companion
- Available: `true`
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

- `20260316T140138Z` · `succeeded` · `2026-04-03T07:57:34.810349Z` · `docs/reports/live-validation-runs/20260316T140138Z`
- `20260314T164647Z` · `succeeded` · `2026-03-14T16:46:57.671520+00:00` · `docs/reports/live-validation-runs/20260314T164647Z`
- `20260314T163430Z` · `succeeded` · `2026-03-14T16:34:42.080370+00:00` · `docs/reports/live-validation-runs/20260314T163430Z`

## Continuation gate

- Status: `policy-go`
- Recommendation: `go`
- Report: `docs/reports/validation-bundle-continuation-policy-gate.json`
- Workflow mode: `hold`
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
