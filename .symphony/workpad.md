# BIG-GO-1475

## Plan
1. Re-inventory the remaining report-related Python helpers after the shadow compare/matrix slice.
2. Port `scripts/benchmark/run_matrix.py` and `scripts/benchmark/soak_local.py` into Go-owned reporting/runtime code plus small Go entrypoints.
3. Update benchmark docs, issue coverage, and in-scope report references to point at the Go entrypoints and checked-in artifacts.
4. Delete the replaced Python helpers and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go validation, commit, and push the branch.

## Acceptance
- The selected active Python helpers (`run_matrix.py` and `soak_local.py`) are deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python file.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helper and the repo-native surfaces it feeds.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
