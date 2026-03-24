# BIGCLAW-183

## Plan
- Inspect the distributed diagnostics heatmap and anomaly clustering implementation to find the remaining gap for the control-center issue scope.
- Tighten the cross-node execution/anomaly rollups only where behavior or report output is incomplete, without expanding beyond `bigclaw-go/internal/api`.
- Add targeted regression coverage for the missing behavior, including markdown/report rendering if the issue affects exported diagnostics output.
- Run focused Go API tests, record exact commands and outcomes here, then commit and push `BIGCLAW-183`.

## Acceptance
- `GET /v2/control-center` returns a cross-node execution heatmap in the distributed diagnostics payload.
- The payload exposes anomaly clusters derived from node/executor execution patterns and cross-node coordination signals.
- The diagnostics markdown/export output includes the cross-node heatmap and anomaly cluster sections with stable, reviewable content.
- Existing control-center/distributed diagnostics behavior remains intact for current fields.
- Targeted tests for the touched API behavior pass locally.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test ./internal/api -run 'TestV2ControlCenter(AppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|IncludesDistributedDiagnostics)'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test ./internal/api`

## Results
- `2026-03-24`: `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test -count=1 ./internal/api -run 'TestV2ControlCenter(AppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary|IncludesDistributedDiagnostics)'` -> `ok  	bigclaw-go/internal/api	5.631s`
- `2026-03-24`: `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-183/bigclaw-go && go test -count=1 ./internal/api` -> `ok  	bigclaw-go/internal/api	7.335s`
