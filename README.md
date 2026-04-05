# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

`bigclaw-go` is the implementation mainline. The root legacy Python package and
root Python test suite have been removed as part of the Go-only retirement
sweep.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `bigclaw-go`: current Go implementation mainline
  - `cmd/bigclawd`: service entrypoint
  - `internal/*`: queue, scheduler, worker, events, API, reporting, and control-plane packages
  - `docs/*`: Go control-plane validation and migration evidence
- `docs/symphony-repo-bootstrap-template.md`: repo-agnostic shared mirror + worktree bootstrap template
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `docs/reports/BIG-GO-1480-python-reality-audit.md`: root Python retirement audit

## Go mainline quick start (recommended)

```bash
cd BigClaw/bigclaw-go
go test ./...
go run ./cmd/bigclawd
curl localhost:8080/healthz
bash ../scripts/ops/bigclawctl github-sync status --json
```

## Local orchestration quick start

BigClaw now defaults to a repo-native local tracker in [`local-issues.json`](./local-issues.json).
Use these entrypoints to keep the remaining Go-mainline migration slices moving without waiting on
Linear issue capacity:

```bash
bash scripts/ops/bigclaw-issue list
bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json
bash scripts/ops/bigclaw-symphony
bash scripts/ops/bigclaw-panel
```

Notes:

- `bash scripts/ops/bigclaw-symphony` starts Symphony against [`workflow.md`](./workflow.md) and
  serves the local issue dashboard at `http://127.0.0.1:4000/`.
- `bash scripts/ops/bigclaw-panel` prints the configured dashboard URL for the current workflow.
- `bash scripts/ops/bigclaw-issue ...` wraps `symphony issue ... --workflow workflow.md` so local
  issue creation and state changes stay pinned to this repository's tracker file.
- `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` promotes the next
  queued local issues to `In Progress` using the canonical order in `docs/parallel-refill-queue.json`.

## Go smoke verify

```bash
cd BigClaw/bigclaw-go
go test ./...
go run ./cmd/bigclawd &
curl localhost:8080/healthz
bash ../scripts/ops/bigclawctl github-sync status --json
```

## Quality gates

Go mainline:

```bash
cd BigClaw/bigclaw-go
go test ./...
```

Go-first bootstrap helper:

```bash
bash scripts/dev_bootstrap.sh
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
The Go-first BigClaw entrypoint is `scripts/ops/bigclawctl`. Active runtime,
control-plane, and validation development belongs in `bigclaw-go/internal/*`
and `bigclaw-go/scripts/*`.
