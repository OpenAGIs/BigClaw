## BIG-GO-1033

### Plan
- Confirm the exact `src/bigclaw` runtime/scheduler/queue/workflow Python files still present in this workspace and map them to existing Go replacements under `bigclaw-go/internal/*`.
- Remove the remaining legacy Python runtime file in scope, then update Python package exports, tests, and documentation so the repository no longer expects that file to exist.
- Add or adjust Go-side validation where needed only if a replacement coverage gap is exposed by the deletion, keeping changes scoped to the runtime migration surface.
- Run targeted validation for the affected package and Go replacement packages, capture exact commands and results, then commit and push the branch.

### Acceptance
- `src/bigclaw/runtime.py` is deleted, reducing the `.py` file count in the requested runtime/scheduler/queue/workflow scope.
- Repository references are updated so no checked-in code or docs still require `src/bigclaw/runtime.py` as a live Python implementation surface.
- Go implementation coverage remains present in `bigclaw-go/internal/worker`, `bigclaw-go/internal/scheduler`, `bigclaw-go/internal/queue`, and `bigclaw-go/internal/workflow`.
- No `pyproject.toml` or `setup.py` exists at repo scope after the change.

### Validation
- `find src/bigclaw \\( -path 'src/bigclaw/runtime*' -o -path 'src/bigclaw/scheduler*' -o -path 'src/bigclaw/queue*' -o -path 'src/bigclaw/workflow*' \\) -type f | sort`
- `rg -n "from bigclaw\\.runtime|import bigclaw\\.runtime|from \\.runtime|src/bigclaw/runtime\\.py|bigclaw/runtime\\.py" . --glob '!**/.git/**' --glob '!local-issues.json'`
- `python3 -m compileall src/bigclaw`
- `cd bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/queue ./internal/workflow`
- `git status --short`
