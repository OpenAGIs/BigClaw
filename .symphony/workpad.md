# BIG-GO-928 Workpad

## Plan

1. Inventory the current Python and non-Go assets behind `tests/test_workspace_bootstrap.py`, `tests/test_planning.py`, and `tests/test_mapping.py`, then map them to existing `bigclaw-go` coverage to identify the smallest missing migration slice.
2. Land first-batch Go migration work scoped to this issue:
   - add Go-side planning models, builders, gate evaluation, and reporting for the behaviors still only covered in Python;
   - add missing Go bootstrap regression tests for cache reuse, stale-seed recovery, cleanup preservation, and validation reporting;
   - confirm `mapping` is already covered and only document its remaining Python asset status.
3. Remove or narrow the targeted Python tests only where an equivalent Go regression test now exists and the remaining Python module is no longer the source of truth for that behavior.
4. Run targeted Go tests for touched packages, capture exact commands and results, then commit and push the issue branch.

## Acceptance

- Produce a clear inventory of current Python/non-Go assets for workspace bootstrap, planning, and mapping.
- Land Go replacements for the uncovered planning and bootstrap behaviors from the target Python tests.
- State the conditions for deleting legacy Python assets and the exact regression commands that protect the migration.

## Validation

- `go test ./bigclaw-go/internal/bootstrap ./bigclaw-go/internal/intake ./bigclaw-go/internal/planning ./bigclaw-go/internal/governance`
- `go test ./bigclaw-go/...` only if targeted coverage reveals package-coupling regressions worth expanding.
- Record final exact commands and pass/fail status in the closeout.

## Validation Results

- `cd bigclaw-go && go test -count=1 ./internal/bootstrap ./internal/intake ./internal/planning ./internal/governance`
  - PASS
  - `ok  	bigclaw-go/internal/bootstrap	3.881s`
  - `ok  	bigclaw-go/internal/intake	0.359s`
  - `ok  	bigclaw-go/internal/planning	1.490s`
  - `ok  	bigclaw-go/internal/governance	1.102s`
- `cd bigclaw-go && go test ./...`
  - PASS
  - representative package results:
  - `ok  	bigclaw-go/cmd/bigclawctl	5.851s`
  - `ok  	bigclaw-go/internal/bootstrap`
  - `ok  	bigclaw-go/internal/contract	3.365s`
  - `ok  	bigclaw-go/internal/planning`
  - `ok  	bigclaw-go/internal/queue	30.056s`
  - `ok  	bigclaw-go/internal/workflow	3.830s`
