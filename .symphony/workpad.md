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
- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
- `bigclaw-go/internal/regression/live_shadow_docs_test.go`

## Plan
1. Keep the sweep scoped to active docs and regression guards that still mention retired Python tooling.
2. Replace stale Python migration-shadow commands with the active Go CLI entrypoints.
3. Remove stale root repository hygiene guidance that still points at `pre-commit`.
4. Retire any remaining active documentation that still presents Python validation commands as current workflow guidance.
5. Tighten regression coverage so these active docs do not regress back to retired Python tooling guidance.
6. Run targeted regression and grep validation, then commit and push the follow-up branch state.

## Acceptance
- Active docs no longer direct developers to retired Python tooling for migration-shadow helpers or root hygiene.
- Active docs no longer present retired Python validation commands as current workflow guidance.
- Regression coverage explicitly guards the touched active docs against reintroducing those references.
- Validation records exact commands and exact results for the targeted regression tests and focused residual-reference searches.
- Changes remain scoped to `BIG-GO-125`.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<" README.md bigclaw-go/docs/migration-shadow.md docs/go-mainline-cutover-handoff.md`
