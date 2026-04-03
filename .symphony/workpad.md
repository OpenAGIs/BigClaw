Issue: BIG-GO-1044

Plan
- Retire the residual UI migration-only Python modules in `src/bigclaw/console_ia.py`, `src/bigclaw/design_system.py`, and `src/bigclaw/ui_review.py`, because the repo already has Go-native replacements and tests under `bigclaw-go/internal/consoleia`, `bigclaw-go/internal/designsystem`, and `bigclaw-go/internal/uireview`.
- Update the remaining planning manifests that still point at those deleted Python sources so the release-control candidate references Go-native validation commands and Go-native evidence targets only.
- Add one scoped regression test under `bigclaw-go/internal/regression` that asserts the deleted Python files stay absent and the Go replacement files/tests stay present.
- Run targeted file-count checks and focused Go tests for `internal/planning` and `internal/regression`, then commit and push the branch.

Acceptance
- Changes stay scoped to the UI top-level Python purge tranche for `console_ia`, `design_system`, and `ui_review`.
- Repository `.py` file count decreases after the selected Python files are removed.
- No planning manifest or regression still points at the deleted UI Python files as active evidence.
- Go-native replacements remain present and are pinned by regression coverage.

Validation
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`
- `cd bigclaw-go && go test ./internal/planning -run 'Test(CandidateBacklogRoundTripPreservesManifestShape|CandidateBacklogRanksReadyItemsAheadOfBlockedWork|EntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence|EntryGateHoldsWhenV2BaselineIsMissingOrNotReady|RenderCandidateBacklogReportSummarizesBacklogAndGateFindings|BuildV3CandidateBacklogMatchesIssuePlanTraceability)' -count=1`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15' -count=1`
- `rg -n "src/bigclaw/(console_ia|design_system|ui_review)\.py|tests/test_(console_ia|design_system|ui_review)\.py" README.md docs bigclaw-go src scripts . -g '*.md' -g '*.go' -g '*.sh' -g '*.json' -g '*.yml'`
- `git diff --stat`
- `git status --short`
