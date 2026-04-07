# BIG-GO-1565

## Plan

- Replace `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` with a Go implementation under `bigclaw-go/scripts/e2e`.
- Replace `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` with a Go implementation under `bigclaw-go/scripts/e2e`.
- Update local callers/tests/docs that reference the deleted Python continuation scripts so the repo uses the Go/native replacements.
- Delete the two Python scripts and the Python-only gate test once equivalent Go coverage is in place.

## Acceptance

- `find . -name '*.py' | wc -l` decreases from the current baseline.
- `bigclaw-go/scripts/e2e/run_all.sh` and checked-in references no longer depend on the deleted continuation Python scripts.
- Go tests cover the continuation scorecard/gate behavior that was previously exercised by the Python gate test.

## Validation

- Baseline: `find . -name '*.py' | wc -l` -> `138`
- Final: `find . -name '*.py' | wc -l` -> `135`
- `cd bigclaw-go && go test ./scripts/e2e/validationbundle/...` -> `ok  	bigclaw-go/scripts/e2e/validationbundle	2.190s`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_scorecard/main.go --output /tmp/bigclaw-1565-scorecard.json` -> exit `0`
- `cd bigclaw-go && go run ./scripts/e2e/validation_bundle_continuation_policy_gate/main.go --scorecard /tmp/bigclaw-1565-scorecard.json --output /tmp/bigclaw-1565-gate.json --enforcement-mode review` -> exit `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveValidationIndexStaysAligned|TestLiveValidationIndexContinuationMetadata|TestContinuationPolicyGateReviewerMetadata'` -> `ok  	bigclaw-go/internal/regression	0.188s`
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py` -> `Ran 3 tests in 2.503s` / `OK`
