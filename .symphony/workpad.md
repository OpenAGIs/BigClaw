## BIG-GO-908 Workpad

### Plan

- [x] Audit the existing Python workspace bootstrap wrappers and the current Go CLI layout to identify the smallest migration slice for `bootstrap`, `cleanup`, and operator-facing `init` coverage.
- [x] Implement Go commands in `bigclaw-go` for the migrated workspace lifecycle path, and repoint the existing ops wrappers to the Go entrypoint without expanding scope beyond workspace/bootstrap handling.
- [x] Add targeted Go tests for the new command surface and update migration documentation with the implementation/validation checklist, regression surface, branch/PR guidance, and risks.
- [ ] Run targeted validation, capture exact commands/results, then commit and push the issue branch.

### Acceptance

- [x] A concrete migration plan and first-batch implementation checklist exist in-repo for the workspace/bootstrap/cleanup/init path.
- [x] Operator-facing workspace bootstrap/cleanup execution is available through a Go command in `bigclaw-go`.
- [x] Validation commands and the expected regression surface are recorded alongside the change.
- [x] Branch/PR recommendation and residual risks are documented.

### Validation

- [x] `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap` -> `ok   bigclaw-go/cmd/bigclawctl 1.861s`, `ok   bigclaw-go/internal/bootstrap 1.296s`
- [x] `python3 -m pytest tests/test_workspace_bootstrap.py tests/test_legacy_shim.py` -> `13 passed in 3.23s`
- [x] `bash scripts/ops/bigclawctl workspace init --report bigclaw-go/docs/reports/workspace-bootstrap-migration-plan.md` -> wrote/refreshed the checked-in migration plan artifact

### Regression Surface

- Workspace bootstrap branch naming and issue identifier sanitization.
- Shared mirror cache initialization and fetch/update behavior.
- Existing workspace cleanup semantics for worktree vs standalone clone modes.
- CLI and ops-wrapper compatibility for current operator invocation paths.

### Notes

- Scope is intentionally limited to workspace lifecycle migration; unrelated Python operational scripts remain unchanged.
- Python validation stays in place for the legacy module/wrapper surface while the first Go implementation slice lands.
- `python -m pytest ...` was not runnable in this environment because `python` is absent; `python3` is the working interpreter on this branch.
