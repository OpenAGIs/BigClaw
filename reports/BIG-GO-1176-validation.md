# BIG-GO-1176 Validation

## Summary

`BIG-GO-1176` records concrete Go-only replacement evidence for the residual
live script surfaces that remain under `scripts/` and `bigclaw-go/scripts/`
after the repository reached a zero-`.py` baseline. This lane does not remove a
new physical Python file because none remain in the checkout; it hardens the
remaining shell/Go entrypoints and documents their supported Go-native
replacements.

## Changes

- added `bigclaw-go/internal/regression/big_go_1176_script_residual_sweep_test.go`
  to fail if Python assets reappear anywhere under the audited `scripts/` or
  `bigclaw-go/scripts/` trees and to pin the retained live script files to this
  issue's residual sweep contract
- updated `bigclaw-go/docs/go-cli-script-migration.md` to include
  `BIG-GO-1176`, the zero-`.py` baseline note, and the audited residual script
  surfaces with their Go-native replacements
- updated `docs/go-cli-script-migration-plan.md` so the repo-root migration plan
  explicitly captures the `BIG-GO-1176` residual live script surface and the
  retained shell wrapper paths

## Validation Commands

```bash
find . -name '*.py' | wc -l
cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1176ResidualScriptSurfacesStayPythonFree|BIGGO1176MigrationDocsCaptureGoOnlyResidualSweep|E2EScriptDirectoryStaysPythonFree|RootOpsDirectoryStaysPythonFree|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'
cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'
git status --short
```

## Validation Results

- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1176ResidualScriptSurfacesStayPythonFree|BIGGO1176MigrationDocsCaptureGoOnlyResidualSweep|E2EScriptDirectoryStaysPythonFree|RootOpsDirectoryStaysPythonFree|BIGGO1160CandidatePythonFilesRemainDeleted|BIGGO1160MigrationDocsListGoReplacements)$'` -> `ok  	bigclaw-go/internal/regression	0.496s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements|RunDevSmokeJSONOutput|RunCreateIssuesCreatesOnlyMissing)$'` -> `ok  	bigclaw-go/cmd/bigclawctl	0.679s`
- `git status --short` before commit -> `.symphony/workpad.md`, `bigclaw-go/docs/go-cli-script-migration.md`, `docs/go-cli-script-migration-plan.md`, and `bigclaw-go/internal/regression/big_go_1176_script_residual_sweep_test.go`

## Acceptance Note

The repository-wide Python count remains `0`, so this lane satisfies the issue
through committed replacement evidence rather than a further numeric decrease.
