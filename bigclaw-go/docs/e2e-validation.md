# BigClaw Go End-to-End Validation

This document covers real cluster smoke validation for the `Kubernetes` and `Ray` executors through the BigClaw control plane.

## Prerequisites

- `go`
- `python3`
- BigClaw Go dependencies installed via `go mod tidy`
- For `Kubernetes`:
  - a reachable cluster
  - `KUBECONFIG` or `BIGCLAW_KUBECONFIG`
  - target namespace permissions to create `Job` and read `Pod` logs
- For `Ray`:
  - a reachable Ray Dashboard / Jobs API endpoint
  - `BIGCLAW_RAY_ADDRESS`, e.g. `ray://127.0.0.1:10001` or `http://127.0.0.1:8265`

## What the scripts do

1. Reuse an already healthy `bigclawd` if one is listening on the requested base URL
2. Otherwise autostart an isolated local `bigclawd` on a fresh loopback port with an isolated queue/audit state directory
3. Submit a task through `POST /tasks`
4. Poll `GET /tasks/{id}` until terminal state
5. Dump `GET /events?task_id=...` for debugging evidence
6. Optionally stream `GET /stream/events` for near-real-time event verification

You can also query `GET /events?trace_id=...` when multiple task IDs belong to the
same trace or shadow-comparison run.

## Verify control-plane API locally

```bash
cd bigclaw-go
go test ./...
go run ./cmd/bigclawd
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/metrics
curl -N http://127.0.0.1:8080/stream/events
```


## SQLite-backed local smoke

Use this to validate the control plane with a durable queue backend before touching a real cluster.

## One-shot full validation

Use this to run the local SQLite smoke plus Kubernetes and Ray validation in one command. All enabled lanes now execute concurrently and export a timestamped repo-native evidence bundle.

```bash
cd bigclaw-go
export KUBECONFIG=/Users/jxrt/.kube/ray-local-config
export BIGCLAW_RAY_ADDRESS=ray://127.0.0.1:10001
export BIGCLAW_KUBERNETES_NAMESPACE=ray
export BIGCLAW_KUBERNETES_IMAGE=alpine:3.20
export BIGCLAW_QUEUE_BACKEND=sqlite
./scripts/e2e/run_all.sh
```

The script writes a consolidated summary to `docs/reports/live-validation-summary.json`, refreshes the canonical component reports for local, Kubernetes, and Ray validation, and creates a timestamped bundle plus index under `docs/reports/live-validation-runs/` and `docs/reports/live-validation-index.md`.

You can then refresh the rolling continuation overlay from the checked-in bundle evidence:

```bash
cd bigclaw-go
python3 scripts/e2e/validation_bundle_continuation_scorecard.py --pretty
```

This writes `docs/reports/validation-bundle-continuation-scorecard.json`, summarizing the recent bundle lineage plus the current shared-queue companion proof. The scorecard is still manual or workflow-triggered rather than always-on.

You can evaluate the checked-in continuation policy gate as a follow-up:

```bash
cd bigclaw-go
python3 scripts/e2e/validation_bundle_continuation_policy_gate.py --pretty
```

This writes `docs/reports/validation-bundle-continuation-policy-gate.json` and currently returns `go` for the checked-in evidence window because the latest indexed bundles now include repeated `ray` coverage across multiple runs.

## Mixed workload matrix

```bash
cd bigclaw-go
export KUBECONFIG=/Users/jxrt/.kube/ray-local-config
export BIGCLAW_RAY_ADDRESS=ray://127.0.0.1:10001
export BIGCLAW_KUBERNETES_NAMESPACE=ray
export BIGCLAW_KUBERNETES_IMAGE=alpine:3.20
export BIGCLAW_QUEUE_BACKEND=sqlite
python3 scripts/e2e/mixed_workload_matrix.py \
  --report-path docs/reports/mixed-workload-matrix-report.json
```

This validates one control-plane instance against a more production-like mix of `local`, tool-routed `kubernetes`, tool-routed `ray`, and high-risk isolation scenarios.

## Multi-node shared queue proof

```bash
cd bigclaw-go
python3 scripts/e2e/multi_node_shared_queue.py \
  --count 200 \
  --submit-workers 8 \
  --report-path docs/reports/multi-node-shared-queue-report.json
```

This starts two `bigclawd` processes against one SQLite queue and verifies there are no duplicate terminal completions across the two nodes.

## Broker failover and replay fault-injection pack

The current repo does not yet ship a broker-backed event log or live failover harness, but the implementation-ready validation matrix now lives in `docs/reports/broker-failover-fault-injection-validation-pack.md`.

Use that pack as the source of truth for:

- broker leader or replica failover scenarios
- replay resume and duplicate-delivery assertions
- checkpoint fencing and stale-writer recovery rules
- the minimum machine-readable report schema required before future broker durability work can be closed honestly

## Multi-subscriber takeover validation matrix

Use this to regenerate the executable local takeover harness report for lease-aware subscriber-group checkpoint ownership.

```bash
cd bigclaw-go
python3 scripts/e2e/subscriber_takeover_fault_matrix.py --pretty
```

This refreshes `docs/reports/multi-subscriber-takeover-validation-report.json` with three deterministic local takeover scenarios, owner timelines, checkpoint transitions, duplicate replay accounting, and stale-writer rejection counts. The remaining live multi-node executability caveats are consolidated in `docs/reports/subscriber-takeover-executability-follow-up-digest.md`.

## Cross-process coordination capability surface

Use this to regenerate the machine-readable coordination surface that ties together the current shared-queue proof, the deterministic takeover harness, and the contract-defined broker-backed target.

```bash
cd bigclaw-go
python3 scripts/e2e/cross_process_coordination_surface.py --pretty
```

This refreshes `docs/reports/cross-process-coordination-capability-surface.json` with the current live local proof metrics, takeover harness summary, capability-by-capability state, and the next runtime hooks for a real distributed coordination proof.

Optional toggles:

```bash
export BIGCLAW_E2E_RUN_LOCAL=0
export BIGCLAW_E2E_RUN_KUBERNETES=1
export BIGCLAW_E2E_RUN_RAY=1
export BIGCLAW_E2E_SUMMARY_REPORT_PATH=docs/reports/live-validation-summary.json
./scripts/e2e/run_all.sh
```

```bash
cd bigclaw-go
export BIGCLAW_QUEUE_BACKEND=sqlite
export BIGCLAW_QUEUE_SQLITE_PATH=./state/queue.db
export BIGCLAW_AUDIT_LOG_PATH=./state/audit.jsonl
python3 scripts/e2e/run_task_smoke.py \
  --autostart \
  --go-root "$PWD" \
  --executor local \
  --title "SQLite smoke" \
  --entrypoint "echo hello from sqlite"
```

This should create:
- `docs/reports/sqlite-smoke-report.json` style output
- JSONL audit events in the configured audit log

If you run multiple local smoke or live-validation processes at the same time, give each process its own `BIGCLAW_QUEUE_SQLITE_PATH` and `BIGCLAW_AUDIT_LOG_PATH` to avoid SQLite lock contention.

## Kubernetes smoke test

```bash
cd bigclaw-go
export KUBECONFIG=/path/to/kubeconfig
export BIGCLAW_KUBERNETES_NAMESPACE=default
export BIGCLAW_KUBERNETES_IMAGE=alpine:3.20
./scripts/e2e/kubernetes_smoke.sh
```

Optional overrides:

```bash
export BIGCLAW_KUBERNETES_SMOKE_IMAGE=ubuntu:24.04
export BIGCLAW_KUBERNETES_SMOKE_ENTRYPOINT='echo custom kubernetes validation'
export BIGCLAW_KUBERNETES_SMOKE_REPORT_PATH=docs/reports/kubernetes-live-smoke-report.json
./scripts/e2e/kubernetes_smoke.sh
```

By default the script writes the latest report to `docs/reports/kubernetes-live-smoke-report.json`.

## Ray smoke test

```bash
cd bigclaw-go
export BIGCLAW_RAY_ADDRESS=ray://127.0.0.1:10001
# BigClaw will normalize ray://... to the local Ray Jobs API on :8265 for submission.
./scripts/e2e/ray_smoke.sh
```

Optional overrides:

```bash
export BIGCLAW_RAY_SMOKE_ENTRYPOINT='python -c "print(123)"'
export BIGCLAW_RAY_RUNTIME_ENV_JSON='{"env_vars":{"BIGCLAW_SMOKE":"1"}}'
export BIGCLAW_RAY_SMOKE_REPORT_PATH=docs/reports/ray-live-smoke-report.json
./scripts/e2e/ray_smoke.sh
```

By default the script writes the latest report to `docs/reports/ray-live-smoke-report.json`.

## Parallel live validation

You can now run Kubernetes and Ray smoke validation in parallel even when `BIGCLAW_QUEUE_BACKEND=sqlite`, because autostarted control-plane processes isolate their HTTP port, queue path, and audit path automatically.

```bash
cd bigclaw-go
export KUBECONFIG=/Users/jxrt/.kube/ray-local-config
export BIGCLAW_RAY_ADDRESS=ray://127.0.0.1:10001
export BIGCLAW_KUBERNETES_NAMESPACE=ray
export BIGCLAW_KUBERNETES_IMAGE=alpine:3.20
export BIGCLAW_QUEUE_BACKEND=sqlite
./scripts/e2e/kubernetes_smoke.sh &
./scripts/e2e/ray_smoke.sh &
wait
```

The latest report payloads include `base_url`, `state_dir`, and `service_log` so each autostarted run can be inspected independently. `run_all.sh` also copies stdout/stderr, service logs, and discovered audit logs into the timestamped bundle for workflow closeout.

## API-level validation commands

Submit a task directly:

```bash
curl -X POST http://127.0.0.1:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "manual-k8s-task",
    "title": "Manual Kubernetes task",
    "required_executor": "kubernetes",
    "container_image": "alpine:3.20",
    "entrypoint": "echo hello from manual task",
    "execution_timeout_seconds": 120
  }'
```

Poll result:

```bash
curl http://127.0.0.1:8080/tasks/manual-k8s-task
curl 'http://127.0.0.1:8080/events?task_id=manual-k8s-task&limit=100'
curl 'http://127.0.0.1:8080/events?trace_id=manual-k8s-task&limit=100'
```

Inspect dead letters and replay them:

```bash
curl http://127.0.0.1:8080/deadletters?limit=20
curl -X POST http://127.0.0.1:8080/deadletters/manual-k8s-task/replay
```

## Expected success shape

- `state` becomes `succeeded`
- `latest_event.type` becomes `task.completed`
- `events` include `task.queued`, `task.leased`, `task.started`, `task.completed`

## Expected failure shape

- `state` becomes `dead_letter` or `retrying`
- `latest_event.payload.message` includes executor error details
- script exits non-zero and prints task/event payloads
