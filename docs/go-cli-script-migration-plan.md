# BIG-GO-902 Script Layer to Go CLI Migration Plan

## Goal

Move the remaining repo-level script automation entrypoints onto `bigclaw-go/cmd/bigclawctl`
subcommands and retire the legacy Python file names once docs and automation have moved to the
Go-first path.

## This Slice

The implemented migration batches in this issue move these entrypoints behind the Go CLI.

### Repo-root entrypoints

- retired `scripts/create_issues.py`; use `bigclawctl create-issues`
- root dev smoke path is Go-only: use `bigclawctl dev-smoke`
- `scripts/ops/bigclaw-symphony` -> `bigclawctl symphony`
- `scripts/ops/bigclaw-issue` -> `bigclawctl issue`
- `scripts/ops/bigclaw-panel` -> `bigclawctl panel`
- retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`
- retired the refill Python wrapper; use `bigclawctl refill`
- retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bigclawctl workspace bootstrap`
- retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bigclawctl workspace bootstrap`
- retired `scripts/ops/symphony_workspace_validate.py`; use `bigclawctl workspace validate`

### `bigclaw-go/scripts/*` automation surface

- `bigclaw-go/scripts/e2e/` operator entrypoints now dispatch through `bigclawctl automation e2e ...`
- benchmark automation is Go-owned via `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`
- migration automation is Go-owned via `bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`

The remaining compatibility layer is intentionally thin:

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

- `scripts/ops/bigclaw-symphony`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`

The remaining shell aliases exist only for operator convenience. Repo-root Python wrapper files
have been retired in favor of direct `bash scripts/ops/bigclawctl ...` entrypoints.

## Remaining Backlog

- Continue shrinking `scripts/dev_bootstrap.sh` so it stays a validation-only shell helper and
  does not reintroduce Python environment management at the repository root.
- Collapse `scripts/ops/bigclawctl` itself from `go run` wrapper into a compiled release binary
  path for production/operator use.
- Update repo docs that still present Python entrypoints as a primary path instead of a shim path.

## Validation Commands

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/legacyshim`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `bash scripts/dev_bootstrap.sh`
- `BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl panel --help`
- `bash scripts/ops/bigclawctl symphony --help`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`

## Regression Surface

- GitHub bootstrap automation:
  `create-issues` must preserve title/body/label parity and skip already-created issues.
- Local tracker shortcuts:
  `issue state/comment` positional shortcuts must still map onto the same local-issues behaviors.
- Root compatibility retirement:
  repo operators must invoke the Go-first repo-root entrypoints and keep the removed Python shims absent:
  `bash scripts/ops/bigclawctl ...` entrypoints.
- BigClaw automation helpers:
  `/healthz`, `/tasks/:id`, and `/events` polling plus report serialization must remain compatible
  for the migrated `bigclaw-go/scripts/*` automation callers.
- Symphony invocation:
  CLI discovery order and workflow binding must still prefer the repo-adjacent checkout before
  falling back to `PATH`.
- Operator docs:
  repo instructions must not imply that the Python scripts are still the implementation mainline.
- GitHub sync cutover:
  `scripts/ops/bigclaw_github_sync.py` must stay deleted and hooks/workflow/CI should invoke
  `bash scripts/ops/bigclawctl github-sync ...` directly.

## Branch and PR Suggestion

- Branch: `feat/BIG-GO-902-go-cli-script-migration`
- PR title: `BIG-GO-902: migrate repo script entrypoints to Go CLI`
- PR description focus:
  - migrated entrypoints and retained shims
  - exact validation commands and results
  - explicit note that `bigclaw-go/scripts/*` is now Python-free and Go-owned

## Risks

- `create-issues` still relies on a static in-repo issue plan map. If the canonical issue list
  changes frequently, the next step should move plan data into versioned JSON/YAML fixtures.
- `scripts/ops/bigclawctl` still uses `go run`, so first-run latency and local Go toolchain
  availability remain operator dependencies.
- `run-task-smoke --autostart` and `soak-local --autostart` still depend on ephemeral port
  reservation before `bigclawd` binds, so local port races remain possible.
- `symphony`/`issue`/`panel` are now implemented in Go but still depend on an external Symphony
  binary; this issue does not change that external dependency boundary.
