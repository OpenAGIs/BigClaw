# BIG-GO-112 Workpad

## Plan

- Confirm which deleted Python test contracts from the residual backlog still lack dedicated replacement-registry artifacts.
- Add a scoped Go-native replacement registry for the remaining deferred contract cluster owned by this issue.
- Add regression coverage that proves the retired Python tests stay absent, the mapped Go/native replacements still exist, and the lane report stays aligned.
- Add a `bigclaw-go/docs/reports/` lane report capturing scope, replacement ownership, and validation evidence for `BIG-GO-112`.
- Run targeted Go validation for the new regression surface and record exact commands and results here and in the lane report.
- Commit the scoped issue changes and push the branch to the remote tracking branch.

## Acceptance

- `.symphony/workpad.md` exists before code edits and captures plan, acceptance, and validation for `BIG-GO-112`.
- `BIG-GO-112` adds only issue-scoped artifacts for the residual Python test cluster and does not modify unrelated sweeps.
- The repo contains a dedicated `BIG-GO-112` replacement-registry path, regression tests, and lane report for the chosen residual test cluster.
- Targeted regression validation passes with exact commands and results recorded.
- The branch is committed and pushed after validation succeeds.

## Validation

- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO112'`
- `cd bigclaw-go && go test -count=1 ./internal/migration ./internal/regression`
- `git status --short`

## Results

- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO112ResidualTestContractSweepM(ManifestMatchesRetiredTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  - `ok  	bigclaw-go/internal/regression	4.466s`
- `cd bigclaw-go && go test -count=1 ./internal/migration ./internal/regression`
  - `?   	bigclaw-go/internal/migration	[no test files]`
  - `FAIL	bigclaw-go/internal/regression`
  - Unrelated existing failure: `TestLiveShadowScorecardBundleStaysAligned` expected a different live-shadow generator script string and reported `GeneratorScript:go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- `git status --short`
  - `M .symphony/workpad.md`
  - `?? bigclaw-go/docs/reports/big-go-112-residual-test-contract-sweep-m.md`
  - `?? bigclaw-go/internal/migration/residual_test_contract_sweep_m.go`
  - `?? bigclaw-go/internal/regression/big_go_112_residual_test_contract_sweep_m_test.go`
