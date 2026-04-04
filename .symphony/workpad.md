# BIG-GO-1195 Workpad

## Plan
- Re-inventory physical Python files across the repository, with explicit focus on `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Replace the stale workpad with lane-specific acceptance criteria, Go replacement path, and exact validation commands for BIG-GO-1195.
- Add a narrow Go regression test that proves the retained Go replacement path still observes an empty Python asset baseline.
- Record the inventory, validation evidence, and residual blocker in lane-specific report artifacts.
- Run targeted validation, then commit and push the lane changes.

## Acceptance
- The remaining Python asset list for BIG-GO-1195 is explicit and auditable.
- The repository still contains no physical `.py` files, including in `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- A Go regression guard for BIG-GO-1195 is committed and passes.
- The Go replacement path and exact validation commands/results are documented in committed artifacts.
- The lane stays scoped to heartbeat Python-asset sweep evidence and hardening.

## Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 -type f -name '*.py' | wc -l`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 -type f -name '*.py' | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195/bigclaw-go && go test ./internal/regression -run TestBIGGO1195LegacyPythonCompileCheckMatchesZeroPythonBaseline$`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 && bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 status --short`

## Validation Results
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 -type f -name '*.py' | wc -l` -> `0`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 -type f -name '*.py' | sort` -> `[no output]`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195/bigclaw-go && go test ./internal/regression -run TestBIGGO1195LegacyPythonCompileCheckMatchesZeroPythonBaseline$` -> `ok  	bigclaw-go/internal/regression	0.426s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1195 && bash scripts/ops/bigclawctl legacy-python compile-check --json` -> `{"files":[],"python":"python3","repo":"/Users/openagi/code/bigclaw-workspaces/BIG-GO-1195","status":"ok"}`

## Residual Risk
- If the repository baseline is already at zero physical Python files, this lane can only harden and document the Go-only state rather than reduce the `.py` count numerically.
