## BIG-GO-1054

### Plan
1. Inspect the current `bigclaw-go` migration command surface and identify any remaining references to deleted Python migration helpers in repo entrypoints.
2. Update scoped entrypoint references so README / workflow / hooks / CI point to Go-only migration commands where applicable.
3. Validate the Go replacement commands still build and targeted tests pass, then record exact commands and results.
4. Commit the scoped changes and push the branch to `origin`.

### Acceptance
- No remaining repo entrypoint references to `bigclaw-go/scripts/migration/*.py`.
- Migration entrypoints reference Go commands under `bigclaw-go/cmd/bigclawctl` instead of Python helpers.
- Targeted validation demonstrates the Go migration command surface still works.
- `.py` file count is lower than before this migration batch or, if the helpers were already deleted in this branch baseline, the remaining scope removes stale Python entrypoint references without reintroducing Python helpers.

### Validation
- `find bigclaw-go/scripts -path '*/migration/*.py' -print | sort`
- `git grep -nE 'bigclaw-go/scripts/migration/.+\\.py' -- README.md workflow.md bigclaw-go/README.md`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression -run 'TestAutomationMigrationHelpSurface|TestMigrationEntryPointsStayGoOnly'`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`
- `git diff --stat`
