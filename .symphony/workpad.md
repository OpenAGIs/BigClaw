# BIG-GO-16 Workpad

## Plan
1. Normalize the restored workspace onto the `BIG-GO-16` branch from repository mainline.
2. Identify residual auxiliary Python support/example files relevant to batch B and determine whether each should be converted or deleted.
3. Apply the minimal scoped changes for this issue only, including any necessary reference cleanup.
4. Run targeted validation for the touched paths and capture exact commands and results.
5. Commit the changes on the issue branch and push to the remote branch.

## Acceptance
- The workspace is on a valid branch for `BIG-GO-16`.
- Residual Python support/example files in scope for batch B are removed or converted with no unrelated edits.
- References affected by those removals/conversions are updated consistently.
- Targeted validation commands pass, or any unavoidable failure is recorded precisely.
- Changes are committed and pushed to the remote branch for `BIG-GO-16`.

## Validation
- Inspect `git status` and `git diff --stat` to confirm the change set is scoped.
- Run targeted repository searches for the removed or converted Python paths to confirm no stale references remain.
- Run the smallest relevant automated tests or checks for the touched area and record exact commands plus results.
- Confirm `git status` is clean after commit and verify the push target branch on `origin`.

### Results
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-16/bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowRuntimeDocsStayAligned|LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|ProductionCorpusMatrixManifestAlignment|ProductionCorpusDriftRollupStaysAligned|ProductionCorpusDigestReferencesRemainIntact|Lane8FollowupDigestsStayAligned|Lane8ShadowMatrixCorpusCoverageStaysAligned|RollbackDocsStayAligned)$'`
  - Result: `ok  	bigclaw-go/internal/regression	0.197s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-16 && rg -n "examples/shadow-(corpus-manifest|task|task-budget|task-validation)\.json|scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard)\.py" bigclaw-go/docs bigclaw-go/scripts/migration/export_live_shadow_bundle bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go bigclaw-go/internal/regression/production_corpus_surface_test.go bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go -S`
  - Result: exit `1` with no matches
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-16 && git diff --stat`
  - Result: scoped to 21 files under `.symphony/` and `bigclaw-go/`, including deletion of the four `bigclaw-go/examples/shadow-*.json` fixtures
