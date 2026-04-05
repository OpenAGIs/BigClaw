# BIG-GO-1356 Workpad

## Plan
- Inspect repo-native Ray smoke entrypoints and checked-in validation artifacts for residual `python -c` usage.
- Replace the default Ray smoke entrypoint with a shell-native command that preserves the smoke semantics without Python.
- Add or extend regression coverage to enforce the Go/shell-native Ray smoke path and verify checked-in evidence no longer records Python entrypoints for this flow.
- Run targeted tests and repository checks, then commit and push the issue branch.

## Acceptance
- Repository reality stays Python-free for physical files: `find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l` remains `0`.
- A concrete native replacement lands for Python entrypoint consolidation removal in the Ray smoke path.
- Checked-in docs/reports and regression coverage align with the native replacement.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l`
- `cd bigclaw-go && go test ./internal/regression ./scripts/e2e/... ./cmd/bigclawctl/...` if package layout permits; otherwise narrow to impacted packages.
- `git diff --stat`

## Execution Notes
- Replaced the active Ray smoke default entrypoint with the shell-native `echo hello from ray` path in `bigclaw-go/scripts/e2e/ray_smoke.sh`.
- Updated the checked-in canonical/latest Ray validation evidence so the current live-validation surfaces no longer record `python -c "print('hello from ray')"`.
- Added regression coverage in `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go` to keep the Ray smoke script, docs, and active evidence Python-free.
- Validation results:
  - `find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l` -> `0`
  - `bash -n bigclaw-go/scripts/e2e/ray_smoke.sh` -> exit `0`
  - `cd bigclaw-go && go test ./internal/regression -run 'Test(E2EMigrationDocListsOnlyActiveEntrypoints|RaySmokeEntrypointStaysNative|ActiveRayValidationEvidenceAvoidsPythonEntrypoints|LiveValidationIndexStaysAligned|LiveValidationSummaryStaysAligned)$'` -> `ok   bigclaw-go/internal/regression 0.853s`
- Git:
  - branch: `BIG-GO-1356`
  - commit: `c24319f2` (`BIG-GO-1356: remove Ray Python smoke entrypoint`)
  - push: `git push -u origin BIG-GO-1356` -> branch tracks `origin/BIG-GO-1356`
