# BIG-GO-968 Workpad

## Plan

1. Lock the real batch-2 scope to the currently remaining `tests/**` Python files.
2. Map each file to an existing Go-native or Go-driven regression surface and decide delete, replace, or keep.
3. Add only the missing Go regression coverage needed to safely remove selected Python tests.
4. Delete the migrated Python tests and keep `conftest.py` only if still required by the remaining Python suite.
5. Run targeted `go test` and repository count checks, then record exact commands and results here.
6. Commit the scoped change set and push `BIG-GO-968` to `origin`.

## Batch 2 File List

- `tests/conftest.py`
- `tests/test_console_ia.py`
- `tests/test_design_system.py`
- `tests/test_evaluation.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_operations.py`
- `tests/test_reports.py`
- `tests/test_ui_review.py`

## Acceptance

- The batch-2 file list matches the actual remaining `tests/**` Python files in this workspace.
- Each file in this batch is explicitly classified as delete, replace, or keep with a concrete reason.
- The selected Python files are removed only after matching Go coverage exists.
- The workpad records exact validation commands and results.
- The final report includes the impact on `tests/**` Python file count and overall repository Python file count.

## Validation

- `go test ./internal/product ./internal/regression`
- `rg --files tests -g '*.py'`
- `rg --files | rg '\.py$' | wc -l`
- `git status --short`

## Notes

- Delete: Python tests already covered by existing Go-native or Go regression tests.
- Replace: Python tests migrated to new Go regression tests that exercise the same Python contract.
- Keep: Python tests still covering broader Python-owned report/evaluation/operations surfaces without a narrow Go replacement in this issue.

## Results

- Deleted:
  - `tests/test_live_shadow_bundle.py`
    - Reason: already covered by existing Go regression surfaces in `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`.

- Replaced with Go-driven regression coverage:
  - `tests/test_console_ia.py`
    - Reason: replaced by `bigclaw-go/internal/regression/python_console_design_contract_test.go` covering Console IA round-trip, gap auditing, rendered IA report, interaction-contract auditing, release-ready BIG-4203 draft, and frame-contract failure cases.
  - `tests/test_design_system.py`
    - Reason: replaced by `bigclaw-go/internal/regression/python_console_design_contract_test.go` covering design-system round-trip, audit scoring/gaps, top-bar audit/report contract, and information-architecture audit/report contract.
  - `tests/test_ui_review.py`
    - Reason: replaced by `bigclaw-go/internal/regression/python_ui_review_contract_test.go` covering UI review pack round-trip, incomplete-pack audit failures, release-ready BIG-4204 pack counts, rendered report/board/html contracts, and bundle export artifacts.
  - `tests/test_evaluation.py`
    - Reason: replaced by `bigclaw-go/internal/regression/python_evaluation_contract_test.go` covering benchmark runner scoring, replay mismatch reporting, suite comparison/report rendering, replay detail rendering, run replay index artifact links, and missing-report fallback behavior.

- Kept:
  - `tests/conftest.py`
    - Reason: still required for the remaining Python tests under `tests/`.
  - `tests/test_operations.py`
    - Reason: still covers broader Python operations/reporting/dashboard composition beyond the narrowed Go-native surfaces touched here.
  - `tests/test_reports.py`
    - Reason: still covers large Python-owned reporting builders, report writers, and closure/checklist composition that do not have a scoped direct Go replacement in this issue.

- Validation commands and results:
  - `go test ./internal/regression -run 'TestLane8Python(ConsoleIA|DesignSystem|UIReview)ContractStaysAligned|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned)'`
    - Result: `ok  	bigclaw-go/internal/regression	0.776s`
  - `go test ./internal/regression -run 'TestLane8PythonEvaluationContractStaysAligned|TestLane8Python(ConsoleIA|DesignSystem|UIReview)ContractStaysAligned|TestLiveShadow(ScorecardBundleStaysAligned|BundleSummaryAndIndexStayAligned)'`
    - Result: `ok  	bigclaw-go/internal/regression	1.246s`
  - `go test ./internal/product`
    - Result: `ok  	bigclaw-go/internal/product	(cached)`
  - `rg --files tests -g '*.py'`
    - Result: `tests/conftest.py`, `tests/test_operations.py`, `tests/test_reports.py`
  - `rg --files tests -g '*.py' | wc -l`
    - Result: `3`
  - `rg --files | rg '\.py$' | wc -l`
    - Result: `83`
  - Baseline counts before changes:
    - `tests/**` Python files: `8`
    - repository Python files: `88`
  - Impact after changes:
    - `tests/**` Python files: `8 -> 3` (`-5`)
    - repository Python files: `88 -> 83` (`-5`)
