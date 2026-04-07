# Task Protocol Spec

## Core Entities

### Task

The Go control plane task model is defined in `internal/domain/task.go` and carries:

- identity: `id`, `title`
- execution routing: `required_executor`, `command`, `args`, `entrypoint`, `container_image`, `working_dir`
- governance: `priority`, `budget_cents`, `risk_level`
- runtime controls: `execution_timeout_seconds`, `environment`, `metadata`, `runtime_env`
- timing: `created_at`, `updated_at`
- state: `queued`, `leased`, `running`, `retrying`, `succeeded`, `failed`, `cancelled`, `dead_letter`

### Lease

The queue lease model is defined in `internal/queue/queue.go`:

- `task_id`
- `worker_id`
- `expires_at`
- `attempt`
- `acquired_at`

### Event Envelope

The event model is defined in `internal/domain/task.go` and used throughout the API, recorder, and event bus:

- `id`
- `type`
- `task_id`
- `timestamp`
- `payload`

## State Transitions

The enforced task-state transitions live in `internal/domain/state_machine.go`.

Allowed examples include:

- `queued -> leased`
- `leased -> running`
- `running -> succeeded`
- `running -> retrying`
- `running -> dead_letter`
- `running -> cancelled`

Illegal transitions are rejected by `ValidateTransition`.

## Executor Contract

The executor contract is defined in `internal/executor/executor.go`:

- `Runner.Kind()` identifies executor type
- `Runner.Capability()` advertises runtime capability
- `Runner.Execute(context.Context, domain.Task)` returns a normalized result

The normalized result supports:

- success completion
- retryable failure
- dead-letter failure
- message and artifact propagation

## Replay and Query Surfaces

The task protocol is observable via:

- `GET /tasks/{id}` for status and event history
- `GET /events?task_id=...` for filtered timelines
- `GET /replay/{id}` for replay-oriented timelines
- `GET /stream/events` for SSE consumers
