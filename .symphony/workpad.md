Issue: BIG-GO-1028

Plan
- Retire `tests/test_observability.py` because the Go-owned observability and run-detail surfaces already cover ledger persistence, closeout serialization, repo sync audit rendering, detail/report rendering, and collaboration round-trips.
- Retire `tests/test_runtime_matrix.py` because the current Go scheduler and worker runtime tests already cover routing, executor lifecycle payloads, acceptance annotation, and orchestration/takeover event flow for the same product area.
- Keep the change set scoped to deleting these migrated Python tests plus the required workpad update only.
- Delete the migrated Python test files so this tranche reduces repository `.py` inventory immediately.
- Run targeted file-count checks and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletions.
- Repository `.py` file count decreases by deleting the two migrated Python test files.
- Repository `.go` file count remains unchanged.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/observability -run 'TestJSONLAuditSinkWritesEvents|TestMissingRequiredFieldsForEventUsesTopLevelAuditIdentifiers|TestMissingRequiredFieldsForEventReturnsSpecGaps|TestJSONLAuditSinkRejectsMalformedKnownAuditEvent|TestRecordSpecEventRejectsMalformedAuditEventsBeforeMutation|TestRecordSpecEventAcceptsWellFormedAuditEvents|TestTraceSummaryAggregatesTimeline|TestTraceSummariesReturnsMostRecentFirst|TestRecorderStoresTaskSnapshotsAndAppliesEventStates'`
- `cd bigclaw-go && go test ./internal/worker -run 'TestRuntimePublishesArtifactsAndWorkflowCloseoutOnCompletion|TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent|TestRuntimeAnnotatesAcceptedRuns|TestRuntimePublishesRejectedDecisionHandoffBeforeRetry'`
- `cd bigclaw-go && go test ./internal/api -run 'TestV2RunDetailExposesToolTraceArtifactsAuditAndReport|TestV2RunDetailCloseoutSummaryFromMetadata'`
- `git diff --stat`
- `git status --short`
