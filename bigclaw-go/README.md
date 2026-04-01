# BigClaw Go

BigClaw Go is an initial control-plane rewrite scaffold aimed at high-concurrency
execution patterns that fit `Kubernetes` and `Ray` style runtimes.

## Scope in this bootstrap

This bootstrap now covers an MVP slice for all current Go rewrite planning tickets:

- `BIG-GO-001` architecture ADR and module boundaries
- `BIG-GO-002` unified task protocol and state machine
- `BIG-GO-003` persistent queue plus lease semantics
- `BIG-GO-004` scheduler kernel and dispatch loop
- `BIG-GO-005` worker runtime, heartbeats, cancellation, timeout handling
- `BIG-GO-006` real Kubernetes `Job` client integration path
- `BIG-GO-007` real Ray Jobs REST API integration path
- `BIG-GO-008` in-process event bus with replay support
- `BIG-GO-009` basic observability and HTTP status surface
- `BIG-GO-010` migration plan documents
- `BIG-GO-011` benchmark scaffolding

## Layout

- `cmd/bigclawd`: service entrypoint
- `internal/api`: health and metrics endpoints
- `internal/config`: runtime config defaults and env loading
- `internal/domain`: task model, events, state machine
- `internal/events`: event bus and replay
- `internal/executor`: executor interfaces and adapters
- `internal/observability`: event recorder and counters
- `internal/queue`: queue contracts, memory queue, file queue, benchmarks
- `internal/scheduler`: routing policy and benchmarks
- `internal/worker`: runtime loop, heartbeats, timeout enforcement
- `docs/adr`: architecture decisions
- `docs/*.md`: migration and benchmark plans

## Real integration configuration

### Event log runtime knobs

- `BIGCLAW_EVENT_LOG_BACKEND`
- `BIGCLAW_EVENT_LOG_TARGET_BACKEND`
- `BIGCLAW_EVENT_LOG_REPLICATION_FACTOR`
- `BIGCLAW_EVENT_LOG_BROKER_DRIVER`
- `BIGCLAW_EVENT_LOG_BROKER_URLS`
- `BIGCLAW_EVENT_LOG_BROKER_TOPIC`
- `BIGCLAW_EVENT_LOG_CONSUMER_GROUP`
- `BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT`
- `BIGCLAW_EVENT_LOG_REPLAY_LIMIT`
- `BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL`

### Event durability contract

- `BIGCLAW_EVENT_BACKEND` with `memory`, `sqlite`, `http`, or `broker`
- `BIGCLAW_EVENT_LOG_DSN` for durable event-log backends
- `BIGCLAW_EVENT_CHECKPOINT_DSN` when checkpoint support is required
- `BIGCLAW_EVENT_RETENTION` for durable replay history retention
- `BIGCLAW_EVENT_REQUIRE_REPLAY`
- `BIGCLAW_EVENT_REQUIRE_CHECKPOINT`
- `BIGCLAW_EVENT_REQUIRE_FILTERING`

### Kubernetes

- `BIGCLAW_KUBECONFIG` or `KUBECONFIG`
- `BIGCLAW_KUBERNETES_NAMESPACE`
- `BIGCLAW_KUBERNETES_IMAGE`
- `BIGCLAW_KUBERNETES_SERVICE_ACCOUNT`
- `BIGCLAW_KUBERNETES_POLL_INTERVAL`
- `BIGCLAW_KUBERNETES_CLEANUP`

### Ray

- `BIGCLAW_RAY_ADDRESS`
- `BIGCLAW_RAY_HTTP_TIMEOUT`
- `BIGCLAW_RAY_POLL_INTERVAL`
- `BIGCLAW_RAY_BEARER_TOKEN`


## End-to-end scripts

- `scripts/e2e/` is now a Go-and-shell-only operator surface; the tranche-1 Python helpers were removed and replaced by `bigclawctl automation e2e ...` commands
- `scripts/e2e/run_all.sh` runs local SQLite, Kubernetes, and Ray validation concurrently, writes a timestamped bundle under `docs/reports/live-validation-runs/`, and refreshes `docs/reports/live-validation-summary.json` plus `docs/reports/live-validation-index.md`
- `scripts/e2e/kubernetes_smoke.sh` runs a real Kubernetes smoke task through BigClaw
- `scripts/e2e/ray_smoke.sh` runs a real Ray Jobs API smoke task through BigClaw
- `go run ./cmd/bigclawctl automation e2e run-task-smoke ...` is the generic submit/poll helper used by all wrappers
- `go run ./cmd/bigclawctl automation e2e export-validation-bundle ...` exports repo-native evidence bundles, latest report copies, and the validation index
- `go run ./cmd/bigclawctl automation migration shadow-compare ...` compares primary vs shadow BigClaw endpoints
- `scripts/benchmark/run_suite.sh` regenerates benchmark evidence
- Full instructions live in `docs/e2e-validation.md` and `docs/migration-shadow.md`

## Run

```bash
cd bigclaw-go
go test ./...
go run ./cmd/bigclawd
curl localhost:8080/healthz
curl localhost:8080/metrics
```

## One-shot validation

Each `run_all.sh` invocation now creates a timestamped evidence bundle under `docs/reports/live-validation-runs/<run-id>/` so local, Kubernetes, and Ray logs stay isolated while the latest canonical reports remain easy to link in review notes.

```bash
cd bigclaw-go
export KUBECONFIG=/Users/jxrt/.kube/ray-local-config
export BIGCLAW_RAY_ADDRESS=ray://127.0.0.1:10001
export BIGCLAW_KUBERNETES_NAMESPACE=ray
export BIGCLAW_KUBERNETES_IMAGE=alpine:3.20
export BIGCLAW_QUEUE_BACKEND=sqlite
./scripts/e2e/run_all.sh
```
