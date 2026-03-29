# BIG-GO-961 Root Config + Root Scripts Material Pass

## In-Scope Asset List

Root config:
- `pyproject.toml` — already absent before this lane; kept absent because the repository root is no longer a Python packaging surface.
- `setup.py` — already absent before this lane; kept absent for the same reason.

Root script Python assets processed in this lane:
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `src/bigclaw/legacy_shim.py`

## Delete / Replace / Keep Rationale

- Deleted `scripts/create_issues.py`: it only proxied to `bash scripts/ops/bigclawctl create-issues`.
- Deleted `scripts/dev_smoke.py`: it only emitted a migration warning and proxied to `bash scripts/ops/bigclawctl dev-smoke`.
- Deleted `scripts/ops/bigclaw_github_sync.py`: it only proxied to `bash scripts/ops/bigclawctl github-sync`.
- Deleted `scripts/ops/bigclaw_refill_queue.py`: it only proxied to `bash scripts/ops/bigclawctl refill`.
- Deleted `scripts/ops/bigclaw_workspace_bootstrap.py`: it only added legacy defaults before dispatching to `bash scripts/ops/bigclawctl workspace bootstrap`.
- Deleted `scripts/ops/symphony_workspace_bootstrap.py`: it only proxied to `bash scripts/ops/bigclawctl workspace`.
- Deleted `scripts/ops/symphony_workspace_validate.py`: it only translated legacy flags before dispatching to `bash scripts/ops/bigclawctl workspace validate`.
- Deleted `src/bigclaw/legacy_shim.py`: it became dead code once the root/script Python wrappers were removed.
- Kept `scripts/ops/bigclawctl`: this is the canonical Go-first shell entrypoint that replaces all deleted Python shims.
- Kept `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and `scripts/ops/bigclaw-symphony`: these are shell aliases, not Python assets, and still provide short operator commands over `bigclawctl`.

## Python File Count Impact

- Repository `.py` files before this lane: `123`
- Repository `.py` files after this lane: `115`
- Net reduction from this lane: `8`

The lane deleted 7 root/script Python wrappers plus 1 dead support module.
