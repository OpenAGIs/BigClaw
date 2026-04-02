# BIG-GO-1055 Validation

Date: 2026-04-02

## Scope

Issue: `BIG-GO-1055`

Title: `Go-replacement Y: remove root packaging entrypoints`

This lane finalizes the root packaging and operator entrypoint cutover by keeping
the repository root free of Python packaging files, deleting the remaining root
Python operator shims, and enforcing Go-only docs, CI, hooks, and bootstrap paths.

## Delivered

- removed the root Python operator shims:
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- kept the root packaging files absent:
  - `pyproject.toml`
  - `setup.py`
- updated root cutover surfaces to use Go-only entrypoints:
  - `README.md`
  - `.github/workflows/ci.yml`
  - `scripts/dev_bootstrap.sh`
  - `.githooks/post-commit`
  - `.githooks/post-rewrite`
- added `bigclaw-go/internal/regression/root_entrypoint_cutover_test.go` so the
  removed packaging and Python shim surfaces cannot silently return

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055 -name '*.py' -type f | wc -l
```

Result:

```text
41
```

Command:

```bash
git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055 ls-tree -r --name-only origin/main | rg '\.py$' | wc -l
```

Result:

```text
46
```

Command:

```bash
comm -23 <(git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055 ls-tree -r --name-only origin/main | sort) <(git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055 ls-tree -r --name-only HEAD | sort) | rg '\.py$' | wc -l
```

Result:

```text
5
```

Interpretation: this lane reduced the repository Python file count by five files
relative to `origin/main` by deleting root Python operator shim entrypoints.

### Packaging and shim removal checks

Command:

```bash
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/pyproject.toml && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/setup.py && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/ops/bigclaw_github_sync.py && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/ops/bigclaw_refill_queue.py && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/ops/bigclaw_workspace_bootstrap.py && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/ops/symphony_workspace_bootstrap.py && \
test ! -e /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/ops/symphony_workspace_validate.py && \
echo removed
```

Result:

```text
removed
```

### Stale root reference scan

Command:

```bash
rg -n "python3 scripts/ops/bigclaw_github_sync\.py|python3 scripts/ops/bigclaw_refill_queue\.py|scripts/ops/\*workspace\*\.py|actions/setup-python|pip install pytest|pytest --cov|BIGCLAW_ENABLE_LEGACY_PYTHON|PYTHONDONTWRITEBYTECODE" \
  /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/README.md \
  /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/.github/workflows \
  /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/scripts/dev_bootstrap.sh \
  /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/.githooks
```

Result:

- exit code `1`
- no matches

### Targeted Go regression tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1055/bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	3.155s
ok  	bigclaw-go/internal/legacyshim	(cached)
ok  	bigclaw-go/internal/regression	(cached)
```

## Commit And Push

- Code migration commit: `8c788463` `BIG-GO-1055: remove root Python packaging entrypoints`
- Evidence refresh commits:
  - `f3a9407a` `BIG-GO-1055: record validation evidence`
  - `0f2e2eab` `BIG-GO-1055: refresh workpad validation`
- Push: `git push origin symphony/BIG-GO-1055` succeeded

## Residual Risk

- The repository still contains migration-only Python source and test assets outside this
  lane's scope.
- For this issue's scope, the root packaging and operator entrypoint cutover is complete and
  enforced by targeted regression coverage.
