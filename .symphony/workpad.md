## BIG-GO-1090

### Plan
1. Replace the four remaining Python operator wrappers in `scripts/ops/` with shell entrypoints that forward into `scripts/ops/bigclawctl` while preserving the legacy argument/default behavior those wrappers currently supply.
2. Remove dead Python shim helper code and update migration docs/README references so the default path is Go/shell only.
3. Add focused Go regression coverage that keeps these entrypoints Python-free and prevents docs from drifting back to removed Python commands.

### Acceptance
- Repository `.py` count drops below the current baseline.
- `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` are removed or no longer the default execution path.
- Operator docs reference the shell/Go entrypoints instead of Python commands for this tranche.

### Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestOpsScriptDirectoryStaysPythonFree|TestOpsMigrationDocsListOnlyActiveEntrypoints|TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'`
- `bash scripts/ops/bigclaw_refill_queue --help`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_validate --help`

### Validation Results
- Baseline `find . -name '*.py' | wc -l` before edits: `23`
- Final `find . -name '*.py' | wc -l`: `19`
- `cd bigclaw-go && go test ./internal/regression -run 'TestOpsScriptDirectoryStaysPythonFree|TestOpsMigrationDocsListOnlyActiveEntrypoints|TestE2EScriptDirectoryStaysPythonFree|TestE2EMigrationDocListsOnlyActiveEntrypoints'` -> `ok   bigclaw-go/internal/regression	0.511s`
- `bash scripts/ops/bigclaw_refill_queue --help` -> exited `0`, printed `usage: bigclawctl refill [flags]`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help` -> exited `0`, printed `usage: bigclawctl workspace bootstrap [flags]`
- `bash scripts/ops/symphony_workspace_bootstrap --help` -> exited `0`, printed `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/symphony_workspace_validate --help` -> exited `0`, printed `usage: bigclawctl workspace validate [flags]`
- `bash -n scripts/ops/bigclaw_refill_queue scripts/ops/bigclaw_workspace_bootstrap scripts/ops/symphony_workspace_bootstrap scripts/ops/symphony_workspace_validate` -> exited `0`
