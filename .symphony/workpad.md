Issue: BIG-GO-1019

Plan
- Inspect the remaining `bigclaw-go/scripts/e2e/**` Python residue and pick a self-contained migration slice directly exercised by `scripts/e2e/run_all.sh`.
- Replace `export_validation_bundle.py`, `validation_bundle_continuation_scorecard.py`, and `validation_bundle_continuation_policy_gate.py` with Go-native `bigclawctl automation e2e` subcommands while preserving report shapes and workflow exit semantics.
- Update `scripts/e2e/run_all.sh`, targeted docs/report references, and regression tests to call the Go entrypoints directly, then remove the migrated Python files and their Python tests.
- Run targeted validation for the migrated continuation/export slice, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs/reports.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- `scripts/e2e/run_all.sh` no longer depends on the migrated Python continuation/export scripts.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `go test ./cmd/bigclawctl -run 'TestAutomation(E2E.*|RunAll.*|ExportValidationBundle.*|ValidationBundleContinuation.*)'`
- `go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `go run ./cmd/bigclawctl automation e2e continuation-scorecard --help`
- `go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
- `git diff --stat && git status --short`
