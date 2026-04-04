# BIG-GO-1175 Workpad

## Plan
- Confirm the repository baseline for live `.py` files and inspect the targeted root/script migration surfaces that still carry Go-replacement evidence.
- Add scoped regression coverage that keeps `scripts/dev_bootstrap.sh` as a shell/Go validation helper and blocks reintroduction of retired root Python entrypoints.
- Update the Go migration docs so `BIG-GO-1175` records the zero-`.py` baseline and the concrete supported replacements for the retained shell helper.
- Run targeted validation commands, capture exact commands and results here, then commit and push the branch.

## Acceptance
- The repository remains at a zero-`.py` baseline as measured by `find . -name '*.py' | wc -l`.
- `scripts/dev_bootstrap.sh` is covered as a retained shell helper that only dispatches through Go-native entrypoints plus the optional legacy compile-check.
- Migration docs explicitly record `BIG-GO-1175` as concrete replacement evidence for the root helper sweep area.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(RootScriptResidualSweep|RootScriptResidualSweepDocs|BIGGO1175DevBootstrapStaysGoOnly|BIGGO1175DocsRecordDevBootstrapReplacementEvidence)$'`
- `bash scripts/ops/bigclawctl dev-smoke`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(RootScriptResidualSweep|RootScriptResidualSweepDocs|BIGGO1175DevBootstrapStaysGoOnly|BIGGO1175DocsRecordDevBootstrapReplacementEvidence)$'` -> `ok  	bigclaw-go/internal/regression	0.470s`
- `bash scripts/ops/bigclawctl dev-smoke` -> `smoke_ok local`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/docs/go-cli-script-migration.md`, `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`, and `docs/go-cli-script-migration-plan.md` modified as expected
