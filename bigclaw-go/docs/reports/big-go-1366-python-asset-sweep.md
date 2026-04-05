# BIG-GO-1366 Python Asset Sweep

## Scope

Heartbeat refill lane `BIG-GO-1366` records the remaining physical Python asset inventory for the repository and lands concrete native replacement evidence for the `bigclaw-go/scripts/e2e` sweep-A surface.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

This checkout therefore lands as a native-replacement sweep rather than a direct Python-file deletion batch because there are no physical `.py` assets left to remove in-branch.

## Native Replacement Landed

- `bigclaw-go/scripts/e2e/run_all.sh` orchestrates the e2e sweep through `go run` invocations of `bigclawctl automation e2e ...` plus the retained shell smoke wrappers
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh` and `bigclaw-go/scripts/e2e/ray_smoke.sh` stay shell-native wrappers around `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` now fakes the `go` binary with a bash stub instead of an embedded `python3` helper

## Validation Commands And Results

- `find . -name '*.py' | wc -l`
  Result: `0`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1366(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|E2EReplacementPathsRemainAvailable|LaneReportCapturesNativeReplacement)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.521s`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationE2EScriptsStayGoOnly|TestAutomationE2EScriptRunAllUsesNativeEntrypoints|TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	1.522s`
