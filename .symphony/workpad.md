# BIG-GO-1085

## Plan
- Confirm the current `scripts/ops/bigclaw_workspace_bootstrap.py` wrapper behavior and search for active references.
- Delete the Python compatibility wrapper so the legacy Python entrypoint is no longer present.
- Run targeted validation for the remaining Go-first workspace bootstrap path and verify the repository Python file count decreases.
- Commit and push the scoped change set.

## Acceptance
- `scripts/ops/bigclaw_workspace_bootstrap.py` is removed from the repository or otherwise no longer has a default execution path.
- The repository `.py` file count decreases compared with the pre-change count.
- The remaining workspace bootstrap command path through `scripts/ops/bigclawctl workspace ...` still executes successfully for a targeted help/smoke command.

## Validation
- `find . -name '*.py' | wc -l`
- `bash scripts/ops/bigclawctl workspace --help`
- `git diff --stat`

## Validation Results
- Pre-change `find . -name '*.py' | wc -l` -> `23`
- Post-change `find . -name '*.py' | wc -l` -> `22`
- `bash scripts/ops/bigclawctl workspace --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `rg -n "scripts/ops/bigclaw_workspace_bootstrap\\.py|bigclaw_workspace_bootstrap\\.py" --glob '!reports/**' --glob '!local-issues.json' .` -> documentation-only matches in `docs/go-cli-script-migration-plan.md`
- `git diff --stat` -> `.symphony/workpad.md | 22 insertions/deletions`, `scripts/ops/bigclaw_workspace_bootstrap.py | 26 deletions`
