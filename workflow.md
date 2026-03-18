---
tracker:
  kind: local
  path: ./local-issues.json
  active_states:
    - Todo
    - In Progress
    - In Review
    - Merging
    - Rework
  terminal_states:
    - Closed
    - Cancelled
    - Canceled
    - Duplicate
    - Done

polling:
  interval_ms: 15000

workspace:
  root: ~/code/symphony-workspaces

server:
  host: 127.0.0.1
  port: 4000

hooks:
  after_create: |
    set -eu
    bash "$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl" workspace bootstrap --workspace "$SYMPHONY_WORKSPACE" --issue "$SYMPHONY_ISSUE_IDENTIFIER" --repo-url "${SYMPHONY_BOOTSTRAP_REPO_URL:-git@github.com:OpenAGIs/BigClaw.git}" --github-url "${SYMPHONY_BOOTSTRAP_GITHUB_URL:-git@github.com:OpenAGIs/BigClaw.git}" --default-branch "${SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH:-main}" --cache-base "${SYMPHONY_BOOTSTRAP_CACHE_BASE:-$HOME/.cache/symphony/repos}" --cache-key "${SYMPHONY_BOOTSTRAP_CACHE_KEY:-openagis-bigclaw}" --json
    git config user.email "dcjcloud@gmail.com"
    git config user.name "native cloud"
    gh auth setup-git || true
    bash scripts/ops/bigclawctl github-sync install
    bash scripts/ops/bigclawctl github-sync status --json
  before_run: |
    set -eu
    bash scripts/ops/bigclawctl github-sync install
    bash scripts/ops/bigclawctl github-sync sync --allow-dirty --json
  after_run: |
    set -eu
    bash scripts/ops/bigclawctl github-sync status --json --require-clean --require-synced || true
  before_remove: |
    set -eu
    bash "$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl" workspace cleanup --workspace "$SYMPHONY_WORKSPACE" --issue "$SYMPHONY_ISSUE_IDENTIFIER" --repo-url "${SYMPHONY_BOOTSTRAP_REPO_URL:-git@github.com:OpenAGIs/BigClaw.git}" --github-url "${SYMPHONY_BOOTSTRAP_GITHUB_URL:-git@github.com:OpenAGIs/BigClaw.git}" --default-branch "${SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH:-main}" --cache-base "${SYMPHONY_BOOTSTRAP_CACHE_BASE:-$HOME/.cache/symphony/repos}" --cache-key "${SYMPHONY_BOOTSTRAP_CACHE_KEY:-openagis-bigclaw}" --json || true
  timeout_ms: 120000

agent:
  max_concurrent_agents: 10
  max_concurrent_agents_by_state:
    Todo: 4
    In Progress: 6
    In Review: 2
  max_turns: 20

codex:
  command: codex --config model_reasoning_effort=medium app-server
  read_timeout_ms: 60000
  approval_policy: never
  thread_sandbox: danger-full-access
  turn_sandbox_policy:
    type: dangerFullAccess
---

You are working on a local tracker issue `{{ issue.identifier }}`.

Primary operating mode:
- Use the local tracker store in `local-issues.json` as the source of truth for planning, state transitions, and completion comments.
- Use Symphony as the parallel orchestration layer whenever the current project has multiple independent slices that can advance concurrently.
- Treat `bigclaw-go` as the sole implementation mainline for new development; only touch `src/bigclaw` when migrating a required surface to Go or explicitly freezing a legacy Python path.
- Prefer 2-4 active child tickets in parallel when the work can be safely decomposed without merge conflicts.
- Keep at least 2 tickets in `In Progress` whenever the project still has parallel-safe `Todo` slices available.
- Use `Backlog` rather than `Todo` for standby slices that should not be picked up immediately; Symphony treats `Todo` as runnable work.
- Keep each parallel slice small, code-backed, and independently verifiable.
- Use `docs/parallel-refill-queue.json` as the canonical refill order and `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json` as the reusable manual/automated refill entrypoint.
- Use `docs/go-mainline-cutover-issue-pack.md` as the canonical project brief behind the local tracker issue set.
- Mirror `elixir/WORKFLOW.md`'s unattended posture: keep ticket state current, keep GitHub current throughout execution, and avoid leaving active work without a synced branch state.

Hook-backed GitHub sync:
- Workspace `after_create` now uses the Go-first `scripts/ops/bigclawctl workspace bootstrap` entrypoint, with repo URL / branch / cache location supplied via `SYMPHONY_BOOTSTRAP_*` env vars.
- Workspace `before_run` re-applies `core.hooksPath=.githooks` and auto-pushes any clean unsynced branch head at the start of every turn.
- Repository `.githooks/post-commit` and `.githooks/post-rewrite` automatically push the active branch and verify local/remote SHA equality after each commit or amend.
- Workspace `after_run` emits a final sync audit and flags dirty or unsynced workspaces in Symphony logs.
- Workspace `before_remove` prunes the issue worktree from the shared seed repo so terminal issue cleanup does not leak worktree metadata.
- The same bootstrap layer is designed to be copied into other repos without renaming the workflow hooks; only the env defaults need to change.
- Use `BIGCLAW_SKIP_AUTO_SYNC=1` only for exceptional local recovery flows; normal issue execution must leave auto-sync enabled.

Execution protocol:
1. Start by checking whether the current project or epic still has open child tickets in `Todo` / `In Progress`.
2. If no suitable child ticket exists, create the next smallest high-value local issue before coding by using `symphony issue create --workflow workflow.md ...` when available or by updating `local-issues.json`.
3. Claim work explicitly in the local tracker by moving the ticket to `In Progress` before implementation.
4. After every substantive code, doc, config, or test update, immediately `git add` + `git commit`; the installed hooks must then auto-push the current issue branch, verify local/remote SHA equality, and keep the PR current throughout execution rather than only at issue closeout.
5. Never end a coding turn with uncommitted or unpushed substantive changes unless blocked by a true external failure; if blocked, record the exact blocker and keep the issue active.
6. Complete code, tests, GitHub push verification, and PR/body refresh before marking the ticket `Done`.
7. Add a local tracker comment with:
   - what changed,
   - validation commands/results,
   - commit sha,
   - PR URL.
8. When one slice is done, immediately pick or create the next parallel-safe slice and continue.
9. If the dashboard shows fewer than 2 running slices while safe `Todo` work exists, move the next ticket(s) to `In Progress` and let Symphony refill capacity on the next polling cycle.
10. When manual or unattended refill is needed, prefer `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json` so queue activation follows the recorded issue order without calling Linear.

GitHub verification is mandatory before and during execution:
- Every substantive code update must be pushed to the configured GitHub branch instead of waiting until final completion.
- Every run must prove that the latest local commit exists on the configured GitHub branch.
- Do not treat `git push` success alone as sufficient; compare local and remote branch SHAs.
- Keep the active PR title/body aligned with the total branch scope after each push.
- If a workspace branch does not yet exist on origin, create it with the first push before continuing implementation.
- If auto-sync fails because the remote moved, resolve the branch divergence immediately before continuing the local issue.

Follow the same execution quality bar as the root workflow, and ensure every run ends with:
1) validation evidence,
2) successful `git push`,
3) local/remote SHA equality confirmation,
4) `git log -1 --stat` output capture.
