# BIG-GO-1057 Workpad

## Plan
- Confirm every live entry surface that still references `scripts/ops/bigclaw_github_sync.py`.
- Remove `scripts/ops/bigclaw_github_sync.py`.
- Update hooks, README, migration docs, and regression tests to use `bash scripts/ops/bigclawctl github-sync ...` instead of the deleted Python wrapper.
- Add or adjust regression coverage so this slice asserts the deleted wrapper stays absent and the Go-first entrypoint remains usable.
- Run targeted validation, record exact commands and results, then commit and push the branch.

## Acceptance
- `scripts/ops/bigclaw_github_sync.py` is deleted from the repo.
- Live operator entry surfaces no longer call the deleted Python wrapper.
- README, hooks, workflow-adjacent docs, and CI-facing references for this entrypoint point at `scripts/ops/bigclawctl` or equivalent shell/Go entry.
- Regression coverage pins the removal so the Python entrypoint does not return.
- `.py` file count decreases relative to the pre-change baseline.

## Validation
- Capture pre/post `.py` file counts with `rg --files . | rg '\\.py$' | wc -l`.
- Run targeted Go tests covering the github-sync CLI and purge regression.
- Run the Go-first github-sync help/status commands through `scripts/ops/bigclawctl`.
- Record exact commands and pass/fail outcomes in the closeout response.
