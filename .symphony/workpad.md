## BIG-GO-1287

### Plan
1. Inventory the repository for remaining physical Python assets and Python-oriented references.
2. Remove or shrink issue-scoped residual references so the repository clearly reflects the Go-only state.
3. Run targeted validation that proves the Python asset count is zero and that referenced Go paths are valid.
4. Commit the scoped changes and push the branch to `origin`.

### Acceptance
- Produce an explicit inventory for the remaining Python asset set in this lane.
- Prefer deleting or shrinking residual Python-facing surface area without expanding scope beyond this issue.
- Document the Go replacement path and the validation commands used.
- Optimize for a lower physical Python file count; if it is already zero, preserve that state and remove stale references.

### Validation
- `find . -name '*.py' -o -name '*.pyi' | sort`
- `rg -n "python|\\.py" README.md workflow.md docs scripts bigclaw-go .github Makefile`
- `git ls-tree -r --name-only HEAD | rg '\\.py$'`
- `go test ./...`
