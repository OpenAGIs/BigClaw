# BIG-GO-1452

## Plan
1. Inventory remaining Python files in the repository, with emphasis on `src/bigclaw/*.py`, `tests/*.py`, `scripts/*.py`, and `bigclaw-go/scripts/*.py`.
2. Select a scoped sweep of Python assets that can be removed or turned into explicit thin wrappers around Go replacements.
3. Implement the sweep and keep the changes narrowly focused on reducing physical Python file count.
4. Run targeted validation, recording exact commands and outcomes.
5. Commit the changes and push branch `BIG-GO-1452` to `origin`.

## Acceptance
- Lane-specific remaining Python asset inventory is documented.
- A batch of physical Python files is deleted, replaced, or reduced to no-behavior compatibility shims.
- Go replacement paths and validation commands are documented.
- Repository Python file count goes down.

## Validation
- Measure Python file count before and after with repository file inventory commands.
- Run targeted tests or command checks that cover each touched path.
- Record exact commands and results for the final report.
