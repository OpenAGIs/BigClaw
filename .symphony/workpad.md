# BIG-GO-1055

## Plan
- Inspect the remaining root-level Python operator shims and every repo surface that still points at them.
- Remove the Python shim entrypoints under `scripts/ops/` so the tracked `.py` count drops as part of this issue.
- Rewrite root-facing documentation and CI/bootstrap surfaces to use the canonical Go-only path: `make ...` and `bash scripts/ops/bigclawctl ...`.
- Add regression coverage that fails if the removed Python entrypoints or root packaging files reappear in root migration surfaces.
- Run targeted repository checks and Go tests, then commit and push the branch.

## Acceptance
- Root packaging entrypoints remain absent: `pyproject.toml` and `setup.py` do not exist at the repository root.
- The Python shim entrypoints `scripts/ops/bigclaw_github_sync.py`, `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` are removed.
- `README.md`, `.github/workflows/ci.yml`, and the root bootstrap path no longer instruct operators or CI to use Python packaging or Python shim entrypoints.
- Validation evidence shows the tracked `.py` file count dropped and the remaining root entrypoint references are Go-only.

## Validation
- `find . -name '*.py' -type f | wc -l`
- `test ! -e pyproject.toml && test ! -e setup.py`
- `test ! -e scripts/ops/bigclaw_github_sync.py && test ! -e scripts/ops/bigclaw_refill_queue.py && test ! -e scripts/ops/bigclaw_workspace_bootstrap.py && test ! -e scripts/ops/symphony_workspace_bootstrap.py && test ! -e scripts/ops/symphony_workspace_validate.py`
- `rg -n "python3 scripts/ops/bigclaw_github_sync\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py|scripts/ops/\\*workspace\\*\\.py|actions/setup-python|pip install pytest|pytest --cov|BIGCLAW_ENABLE_LEGACY_PYTHON|PYTHONDONTWRITEBYTECODE" README.md .github/workflows scripts/dev_bootstrap.sh .githooks`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`

## Execution Result
- Removed the five repo-root Python operator shim files under `scripts/ops/`.
- Switched root README, CI, hooks, and bootstrap guidance to Go-only entrypoints.
- Added `bigclaw-go/internal/regression/root_entrypoint_cutover_test.go` to keep the cutover surfaces aligned.

## Validation Result
- `printf 'before='; git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l; printf 'after='; find . -name '*.py' -type f | wc -l; printf 'deleted_in_diff='; git diff --diff-filter=D --name-only | rg '\.py$' | wc -l`
  - passed: `before=46`, `after=41`, `deleted_in_diff=5`
- `test ! -e pyproject.toml && test ! -e setup.py && test ! -e scripts/ops/bigclaw_github_sync.py && test ! -e scripts/ops/bigclaw_refill_queue.py && test ! -e scripts/ops/bigclaw_workspace_bootstrap.py && test ! -e scripts/ops/symphony_workspace_bootstrap.py && test ! -e scripts/ops/symphony_workspace_validate.py && echo removed`
  - passed: `removed`
- `rg -n "python3 scripts/ops/bigclaw_github_sync\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py|scripts/ops/\\*workspace\\*\\.py|actions/setup-python|pip install pytest|pytest --cov|BIGCLAW_ENABLE_LEGACY_PYTHON|PYTHONDONTWRITEBYTECODE" README.md .github/workflows scripts/dev_bootstrap.sh .githooks`
  - passed with exit code `1` and no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
  - passed
- `bash scripts/dev_bootstrap.sh`
  - passed
