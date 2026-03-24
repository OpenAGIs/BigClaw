# BIGCLAW-178

## Plan
- Inspect existing distributed policy, scheduler isolation, and reporting surfaces.
- Extend policy/runtime types to express tenant isolation and ownership constraints.
- Surface cross-tenant boundary details in scheduler reports/events.
- Add targeted tests to block invalid cross-tenant allocations and verify reporting.
- Run targeted test commands, record exact results, then commit and push.

## Acceptance
- Policy can express tenant isolation.
- Scheduling report shows cross-tenant boundary details.
- Tests prove violating allocations are blocked.

## Validation
- go test targeted scheduler/worker/reporting packages covering isolation policy and report output.
- Review git diff to keep scope limited to BIGCLAW-178.

## Results
- `go test ./internal/scheduler -run 'TestScheduler(EnforcesTenantIsolationAndOwnerMatch|TaskPolicyTightensIsolationAgainstSharedDefaults)' -count=1`
  - `ok  	bigclaw-go/internal/scheduler	1.350s`
- `go test ./internal/worker -run 'TestRuntime(PublishesRejectedDecisionHandoffBeforeRetry|PublishesTaskPolicyIsolationOnBlockedEvent)' -count=1`
  - `ok  	bigclaw-go/internal/worker	2.336s`
- `go test ./internal/api -run 'TestV2ControlCenterDistributedDiagnosticsShowIsolationBoundaries' -count=1`
  - `ok  	bigclaw-go/internal/api	2.926s`
