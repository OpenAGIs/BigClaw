## BIGCLAW-187 Workpad

### Plan

- [x] Inspect the distributed diagnostics builders, reviewer surfaces, and replay-validation report patterns.
- [x] Add a repo-native distributed diagnostics surface for event compression and replay validation.
- [x] Thread the new surface through the distributed diagnostics JSON and markdown exports.
- [x] Add focused regression/API coverage for the new surface.
- [in_progress] Run targeted Go tests, then commit and push the branch.

### Acceptance Criteria

- [x] Distributed diagnostics expose a checked-in event-compression and replay-validation surface.
- [x] The new surface is available in both JSON responses and markdown export output.
- [x] A repo-native report captures compression summary, replay validation details, artifacts, and limitations.
- [x] Focused regression and API tests validate the new surface end to end.
- [x] Exact validation commands and outcomes are recorded for the final report.

### Validation

- [x] `cd bigclaw-go && go test ./internal/api ./internal/regression`

### Notes

- Scope is intentionally limited to the distributed diagnostics/reporting surface and its checked-in evidence bundle.
