# BIG-GO-1364 Workpad

## Plan

1. Add a Go-native migration registry for a scoped "tests legacy contract removal sweep A" slice covering already-retired Python contract tests and their checked-in replacements.
2. Add a dedicated `BIG-GO-1364` regression test that validates the registry contents, replacement paths, and lane report.
3. Add a lane report documenting baseline Python count, the replacement evidence, and the exact validation commands run.
4. Run targeted regression tests plus the repository Python-count command, then commit and push the branch.

## Acceptance

- Keep changes scoped to `BIG-GO-1364`.
- Land concrete Go/native replacement evidence in git because repository-wide Python count is already `0`.
- Preserve `find . -name '*.py' | wc -l` at `0`.
- Add a dedicated regression guard and report for the legacy contract test replacement sweep.
- Record exact validation commands and results.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1364LegacyContractTestReplacement'`

## Execution Notes

- 2026-04-05: The checked-out workspace was already at a repository-wide Python count of `0`, so this lane had to satisfy acceptance by landing concrete Go/native replacement evidence rather than reducing the `.py` count further.
- 2026-04-05: Added `bigclaw-go/internal/migration/legacy_contract_tests.go` as the Go-native replacement registry for legacy contract-test sweep A.
- 2026-04-05: Added `bigclaw-go/internal/regression/big_go_1364_legacy_contract_test_replacement_test.go` and `bigclaw-go/docs/reports/big-go-1364-legacy-contract-test-replacement.md` to lock the replacement map and evidence in place.
- 2026-04-05: Ran `find . -name '*.py' | wc -l` and observed `0`.
- 2026-04-05: Ran `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1364LegacyContractTestReplacement(ManifestMatchesRetiredTests|PathsExist|LaneReportCapturesReplacementState)$'` and observed `ok  	bigclaw-go/internal/regression	0.634s`.
