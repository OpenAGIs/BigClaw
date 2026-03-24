## BIGCLAW-180 Workpad

### Plan

- [x] Audit `bigclaw-go/internal/api/distributed.go` and keep the implementation scoped to the distributed diagnostics response and markdown export.
- [x] Add a machine-readable diagnostics snapshot section that summarizes the filtered task slice, state/risk/executor mix, and snapshot timing metadata.
- [x] Add a cross-task comparison section that compares tasks within the filtered slice on executor, state, risk, priority, timing, and event counts.
- [x] Extend markdown export coverage so snapshot and comparison data are included in `GET /v2/reports/distributed/export`.
- [x] Add targeted regression tests for JSON payloads and markdown export, then run only the relevant Go tests.
- [ ] Commit and push the issue branch `symphony/BIGCLAW-180`.

### Acceptance

- [x] `GET /v2/reports/distributed` returns a machine-readable `diagnostics_snapshot` section for the currently filtered task slice.
- [x] `GET /v2/reports/distributed` returns a machine-readable `cross_task_comparison` section derived from the same filtered slice.
- [x] `GET /v2/reports/distributed/export` renders both snapshot and cross-task comparison sections in markdown.
- [x] The new sections remain bounded to the active filter slice and do not change unrelated report surfaces.
- [x] Targeted Go tests cover JSON payload shape and markdown export content for the new sections.

### Validation

- [x] `cd bigclaw-go && go test ./internal/api -run 'TestV2DistributedReport(BuildsCapacityViewAndMarkdownExport|AppliesTimeWindowFiltersToResponseAndExport)'`
  Result: `ok  	bigclaw-go/internal/api	5.307s`
- [x] `cd bigclaw-go && go test ./internal/api -run 'TestV2DistributedReport(IncludesRetentionExpirySurface|IncludesProviderLiveHandoffIsolationSurface|IncludesBrokerBootstrapSurface|IncludesBrokerReviewBundle)'`
  Result: `ok  	bigclaw-go/internal/api	5.594s`
