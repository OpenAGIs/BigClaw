## BIGCLAW-188 Workpad

### Plan

- Inspect the Go control-center payload and the checked-in capacity certification evidence to find the narrowest existing surface for a cross-batch throughput and cost comparison view.
- Add a deterministic comparison payload derived from the checked-in capacity certification soak lanes, keeping the change scoped to the control-center admission/evidence surface.
- Extend targeted Go tests so the new comparison view is regression-covered from the public `/v2/control-center` response.
- Run only the targeted Go tests affected by this slice, record the exact commands and outcomes, then commit and push `BIGCLAW-188`.

### Acceptance Criteria

- `/v2/control-center` exposes a cross-batch comparison view that lets operators compare checked-in batch lanes on throughput and cost-related metrics from repo-native evidence.
- The comparison view is deterministic, sourced from existing checked-in benchmark/capacity artifacts, and does not introduce a parallel evidence store or runtime dependency.
- Existing admission policy summary behavior remains intact while the new comparison view is covered by focused Go tests.

### Validation

- `cd bigclaw-go && go test ./internal/api -run 'TestV2ControlCenterIncludesAdmissionPolicySummary|TestV2ControlCenterIncludesCrossBatchThroughputAndCostComparison'`

### Notes

- Scope is intentionally limited to the Go control-center payload and its tests.
- Cost will be represented using deterministic batch-level evidence from the soak matrix so the comparison remains auditable from checked-in artifacts.
- Validation result: `cd bigclaw-go && go test ./internal/api -run 'TestV2ControlCenterIncludesAdmissionPolicySummary|TestV2ControlCenterIncludesCrossBatchThroughputAndCostComparison'` -> `ok  	bigclaw-go/internal/api	0.366s`
