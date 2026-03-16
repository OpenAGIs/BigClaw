# Worker Runtime Closeout Digest

## Purpose

This digest is the canonical closeout surface for the Go control-plane worker lifecycle, task state machine, and task protocol contract.

It replaces scattered closeout claims with one repo-native review point backed by current code and automated tests.

## Automated Evidence

- `internal/worker/runtime.go`
- `internal/worker/runtime_test.go`
- `internal/domain/task.go`
- `internal/domain/state_machine.go`
- `internal/domain/state_machine_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`

## Runtime Contract

### Task States

The runtime contract currently uses these task states from `internal/domain/task.go`:

- `queued`
- `leased`
- `running`
- `blocked`
- `retrying`
- `succeeded`
- `failed`
- `cancelled`
- `dead_letter`

### Allowed State Transitions

The enforced transition map in `internal/domain/state_machine.go` currently allows:

- `queued -> leased`
- `queued -> cancelled`
- `leased -> running`
- `leased -> retrying`
- `leased -> cancelled`
- `running -> succeeded`
- `running -> failed`
- `running -> blocked`
- `running -> retrying`
- `running -> cancelled`
- `running -> dead_letter`
- `blocked -> retrying`
- `blocked -> cancelled`
- `retrying -> queued`
- `retrying -> dead_letter`
- `retrying -> cancelled`
- `failed -> retrying`
- `failed -> dead_letter`

Illegal transitions are rejected by `ValidateTransition`.

### Event To State Projection

The externally visible task state projection in `TaskStateFromEventType` currently maps:

- `task.queued -> queued`
- `task.leased -> leased`
- `scheduler.routed -> leased`
- `task.started -> running`
- `task.retried -> retrying`
- `task.completed -> succeeded`
- `task.preempted -> cancelled`
- `task.cancelled -> cancelled`
- `task.dead_lettered -> dead_letter`

## Verified Lifecycle Behaviors

Automated coverage in `internal/worker/runtime_test.go` and `internal/api/server_test.go` verifies that:

- runnable work progresses through `queued -> leased -> running -> succeeded`
- long-running work renews queue leases while executing
- missing trace IDs default to the task ID for event correlation
- timeouts requeue work with a retry event instead of dropping it
- in-flight cancellation completes as `cancelled` and updates worker counters
- missing executor registrations move work to `dead_letter`
- control-plane pause leaves work queued and exposes a `paused` worker snapshot
- active human takeover requeues work, records a takeover audit event, and stores the task as `blocked`
- urgent work can preempt a lower-priority run and surfaces both preemption and cancellation evidence
- runtime events include executor selection, required tools, completion messages, and artifact references

## Stable Review Surfaces

The current repo-native review surfaces for this contract are:

- `GET /tasks/{id}` for projected task status and task metadata
- `GET /events?task_id=...` for filtered task timelines
- `GET /events?trace_id=...` for trace-scoped timelines
- `GET /replay/{id}` for replay-oriented task history with delivery metadata
- `GET /stream/events` for SSE consumers
- `GET /debug/status` for worker snapshot, worker-pool snapshot, control snapshot, event durability, event-log backend capabilities, retention watermark, and checkpoint reset diagnostics

The worker snapshot returned through `GET /debug/status` is currently expected to expose:

- `worker_id`
- `state`
- `current_task_id`
- `current_trace_id`
- `current_executor`
- `last_heartbeat_at`
- `last_started_at`
- `last_finished_at`
- `last_result`
- `lease_renewals`
- `successful_runs`
- `retried_runs`
- `dead_letter_runs`
- `cancelled_runs`
- `preemption_active`
- `current_preemption_task_id`
- `current_preemption_worker_id`
- `last_preempted_task_id`
- `last_preemption_at`
- `last_preemption_reason`
- `preemptions_issued`
- `last_transition`

## Focused Companion Reports

These focused reports remain in place, but this digest is the canonical closeout reference:

- `docs/reports/worker-lifecycle-validation-report.md`
- `docs/reports/state-machine-validation-report.md`
- `docs/reports/task-protocol-spec.md`
