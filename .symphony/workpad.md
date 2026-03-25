# BIGCLAW-198 Workpad

## Plan
1. Add a Go-native parallel tenant reconciliation report for cost, throughput, and success rate aggregation.
2. Expose the report through a focused API route with markdown export.
3. Add targeted reporting and API coverage for tenant totals and rendered output.
4. Run focused validation, capture exact commands and outcomes, then commit and push the issue branch.

## Acceptance
- The implementation is scoped to the parallel tenant reconciliation reporting slice.
- The report exposes tenant-level cost, throughput, and success-rate metrics plus reconciled totals.
- The new reporting surface is reachable through the Go API and exportable as markdown.
- Targeted automated tests pass for the touched reporting and API packages.

## Validation
- `cd bigclaw-go && go test ./internal/reporting ./internal/api`
  Result: passed
- `gofmt -w bigclaw-go/internal/reporting/reporting.go bigclaw-go/internal/reporting/reporting_test.go bigclaw-go/internal/api/expansion.go bigclaw-go/internal/api/expansion_test.go bigclaw-go/internal/api/server.go`
  Result: completed

## Notes
- Added `ParallelTenantReconciliationReport` builder/render/write support in `bigclaw-go/internal/reporting`.
- Added `/v2/reports/parallel/tenant-reconciliation` and `/v2/reports/parallel/tenant-reconciliation/export` in the Go API.
- Added focused tests for tenant cost, throughput, success-rate aggregation, and markdown export behavior.
