# BIG-GO-1475

## Plan
1. Re-inventory the remaining report-related Python helpers after the continuation-helper slice.
2. Port the active `scripts/e2e/run_task_smoke.py` report helper to a Go-owned entrypoint and shared reporting package code.
3. Update shell wrappers, docs, and issue-coverage references to point at the Go entrypoint.
4. Delete the replaced Python helper and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go and report-consumer validation, commit, and push the branch.

## Acceptance
- The selected active Python helper (`run_task_smoke.py`) is deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python file.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helper and regression/report-consumer surfaces it feeds.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
