# BIGCLAW-178

## Plan

1. Expose tenant and ownership isolation intent in `bigclaw-go/internal/policy` so task policy output can express tenant-scoped scheduling and owner matching.
2. Tighten distributed diagnostics in `bigclaw-go/internal/api/distributed.go` so reports show explicit cross-tenant and cross-owner boundary pairs in addition to violation counts.
3. Add focused tests in `bigclaw-go/internal/policy` and `bigclaw-go/internal/api` covering policy expression, blocked cross-tenant placement visibility, and markdown/report output.
4. Run the narrowest relevant Go test targets for the touched packages, then record exact commands and outcomes below.
5. Commit and push the scoped issue branch changes.

## Acceptance

1. Policy output can express tenant isolation mode and owner-matching requirements.
2. Distributed diagnostics and markdown report show cross-tenant boundary details, not just aggregate counts.
3. Tests prove invalid cross-tenant or ownership-breaking assignments are rejected and surfaced in diagnostics.

## Validation

- Run only the narrowest Go test targets that exercise the touched packages:
  - `go test ./internal/policy ./internal/api`
- Record exact commands and pass/fail outcomes in this file after execution.

## Test Log

- `cd /Users/openagi/code/bigclaw-workspaces/BIGCLAW-178/bigclaw-go && go test ./internal/policy ./internal/api`
  - `ok  	bigclaw-go/internal/policy	(cached)`
  - `ok  	bigclaw-go/internal/api	4.822s`
