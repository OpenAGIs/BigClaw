# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

`bigclaw-go` is the current implementation mainline for new development. The
repository root now exposes Go-only build entrypoints; the last physical
`src/bigclaw` package asset has been retired and the root is no longer a Python
package surface.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `bigclaw-go`: current Go implementation mainline
  - `cmd/bigclawd`: service entrypoint
  - `internal/*`: queue, scheduler, worker, events, API, reporting, and control-plane packages
  - `docs/*`: Go control-plane validation and migration evidence
- `docs/symphony-repo-bootstrap-template.md`: repo-agnostic shared mirror + worktree bootstrap template
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: retired physical package path; kept only as a historical reference in migration documentation and validation reports
- `tests/`: unit tests

## Root Go quick start (recommended)

```bash
cd BigClaw
make test
make build
make run
curl localhost:8080/healthz
bash scripts/ops/bigclawctl github-sync status --json
bash scripts/ops/bigclawctl dev-smoke
```

## Local orchestration quick start

BigClaw now defaults to a repo-native local tracker in [`local-issues.json`](./local-issues.json).
Use these entrypoints to keep the remaining Go-mainline migration slices moving without waiting on
Linear issue capacity:

```bash
bash scripts/ops/bigclawctl issue list
bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json
bash scripts/ops/bigclawctl symphony
bash scripts/ops/bigclawctl panel
```

Notes:

- `bash scripts/ops/bigclawctl symphony` starts Symphony against [`workflow.md`](./workflow.md) and
  serves the local issue dashboard at `http://127.0.0.1:4000/`.
- `bash scripts/ops/bigclawctl panel` prints the configured dashboard URL for the current workflow.
- `bash scripts/ops/bigclawctl issue ...` wraps `symphony issue ... --workflow workflow.md` so local
  issue creation and state changes stay pinned to this repository's tracker file.
- `python3 bigclaw-go/scripts/e2e/run_task_smoke.py`,
  `python3 bigclaw-go/scripts/benchmark/soak_local.py`, and
  `python3 bigclaw-go/scripts/migration/shadow_compare.py` now forward into
  `bigclawctl automation ...`; the migration matrix lives in
  [`bigclaw-go/docs/go-cli-script-migration.md`](./bigclaw-go/docs/go-cli-script-migration.md).
- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-symphony`, and `scripts/ops/bigclaw-panel` are
  retained as compatibility wrappers, but the preferred operator path is now `scripts/ops/bigclawctl`.
- `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` promotes the next
  queued local issues to `In Progress` using the canonical order in `docs/parallel-refill-queue.json`.

## Legacy Python migration note

Do not use Python packaging from the repository root. The repository no longer
ships a physical `src/bigclaw` package tree; remaining Python references in the
repo are historical migration evidence or Go-invoked compatibility tooling.

Use the Go-first bootstrap helper when you need the documented compatibility
checks:

```bash
bash scripts/dev_bootstrap.sh
```

## Go smoke verify

```bash
cd BigClaw
make test
make run &
curl localhost:8080/healthz
bash scripts/ops/bigclawctl github-sync status --json
```

## Quality gates

Go mainline:

```bash
make test
make build
```

Go-first bootstrap helper:

```bash
bash scripts/dev_bootstrap.sh
```

Repository compatibility checks:

```bash
bash scripts/ops/bigclawctl legacy-python compile-check --json
pre-commit run --all-files
```

## Quick verify

```bash
git log -1 --stat
git remote -v
git push origin main
```

Repository: https://github.com/OpenAGIs/BigClaw

## Repo-agnostic bootstrap template

Use `docs/symphony-repo-bootstrap-template.md` when you want another Symphony-managed repo to
reuse the same local mirror + `git worktree` pattern without inheriting BigClaw-specific names.
The root Go-only build entrypoints are `make test`, `make build`, and `make run`;
the Go-first operator entrypoint is `scripts/ops/bigclawctl`.

The old `python -m bigclaw` and `python -m bigclaw serve` entrypoints are
retired; use `bash scripts/ops/bigclawctl` and
`go run ./bigclaw-go/cmd/bigclawd` for the active operator and local server
paths. Active runtime development belongs in `bigclaw-go/internal/*`.
