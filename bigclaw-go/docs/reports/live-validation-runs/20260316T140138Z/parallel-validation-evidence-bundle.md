# Parallel Validation Evidence Bundle

- Run ID: `20260316T140138Z`
- Generated at: `2026-04-03T07:57:34.810349Z`
- Status: `succeeded`

## Matrix

- Lane count: `3`
- Enabled lanes: `3`
- Succeeded lanes: `3`
- Failing lanes: `0`
- Root causes localized: `3`

- Lane `local` executor=`local` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json`
- Lane `local` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log` kind=`stderr_log` message=``
- Lane `k8s` executor=`kubernetes` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json`
- Lane `k8s` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log` kind=`stderr_log` message=``
- Lane `ray` executor=`ray` status=`succeeded` enabled=`true` report=`docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Lane `ray` root cause: event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log` kind=`stderr_log` message=``

## Lane Details

### local
- Executor: `local`
- Enabled: `true`
- Status: `succeeded`
- Task ID: `local-smoke-1773669703`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json`
- Canonical report: `docs/reports/sqlite-smoke-report.json`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/local.stderr.log` kind=`stderr_log`

### k8s
- Executor: `kubernetes`
- Enabled: `true`
- Status: `succeeded`
- Task ID: `kubernetes-smoke-1773669703`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json`
- Canonical report: `docs/reports/kubernetes-live-smoke-report.json`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/kubernetes.stderr.log` kind=`stderr_log`

### ray
- Executor: `ray`
- Enabled: `true`
- Status: `succeeded`
- Task ID: `ray-smoke-1773669703`
- Bundle report: `docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Canonical report: `docs/reports/ray-live-smoke-report.json`
- Failure root cause: status=`not_triggered` event=`task.completed` location=`docs/reports/live-validation-runs/20260316T140138Z/ray.stderr.log` kind=`stderr_log`
