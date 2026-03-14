# BigClaw Go Migration Plan

## Goals

- Keep the old implementation available during bootstrap
- Add a compatibility layer around task protocol and issue writeback
- Support shadow traffic before cutover
- Provide a fast rollback path by task type or tenant

## Phases

1. Freeze task protocol and state machine
2. Run Go control plane in shadow mode
3. Compare queue, routing, and completion outcomes
4. Shift low-risk task classes first
5. Expand to Kubernetes and Ray-backed workloads
6. Retire legacy scheduler only after parity evidence exists

## Rollback

- Disable Go dispatcher by config flag
- Stop new leases from the Go control plane
- Hand back eligible tasks to legacy scheduler
- Keep audit trail and replay logs for every shadow run
