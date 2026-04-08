# BIG-GO-125 Workpad

## Context
- Issue: `BIG-GO-125`
- Title: `Residual tooling Python sweep H`
- Goal: remove or replace residual Python tooling guidance that still appears in active developer-facing tooling and migration helper surfaces.
- Current repo state on entry: repository-wide physical `.py` inventory is already `0`, so this lane is focused on active residual references rather than deleting live Python source files.

## Scope
- `.symphony/workpad.md`
- `README.md`
- `docs/go-mainline-cutover-handoff.md`
- `docs/symphony-repo-bootstrap-template.md`
- `docs/parallel-refill-queue.md`
- `docs/go-cli-script-migration-plan.md`
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
- `bigclaw-go/docs/go-cli-script-migration.md`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/docs/reports/live-validation-index.json`
- `bigclaw-go/docs/reports/live-validation-summary.json`
- `bigclaw-go/docs/reports/ray-live-smoke-report.json`
- `bigclaw-go/docs/reports/ray-live-jobs.json`
- `bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md`
- `bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
- `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
- `bigclaw-go/internal/regression/live_shadow_docs_test.go`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/internal/regression/live_validation_summary_test.go`
- `bigclaw-go/internal/regression/live_validation_index_test.go`

## Plan
1. Keep the sweep scoped to active docs and regression guards that still mention retired Python tooling.
2. Replace stale Python migration-shadow commands with the active Go CLI entrypoints in both active docs and generated reviewer indexes that remain checked in.
3. Remove stale root repository hygiene guidance that still points at `pre-commit`.
4. Retire any remaining active documentation that still presents Python validation commands as current workflow guidance.
5. Align checked-in live-validation reviewer artifacts with the shell-native Ray smoke default already documented elsewhere in the repo.
6. Normalize skipped live-validation bundles so they do not imply Ray smoke report artifacts were produced when the lane was disabled.
7. Sweep adjacent checked-in reviewer evidence that still embeds inline-Python Ray entrypoints outside the main live-validation summary/index surfaces.
8. Remove any remaining live template wording that still frames current bootstrap guidance in Python-specific terms.
9. Normalize any remaining active planning surfaces that still phrase the default workflow around Python-specific paths instead of Go-first legacy wording.
10. Normalize maintained compatibility reports so they use Go-first legacy runtime wording instead of Python-specific mainline phrasing where archival specificity is not required.
11. Tighten any remaining active Go CLI migration wording so current-state replacement sections use generic legacy-script language instead of unnecessary Python-specific phrasing.
12. Tighten the root Go CLI migration plan so active replacement bullets use generic legacy-script wording instead of unnecessary Python-specific labels.
13. Tighten regression coverage so these active docs and reviewer artifacts do not regress back to retired Python tooling guidance or misleading skipped-lane report links.
14. Run targeted regression and grep validation, then commit and push the follow-up branch state.

## Acceptance
- Active docs and checked-in reviewer indexes/summaries no longer direct developers to retired Python tooling for migration-shadow helpers or root hygiene.
- Checked-in live-validation reviewer artifacts no longer advertise the retired inline-Python Ray smoke default where the active docs already require the shell-native replacement.
- Checked-in skipped live-validation bundles no longer point reviewers at Ray smoke report artifacts that were never produced.
- Adjacent checked-in reviewer evidence no longer embeds stale inline-Python Ray entrypoints outside the canonical live-validation summary/index surfaces.
- Live bootstrap templates no longer frame the current workspace bootstrap path as Python-specific guidance.
- Active refill planning guidance no longer frames the default repo workflow around Python-specific path language.
- Maintained compatibility reports no longer frame the current mainline contract around Python-specific runtime wording where generic legacy wording is sufficient.
- Active Go CLI migration docs no longer use unnecessary Python-specific phrasing in current-state replacement sections when generic legacy-script wording is sufficient.
- The root Go CLI migration plan no longer uses unnecessary Python-specific phrasing in current-state replacement bullets when generic legacy-script wording is sufficient.
- Active docs no longer present retired Python validation commands as current workflow guidance.
- Regression coverage explicitly guards the touched active docs against reintroducing those references.
- Validation records exact commands and exact results for the targeted regression tests and focused residual-reference searches.
- Changes remain scoped to `BIG-GO-125`.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'ray-live-smoke-report.json|executor disabled; no Ray smoke report was produced for this bundle' bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c|sh -c '\''echo (gpu via ray|required ray|ray driver snapshot)'\''' bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i '\bpython\b|workspace_bootstrap\.py|workspace_bootstrap_cli\.py' docs/symphony-repo-bootstrap-template.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'Python paths are migration-only unless explicitly marked otherwise|legacy migration-only paths stay out of the default developer workflow unless explicitly marked otherwise' docs/parallel-refill-queue.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'legacy Python runtime surface|legacy pre-cutover runtime surface' bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'Python-side tests|Python script output|retired script-side tests|legacy script output' bigclaw-go/docs/go-cli-script-migration.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'retired the refill Python wrapper|retired the refill wrapper|retired benchmark Python helpers|retired benchmark script helpers|retired migration Python helpers|retired migration script helpers|root Python workspace shims|root workspace shims' docs/go-cli-script-migration-plan.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && git push origin BIG-GO-125`

## Final Blocker
- None.
