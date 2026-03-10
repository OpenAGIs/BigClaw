---
tracker:
  kind: linear
  api_key: "$LINEAR_API_KEY"
  project_slug: "a49a53e934bc"
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
  interval_ms: 60000

workspace:
  root: ~/code/symphony-workspaces

hooks:
  after_create: |
    git clone https://$GITHUB_TOKEN@github.com/OpenAGIs/BigClaw.git .
    git config user.email "dcjcloud@gmail.com"
    git config user.name "native cloud"
    git remote set-url origin https://$GITHUB_TOKEN@github.com/OpenAGIs/BigClaw.git
    # [ROBUST PUSH POLICY]: Always verify push completion before ticket completion.
    # Instruction: Run `git push origin <branch> && git log -1 --stat`.

agent:
  max_concurrent_agents: 10
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

Follow the same execution quality bar as the root workflow, and ensure every run ends with:
1) validation evidence,
2) successful `git push`,
3) `git log -1 --stat` output capture.
