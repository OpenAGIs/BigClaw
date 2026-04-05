# BIG-GO-1366 Workpad

## Plan

1. Inspect `bigclaw-go/scripts/e2e` and existing regression/report patterns for prior Python-removal refill lanes.
2. Add `BIG-GO-1366` regression coverage scoped to the `bigclaw-go/scripts/e2e` replacement surface.
3. Add the lane report documenting the Go or shell-native replacements and validation evidence.
4. Run targeted validation, then commit and push the change set.

## Acceptance

- Keep the issue scope limited to `bigclaw-go/scripts` e2e Python replacement sweep A.
- Land concrete Go or native replacement evidence in git for `bigclaw-go/scripts/e2e`, since the repository is already at zero `.py` files.
- Preserve repository reality that `find . -name '*.py' | wc -l` is `0`.
- Record exact validation commands and results in the issue report and final closeout.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1366'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationE2EScriptsStayGoOnly|TestAutomationE2EScriptRunAllUsesNativeEntrypoints'`
