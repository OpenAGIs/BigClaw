# BIG-GO-902 Script Layer to Go CLI Migration Plan

## Goal

Move the remaining repo-level script automation entrypoints onto `bigclaw-go/cmd/bigclawctl`
subcommands, while preserving the existing file names as compatibility shims during the
operator cutover window.

## This Slice

The implemented migration batches in this issue move the following repo-root entrypoints behind the
Go CLI:

- `scripts/create_issues.py` -> `bigclawctl create-issues`
- `scripts/dev_smoke.py` -> `bigclawctl dev-smoke`
- `scripts/ops/bigclaw-symphony` -> `bigclawctl symphony`
- `scripts/ops/bigclaw-issue` -> `bigclawctl issue`
- `scripts/ops/bigclaw-panel` -> `bigclawctl panel`
- `scripts/ops/bigclaw_github_sync.py` -> `bigclawctl github-sync`
- `scripts/ops/bigclaw_refill_queue.py` -> `bigclawctl refill`
- `scripts/ops/bigclaw_workspace_bootstrap.py` -> `bigclawctl workspace ...`
- `scripts/ops/symphony_workspace_bootstrap.py` -> `bigclawctl workspace ...`
- `scripts/ops/symphony_workspace_validate.py` -> `bigclawctl workspace validate`

The compatibility layer is intentionally thin:

- Python root shims only proxy into `scripts/ops/bigclawctl`.
- Python `scripts/ops/*_*.py` shims only translate legacy flags/defaults before dispatching into `scripts/ops/bigclawctl`.
- Bash ops aliases only proxy into `scripts/ops/bigclawctl`.
- Behavioral ownership now lives in Go under `bigclaw-go/cmd/bigclawctl`.

## First-Batch Change List

### Added Go CLI commands

- `create-issues`
  - ports the canned v1 and v2-ops GitHub issue bootstrap plans from Python into Go
  - keeps `GITHUB_TOKEN`/`BIGCLAW_PLAN` compatible defaults
  - supports `--json` for automation callers
- `dev-smoke`
  - replaces the frozen Python scheduler smoke path with a Go scheduler decision check
  - preserves the `smoke_ok <executor>` human output for compatibility callers
- `symphony`
  - centralizes repo-root workflow resolution, bootstrap env injection, and CLI discovery
- `issue`
  - preserves the local tracker shortcuts for `list/create/state/comment`
  - falls back to `symphony issue --workflow workflow.md` for the remaining surface
- `panel`
  - proxies `symphony panel --workflow workflow.md`

### Compatibility shims kept in place

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `scripts/ops/bigclaw-symphony`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`

These shims should remain until operator docs and external automation references are updated to
invoke `bash scripts/ops/bigclawctl ...` directly.

## Remaining Backlog

- Evaluate whether `scripts/dev_bootstrap.sh` should become a Go-owned bootstrap command or remain
  a shell-only environment bootstrapper.
- Collapse `scripts/ops/bigclawctl` itself from `go run` wrapper into a compiled release binary
  path for production/operator use.
- Audit `bigclaw-go/scripts/*` migration helpers and E2E utilities separately. They are out of
  scope for this repo-root script migration slice.
- Update repo docs that still present Python entrypoints as a primary path instead of a shim path.

## Validation Commands

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `PYTHONPATH=src python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py`
- `bash scripts/ops/bigclawctl dev-smoke`
- `PYTHONPATH=src python3 scripts/dev_smoke.py`
- `python3 scripts/create_issues.py --help`
- `PYTHONPATH=src python3 scripts/ops/bigclaw_github_sync.py status --json`
- `PYTHONPATH=src python3 scripts/ops/bigclaw_refill_queue.py --help`
- `PYTHONPATH=src python3 scripts/ops/symphony_workspace_validate.py --help`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl panel --help`
- `bash scripts/ops/bigclawctl symphony --help`

## Regression Surface

- GitHub bootstrap automation:
  `create-issues` must preserve title/body/label parity and skip already-created issues.
- Local tracker shortcuts:
  `issue state/comment` positional shortcuts must still map onto the same local-issues behaviors.
- Legacy workspace wrappers:
  `--issues`, `--report-file`, and `--no-cleanup` still need to translate to the Go workspace
  validation flags without changing existing automation call sites.
- Symphony invocation:
  CLI discovery order and workflow binding must still prefer the repo-adjacent checkout before
  falling back to `PATH`.
- Operator docs:
  repo instructions must not imply that the Python scripts are still the implementation mainline.

## Branch and PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `BIG-GO-902: migrate repo script entrypoints to Go CLI`
- PR description focus:
  - migrated entrypoints and retained shims
  - exact validation commands and results
  - explicit note that `bigclaw-go/scripts/*` is deferred to a follow-up migration lane

## Risks

- `create-issues` still relies on a static in-repo issue plan map. If the canonical issue list
  changes frequently, the next step should move plan data into versioned JSON/YAML fixtures.
- Some compatibility wrappers still require `PYTHONPATH=src` when run directly. The preferred
  operator path remains `bash scripts/ops/bigclawctl ...` until packaging or console-script
  installation is standardized.
- `scripts/ops/bigclawctl` still uses `go run`, so first-run latency and local Go toolchain
  availability remain operator dependencies.
- `symphony`/`issue`/`panel` are now implemented in Go but still depend on an external Symphony
  binary; this issue does not change that external dependency boundary.
