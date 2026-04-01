# BIG-GO-1066 Workpad

## Plan
- Confirm the remaining physical Python assets in the requested root-script sweep scope and record the pre-change `.py` baseline.
- Delete the residual `scripts/ops/*.py` compatibility shims now that `scripts/ops/bigclawctl` is the supported Go-first operator entrypoint.
- Update current README and migration docs so they no longer instruct operators to invoke the deleted Python wrappers.
- Extend Go regression coverage so this tranche asserts the deleted script paths stay absent and the Go replacements remain present.
- Run targeted validation, record exact commands/results, then commit and push the issue branch.

## Acceptance
- The current batch asset list is explicit: `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py`.
- Those Python files are removed instead of retained as Python compatibility wrappers.
- Current repo docs point operators at `bash scripts/ops/bigclawctl refill ...` and `bash scripts/ops/bigclawctl workspace ...` rather than the deleted Python entrypoints.
- Regression coverage fails if the deleted Python files reappear and still verifies the Go replacement packages/files exist.
- The repo-wide `.py` file count decreases from the pre-change baseline.

## Validation
- `git checkout -b feat/BIG-GO-1066-root-scripts-residual-sweep`
- `rg --files . | rg '\\.py$' | wc -l`
- `cd bigclaw-go && go test ./internal/regression ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `rg --files . | rg '\\.py$' | wc -l`
- `git status --short`
- `git add ... && git commit ...`
- `git push -u origin feat/BIG-GO-1066-root-scripts-residual-sweep`

## Risks
- Historical reports and archived local issue comments will still mention the deleted Python wrappers because they are evidence artifacts, not live operator docs.
- Any external automation still hard-coded to the deleted `.py` paths will break and must switch to `scripts/ops/bigclawctl`.
