# Bootstrap cache validation

Use this validation flow to prove that Symphony reuses one shared local mirror + seed repo instead
of re-downloading the same GitHub repository for every parallel issue workspace.

## Validation command

```bash
bash scripts/ops/bigclawctl workspace validate \
  --repo-url "$SYMPHONY_BOOTSTRAP_REPO_URL" \
  --workspace-root ./tmp/bootstrap-validation \
  --issues OPE-272,OPE-273,OPE-274 \
  --cache-base "${SYMPHONY_BOOTSTRAP_CACHE_BASE:-$HOME/.cache/symphony/repos}" \
  --cache-key "${SYMPHONY_BOOTSTRAP_CACHE_KEY:-}" \
  --report reports/bootstrap-cache-validation.json \
  --json
```

## Expected evidence

- `summary.single_cache_root_reused=true`
- `summary.mirror_creations=1`
- `summary.seed_creations=1`
- `summary.clone_suppressed_after_first=true`
- `summary.cache_reused_after_first=true`
- Every workspace result reports the same `cache_root`, `mirror_path`, and `seed_path`

## Observability fields

Each bootstrap result now reports:

- `cache_root`
- `cache_key`
- `mirror_path`
- `seed_path`
- `cache_reused`
- `clone_suppressed`
- `mirror_created`
- `seed_created`
- `workspace_mode`

These fields distinguish a cold cache warm-up from later worktree reuse and make it obvious when a
workspace avoided a repeated full clone.
