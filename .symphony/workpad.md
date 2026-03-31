Issue: BIG-GO-1030

Plan
- Remove the residual repo-root Python operator wrappers in `scripts/ops/*.py` and retire `src/bigclaw/legacy_shim.py` now that `scripts/ops/bigclawctl` is the Go-first entrypoint.
- Retire the frozen `python -m bigclaw` package entrypoint in `src/bigclaw/__main__.py` and align the remaining compatibility manifest/tests with that retired state.
- Update directly coupled Go tests and legacy compile-check fixtures so they only cover the remaining Python compatibility files that still exist.
- Refresh repo docs that still present the deleted Python wrappers as valid entrypoints.
- Run targeted validation around the Go legacy-shim package and the `bigclawctl` workspace/github-sync/refill entrypoints, then capture exact commands and results.
- Commit the scoped change set and push it to the remote branch.

Acceptance
- Changes stay scoped to residual repo-root Python compatibility assets plus directly coupled docs/tests.
- Repository `.py` file count decreases as a direct result of this issue.
- The repo no longer carries an executable `python -m bigclaw` package entrypoint.
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
