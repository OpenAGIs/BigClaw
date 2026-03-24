# BIGCLAW-183

## Plan
- Inspect the control-center distributed diagnostics payload and tests to identify the narrowest extension point for cross-node execution visibility.
- Add a node-aware execution heatmap and anomaly clustering surface to the existing control-center distributed diagnostics response.
- Cover the new response shape with targeted API tests and keep markdown/export output aligned if the new diagnostics are rendered there.
- Run targeted Go tests for the touched API package, record exact commands and results, then commit and push the issue branch.

## Acceptance
- `GET /v2/control-center` returns a cross-node execution heatmap in the distributed diagnostics payload.
- The new diagnostics expose anomaly clusters derived from node/executor execution patterns and coordination signals.
- Existing control-center/distributed diagnostics behavior remains intact for current fields.
- Targeted tests cover the new payload and pass locally.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test ./internal/api -run 'TestV2ControlCenter(AppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|IncludesDistributedDiagnostics)'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test ./internal/api`

## Results
- `2026-03-24`: `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test -count=1 ./internal/api -run 'TestV2ControlCenter(AppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|IncludesDistributedDiagnostics)'` -> `ok  	bigclaw-go/internal/api	3.371s`
- `2026-03-24`: `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test -count=1 ./internal/api` -> `ok  	bigclaw-go/internal/api	5.079s`
