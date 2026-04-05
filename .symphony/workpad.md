# BIG-GO-1357 Workpad

## Plan

1. Inspect the current validation-script surface and confirm whether Python-file count can still decrease.
2. Replace the remaining `bigclaw-go/scripts/e2e/run_all.sh` orchestration body with a Go-native `bigclawctl automation e2e run-all` entrypoint while keeping compatibility for existing operators.
3. Update regression/docs coverage so the validation workflow points at the Go-native entrypoint and verifies the shim behavior.
4. Run targeted validation commands, capture exact commands and outcomes here, then commit and push the branch.

## Acceptance

- Keep changes scoped to the validation-script shrink pass.
- Since `find . -name '*.py' | wc -l` is already `0`, land a concrete Go/native replacement in git for the validation workflow.
- `scripts/e2e/run_all.sh` should shrink to a compatibility wrapper or be removed in favor of the Go-native command.
- Targeted Go tests covering the new command and regression/docs expectations pass.

## Validation

- Baseline: `find . -name '*.py' | wc -l`
- Targeted tests:
  - `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomationE2ERunAll|TestLegacyShimHelp|TestAutomationE2E'`
  - `cd bigclaw-go && go test ./internal/regression -run 'TestE2E|TestRootScriptResidualSweep'`
- Optional command smoke:
  - `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-all --help`

## Results

- Baseline Python count: `find . -name '*.py' | wc -l` -> `0`
- Landed Go-native replacement: `bigclawctl automation e2e run-all` now owns the live-validation orchestration; `bigclaw-go/scripts/e2e/run_all.sh` is reduced to an exec shim.
- Validation:
  - `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomationUsageListsBIGGO1160GoReplacements|TestAutomationE2ERunAllUsesGoBundleCommandsAndDefaultsHoldMode|TestRunAllShimDelegatesToGoRunAll'` -> `ok  	bigclaw-go/cmd/bigclawctl	2.267s`
  - `cd bigclaw-go && go test ./internal/regression -run 'TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok  	bigclaw-go/internal/regression	0.439s`
  - `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-all --help` -> `exit 0`, usage printed with the new `run-all` flags.
