# BIG-GO-921

## Plan
1. Inventory all Python packaging/build assets and references (`pyproject.toml`, `setup.py`, setuptools-related docs/scripts/tests).
2. Design a Go-only replacement surface that makes the migration state explicit and provides repo-visible inventory/removal criteria.
3. Implement the first Go surface and update affected docs/tests to point at the Go path instead of Python packaging.
4. Run targeted validation, capture exact commands/results, then commit and push the branch.

## Acceptance
- Current Python/non-Go packaging assets are explicitly inventoried.
- A Go replacement or migration path exists for the Python packaging/build surface.
- Initial Go implementation lands in-repo with scoped updates.
- Conditions for deleting old Python assets and regression validation commands are documented.

## Validation
- `go test` for the new/updated Go package(s).
- `go test` for any impacted regression or repo-surface tests.
- Manual inspection that old Python packaging files are no longer required by active repo guidance.
