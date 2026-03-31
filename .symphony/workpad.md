Issue: BIG-GO-1034

Scope
- Remove migrated Python governance/reporting/observability/memory/cost-control modules from `src/bigclaw`.
- Remove direct Python tests that only validate those deleted modules.
- Clean Python package references that would otherwise point at removed modules.
- Use existing Go implementations in `bigclaw-go/internal/governance`, `bigclaw-go/internal/reporting`, `bigclaw-go/internal/observability`, and `bigclaw-go/internal/costcontrol` as the replacement surface.
- Add or adjust Go validation only if needed to cover deleted Python behavior gaps inside this slice.

Acceptance
- Python file count in the targeted migration slice decreases.
- Go implementation remains present for governance/reporting/observability/cost-control, with Go tests covering the retained replacement surface.
- No `pyproject.toml` or `setup.py` remains at repo root.
- Commit explains which Python files were deleted and which Go files validate the replacement surface.

Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034/bigclaw-go && go test ./internal/governance ./internal/reporting ./internal/observability ./internal/costcontrol`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && git diff --stat`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && rg --files src/bigclaw tests | rg 'governance|reports|observability|memory|cost_control'`

Notes
- Keep changes scoped to the migrated module slice; do not rewrite unrelated Python subsystems in this issue.
- Python modules that still depend on observability/reporting outside this slice are not being ported here; only direct references to deleted modules will be cleaned.
