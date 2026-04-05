# BIG-GO-1414

## Plan
- Inventory remaining Python assets, prioritizing `src/bigclaw/*.py`, `tests/*.py`, `scripts/*.py`, and `bigclaw-go/scripts/*.py`.
- Pick a scoped batch for this lane where Go replacements already exist or the Python files are removable without changing behavior.
- Remove or shrink those Python assets, keeping all changes limited to this issue.
- Validate the replacement path with targeted commands and confirm the Python file count decreases.
- Commit the lane changes and push `BIG-GO-1414` to `origin`.

## Acceptance
- Document the lane-relevant remaining Python asset list.
- Reduce the repository Python file count by deleting or replacing a batch of physical Python files.
- State the Go replacement path and exact validation commands.
- Keep the diff scoped to this issue.

## Validation
- Baseline and final Python inventories with `rg --files -g '*.py'`.
- Run targeted checks covering the removed or replaced assets.
- Record exact commands and pass/fail results in the final report.
