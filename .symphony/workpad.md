# BIG-GO-146 Workpad

## Plan

1. Inspect existing Python-sweep regression/report patterns and identify the
   residual support-asset surfaces that match this lane: examples, fixtures,
   demos, and support helpers.
2. Add a focused regression test covering the targeted residual surfaces and
   required Go/native replacement paths.
3. Add the matching lane report under `bigclaw-go/docs/reports/` documenting
   the audited directories, current inventory, and validation commands/results.
4. Run targeted validation commands, record exact commands and outcomes, then
   commit and push the scoped changes.

## Acceptance

- A new `BIG-GO-146` regression test exists under
  `bigclaw-go/internal/regression/`.
- A new `BIG-GO-146` lane report exists under `bigclaw-go/docs/reports/`.
- The lane remains scoped to residual support assets Python sweep coverage.
- Targeted regression tests pass.
- The exact validation commands and results are captured in the report and final
  handoff.
- Changes are committed and pushed to the remote branch.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find bigclaw-go/examples bigclaw-go/testdata bigclaw-go/docs scripts/ops -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO146'`
