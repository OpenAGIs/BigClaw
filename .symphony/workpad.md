Issue: BIG-GO-1031

Plan
- Remove the remaining root-level Python operator shim entrypoints in `scripts/ops/*.py` that keep the repo presenting Python-first execution surfaces.
- Delete `src/bigclaw/legacy_shim.py` once the shim wrappers are gone, and update directly coupled Go tests that still freeze the removed Python shim file list.
- Tighten root documentation so the repository advertises Go-only build entrypoints and no longer recommends Python wrapper commands from the root.
- Run scoped validation for file-count deltas, Go tests, and operator entrypoint help/status checks; record exact commands and results.
- Commit the scoped change set and push it to the remote branch for `BIG-GO-1031`.

Acceptance
- `pyproject.toml` and `setup.py` remain absent from the repository tree.
- Python file count decreases within the scope of this issue by deleting the root operator shim `.py` files and their shared shim helper.
- Root build/operator documentation points to Go-only build entrypoints and `scripts/ops/bigclawctl` instead of Python packaging or Python shim commands.
- Directly coupled Go tests continue to pass after removing the deleted Python shim surface.

Validation
- `test ! -e pyproject.toml && test ! -e setup.py`
- `find scripts/ops -maxdepth 1 -name '*.py' | sort`
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | sort | wc -l`
- `find . -name '*.go' | sort | wc -l`
- `gofmt -w bigclaw-go/internal/legacyshim/compilecheck.go bigclaw-go/internal/legacyshim/compilecheck_test.go bigclaw-go/cmd/bigclawctl/main_test.go bigclaw-go/internal/regression/go_only_build_surface_test.go`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl ./internal/regression -run 'Test(FrozenCompileCheckFilesUsesFrozenShimList|CompileCheckRunsPyCompileAgainstFrozenShimList|RunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens|RootGoOnlyBuildSurfaceStaysAligned)'`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl --help`
- `git diff --stat && git status --short`
