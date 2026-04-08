Issue: BIG-GO-136

Plan
- Identify lingering Python support assets that are not part of the frozen legacy runtime surface and fit the "examples, fixtures, demos, and support helpers" sweep scope.
- Remove the remaining Python support/helper scripts under `bigclaw-go/scripts/` that still back Go-only validation and migration evidence, along with stale references to them in Go docs and reports.
- Keep core Python package sources under `src/` and repo-management scripts under `scripts/ops/` untouched unless they are directly required by the scoped support-asset cleanup.
- Run targeted repository searches plus focused Go tests for any replacement logic or documentation-adjacent generators that remain.
- Commit the scoped changes and push branch `BIG-GO-136` to `origin`.

Acceptance
- Lingering Python support assets covered by this sweep are removed or no longer referenced by active documentation in the touched area.
- No new unrelated codepaths or migrations are introduced outside the targeted support-asset cleanup.
- Validation commands and exact results are recorded.
- Changes are committed on `BIG-GO-136` and pushed to the remote branch.

Validation
- `rg --files -g '*.py' bigclaw-go/scripts`
- `rg -n "scripts/(migration|benchmark|e2e)/.*\\.py|python3 scripts/(migration|benchmark|e2e)" bigclaw-go/docs bigclaw-go/scripts`
- Focused Go tests for any touched package replacements or updated generators, likely `go test ./...` for the affected `bigclaw-go` script/test packages if the sweep leaves Go coverage there.
- `git diff --stat`

Results
- `rg -n "scripts/migration/|python3 scripts/migration|examples/shadow-|bigclaw-go/examples/shadow-" bigclaw-go/docs bigclaw-go/internal bigclaw-go/scripts bigclaw-go/examples` -> exit 1 after cleanup; no remaining migration-helper or deleted example references in active Go docs/tests/scripts.
- `rg --files bigclaw-go/scripts -g '*.py'` -> only `benchmark/` and `e2e/` Python helpers remain; `bigclaw-go/scripts/migration/*.py` is gone.
- `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadow|TestRollback|TestProductionCorpus'` -> `ok  	bigclaw-go/internal/regression	5.681s`
- `git diff --stat` -> 25 files changed, 114 insertions, 1352 deletions.
