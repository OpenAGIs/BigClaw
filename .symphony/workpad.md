# BIGCLAW-171

## Plan
- inspect the existing control-center worker pool summary and distributed diagnostics report paths
- add any missing worker-pool aggregate fields needed to expose executor distribution, node health, and capacity utilization cleanly in control-center payloads and diagnostics reports
- render worker pool summary plus node-level rollups inside the distributed diagnostics markdown so the control center has a report-backed view of the same data
- add focused unit coverage for worker pool summary computation
- add focused report rendering coverage for worker pool summary and node-aware markdown sections
- run targeted Go tests for the touched API package
- commit and push the scoped branch changes

## Acceptance
- control center payload exposes worker pool summary with node-level statistics
- payload includes executor distribution, node health, and capacity utilization
- distributed diagnostics report renders the worker pool summary and node-aware details
- new tests cover summary computation and report rendering

## Validation
- `cd bigclaw-go && go test ./internal/api -run 'TestWorkerPoolSummary|TestV2ControlCenterAppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|TestBuildDistributedDiagnosticsIncludesWorkerPoolSummary|TestRenderDistributedDiagnosticsMarkdownIncludesWorkerPoolSummary'`
- `cd bigclaw-go && go test ./internal/api`

## Validation Results
- `cd bigclaw-go && go test ./internal/api -run 'TestWorkerPoolSummary|TestV2ControlCenterAppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|TestBuildDistributedDiagnosticsIncludesWorkerPoolSummary|TestRenderDistributedDiagnosticsMarkdownIncludesWorkerPoolSummary'` -> `ok   bigclaw-go/internal/api 1.636s`
- `cd bigclaw-go && go test ./internal/api` -> `ok   bigclaw-go/internal/api 2.767s`
