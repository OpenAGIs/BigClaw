# BIG-GO-1475

## Plan
1. Re-inventory the remaining report-related Python helpers after the live-validation bundle slice.
2. Port `scripts/migration/shadow_compare.py` and `scripts/migration/shadow_matrix.py` into Go-owned reporting code plus small Go entrypoints.
3. Update migration docs, issue coverage, and remaining test/report surfaces to point at the Go entrypoints and checked-in artifacts.
4. Delete the replaced Python helpers and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go validation, commit, and push the branch.

## Acceptance
- The selected active Python helpers (`shadow_compare.py` and `shadow_matrix.py`) are deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python file.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helper and the repo-native surfaces it feeds.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
