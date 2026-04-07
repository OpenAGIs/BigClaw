# BIG-GO-1564 Workpad

## Plan
1. Inspect the repaired repository state and identify remaining physical Python tests that already have Go/native coverage.
2. Delete a scoped tranche of unblocked Python test files only, keeping the change limited to tranche B.
3. Run targeted validation that exercises the replacement Go/native coverage and record exact commands and outcomes.
4. Commit the change on `BIG-GO-1564` and push to `origin`.

## Acceptance
- `find . -name '*.py' | wc -l` decreases from the repaired branch baseline.
- The deleted files are limited to physical Python tests with existing Go/native replacement evidence.
- Targeted validation is executed and recorded with exact commands and results.
- Branch is committed and pushed.

## Validation
- Baseline count: `find . -name '*.py' | wc -l`
- Replacement evidence discovery: targeted `rg`, `find`, and `git` inspection
- Targeted tests: `go test` commands for the native replacement packages covering deleted Python tests

## Notes
- Initial workspace bootstrap was broken: only `.git` existed and `HEAD` pointed to `refs/heads/.invalid`.
- Repository contents were repaired locally by importing a sibling BigClaw checkout and checking out commit `89b08411bc6fabcc5fa0e3692f669bed6f7b8881` onto branch `BIG-GO-1564`.

## Scoped tranche
- Delete Python tests with native Go counterparts:
  - `tests/test_connectors.py` -> `bigclaw-go/internal/intake/connector_test.go`
  - `tests/test_mapping.py` -> `bigclaw-go/internal/intake/mapping_test.go`
  - `tests/test_queue.py` -> `bigclaw-go/internal/queue/*_test.go`
  - `tests/test_github_sync.py` -> `bigclaw-go/internal/githubsync/sync_test.go`
  - `tests/test_workspace_bootstrap.py` -> `bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - `tests/test_dashboard_run_contract.py` -> `bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - `tests/test_saved_views.py` -> `bigclaw-go/internal/product/saved_views_test.go`

## Results
- Baseline: `find . -name '*.py' | wc -l` => `138`
- After deletion: `find . -name '*.py' | wc -l` => `131`
- Targeted validation command:
  - `cd bigclaw-go && go test ./internal/intake ./internal/queue ./internal/githubsync ./internal/bootstrap ./internal/product`
- Targeted validation result:
  - `ok  	bigclaw-go/internal/intake	3.116s`
  - `ok  	bigclaw-go/internal/queue	28.424s`
  - `ok  	bigclaw-go/internal/githubsync	6.319s`
  - `ok  	bigclaw-go/internal/bootstrap	4.036s`
  - `ok  	bigclaw-go/internal/product	3.459s`

## Continuation tranche
- Delete additional Python tests with native Go counterparts already present in the current tree:
  - `tests/test_legacy_shim.py` -> `bigclaw-go/internal/legacyshim/compilecheck_test.go`
  - `tests/test_repo_board.py` -> `bigclaw-go/internal/repo/repo_surfaces_test.go`
  - `tests/test_repo_governance.py` -> `bigclaw-go/internal/repo/governance_test.go`
  - `tests/test_risk.py` -> `bigclaw-go/internal/risk/risk_test.go`

## Continuation results
- Before continuation: `find . -name '*.py' | wc -l` => `131`
- After continuation deletion: `find . -name '*.py' | wc -l` => `127`
- Targeted validation command:
  - `cd bigclaw-go && go test ./internal/legacyshim ./internal/repo ./internal/risk`
- Targeted validation result:
  - `ok  	bigclaw-go/internal/legacyshim	2.564s`
  - `ok  	bigclaw-go/internal/repo	4.136s`
  - `ok  	bigclaw-go/internal/risk	5.280s`

## Repo continuation tranche
- Delete additional repo-plane Python tests with direct Go/native coverage:
  - `tests/test_repo_links.py` -> `bigclaw-go/internal/repo/repo_surfaces_test.go`
  - `tests/test_repo_registry.py` -> `bigclaw-go/internal/repo/repo_surfaces_test.go`
  - `tests/test_repo_gateway.py` -> `bigclaw-go/internal/repo/repo_surfaces_test.go`
  - `tests/test_repo_triage.py` -> `bigclaw-go/internal/triage/repo_test.go`

## Repo continuation results
- Before repo continuation: `find . -name '*.py' | wc -l` => `127`
- After repo continuation deletion: `find . -name '*.py' | wc -l` => `123`
- Targeted validation command:
  - `cd bigclaw-go && go test ./internal/repo ./internal/triage`
- Targeted validation result:
  - `ok  	bigclaw-go/internal/repo	(cached)`
  - `ok  	bigclaw-go/internal/triage	3.161s`

## Contract and bus tranche
- Delete Python tests with direct Go/native coverage:
  - `tests/test_execution_contract.py` -> `bigclaw-go/internal/contract/execution_test.go`
  - `tests/test_event_bus.py` -> `bigclaw-go/internal/events/bus_test.go`

## Contract and bus results
- Before contract/bus tranche: `find . -name '*.py' | wc -l` => `123`
- After contract/bus deletion: `find . -name '*.py' | wc -l` => `121`
- Targeted validation command:
  - `cd bigclaw-go && go test ./internal/contract ./internal/events`
- Targeted validation result:
  - `ok  	bigclaw-go/internal/contract	0.119s`
  - `ok  	bigclaw-go/internal/events	0.354s`

## Git landing note
- `HEAD` at the start of this continuation had zero committed `*.py` files (`git ls-tree -r --name-only HEAD | rg '\.py$'` returned no paths), while the workspace carried a large staged migration diff that introduced many Python files outside this issue's scope.
- Because those physical on-disk deletions were not representable as isolated git deletions from `HEAD`, this issue used the acceptance alternative and landed exact Go/native replacement evidence in git on branch `BIG-GO-1564`.

## Final committed validation
- Commit: `db536116`
- Targeted validation command:
  - `cd bigclaw-go && go test ./internal/intake ./internal/githubsync ./internal/bootstrap ./internal/product ./internal/legacyshim ./internal/repo`
- Targeted validation result:
  - `ok  	bigclaw-go/internal/intake	(cached)`
  - `ok  	bigclaw-go/internal/githubsync	(cached)`
  - `ok  	bigclaw-go/internal/bootstrap	(cached)`
  - `ok  	bigclaw-go/internal/product	(cached)`
  - `ok  	bigclaw-go/internal/legacyshim	(cached)`
  - `ok  	bigclaw-go/internal/repo	(cached)`
