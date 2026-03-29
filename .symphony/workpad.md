# BIG-GO-943 Workpad

## Plan

1. Audit `src/bigclaw` legacy runtime/service/scheduler/workflow/orchestration/queue modules against the checked-in Go control-plane packages and lock the lane file list.
2. Land repo-native migration artifacts for this lane: a Go-side report, issue-coverage updates, and regression coverage that pins the mapping from frozen Python surfaces to Go replacements or delete plans.
3. Tighten the legacy compile-check shim list so the remaining Python compatibility shells in this lane stay explicitly frozen and syntax-checked.
4. Run targeted Go tests for the touched packages, capture exact commands/results, then commit and push a scoped branch for `BIG-GO-943`.

## Acceptance

- Lane file list for runtime/service/scheduler/workflow/orchestration/queue is explicit and checked into the repo.
- Each lane file has either a concrete Go replacement path or an explicit delete/defer plan.
- Validation commands and residual risks are documented in the checked-in artifacts.
- Python assets for this lane do not grow; remaining files are marked as frozen compatibility shims with Go ownership.

## Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression`
- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/scheduler ./internal/workflow ./internal/queue ./internal/worker`
