# BIG-GO-1074 Workpad

## Plan
- Confirm every live workflow/bootstrap/sync/refill surface that still routes through the Python compatibility entrypoints.
- Replace those workflow-facing entrypoints with non-Python launchers that invoke `bash scripts/ops/bigclawctl ...` directly.
- Delete the Python bootstrap/refill wrappers once their default execution paths are removed.
- Refresh focused docs and regression coverage so the workflow contract pins the Go-only entrypoints and the deleted Python files stay absent.
- Run targeted validation, record exact commands and results, then commit and push `BIG-GO-1074`.

## Acceptance
- `workflow.md` no longer depends on Python bootstrap or refill entrypoints for the documented unattended workflow path.
- The default operator path for bootstrap, sync, and refill is a non-Python entrypoint.
- At least the targeted Python bootstrap/refill shim files removed by this slice are deleted from the repo.
- Regression coverage fails if the removed Python entrypoints reappear or if workflow-facing guidance regresses to Python-first commands.
- Repository `.py` count is lower after the change than before it.

## Validation
- Baseline and final Python counts: `rg --files . | rg '\.py$' | wc -l`
- Workflow/doc surface audit: `rg -n "python3 scripts/ops/(bigclaw_refill_queue|bigclaw_workspace_bootstrap|symphony_workspace_bootstrap)\\.py|bash scripts/ops/bigclawctl (workspace|github-sync|refill)" workflow.md README.md docs bigclaw-go`
- Targeted Go tests: `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
- Direct entrypoint smoke checks:
  - `bash scripts/ops/bigclawctl workspace --help`
  - `bash scripts/ops/bigclawctl github-sync --help`
  - `bash scripts/ops/bigclawctl refill --help`
- Record exact command outputs/results in the closeout summary.
