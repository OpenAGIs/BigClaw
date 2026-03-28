# BIG-GO-930

## Plan
1. Inventory remaining Python and other non-Go assets still present in the repo and identify which are legacy versus still required evidence/docs.
2. Map legacy Python entrypoints/scripts/tests to existing Go replacements or document explicit migration/delete conditions where no direct replacement exists.
3. Remove scoped residual Python assets that are superseded by Go-only implementations or are no longer part of the target repo surface.
4. Update repository documentation to define Go-only status, remaining non-Go exceptions, validation commands, and main-merge conditions.
5. Run targeted validation commands, capture exact results, then commit and push the scoped changes.

## Acceptance
- Current Python/non-Go asset inventory is explicit.
- Go replacement or migration path is documented for the inventory.
- First batch of direct Go-only cleanup is landed where safely scoped.
- Conditions for deleting legacy Python assets and regression validation commands are documented.

## Validation
- `go test ./...` from `bigclaw-go/`
- Targeted repository scans such as `rg --files -g '*.py' -g '*.sh'`
- Any focused checks introduced by the cleanup

## Result
- Deleted the migrated Python-only automation shims:
  - `bigclaw-go/scripts/e2e/run_task_smoke.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`
  - `bigclaw-go/scripts/migration/shadow_compare.py`
- Moved checked-in callers to direct Go CLI invocations.
- Added `docs/go-only-repo-cutover.md` with inventory, delete conditions, validation, and merge strategy.
- Added `bigclaw-go/internal/regression/go_only_repo_cutover_test.go` to keep the cutover doc and removed shim set aligned.

## Validation Results
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression` -> passed
- `cd bigclaw-go && python3 scripts/e2e/run_all_test.py` -> passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help` -> passed
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help` -> passed
- `cd bigclaw-go && go test ./...` -> passed
- `test ! -e bigclaw-go/scripts/e2e/run_task_smoke.py && test ! -e bigclaw-go/scripts/benchmark/soak_local.py && test ! -e bigclaw-go/scripts/migration/shadow_compare.py` -> passed
- `rg --files -g '*.py' -g '*.sh' | sort` -> passed; inventory captured for the cutover doc
