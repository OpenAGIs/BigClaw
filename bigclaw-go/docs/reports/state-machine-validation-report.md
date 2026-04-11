# State Machine Validation Report

## Objective

Validate that the unified Go task state machine accepts legal transitions and rejects illegal transitions.

## Evidence

Primary automated coverage lives in:

- `internal/domain/state_machine_test.go`
- `internal/api/server_test.go`
- `internal/worker/runtime_test.go`

## Verified Behaviors

- `queued -> leased` is accepted
- `leased -> running` is accepted
- `running -> succeeded` is accepted
- illegal jumps such as `queued -> succeeded` are rejected
- API status responses translate event types back into externally visible task states
- worker runtime emits lifecycle events in queue -> lease -> start -> terminal order

## Result

- The current task-state contract is stable enough for scheduler, worker, API, recorder, and executor integration.
- Illegal transition rejection is covered by automated tests.
- Event-driven status projection matches the internal state model.
