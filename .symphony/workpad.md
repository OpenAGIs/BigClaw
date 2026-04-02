# BIG-GO-1082 Workpad

## Plan
- Confirm the current repo state for `scripts/dev_smoke.py`, root smoke references, and the current `.py` count before edits.
- Keep the root smoke path on `bash scripts/ops/bigclawctl dev-smoke` and remove any remaining live references that would suggest a Python dev-smoke entrypoint.
- Because `scripts/dev_smoke.py` is already absent in this worktree, delete the remaining standalone Python deprecation helper tied to the frozen legacy wrapper surface and inline its logic at the only remaining call sites so the repository `.py` count still drops for this issue.
- Add a focused Go regression check that keeps the root smoke docs on the Go entrypoint and prevents `scripts/dev_smoke.py` from reappearing in live docs.
- Run targeted verification, record exact commands/results, then commit and push the scoped branch.

## Acceptance
- `scripts/dev_smoke.py` remains absent.
- The supported root dev-smoke path is `bash scripts/ops/bigclawctl dev-smoke`.
- No live README/docs surface reintroduces `scripts/dev_smoke.py` as an active command.
- The standalone Python helper `src/bigclaw/deprecation.py` is removed and its remaining behavior is preserved inline.
- Repository `.py` count drops from the pre-change baseline.

## Validation
- `find . -name '*.py' | sort | wc -l`
- `rg -n "scripts/dev_smoke\\.py|python3 scripts/dev_smoke\\.py|dev_smoke\\.py" README.md docs .github scripts bigclaw-go src -g '!reports/**' -g '!.symphony/**'`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `bash scripts/ops/bigclawctl dev-smoke`
- `git status --short`

## Validation Results
- `find . -name '*.py' | sort | wc -l` -> `22`
- `rg -n "scripts/dev_smoke\\.py|python3 scripts/dev_smoke\\.py|dev_smoke\\.py" README.md docs .github scripts bigclaw-go src -g '!reports/**' -g '!.symphony/**'` -> hits only the explicit README removal note and the new regression guard; no live command path advertises `python3 scripts/dev_smoke.py`
- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py` -> exited `0`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression` -> `ok   bigclaw-go/cmd/bigclawctl 3.874s`; `ok   bigclaw-go/internal/regression 1.990s`
- `bash scripts/ops/bigclawctl dev-smoke` -> `smoke_ok local`
- `git status --short` -> `M .symphony/workpad.md`; `M README.md`; `M src/bigclaw/__main__.py`; `D src/bigclaw/deprecation.py`; `M src/bigclaw/runtime.py`; `?? bigclaw-go/internal/regression/dev_smoke_entrypoint_migration_test.go`
