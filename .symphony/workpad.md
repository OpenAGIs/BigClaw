# BIG-GO-926

## Plan
- Inventory the current Python and non-Go assets for `tests/test_reports.py`, `tests/test_validation*.py`, and `tests/test_evaluation.py`, then map each area to an existing or new Go package under `bigclaw-go`.
- Implement the first Go replacements in the smallest scoped packages that cover reporting, evaluation, and validation policy behavior needed by the migrated tests.
- Add Go tests that preserve the relevant behavioral coverage from the Python suite, remove migrated Python tests/assets when safe, and keep any untouched legacy assets explicit in the inventory.
- Run targeted regression commands for the new Go tests and any impacted existing tests, then commit and push the branch.

## Acceptance
- Current Python and non-Go assets in scope are explicitly listed in repo changes.
- Go replacement implementation or migration plan exists for reporting, evaluation, and validation policy coverage.
- First batch of Go implementation and migrated tests lands in this branch.
- Conditions for deleting old Python assets and exact regression commands are documented.

## Asset Inventory
- Deleted root Python test entrypoints now covered by Go:
  - `tests/test_reports.py`
  - `tests/test_evaluation.py`
  - `tests/test_validation_policy.py`
- New Go replacement coverage:
  - `bigclaw-go/internal/reporting/migration_suite.go`
  - `bigclaw-go/internal/reporting/migration_suite_test.go`
- Remaining Python / non-Go assets still in scope:
  - `tests/test_validation_bundle_continuation_policy_gate.py`
  - `tests/test_validation_bundle_continuation_scorecard.py`
  - `src/bigclaw/reports.py`
  - `src/bigclaw/evaluation.py`
  - `src/bigclaw/validation_policy.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - checked-in report fixtures under `bigclaw-go/docs/reports/validation-bundle-continuation*.json` and dependent live-validation docs

## Remaining Migration Plan
- `reports` / `evaluation` / `validation_policy` root Python test coverage is migrated into `bigclaw-go/internal/reporting`.
- `tests/test_validation_bundle_continuation_policy_gate.py` and `tests/test_validation_bundle_continuation_scorecard.py` remain because they still validate Python script entrypoints in `bigclaw-go/scripts/e2e/`.
- Safe deletion conditions for remaining Python assets:
  - delete root continuation wrapper tests only after equivalent Go tests cover the script behavior directly;
  - delete `src/bigclaw/reports.py` and `src/bigclaw/evaluation.py` only after their remaining importers under `tests/` are migrated or removed;
  - delete `src/bigclaw/validation_policy.py` only after package references are removed and the Go replacement is the sole execution path;
  - delete `bigclaw-go/scripts/e2e/*.py` only after the scripts themselves are replaced by Go executables/tests and docs/runbooks stop invoking Python.

## Validation
- `cd bigclaw-go && go test ./internal/reporting -count=1`
  - result: `ok  	bigclaw-go/internal/reporting	0.422s`
- `python3 -m pytest tests/test_validation_bundle_continuation_policy_gate.py tests/test_validation_bundle_continuation_scorecard.py -q`
  - result: `7 passed`
- `git status --short`
