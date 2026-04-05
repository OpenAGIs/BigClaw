# BIG-GO-1361

## Plan
- Inspect bigclaw-go core module paths for stale Python-era references and identify a scoped Go/native cleanup.
- Update only the relevant core sweep artifacts to reflect repository reality and Go/native replacements.
- Run targeted validation for touched areas and record exact commands and outcomes.
- Commit and push the changes to the remote branch.

## Acceptance
- Keep changes scoped to BIG-GO-1361.
- Because `find . -name '*.py' | wc -l` is already 0 in this workspace, satisfy acceptance via concrete Go/native replacement evidence landing in git.
- Remove or replace stale Python-oriented core module references where found.

## Validation
- `find . -name '*.py' | wc -l`
- `rg -n "python|\\.py|src/bigclaw" bigclaw-go .github scripts docs`
- Targeted tests/builds for the touched Go/native area.

## Execution Notes
- 2026-04-05: The checked-out workspace already had `0` physical `.py` files, so this lane follows the concrete Go/native replacement-evidence path.
- 2026-04-05: Added `bigclaw-go/internal/migration/legacy_core_modules.go` to record the retired `src/bigclaw` core modules and their active Go owners.
- 2026-04-05: Added `bigclaw-go/internal/regression/big_go_1361_legacy_core_module_replacement_test.go` and `bigclaw-go/docs/reports/big-go-1361-legacy-core-module-replacement.md` as the lane guard and report.
- 2026-04-05: Ran `find . -name '*.py' | wc -l` and observed `0`.
- 2026-04-05: Ran `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1361LegacyCoreModuleReplacement(ManifestMatchesRetiredModules|ReplacementPathsExist|LaneReportCapturesReplacementState)$'` and observed `ok  	bigclaw-go/internal/regression	1.082s`.
