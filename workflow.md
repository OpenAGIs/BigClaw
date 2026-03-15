---
tracker:
  kind: linear
  api_key: "$LINEAR_API_KEY"
  project_slug: "8a198fec793e"
  active_states:
    - Todo
    - In Progress
    - In Review
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

hooks:
  after_create: |
    git clone https://$GITHUB_TOKEN@github.com/OpenAGIs/BigClaw.git .
    git config user.email "dcjcloud@gmail.com"
    git config user.name "native cloud"
    git remote set-url origin https://$GITHUB_TOKEN@github.com/OpenAGIs/BigClaw.git
    # [ROBUST PUSH POLICY]: Always verify push completion before ticket completion.
    # Instruction:
    #   1. git push origin <branch>
    #   2. local_sha=$(git rev-parse HEAD)
    #   3. remote_sha=$(git ls-remote --heads origin <branch> | awk '{print $1}')
    #   4. test "$local_sha" = "$remote_sha"
    #   5. git log -1 --stat

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

You are working on a Linear ticket `{{ issue.identifier }}`.

Primary operating mode:
- Use Linear as the source of truth for planning, state transitions, and completion comments.
- Use Symphony as the parallel orchestration layer whenever the current project has multiple independent slices that can advance concurrently.
- Prefer 2-4 active child tickets in parallel when the work can be safely decomposed without merge conflicts.
- Keep at least 2 tickets in `In Progress` whenever the project still has parallel-safe `Todo` slices available.
- Keep each parallel slice small, code-backed, and independently verifiable.
- Use `docs/parallel-refill-queue.json` as the canonical refill order and `scripts/ops/bigclaw_refill_queue.py` as the reusable manual/automated refill entrypoint.

Execution protocol:
1. Start by checking whether the current project or epic still has open child tickets in `Todo` / `In Progress`.
2. If no suitable child ticket exists, create the next smallest high-value Linear issue before coding.
3. Claim work explicitly in Linear by moving the ticket to `In Progress` before implementation.
4. Complete code, tests, GitHub push verification, and PR/body refresh before marking the ticket `Done`.
5. Add a Linear comment with:
   - what changed,
   - validation commands/results,
   - commit sha,
   - PR URL.
6. When one slice is done, immediately pick or create the next parallel-safe slice and continue.
7. If the dashboard shows fewer than 2 running slices while safe `Todo` work exists, move the next ticket(s) to `In Progress` and let Symphony refill capacity on the next polling cycle.
8. When manual or unattended refill is needed, prefer `python3 scripts/ops/bigclaw_refill_queue.py --apply --watch` so queue activation follows the recorded issue order.

GitHub verification is mandatory before completion:
- Every run must prove that the latest local commit exists on the configured GitHub branch.
- Do not treat `git push` success alone as sufficient; compare local and remote branch SHAs.
- Keep the active PR title/body aligned with the total branch scope after each push.

Follow the same execution quality bar as the root workflow, and ensure every run ends with:
1) validation evidence,
2) successful `git push`,
3) local/remote SHA equality confirmation,
4) `git log -1 --stat` output capture.
