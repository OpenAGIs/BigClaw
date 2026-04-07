# BIG-GO-1576

## Plan
- Materialize the candidate Python asset set and trace current call sites, tests, and Go replacements.
- Remove Python files that are already obsolete or can be replaced directly with existing Go entrypoints.
- Where immediate deletion is not safe, reduce Python to a thin compatibility shim that delegates to Go or a stable shell entrypoint.
- Update tests and documentation references only as needed to keep scope limited to this sweep.
- Run targeted validation for touched paths, capture exact commands/results, then commit and push `BIG-GO-1576`.

## Acceptance
- Produce the explicit list of Python files covered in this sweep.
- Prefer deletion, migration, or replacement with Go implementations / Go commands.
- Any Python file that remains must be a thin compatibility layer with a concrete removal condition documented in code or issue-facing notes.
- Record exact validation commands, outcomes, and residual risks.

## Validation
- Inspect file inventory with `rg --files` / `git grep` against the candidate list.
- Run targeted unit or integration tests covering any touched replacements or shims.
- Run focused search checks to confirm removed Python assets no longer exist or are reduced to compatibility-only logic.
- Review `git diff --stat` and `git status --short` before commit.

## Validation Results
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort` -> no output
- `find src/bigclaw tests scripts/ops bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort` -> no output
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1576'` -> `ok  	bigclaw-go/internal/regression	0.374s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestTopLevelModulePurgeTranche(1|8|14|17)|TestPythonTestTranche17Removed|TestRootScriptResidualSweep|TestE2EScriptDirectoryStaysPythonFree'` -> `ok  	bigclaw-go/internal/regression	0.169s`
