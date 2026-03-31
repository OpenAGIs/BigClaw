Issue: BIG-GO-1030

Plan
- Remove the residual repo-root Python operator wrappers in `scripts/ops/*.py` and retire `src/bigclaw/legacy_shim.py` now that `scripts/ops/bigclawctl` is the Go-first entrypoint.
- Retire the frozen `python -m bigclaw` package entrypoint in `src/bigclaw/__main__.py` and align the remaining compatibility manifest/tests with that retired state.
- Inline the last deprecation helper into `src/bigclaw/runtime.py`, retire `src/bigclaw/deprecation.py`, and keep `bigclawctl legacy-python compile-check` valid when no frozen Python files remain.
- Remove dead Python-only validation-policy code that is no longer referenced anywhere in the repo.
- Remove additional orphan Python modules that have no package exports, tests, or repo consumers.
- Remove dead Python mirrors whose active ownership already lives in Go and whose repo references are documentation-only.
- Remove the dead standalone workspace bootstrap CLI module now that active bootstrap behavior lives in `workspace_bootstrap.py` and `scripts/ops/bigclawctl`.
- Remove the dead workspace bootstrap validation helper module that is only referenced by its own Python regression.
- Remove orphan Python model/report modules that survive only through unused package exports.
- Remove isolated Python persistence helpers that only remain to support their own legacy tests.
- Remove isolated Python contract modules that only remain through stale package exports and dedicated test files.
- Remove isolated Python Git automation helpers that are fully superseded by the Go CLI path and only remain through dedicated legacy tests.
- Remove isolated Python repo-review helper lanes that are fully mirrored by Go ownership and only remain through dedicated legacy tests.
- Inline the last tiny repo-link helper into its only Python consumer and retire the standalone helper module/test pair.
- Remove the stale Python intake mapping export now that Go owns the active mapping contract and no Python consumers remain.
- Retire the isolated Python event-bus helper lane that now only survives through package exports and a dedicated test file.
- Retire the isolated Python repo discussion board helper lane now that only dedicated Python tests and stale docs still reference it and the Go repo surface already carries equivalent coverage.
- Retire the isolated Python repo commit helper module when it has no in-repo imports, no tests, and only historical docs still mention it.
- Retire the isolated Python saved-view helper lane once the only remaining usage is package exports, dedicated tests, and planning/docs references that can point at the Go-owned saved-view surface instead.
- Retire the isolated Python intake connector and workflow-definition helpers once they only remain through package exports, dedicated tests, and documentation that already points to Go-owned parity surfaces.
- Collapse the shared Python run-detail renderer into `reports.py` so the repo no longer carries a standalone helper module for that HTML surface.
- Retire the standalone Python saved-views model/audit helper lane once planning evidence and validation point at the Go-owned product surface.
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
- Standalone Python CLI leaf modules with no imports or tests are retired instead of being kept as unused repo assets.
- Python validation/helper modules that only exist to support isolated legacy tests are retired with those test slices.
- Python modules that are only reachable via stale `__init__` exports are removed together with those exports.
- Python modules with no imports or exports and only one dedicated test slice are retired with that test slice.
- Python contract/report surfaces with no runtime consumers are retired together with their export and regression-only test coverage.
- Python Git/ops helpers with no package exports and only dedicated legacy tests are retired with those tests once the Go CLI path is already validated.
- Python repo-side helper modules with no runtime consumers and only dedicated legacy tests are retired together to keep the physical tree aligned with Go-owned repo surfaces.
- Tiny Python helper modules with one runtime consumer are inlined and retired when that is cheaper than carrying another standalone `.py` file.
- Python export-only compatibility modules are retired once their last in-repo consumers disappear.
- Python event/helper lanes with no runtime consumers are retired together with their package-export surface and dedicated tests.
- Python repo discussion board helpers that are not exported and only survive via dedicated Python tests are retired together with those tests once Go-owned repo surface coverage already exists.
- Python repo commit helper modules with no package exports, runtime imports, or tests are removed instead of being kept as dead physical assets.
- Python saved-view helpers that are no longer imported by runtime code are removed together with their dedicated tests and stale Python evidence links, while planning/docs are repointed to Go ownership.
- Python intake connector and workflow-definition helpers with no live runtime consumers are removed together with stale exports/tests, while cutover docs continue to point to Go ownership.
- Shared Python helper modules that only exist to serve one remaining ownership cluster are inlined into that owner module and then deleted.
- Python product/helper lanes are retired when Go already owns the active implementation and the remaining Python references are only planning or regression scaffolding.
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
- `python3 -m pytest tests/test_reports.py tests/test_observability.py -q`
- `git diff --stat && git status --short`
