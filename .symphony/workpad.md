## BIG-GO-907 Workpad

### Plan

- [x] Audit the remaining non-Go validation/reporting tooling and lock the first migration slice to the validation bundle continuation scorecard and continuation policy gate.
- [x] Implement repo-native Go generation for the continuation scorecard and continuation policy gate.
- [x] Update repo docs and migration reporting so the first-batch implementation list, validation commands, regression surface, branch/PR guidance, and risks are recorded in-repo.
- [x] Run targeted validation commands and capture exact outcomes.
- [ ] Commit and push the scoped branch.

### Acceptance

- [x] An executable migration plan exists in-repo and identifies the first implementation batch.
- [x] The validation bundle continuation scorecard and continuation policy gate run through Go tooling.
- [x] Validation commands and regression surface are explicit in repo docs.
- [x] Branch/PR recommendation and concrete risks are documented in-repo.

### Validation

- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-907/bigclaw-go && go test ./internal/migration ./cmd/bigclawctl`
  Result: passed
- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-907 && python3 -m pytest bigclaw-go/scripts/e2e/run_all_test.py -q`
  Result: passed
- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-907/bigclaw-go && go test ./internal/regression -run TestE2EValidationDocsStayAligned`
  Result: passed
- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-907/bigclaw-go && go run ./cmd/bigclawctl migration validation-continuation-scorecard --output docs/reports/validation-bundle-continuation-scorecard.json`
  Result: passed
- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-907/bigclaw-go && go run ./cmd/bigclawctl migration validation-continuation-policy-gate --scorecard docs/reports/validation-bundle-continuation-scorecard.json --output docs/reports/validation-bundle-continuation-policy-gate.json --max-latest-age-hours 9999`
  Result: passed

### Notes

- Scope is intentionally limited to the continuation scorecard/gate path and migration planning/reporting updates needed by `BIG-GO-907`.
- Remaining Python migration backlog for bundle export, live-shadow export, and evaluation-heavy helpers is recorded in `bigclaw-go/docs/reports/validation-reporting-go-migration-plan.md`.
