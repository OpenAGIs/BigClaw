## BIG-GO-1025

### Plan
- Identify a tranche-3 subset of `src/bigclaw` Python modules that already have Go-owned replacements and no remaining live Python consumers beyond package exports, tests, and migration docs.
- Remove those Python modules plus any Python-only regression tests that only cover the deleted residual surfaces.
- Update package exports and migration documentation so the repository reflects the removed Python assets and the existing Go ownership.
- Run targeted validation, record exact commands and outcomes, then commit and push the scoped change set.

### Acceptance
- Changes stay scoped to `BIG-GO-1025` and focus on repository-level Python residuals under `src/bigclaw`.
- The `.py` file count under `src/bigclaw` is reduced by deleting migrated residual modules rather than adding new Python assets.
- The closeout reports the before/after Python and Go file counts plus confirms `pyproject.toml` and `setup.py` remain absent.
- Validation covers the affected Go and repo metadata surfaces with exact commands and results captured.

### Validation
- `rg --files src/bigclaw -g '*.py' | wc -l`
- `rg --files bigclaw-go -g '*.go' | wc -l`
- `cd bigclaw-go && go test ./internal/repo ./internal/product`
- `git diff --stat`
- `git status --short`
