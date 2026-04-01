# BIG-GO-1070 Validation

Date: 2026-04-02

## Scope

Issue: `BIG-GO-1070`

Title: `other python assets + packaging cleanup`

This lane closes the remaining packaging-adjacent Python wrapper cleanup that was
still physically present in the repository after `pyproject.toml` and `setup.py`
had already been removed in earlier work. The scope for this batch is:

- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `src/bigclaw/legacy_shim.py`

The repo-side goal is to reduce tracked Python files directly, remove default
operator execution through Python compatibility wrappers, and keep the Go-first
replacement path verifiable through `scripts/ops/bigclawctl`.

## Delivered

- deleted the four operator-facing Python wrappers under `scripts/ops/`
- deleted `src/bigclaw/legacy_shim.py`
- deleted the dead Go helper/test pair that only mirrored the removed Python
  compatibility behavior:
  - `bigclaw-go/internal/legacyshim/wrappers.go`
  - `bigclaw-go/internal/legacyshim/wrappers_test.go`
- narrowed `bigclawctl legacy-python compile-check` to the Python file that still
  exists in this surface:
  - `src/bigclaw/__main__.py`
- expanded the regression purge list in
  `bigclaw-go/internal/regression/top_level_module_purge_tranche1_test.go` so the
  deleted Python wrapper paths stay absent
- updated live operator documentation to use:
  - `bash scripts/ops/bigclawctl refill ...`
  - `bash scripts/ops/bigclawctl workspace ...`

## Validation

### Python file counts

Command:

```bash
rg --files . | rg '\.py$' | wc -l
```

Baseline result before deletion:

```text
43
```

Final result after deletion:

```text
38
```

Net effect for this lane:

```text
-5
```

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl
```

Result:

```text
ok  	bigclaw-go/internal/legacyshim	(cached)
ok  	bigclaw-go/internal/regression	3.220s
ok  	bigclaw-go/cmd/bigclawctl	(cached)
```

### Go-first operator command checks

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/scripts/ops/bigclawctl refill --help
```

Result: exit code `0`, printed `usage: bigclawctl refill [flags]`

Command:

```bash
BIGCLAW_BOOTSTRAP_REPO_URL=git@github.com:OpenAGIs/BigClaw.git BIGCLAW_BOOTSTRAP_CACHE_KEY=openagis-bigclaw bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/scripts/ops/bigclawctl workspace bootstrap --help
```

Result: exit code `0`, printed `usage: bigclawctl workspace bootstrap [flags]`

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/scripts/ops/bigclawctl workspace validate --help
```

Result: exit code `0`, printed `usage: bigclawctl workspace validate [flags]`

### Frozen Python compile check

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/scripts/ops/bigclawctl legacy-python compile-check --json
```

Result:

```json
{
  "files": [
    "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1070/src/bigclaw/__main__.py"
  ],
  "python": "python3",
  "repo": "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1070",
  "status": "ok"
}
```

## Commit And Push

- Code migration commit:
  - `06789289080b536c02beb749882bbefa25fedec8`
- Evidence commit:
  - `08ef2562d95dea2a50eabcd6fe6478d4fbc90f74`
- Push target:
  - `origin/symphony/BIG-GO-1070`

## Residual Risk

- Historical reports and tracker comments still mention the deleted Python wrapper
  paths. Those references are archival only and are not live operator entry
  surfaces.
- The repository still contains other Python modules outside this issue’s narrow
  packaging/operator-wrapper scope; follow-on lanes handle those broader removals.
