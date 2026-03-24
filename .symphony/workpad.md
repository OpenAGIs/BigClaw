# BIGCLAW-178

## Plan

1. Extend `bigclaw-go/internal/scheduler/policy_store.go` so distributed routing policy can express tenant isolation mode plus ownership requirements.
2. Enforce those rules in `bigclaw-go/internal/scheduler/scheduler.go` using task tenant/owner metadata and quota tenant context, with explicit rejection reasons for cross-tenant or owner-boundary violations.
3. Add distributed diagnostics fields and markdown rendering in `bigclaw-go/internal/api/distributed.go` so reports show tenant isolation boundaries and violations.
4. Add focused Go tests in scheduler/policy/api packages for policy parsing, accepted same-tenant ownership routing, blocked cross-tenant placement, and report visibility.
5. Run the narrowest relevant Go test targets, record exact commands and results below, then commit and push the branch.

## Acceptance

1. Scheduler policy definitions can express tenant isolation and ownership constraints.
2. Distributed diagnostics/report output indicates tenant-boundary enforcement or violations.
3. Tests prove invalid cross-tenant or ownership-breaking assignments are rejected.

## Validation

- Run only the narrowest Go test targets that exercise scheduler policy parsing/enforcement and distributed diagnostics output:
  - `go test ./internal/scheduler`
  - `go test ./internal/api`
- Record exact commands and pass/fail outcomes in this file after execution.

## Test Log

- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-178/bigclaw-go && go test ./internal/scheduler ./internal/api ./internal/worker`
  - `ok  	bigclaw-go/internal/scheduler	2.276s`
  - `ok  	bigclaw-go/internal/api	6.866s`
  - `ok  	bigclaw-go/internal/worker	5.692s`
