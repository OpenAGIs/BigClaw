# BIG-GO-1036 Validation

- Title: Go-replacement F: replace Python tests with Go tests tranche 1
- Branch: `symphony/BIG-GO-1036`
- Commit: `92e32f716b2df0a11b776c44977d209ad8eea0a2`

## Completed

- Added `bigclaw-go/internal/regression/ui_review_pack_contract_test.go` to preserve the deterministic `BIG-4204` UI review pack contract in Go.
- Updated `src/bigclaw/planning.py` so release-control review-pack validation/evidence points at the Go regression instead of the deleted Python test.
- Deleted `tests/test_ui_review.py`.
- Deleted `tests/conftest.py`.

## Python Deletions

- `tests/test_ui_review.py`
- `tests/conftest.py`

## Go Additions

- `bigclaw-go/internal/regression/ui_review_pack_contract_test.go`

## Validation

- `cd bigclaw-go && go test ./internal/regression -run TestUIReviewPackContractStaysAligned`
  - `ok  	bigclaw-go/internal/regression	0.965s`
- `python3 -m py_compile src/bigclaw/planning.py`
  - passed
- `rg --files tests . | rg '(^|/)tests/.*\.py$|(^|/)tests/conftest\.py$' || true`
  - no output
- `git ls-remote --heads origin symphony/BIG-GO-1036`
  - `92e32f716b2df0a11b776c44977d209ad8eea0a2 refs/heads/symphony/BIG-GO-1036`

## Scope Notes

- `pyproject.toml` and `setup.py` are absent in this checkout.
- This tranche stayed scoped to the remaining UI review Python test surface and its active planning references.
