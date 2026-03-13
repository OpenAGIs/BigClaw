# BigClaw v2.0 Phase 1 Operations Foundation Report

Date: 2026-03-13

## Scope

This change set establishes a backend-facing Phase 1 foundation for BigClaw v2.0 planned work inside `bigclaw-go`.

Primary issue alignment:
- `OPE-69` / `BIG-801`: engineering dashboard aggregation
- `OPE-70` / `BIG-802`: queue and control-center operations
- `OPE-71` / `BIG-803`: premium orchestration policy surface
- `OPE-72` / `BIG-804`: run detail and replay-oriented data plane
- Partial `OPE-73` / `BIG-805`: human takeover and collaboration notes

## Delivered backend surfaces

### Control foundation
- Added `internal/control/controller.go`
- Supports:
  - global pause / resume state
  - per-task human takeover records
  - reviewer / owner tracking
  - collaboration notes timeline
  - active takeover listing for operational views
  - transfer-to-human alias handling in the control-center action surface

### Premium policy surface
- Added `internal/policy/policy.go`
- Resolves per-task orchestration plans into:
  - `standard` vs `premium`
  - dedicated queue lane
  - concurrency profile
  - advanced approval flag
  - multi-agent graph eligibility
  - dedicated browser / VM pool flags
  - isolation mode and routing reason

### Task snapshot persistence
- Extended `internal/observability/recorder.go`
- Recorder now keeps durable in-memory task snapshots alongside event history
- Task snapshots survive queue `Ack` and power:
  - dashboard aggregation
  - run detail retrieval
  - control-center recent task views
  - task lifecycle recovery after replay / takeover actions

### v2 API endpoints
- Added `GET /v2/dashboard/engineering`
- Added `GET /v2/control-center`
- Added `GET /v2/control-center/audit`
- Added `POST /v2/control-center/actions`
- Added `GET /v2/runs/{task_id}`
- Added `GET /v2/runs/{task_id}/audit`
- Added `GET /v2/runs/{task_id}/report`
- Added queue-backed task inspection in the control center for live priority / lease / worker visibility
- Added filtered queue / risk / budget / priority summaries plus worker-pool packaging for the operations surface

### Worker runtime control integration
- Extended `internal/worker/runtime.go`
- Runtime now:
  - skips leasing when the control plane is paused
  - requeues tasks under active human takeover
  - records takeover deferral in audit history

## API intent summary

### `GET /v2/dashboard/engineering`
Provides:
- team / project / tenant / time-window filtering
- active runs
- blockers
- budget totals
- SLA-risk counts
- premium plan counts
- ticket → PR → merge funnel
- recent task overviews with policy context

### `GET /v2/control-center`
Provides:
- control-plane pause state
- queue depth and filtered queue views
- budget / risk / priority summaries for the live queue
- dead-letter inventory
- active takeovers
- recent task list
- worker-pool summary when available
- recent control-action audit entries
- authorization envelope with the caller role and allowed mutating actions

### `POST /v2/control-center/actions`
Supports:
- `pause`
- `resume`
- `replay_deadletter` / `retry`
- `cancel`
- `takeover` / `transfer_to_human`
- `release_takeover`
- `annotate`

### `GET /v2/control-center/audit`
Provides:
- filtered action audit retrieval by task, actor, or action type
- normalized action names for pause / resume / retry / cancel / takeover flows
- role-tagged action history for downstream operational review
- direct payload reuse from recorded audit events

### `GET /v2/runs/{task_id}`
Provides:
- task snapshot
- lifecycle state and derived failure reason
- policy summary
- collaboration / takeover context
- trace summary
- timeline / events
- validation status, acceptance criteria, and validation plan
- tool traces across declared tools, scheduler routing, and executor lifecycle
- artifact references covering executor outputs, workpad, issue, and PR links
- run-scoped audit summary / notes timeline
- downloadable markdown run report and replay / event / trace links
- workpad metadata

## Validation

Command run:
- `go test ./...`

Result:
- all packages passed on 2026-03-13

## Key implementation references
- `internal/api/server.go`
- `internal/api/v2.go`
- `internal/control/controller.go`
- `internal/policy/policy.go`
- `internal/observability/recorder.go`
- `internal/worker/runtime.go`
