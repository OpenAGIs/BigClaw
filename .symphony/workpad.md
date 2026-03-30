Issue: BIG-GO-1019

Plan
- Port `external_store_validation.py` into a Go-native `bigclawctl automation e2e external-store-validation` command.
- Preserve the live lane behavior and canonical report schema: remote HTTP event-log replay, checkpoint reset history, retention boundary visibility, and shared-lease takeover checks.
- Update directly coupled docs and regression text to reference the Go entrypoint, regenerate the canonical report, and remove the Python script.
- Run targeted validation for the external-store tranche, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- External-store validation remains invokable through a Go-native CLI path that writes the same canonical report surface.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `gofmt -w bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command_test.go bigclaw-go/cmd/bigclawctl/automation_commands.go`
- `go test ./cmd/bigclawctl -run 'TestAutomationExternalStoreValidationBuildBackendMatrix|TestAutomationExternalStoreValidationWritesReport'`
- `go run ./cmd/bigclawctl automation e2e external-store-validation --help`
- `go run ./cmd/bigclawctl automation e2e external-store-validation --report-path docs/reports/external-store-validation-report.json`
- `go test ./internal/regression -run 'TestExternalStoreValidationReportStaysAligned'`
- `find bigclaw-go/scripts -name '*.py' | sort | wc -l`
- `git diff --stat && git status --short`
