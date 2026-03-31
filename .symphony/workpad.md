Issue: BIG-GO-1027

Plan
- Audit remaining repo-root Python tests and choose a small tranche whose behavior is already implemented in `bigclaw-go` so the migration stays scoped and lowers `.py` count immediately.
- Patch Go-native tests only where the Python suite still asserts behavior that is not yet pinned in Go.
- Delete the migrated Python test files from `tests/` once equivalent Go coverage exists.
- Run targeted Go validation plus file-count/package-surface checks, then commit and push the branch.

Acceptance
- Changes remain scoped to the selected residual Python tests tranche and directly corresponding Go test coverage.
- Repository `.py` file count decreases after removing migrated Python tests.
- Targeted Go tests cover the migrated dashboard run contract, saved views, and execution contract behaviors.
- Final report includes the exact impact on `.py` count, `.go` count, and confirms whether `pyproject.toml` / `setup.py` changed.

Validation
- `find tests -maxdepth 1 -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `gofmt -w bigclaw-go/internal/product/dashboard_run_contract_test.go bigclaw-go/internal/product/saved_views_test.go bigclaw-go/internal/contract/execution_test.go`
- `cd bigclaw-go && go test ./internal/product -run 'Test(BuildDefaultDashboardRunContractIsReleaseReady|DashboardRunContractAuditDetectsMissingPaths|RenderDashboardRunContractReport|BuildSavedViewCatalog|RenderSavedViewReport|AuditSavedViewCatalog)' -count=1`
- `cd bigclaw-go && go test ./internal/contract -run 'Test(ExecutionContractAuditAcceptsWellFormedContract|ExecutionContractAuditSurfacesContractGaps|ExecutionContractRoundTripAndPermissionMatrix|RenderExecutionContractReportIncludesRoleMatrix|OperationsAPIContract)' -count=1`
- `git diff --stat`
- `git status --short`
