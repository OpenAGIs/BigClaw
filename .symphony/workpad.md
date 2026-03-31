Issue: BIG-GO-1029

Plan
- Remove the remaining Python operator wrapper scripts in `scripts/ops` that still act as residual migration shims over the Go CLI.
- Replace them with shell-native wrappers that preserve the same command behavior through `scripts/ops/bigclawctl`, including legacy workspace flag/default translation where needed.
- Update directly coupled docs and tests so the repo no longer advertises or validates the deleted `.py` entrypoints.
- Run targeted validation for the wrapper and documentation surface, then capture the file-count and packaging impact before commit/push.

Acceptance
- Changes stay scoped to the remaining repo-level residual Python assets that still back the Go script/operator surface plus directly coupled tests/docs.
- The repository `.py` file count decreases from this change.
- The operator wrapper surface remains executable through non-Python scripts with equivalent behavior for `github-sync`, `refill`, and workspace bootstrap/validate entrypoints.
- Final report states the impact on `py`/`go` file counts and on `pyproject.toml`/`setup.py` or `setup.cfg`.

Validation
- `find scripts/ops -maxdepth 1 -type f | sort`
- `find . -type f \( -name '*.py' -o -name '*.go' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0} /\\.py$/{py++} /\\.go$/{go++} END{print "py="py" go="go}'`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclaw-github-sync --help`
- `bash scripts/ops/bigclaw-refill-queue --help`
- `bash scripts/ops/bigclaw-workspace-bootstrap --help`
- `bash scripts/ops/symphony-workspace-bootstrap --help`
- `bash scripts/ops/symphony-workspace-validate --help`
- `git status --short`
