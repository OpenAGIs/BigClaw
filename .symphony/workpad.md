# BIG-GO-1056

## Plan
- Inspect the root smoke migration surface and confirm whether `scripts/dev_smoke.py` still exists or only stale references remain.
- Remove root-facing references to the deleted Python dev smoke path and point operators to the Go `bigclawctl dev-smoke` replacement.
- Keep the bootstrap helper aligned with the Go-only root smoke path so legacy Python validation no longer treats the deleted script as an entrypoint.
- Rebase onto current `origin/main`, resolve any closeout-only workpad conflicts, and keep PR `#219` mergeable.
- Run targeted validation for reference removal, `.py` count, and the active Go smoke command, then push the updated issue branch.

## Acceptance
- `scripts/dev_smoke.py` is absent from the repository and no root README / workflow / hooks / CI / bootstrap surface directs operators to it.
- Root smoke guidance uses `bash scripts/ops/bigclawctl dev-smoke` as the only supported dev smoke entrypoint.
- Validation captures the current tracked `.py` file count and confirms the Go smoke replacement still succeeds.
- PR `#219` is rebased onto current `main` without merge conflicts.

## Validation
- `find . -path '*/.git' -prune -o -name 'dev_smoke.py' -print`
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort | wc -l`
- `rg -n "scripts/dev_smoke\\.py|python3 scripts/dev_smoke\\.py|PYTHONPATH=src python3 scripts/dev_smoke\\.py" README.md docs scripts .github .githooks bigclaw-go`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/dev_bootstrap.sh`

## Validation Result
- `find . -path '*/.git' -prune -o -name 'dev_smoke.py' -print`
  - passed; no output and `scripts/dev_smoke.py` is absent
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort | wc -l`
  - passed; current tracked `.py` count is `45`
- `rg -n "scripts/dev_smoke\\.py|python3 scripts/dev_smoke\\.py|PYTHONPATH=src python3 scripts/dev_smoke\\.py" README.md docs scripts .github .githooks bigclaw-go`
  - passed; no remaining matches in the scoped root/docs/workflow/hooks surfaces
- `bash scripts/ops/bigclawctl dev-smoke`
  - passed; output `smoke_ok local`
- `bash scripts/dev_bootstrap.sh`
  - passed; `go test ./cmd/bigclawctl` succeeded and the helper reported the updated Go-first bootstrap message

## Execution Result
- Branch: `symphony/BIG-GO-1056`
- PR: `https://github.com/OpenAGIs/BigClaw/pull/219`
- Rebase note: resolved `.symphony/workpad.md` conflicts caused by later mainline closeout updates and preserved the `BIG-GO-1056` issue-local workpad state.
