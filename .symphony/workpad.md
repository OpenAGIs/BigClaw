# BIG-GO-1494

## Plan

1. Bootstrap the repository from `origin` because the provided worktree is uninitialized.
2. Measure the exact pre-change Python file count and inventory any files under `scripts/ops` or `root/scripts/ops` that are Python wrappers, including files marked `compat` or `shim`.
3. Remove the physical Python wrapper files that still exist for this area, keeping the change scoped to this issue and preserving the intended Go ownership or documenting delete conditions.
4. Update any directly related references or tests only if required by the wrapper removal.
5. Run targeted validation, capture exact commands and results, then commit and push the branch.

## Acceptance

- Actual physical `.py` file count in the repository decreases.
- Python wrapper files under the targeted `scripts/ops` area are eliminated even if their names or comments indicate `compat` or `shim`.
- Final notes include exact before/after Python file counts, the deleted file list, and the Go ownership or delete conditions for each removed wrapper.
- Changes remain scoped to BIG-GO-1494.
- A commit is created and pushed to the remote branch.

## Validation

- `find . -type f -name '*.py' | wc -l`
- `find . -type f \\( -path './scripts/ops/*.py' -o -path './root/scripts/ops/*.py' \\) | sort`
- Targeted repo checks or tests covering the affected scripts/ops ownership path after removal.
- `git status --short`
