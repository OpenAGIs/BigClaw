## Codex Workpad

### Plan

- [x] Audit the existing Go distributed diagnostics and control-center entrypoints for routing and reviewer-navigation payloads that already overlap with the issue scope.
- [x] Add a scoped aggregation surface for parallel agent routing and distributed diagnostics entrypoints in the Go API layer.
- [x] Cover the new aggregation contract with targeted Go tests and keep the change limited to the active diagnostics/reporting slice.

### Acceptance Criteria

- [x] The Go API exposes a single aggregated payload that makes the parallel routing and distributed diagnostics entrypoints discoverable together.
- [x] Existing distributed diagnostics report/export payloads continue to render and now include the new aggregation surface where appropriate.
- [x] Targeted Go API/regression tests pass for the new aggregation contract.

### Validation

- [x] `cd bigclaw-go && go test ./internal/api ./internal/regression/...`

### Validation Log

- Command: `cd bigclaw-go && go test ./internal/api ./internal/regression/...`
- Result: `ok  	bigclaw-go/internal/api	4.772s`
- Result: `ok  	bigclaw-go/internal/regression	3.493s`

### Notes

- [ ] Keep the change scoped to `BIGCLAW-195`; avoid unrelated runtime, scheduler, or queue semantics changes.
- [ ] Record the exact validation commands and results after implementation.
