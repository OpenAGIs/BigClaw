# BIG-GO-1475

## Plan
1. Port `scripts/e2e/external_store_validation.py` into Go-owned reporting/runtime code plus a small Go entrypoint.
2. Update the external-store validation doc and regression references to point at the Go entrypoint and retained JSON artifact.
3. Add Go coverage for the external-store report-shape helpers and keep the checked-in external-store regression surface aligned.
4. Delete the replaced Python helper and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go validation, commit, and push the branch.

## Acceptance
- The selected active Python helper (`external_store_validation.py`) is deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python file.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helper and the repo-native surfaces it feeds.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
