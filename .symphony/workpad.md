# BIG-GO-1475

## Plan
1. Re-inventory the remaining report-related Python helpers after the `run_task_smoke` slice.
2. Port `scripts/migration/live_shadow_scorecard.py` and `scripts/migration/export_live_shadow_bundle.py` into Go-owned reporting code plus small Go entrypoints.
3. Update migration docs, checked-in report artifacts, regression expectations, and closeout command references to point at the Go entrypoints.
4. Delete the replaced Python helpers and refresh the issue validation report with the next physical Python file-count reduction.
5. Run targeted Go and report-consumer validation, commit, and push the branch.

## Acceptance
- The selected active Python helpers (`live_shadow_scorecard.py` and `export_live_shadow_bundle.py`) are deleted, not just documented.
- Replacement Go ownership is explicit in repo-native scripts, docs, and validation artifacts.
- In-scope report import paths and helper references no longer point at the removed Python files.
- Validation proves the repository moved closer to Go-only by reducing the physical Python file count again.

## Validation
- Capture the pre/post Python file inventory with exact commands.
- Run targeted `go test` coverage for the new reporting helpers and regression/report-consumer surfaces they feed.
- Record exact command results in `reports/BIG-GO-1475-validation.md`.
