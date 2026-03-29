# BIG-GO-948 Workpad

## Plan

1. Inventory the remaining `tests/**` Python files and map them against existing `bigclaw-go` Go tests to identify the lane-owned files that still lack Go coverage.
2. Inspect the selected Python tests and the corresponding Go packages to choose the smallest scoped migration slice that can be completed end-to-end in this issue.
3. Implement the missing Go tests or, where direct migration is out of scope, document the concrete deletion or follow-up plan in-repo while keeping changes limited to this lane.
4. Remove the migrated Python test assets that now have Go replacements and keep any untouched Python tests outside this lane unchanged.
5. Run targeted validation commands for the touched Go packages, record exact commands and results, then commit and push the branch.

## Acceptance

- Produce an explicit file list for the `BIG-GO-948` lane.
- Land Go test replacements for the selected remaining Python tests, or document a concrete delete/follow-up plan for any files that cannot be removed in this lane.
- Record exact validation commands, results, and residual risks.
- Reduce Python / non-Go test assets in the repository without widening scope beyond this issue.

## Validation

- `go test` for the exact `bigclaw-go` packages touched by this lane.
- Targeted execution of any new or expanded Go tests covering the migrated Python scenarios.
- `git status --short` to verify the scoped file set before commit.

## Results

- Migrated 13 Python tests to Go-owned coverage and deleted the Python files:
  - `test_cross_process_coordination_surface.py`
  - `test_followup_digests.py`
  - `test_live_shadow_scorecard.py`
  - `test_shadow_matrix_corpus.py`
  - `test_subscriber_takeover_harness.py`
  - `test_validation_bundle_continuation_scorecard.py`
  - `test_parallel_refill.py`
  - `test_roadmap.py`
  - `test_cost_control.py`
  - `test_deprecation.py`
  - `test_legacy_shim.py`
  - `test_service.py`
- Added Go replacements in:
  - `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`
  - `bigclaw-go/internal/refill/queue_repo_fixture_test.go`
  - `bigclaw-go/internal/regression/roadmap_contract_test.go`
  - `bigclaw-go/internal/regression/deprecation_contract_test.go`
  - `bigclaw-go/internal/costcontrol/controller.go`
  - `bigclaw-go/internal/costcontrol/controller_test.go`
  - `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
  - `bigclaw-go/internal/legacyshim/wrappers.go`
  - `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`
  - `bigclaw-go/internal/service/server.go`
  - `bigclaw-go/internal/service/server_test.go`
  - `bigclaw-go/internal/pilot/report.go`
  - `bigclaw-go/internal/pilot/report_test.go`
- Pushed commits:
  - `b59e941` `test: migrate lane8 remaining python report tests`
  - `cfcd50e` `test: migrate parallel refill queue fixture to go`
  - `868b503` `test: migrate execution pack roadmap checks to go`
  - `911a1d6` `docs: record remaining python test migration plan`
  - `bdd3aa4` `test: migrate cost control checks to go`
  - `0334358` `test: migrate deprecation compatibility checks to go`
  - `29553fc` `test: migrate legacy shim contracts to go`
- Remaining Python tests in `tests/` now require broader Go-native implementation or new contract surfaces rather than direct fixture parity moves.
- Next scoped slice: migrate `tests/test_pilot.py` into a small Go-native pilot package that covers KPI pass-rate math and report rendering without pulling over the broader Python workflow runtime.
- Migrated `tests/test_pilot.py` to `bigclaw-go/internal/pilot/report.go` and `bigclaw-go/internal/pilot/report_test.go`; deleted the Python test after landing equivalent Go coverage for KPI readiness and report rendering.
- Validation result:
  - `cd bigclaw-go && go test ./internal/pilot -run 'TestImplementationResultReadyWhenKPIsPassAndNoIncidents|TestRenderPilotImplementationReportContainsReadinessFields'` -> `ok  	bigclaw-go/internal/pilot	0.789s`
