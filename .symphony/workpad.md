# BIG-GO-1162 Workpad

## Plan
- Confirm the current repository baseline for real Python files and verify whether the `tests/*.py` tranche from this lane is already absent.
- Add issue-specific migration documentation and regression coverage for the retired root `tests/*.py` tranche and its supported Go-native replacements.
- Run targeted validation, capture exact command lines and results, then commit and push the scoped change set.

## Acceptance
- The `BIG-GO-1162` candidate Python test files remain absent from the repository.
- The supported Go replacement or compatibility paths for the covered test tranche are documented and regression-checked.
- `find . -name '*.py' | wc -l` is revalidated on this branch and remains at the repo's current zero-file floor.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(PythonTestTranche17Removed|BIGGO1162CandidatePythonTestsRemainDeleted|BIGGO1162MigrationDocsListGoReplacements)$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(PythonTestTranche17Removed|BIGGO1162CandidatePythonTestsRemainDeleted|BIGGO1162MigrationDocsListGoReplacements)$'` -> `ok  	bigclaw-go/internal/regression	0.839s`
- `git status --short` -> `M .symphony/workpad.md`, `M bigclaw-go/docs/go-cli-script-migration.md`, `M docs/go-cli-script-migration-plan.md`, `?? bigclaw-go/internal/regression/big_go_1162_python_test_sweep_test.go`

## Residual Risk
- The workspace already starts from `0` real `.py` files, so this issue can harden deletion and Go-replacement coverage for the retired test tranche but cannot numerically lower the count below zero.
