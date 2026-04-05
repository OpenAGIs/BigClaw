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
