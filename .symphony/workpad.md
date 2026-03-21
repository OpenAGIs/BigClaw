## Codex Workpad

```text
jxrt:/Users/jxrt/Desktop/symphony-main/BigClaw@feat/bigclaw-go-local-mainline
```

### Plan

- [x] Audit the remaining local tracker refill surface for Linear-specific type names in the Go mainline.
- [x] Rename the refill issue model to tracker-neutral naming in `bigclaw-go/internal/refill/*` and `cmd/bigclawctl`.
- [ ] Validate the renamed refill surface with targeted Go tests.

### Acceptance Criteria

- [x] The Go refill/local issue store packages no longer expose `LinearIssue` as their core issue type.
- [x] `bigclawctl refill` still works with both local and Linear-backed issue sources after the rename.
- [x] `go test ./cmd/bigclawctl ./internal/refill/...` passes.

### Validation

- [x] `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/refill/...`

### Notes

- 2026-03-19: This slice is a bounded `BIG-GOM-307` follow-up aimed at removing Linear-only operator vocabulary from the active Go refill path before tackling larger workflow/runtime migrations.
- 2026-03-19: Targeted refill tests passed after renaming the shared issue model to `TrackedIssue`.
