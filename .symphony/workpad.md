# BIG-GO-1036 Workpad

## Plan
- Replace `tests/test_ui_review.py` with a scoped Go regression test that validates the checked-in UI review pack contract from `src/bigclaw/ui_review.py`.
- Keep the implementation narrow: no broad Go production port of `ui_review`, only Go-side regression coverage needed to remove the Python test file in this tranche.
- Update stale repo references that still name `tests/test_ui_review.py` so validation plans and evidence links remain accurate after deletion.
- Delete the now-unneeded Python test harness files in `tests/`.
- Run targeted Go validation, capture exact commands and results here, then commit and push the branch.

## Scoped Tranche
- `tests/test_ui_review.py`
- `tests/conftest.py`
- `bigclaw-go/internal/regression/ui_review_pack_contract_test.go`
- `src/bigclaw/planning.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/regression` and covers the deterministic review-pack contract that the deleted Python test guarded.
- Replacement coverage explicitly checks stable `src/bigclaw/ui_review.py` review-pack markers for:
  - `BIG-4204` pack identity and core counts
  - review summary, coverage, signoff, escalation, freeze, exception, workload, and timeline contract strings
  - bundle writer artifact path definitions
- Validation and evidence references no longer point to the deleted Python test file.
- Changes remain scoped to this tranche only.

## Validation
- `cd bigclaw-go && go test ./internal/regression -run TestUIReviewPackContractStaysAligned`
  - `ok  	bigclaw-go/internal/regression	0.965s`
- `python3 -m py_compile src/bigclaw/planning.py`
  - passed
- `rg --files tests . | rg '(^|/)tests/.*\.py$|(^|/)tests/conftest\.py$' || true`
  - no output
- `git status --short`
  - `M .symphony/workpad.md`
  - `M src/bigclaw/planning.py`
  - `D tests/conftest.py`
  - `D tests/test_ui_review.py`
  - `?? bigclaw-go/internal/regression/ui_review_pack_contract_test.go`

## Completed
- Added `bigclaw-go/internal/regression/ui_review_pack_contract_test.go` to preserve the deterministic `BIG-4204` review-pack contract in Go.
- Updated `src/bigclaw/planning.py` so the release-control candidate now points review-pack validation/evidence at the Go regression instead of the deleted Python test.
- Deleted `tests/test_ui_review.py` and the now-unused `tests/conftest.py`.
