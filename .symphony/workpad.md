# BIG-GO-903 Workpad

## Plan

1. Inspect the existing Python `tests/` surface, Go `bigclaw-go/internal/...` test surfaces, and migration/readiness docs to identify the current harness split and repo-native documentation patterns.
2. Add a repo-native harness migration plan artifact that breaks the current pytest-heavy surface into Go unit/regression tests, golden/doc alignment coverage, and integration harness lanes, including first-batch implementation targets, validation commands, regression coverage, risks, and branch/PR guidance.
3. Link the new artifact from the relevant migration/readiness/index docs and add regression tests that keep the plan discoverable and structurally aligned.
4. Run targeted validation commands covering the new/updated tests, record exact commands and outcomes, then commit and push the scoped change set to a remote issue branch.

## Acceptance

- Produce an executable migration plan for the primary pytest surface split into Go test, golden, and integration harness lanes.
- Include a first-batch implementation or retrofit checklist with concrete repo paths.
- Document exact validation commands, regression surface, branch/PR recommendation, and risk framing.
- Keep the changes scoped to this issue and enforce them with repo-native tests.

## Validation

- `python3 -m pytest tests/test_followup_digests.py`
- `cd bigclaw-go && go test ./internal/regression`

## Results

- `python3 -m pytest tests/test_harness_migration_plan.py tests/test_followup_digests.py` -> `4 passed in 0.02s`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.912s`
