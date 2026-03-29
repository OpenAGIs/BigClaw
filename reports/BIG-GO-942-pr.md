# BIG-GO-942 PR Draft

## Suggested Title

`BIG-GO-942: migrate lane2 root script wrappers to Go CLI`

## Suggested Body

### Summary

- replace the remaining lane-scoped root `scripts/**` Python implementations with shell wrappers
  over `scripts/ops/bigclawctl`
- preserve legacy wrapper behavior for workspace bootstrap defaults and workspace validate flag
  translation
- add issue-scoped validation, closeout, and status artifacts for `BIG-GO-942`

### Delivered File List

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

### Validation

```bash
cd bigclaw-go && go test ./cmd/bigclawctl
python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py
bash scripts/dev_smoke.py
bash scripts/create_issues.py --help
bash scripts/ops/bigclaw_refill_queue.py --help
bash scripts/ops/bigclaw_github_sync.py status --json
BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py bootstrap --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py cleanup --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json
bash scripts/ops/symphony_workspace_validate.py --repo-url <tmp bare repo> --workspace-root <tmp>/validate --issues COMPAT-VAL-1 COMPAT-VAL-2 --report-file <tmp>/report.json --no-cleanup --json
```

### Reviewer Notes

- the retained legacy file paths still end in `.py`, but they are now shell wrappers rather than
  Python entrypoints
- callers still using `python3 <wrapper>.py` must switch to `bash <wrapper>.py` or
  `bash scripts/ops/bigclawctl ...`
- `scripts/ops/bigclawctl` still uses `go run`, so local Go toolchain availability remains an
  operator dependency

## Reviewer Links

- Compare URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-942?expand=1`
- PR seed URL:
  - `https://github.com/OpenAGIs/BigClaw/pull/new/symphony/BIG-GO-942`
