## Plan

1. Hydrate this issue worktree from a local materialized BigClaw checkout and create a dedicated `BIG-GO-1478` branch.
2. Inventory remaining physical `.py` assets in docs/examples/manifests and identify which files can be migrated to Go-owned replacements or retired.
3. Make the smallest scoped changes that reduce the repository's Python file count while preserving or clarifying developer guidance.
4. Update any affected documentation/manifests so ownership moves to Go paths or explicit delete conditions are documented.
5. Run targeted validation proving the `.py` inventory decreased and the affected docs/examples/manifests remain coherent.
6. Commit and push the branch.

## Acceptance

- Actual tracked `.py` files in the repository decrease.
- The changed docs/examples/manifests explicitly document migrated/deleted Python assets and their Go replacement ownership or deletion rationale.
- Validation output shows the repository moved closer to Go-only.
- Changes stay scoped to the remaining Python docs/examples/manifests cleanup for this issue.

## Validation

- `git ls-files '*.py'` before and after changes.
- Targeted searches over docs/examples/manifests to confirm references now point at Go replacements or removed assets.
- Any repository-specific targeted checks needed for touched files.
- `git status --short`, commit creation, and push of the issue branch.
