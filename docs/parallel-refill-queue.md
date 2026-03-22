# BigClaw v5.3 Go Mainline Refill Queue

This file is the human-readable companion to `docs/parallel-refill-queue.json`.
It records the current Go-mainline cutover backlog slices and the refill order
used by the repo-native local tracker in `local-issues.json`.

Linear issue creation is still blocked by workspace issue limits, but BigClaw no
longer waits on Linear to keep issue execution moving.

## Trigger

- Manual one-shot refill:
  - `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json`
- Continuous refill watcher:
  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json`
- Optional dashboard refresh after promotion:
  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json --refresh-url http://127.0.0.1:4000/api/v1/refresh`
- Local issue CLI:
  - `bash scripts/ops/bigclaw-issue list`
  - `bash scripts/ops/bigclaw-issue state BIG-GOM-303 "In Progress"`
- Local dashboard/orchestrator:
  - `bash scripts/ops/bigclaw-symphony`
  - `bash scripts/ops/bigclaw-panel`

## Policy

- Target: keep `2` issues in `In Progress` when issue capacity is available again.
- Target: keep `2` issues in `In Progress` in the local tracker unless a higher
  parallelism cap is explicitly chosen for a branch-safe batch.
- Promote only issues currently in `Backlog` or `Todo`.
- Use the queue order below as the single source of truth for refill priority.
- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.
- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.
- `local-issues.json` is the authoritative issue state backend for ongoing work.
- Use `docs/go-mainline-cutover-issue-pack.md` as the detailed project brief behind this queue.

## Repo Validation

- Current mainline expectation:
  - new implementation work lands in `bigclaw-go`
  - Python paths are migration-only unless explicitly marked otherwise
- Current tracker expectation:
  - issue state lives in `local-issues.json`
  - queue promotion is handled by `bigclawctl refill`
- Repo-native cutover plan:
  - `docs/go-mainline-cutover-issue-pack.md`

## Current batch

- Current repo tranche status as of March 22, 2026:
  - the Go-mainline cutover tranche is complete and merged to `main`
  - `BIG-PAR-220`, `BIG-PAR-221`, `BIG-PAR-222`, `BIG-PAR-223`, `BIG-PAR-224`, `BIG-PAR-225`, `BIG-PAR-226`, `BIG-PAR-227`, `BIG-PAR-228`, `BIG-PAR-229`, `BIG-PAR-230`, and `BIG-PAR-231` are now closed in the repo-native tracker
  - no active follow-up slice remains in the current repo-native queue snapshot
  - run `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` to confirm whether any additional `Todo` slices should be promoted
- Completed slices:
  - `BIG-GOM-301` — unified domain model and intake contract migration
  - `BIG-GOM-302` — risk, policy, and approval semantics migration
  - `BIG-GOM-303` — workflow orchestration and scheduler loop migration
  - `BIG-GOM-304` — observability, reporting, and weekly operations surface migration
  - `BIG-GOM-305` — control center, triage, and operations view migration
  - `BIG-GOM-306` — repo collaboration and lineage surface migration
  - `BIG-GOM-307` — workflow, bootstrap, and GitHub sync toolchain migration
  - `BIG-GOM-308` — Python deprecation and Go-only mainline switch
- Historical first runnable batch once issue creation was available:
  - `BIG-GOM-301` — unified domain model and intake contract migration
  - `BIG-GOM-302` — risk, policy, and approval semantics migration
  - `BIG-GOM-303` — workflow orchestration and scheduler loop migration
  - `BIG-GOM-304` — observability, reporting, and weekly operations surface migration

## Canonical refill order

1. `BIG-GOM-301`
2. `BIG-GOM-302`
3. `BIG-GOM-303`
4. `BIG-GOM-304`
5. `BIG-GOM-305`
6. `BIG-GOM-306`
7. `BIG-GOM-307`
8. `BIG-GOM-308`
9. `BIG-PAR-219`
10. `BIG-PAR-220`
11. `BIG-PAR-221`
12. `BIG-PAR-222`
13. `BIG-PAR-223`
14. `BIG-PAR-224`
15. `BIG-PAR-225`
16. `BIG-PAR-226`
17. `BIG-PAR-227`
18. `BIG-PAR-228`
19. `BIG-PAR-229`
20. `BIG-PAR-230`
21. `BIG-PAR-231`
