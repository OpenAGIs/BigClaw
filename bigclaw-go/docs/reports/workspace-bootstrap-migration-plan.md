# Workspace bootstrap/cleanup/init migration plan

## Scope

- Keep `bigclaw-go/internal/bootstrap` as the single implementation for workspace bootstrap, cleanup, and cache validation.
- Use `bigclaw-go/cmd/bigclawctl workspace` as the only operator-facing CLI surface for bootstrap lifecycle operations.
- Retain shell wrappers in `scripts/ops` as thin compatibility entrypoints while avoiding new Python implementation logic.
- Freeze `src/bigclaw/workspace_bootstrap*.py` as legacy compatibility paths until downstream repos fully stop importing them.

## First Batch Implementation

- Land `bigclawctl workspace init` so operators can emit a repo-native migration plan and first-batch checklist from the Go CLI.
- Record the plan output in `bigclaw-go/docs/reports/workspace-bootstrap-migration-plan.md` for branch review and PR handoff.
- Update `docs/symphony-repo-bootstrap-template.md` to describe the Go-only bootstrap files and the `workspace init` adoption step.
- Cover the new CLI surface with focused tests in `bigclaw-go/cmd/bigclawctl/main_test.go` and `bigclaw-go/internal/bootstrap/bootstrap_test.go`.

## Validation Commands

- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap`
  - Covers workspace CLI output and bootstrap/cache lifecycle behavior.
- `python -m pytest tests/test_workspace_bootstrap.py tests/test_legacy_shim.py`
  - Confirms legacy compatibility paths still align with the Go-first workspace flow.
- `bash scripts/ops/bigclawctl workspace init --report bigclaw-go/docs/reports/workspace-bootstrap-migration-plan.md`
  - Refreshes the checked-in migration artifact from the canonical Go command.

## Regression Surface

- Workspace branch naming and issue identifier sanitization remain stable for both bootstrap and cleanup.
- Shared mirror and seed cache reuse still suppress repeated remote clones after the first workspace.
- Cleanup still prunes worktree registrations and issue branches without deleting the shared cache.
- Existing `scripts/ops/*workspace*` entrypoints remain compatible with operator automation and workflow hooks.

## Branch Recommendation

- Use branch `symphony/BIG-GO-908` and keep all workspace lifecycle migration changes scoped to Go bootstrap surfaces plus migration docs.

## PR Recommendation

- Open a PR titled `BIG-GO-908: migrate workspace bootstrap lifecycle to Go commands` and call out the remaining Python modules as frozen compatibility shims, not active implementation.

## Risks

- Downstream repos may still import the Python bootstrap modules directly, so removing those modules now would be a breaking change.
- Workflow operators can bypass the new Go plan/init step unless templates and adoption docs point to `bigclawctl workspace init`.
- Bootstrap behavior depends on local `git` features such as `worktree remove --force`; older Git installations remain an environmental risk outside this slice.
