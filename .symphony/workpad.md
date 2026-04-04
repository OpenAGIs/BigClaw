# BIG-GO-1165 Workpad

## Plan
- Inventory the issue candidate Python files and map each one to an existing Go command, Go package surface, or a new Go compatibility entrypoint that can replace it.
- Retire a large sweep of real Python assets in scope by deleting migrated Python files and updating regression coverage so the repo tracks the new Go-only surface.
- Validate the Go replacement and compatibility path with targeted `go test` runs plus repo-level residual checks, then commit and push the lane branch.

## Acceptance
- The issue candidate set is covered by this sweep, with real Python files retired from the repository rather than left as wrappers.
- Each retired Python entrypoint has a verified Go replacement or an updated regression/compatibility surface proving the repo-native Go path.
- `find . -name '*.py' | wc -l` decreases compared with the pre-change baseline of `138`.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./...`
- Additional targeted `go test` commands for any new or modified command/regression packages touched by this sweep.
- `git status --short`

## Results
- `find . -name '*.py' | wc -l` -> `108` after retiring the Go-backed E2E smoke and continuation helpers, down from the pre-change baseline of `138`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1165|TestExternalStoreValidationReportStaysAligned|TestCrossProcessCoordinationReadinessDocsStayAligned|TestBrokerValidationSummaryStaysAligned' -count=1` -> `ok  	bigclaw-go/internal/regression	0.490s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1165|TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned|TestExternalStoreValidationReportStaysAligned|TestCrossProcessCoordinationReadinessDocsStayAligned|TestBrokerValidationSummaryStaysAligned' -count=1` -> `ok  	bigclaw-go/internal/regression	0.944s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1165|TestLiveValidationIndexStaysAligned|TestRuntimeReportFollowUpDocsStayAligned|TestTakeoverFollowUpDigestReferences|TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned|TestExternalStoreValidationReportStaysAligned|TestCrossProcessCoordinationReadinessDocsStayAligned|TestBrokerValidationSummaryStaysAligned' -count=1` -> `ok  	bigclaw-go/internal/regression	0.839s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -count=1` -> `ok  	bigclaw-go/cmd/bigclawctl	0.494s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1165|TestE2EValidationDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveValidationSummaryStaysAligned|TestRuntimeReportFollowUpDocsStayAligned' -count=1` -> `ok  	bigclaw-go/internal/regression	0.518s`
- `python3 -m pytest -q tests/test_followup_digests.py` -> `.. [100%]`
