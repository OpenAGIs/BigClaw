# BIG-GO-1475

## Plan
1. Re-inventory the remaining report-related Python helpers after the coordination-surface slice.
2. Port `scripts/e2e/broker_failover_stub_matrix.py` into Go-owned reporting code plus a small Go entrypoint.
3. Update in-scope durability and e2e references to point at the Go entrypoint and checked-in broker proof artifacts.
4. Delete the replaced Python helper and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go validation, commit, and push the branch.

## Acceptance
- The selected active Python helper (`broker_failover_stub_matrix.py`) is deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python file.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helper and the repo-native surfaces it feeds.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
