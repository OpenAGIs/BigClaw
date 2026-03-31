Issue: BIG-GO-1030

Plan
- Remove the residual repo-root Python operator wrappers in `scripts/ops/*.py` and retire `src/bigclaw/legacy_shim.py` now that `scripts/ops/bigclawctl` is the Go-first entrypoint.
- Retire the frozen `python -m bigclaw` package entrypoint in `src/bigclaw/__main__.py` and align the remaining compatibility manifest/tests with that retired state.
- Inline the last deprecation helper into `src/bigclaw/runtime.py`, retire `src/bigclaw/deprecation.py`, and keep `bigclawctl legacy-python compile-check` valid when no frozen Python files remain.
- Remove dead Python-only validation-policy code that is no longer referenced anywhere in the repo.
- Remove additional orphan Python modules that have no package exports, tests, or repo consumers.
- Remove dead Python mirrors whose active ownership already lives in Go and whose repo references are documentation-only.
- Update directly coupled Go tests and legacy compile-check fixtures so they only cover the remaining Python compatibility files that still exist.
- Refresh repo docs that still present the deleted Python wrappers as valid entrypoints.
- Run targeted validation around the Go legacy-shim package and the `bigclawctl` workspace/github-sync/refill entrypoints, then capture exact commands and results.
- Commit the scoped change set and push it to the remote branch.

Acceptance
- Changes stay scoped to residual repo-root Python compatibility assets plus directly coupled docs/tests.
- Repository `.py` file count decreases as a direct result of this issue.
- The repo no longer carries an executable `python -m bigclaw` package entrypoint.
- The repo no longer carries standalone Python deprecation/helper shim files for the retired package entrypoints.
- Dead isolated Python-only modules without runtime consumers are removed instead of being left as orphan assets.
- Orphan Python source files with no repo references are retired to keep the physical tree aligned with active ownership.
- Python mirrors that only duplicate Go-owned queue/tooling behavior are removed when they no longer serve tests or imports.
- Supported operator paths point to `scripts/ops/bigclawctl` instead of deleted Python wrappers.
- Final report states the impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` presence.

Validation
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `gofmt -w bigclaw-go/internal/legacyshim/compilecheck.go bigclaw-go/internal/legacyshim/compilecheck_test.go bigclaw-go/internal/legacyshim/wrappers_test.go bigclaw-go/cmd/bigclawctl/main_test.go`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl -run 'TestFrozenCompileCheckFilesUsesFrozenShimList|TestCompileCheckRunsPyCompileAgainstFrozenShimList|TestCompileCheckReturnsCompilerOutputOnFailure|TestRunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens'`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl refill --local-issues local-issues.json`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `cd bigclaw-go && go test ./internal/regression -run TestLegacyMainlineCompatibilityManifestStaysAligned`
- `git diff --stat && git status --short`
