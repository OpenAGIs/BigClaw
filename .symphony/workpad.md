## Codex Workpad

```text
/Users/openagi/code/bigclaw-workspaces/BIG-GO-902
```

### Plan

- [x] Audit `scripts/*.py` and common `scripts/ops/*` automation entrypoints against the current Go CLI surface.
- [x] Implement the first Go CLI migration batch for the remaining high-frequency script entrypoints and switch compatibility shims to the Go subcommands.
- [x] Add a scoped migration plan documenting the remaining backlog, validation commands, regression surface, branch/PR guidance, and risks.
- [x] Run targeted tests for the migrated Go CLI commands, record exact commands/results, then commit and push the issue branch.

### Acceptance Criteria

- [x] A concrete migration plan exists for moving the script layer to Go CLI, including the first implementation batch and remaining compatibility-layer plan.
- [x] Common automation entrypoints migrated in this slice execute through `bigclaw-go/cmd/bigclawctl` subcommands instead of Python/Bash-only logic.
- [x] Validation commands and regression surface are explicitly documented for this migration slice.
- [x] Targeted tests covering the new Go CLI behavior pass.
- [x] Changes are committed and pushed to a remote issue branch.

### Validation

- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go test ./cmd/bigclawctl`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclawctl dev-smoke` -> `smoke_ok local`
- [x] `PYTHONPATH=src python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/dev_smoke.py` -> `smoke_ok local` with expected deprecation warning
- [x] `python3 /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/create_issues.py --help` -> usage for `bigclawctl create-issues`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclawctl issue --help` -> usage for `bigclawctl issue`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw-panel --help` -> usage for `bigclawctl panel`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw-symphony --help` -> usage for `bigclawctl symphony`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclaw-issue list` -> exits 0 against repo-local tracker
- [x] Validation report written to `/Users/openagi/code/bigclaw-workspaces/BIG-GO-902/reports/BIG-GO-902-validation.md`
- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go test ./internal/refill`
- [x] `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status` -> exit 0 and rewrote `docs/parallel-refill-queue.md`
- [x] PR draft written to `/Users/openagi/code/bigclaw-workspaces/BIG-GO-902/reports/BIG-GO-902-pr.md`
- [x] 2026-03-27 web check found no public PR result for the branch and the PR seed URL redirected to GitHub sign-in
- [x] Compare URL recorded for reviewer diff access without needing local git setup
- [x] Closeout index written to `/Users/openagi/code/bigclaw-workspaces/BIG-GO-902/reports/BIG-GO-902-closeout.md`

### Notes

- 2026-03-27: Scope is limited to `BIG-GO-902` script/automation entrypoint migration into the existing Go CLI, with compatibility shims preserved where that minimizes operator churn.
- 2026-03-27: First batch migrated `create-issues`, `dev-smoke`, `symphony`, `issue`, and `panel` into `bigclaw-go/cmd/bigclawctl`, and documented the deferred backlog separately in `docs/go-cli-script-migration-plan.md`.
- 2026-03-27: Committed as `99bb530` and pushed to `origin/feat/BIG-GO-902-go-cli-script-migration`; PR seed URL is `https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-902-go-cli-script-migration`.
- 2026-03-27: Added repo-local tracker closeout for `BIG-GO-902` in `local-issues.json` with a completion comment, branch pointer, validation summary, and deferred-risk note.
- 2026-03-27: No GitHub CLI/token is configured in this workspace, so the branch was pushed and the PR creation path was left at the deterministic seed URL instead of opening the PR via API.
- 2026-03-27: Added `reports/BIG-GO-902-validation.md` so reviewers have a single issue-scoped evidence pack covering delivered scope, exact commands, results, branch, PR seed URL, and deferred risks.
- 2026-03-27: Follow-up docs pass switched the preferred operator examples from wrapper entrypoints (`bigclaw-issue`, `bigclaw-symphony`, `bigclaw-panel`) to direct `scripts/ops/bigclawctl` commands while keeping the wrappers documented as compatibility shims.
- 2026-03-27: Added `reports/BIG-GO-902-pr.md` with a ready-to-paste PR title/body because this workspace can push branches but cannot authenticate to create the GitHub PR directly.
- 2026-03-27: Browser verification found no public PR search hit for this branch/title, and the deterministic PR seed URL redirected to GitHub sign-in, so the remaining blocker is still external GitHub authentication rather than missing repo artifacts.
- 2026-03-27: Added the direct GitHub compare URL so reviewers can inspect the branch diff even before a PR is opened.
- 2026-03-27: Added `reports/BIG-GO-902-closeout.md` as a single-file handoff index for branch, links, validation, artifacts, and the remaining external blocker.
