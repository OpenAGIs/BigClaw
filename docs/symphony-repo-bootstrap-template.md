# Symphony repo-agnostic bootstrap template

This template turns Symphony workspace creation into a shared local mirror + `git worktree`
flow, so parallel issue workspaces reuse one cached repository instead of re-cloning on every
issue.

## Required files

Copy these files into the target repository:

- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh` (optional validation wrapper)

The shell entrypoint is generic. Go-owned bootstrap behavior lives behind
`bash scripts/ops/bigclawctl workspace ...`; repositories should not add Python
compatibility shims, `conftest.py`, `pytest.ini`, or other Python bootstrap
manifests when adopting this template.

## Workflow hook template

```yaml
hooks:
  after_create: |
    bash "$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl" workspace bootstrap \
      --workspace "$SYMPHONY_WORKSPACE" \
      --issue "$SYMPHONY_ISSUE_IDENTIFIER" \
      --repo-url "${SYMPHONY_BOOTSTRAP_REPO_URL}" \
      --default-branch "${SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH:-main}" \
      --cache-base "${SYMPHONY_BOOTSTRAP_CACHE_BASE:-$HOME/.cache/symphony/repos}" \
      --cache-key "${SYMPHONY_BOOTSTRAP_CACHE_KEY:-}" \
      --json
  before_remove: |
    bash "$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl" workspace cleanup \
      --workspace "$SYMPHONY_WORKSPACE" \
      --issue "$SYMPHONY_ISSUE_IDENTIFIER" \
      --repo-url "${SYMPHONY_BOOTSTRAP_REPO_URL}" \
      --default-branch "${SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH:-main}" \
      --cache-base "${SYMPHONY_BOOTSTRAP_CACHE_BASE:-$HOME/.cache/symphony/repos}" \
      --cache-key "${SYMPHONY_BOOTSTRAP_CACHE_KEY:-}" \
      --json || true
```

## Environment contract

- `SYMPHONY_BOOTSTRAP_REPO_URL`: required canonical Git remote for the repo.
- `SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH`: optional, defaults to `main`.
- `SYMPHONY_BOOTSTRAP_CACHE_BASE`: optional shared cache root for all mirrored repos.
- `SYMPHONY_BOOTSTRAP_CACHE_KEY`: optional stable cache key. Leave empty to derive it from the
  repo URL.

## Why this avoids repeated downloads

- The first issue for a repo creates one bare mirror under `cache-base/<cache-key>/mirror.git`.
- The same cache root also keeps one shared seed checkout under `cache-base/<cache-key>/seed`.
- Every issue workspace is then created as a lightweight `git worktree` from that seed repo.
- Terminal issue cleanup removes the worktree metadata, but preserves the shared mirror and seed.

## Observable result fields

Bootstrap JSON output includes `cache_root`, `cache_key`, `cache_reused`, `clone_suppressed`,
`mirror_created`, `seed_created`, and `workspace_mode` so operators can tell whether a run warmed
the cache or reused it.

## Validation workflow

Use `bash scripts/ops/bigclawctl workspace validate` plus a 3-issue sample to confirm a repo warms one
mirror/seed cache and then suppresses repeated remote clones for later workspaces.

## BigClaw example

BigClaw uses the same template with defaults baked in:

- repo URL default: `git@github.com:OpenAGIs/BigClaw.git`
- cache key default: `openagis-bigclaw`
- workflow example: `workflow.md`
- Go bootstrap ownership: `scripts/ops/bigclawctl` and `bigclaw-go/internal/bootstrap/*`
