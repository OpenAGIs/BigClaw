# BIG-GO-1047 Workpad

## Plan
- Audit Python residues for workflow/runtime/scheduler/queue and identify direct in-repo consumers.
- Decouple remaining Python modules from legacy runtime shims where required to keep package imports coherent.
- Delete legacy Python implementation/test files for workflow/runtime/scheduler/queue surfaces and rely on Go mainline packages.
- Add or adjust Go tests only if coverage is needed for removed Python behavior.
- Run targeted validation, capture exact commands/results, verify Python file count decreases.
- Commit and push scoped changes to the remote branch.

## Acceptance
- Python files implementing or directly testing workflow/runtime/scheduler/queue legacy surfaces are removed.
- Remaining Python modules no longer require deleted legacy shim modules to import.
- Go workflow/scheduler/queue packages remain validated through targeted `go test` runs.
- Repository `find . -name "*.py" | wc -l` is lower than before the change.
- Commit message enumerates deleted Python files and added Go files/tests if any.

## Validation
- `find . -name "*.py" | wc -l`
- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/workflow`
- Package-level spot check for remaining Python imports if needed.
