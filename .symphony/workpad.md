# BIG-GO-929 Workpad

## Plan
1. Inventory the Python assets under `tests/test_control_center.py`, `tests/test_service.py`, and `tests/test_queue.py`, then map each assertion to `bigclaw-go/internal/reporting`, `bigclaw-go/internal/queue`, or a new scoped Go package.
2. Extend `bigclaw-go/internal/reporting` so queue control center coverage matches the migrated Python expectations, including action coverage and shared-view empty-state rendering.
3. Add a scoped `bigclaw-go/internal/service` package for the legacy governance/server test surface and cover it with targeted Go tests.
4. Document the migration inventory, Go replacement status, Python deletion criteria, and exact regression commands in issue-scoped docs.
5. Run targeted Go tests, inspect the diff for issue scope, then commit and push a dedicated `BIG-GO-929` branch.

## Acceptance
- The current Python/non-Go assets for the three target tests are explicitly listed.
- Each migrated Python assertion has either a landed Go replacement or a documented direct mapping to existing Go coverage.
- First-batch Go implementations and tests land for missing `control center/service/queue` coverage.
- Conditions for deleting the legacy Python assets and the exact regression commands are documented in-repo.
- The branch is committed and pushed after targeted validation passes.

## Validation
- `go test ./bigclaw-go/internal/queue`
- `go test ./bigclaw-go/internal/reporting`
- `go test ./bigclaw-go/internal/service`
- `git diff --stat origin/main...HEAD`

## Validation Results
- `cd bigclaw-go && go test ./internal/queue ./internal/reporting ./internal/control ./internal/repo ./internal/service` -> passed
- `cd bigclaw-go && go test ./...` -> passed
- `git status --short` -> only `.symphony/workpad.md`, `bigclaw-go/internal/repo/*`, `bigclaw-go/internal/reporting/*`, new `bigclaw-go/internal/service/*`, and `bigclaw-go/docs/migration-control-service-queue.md` are in scope
