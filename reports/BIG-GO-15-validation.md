# BIG-GO-15 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-15`

Title: `Sweep packaging/tooling residuals batch B`

This lane removed the remaining root Python-centric packaging and tooling
residuals that were still present on `origin/main`, then refreshed the root CI
and README guidance so they only reference the supported Go-first entrypoints.

## Delivered

- Removed `.pre-commit-config.yaml` from the repository root.
- Removed root Python build-artifact ignore entries from `.gitignore`:
  `__pycache__/`, `*.py[cod]`, and `*.egg-info/`.
- Updated `.github/workflows/ci.yml` to validate the root Go entrypoints via
  `make test`, `make build`, and `bash scripts/ops/bigclawctl ...`.
- Updated `README.md` so repository hygiene uses `git diff --check` instead of
  `pre-commit`.
- Replaced `.symphony/workpad.md` with the BIG-GO-15 plan, acceptance
  criteria, and validation commands.

## Removed Root Tooling Residuals

- `.pre-commit-config.yaml`
- `.gitignore` entries: `__pycache__/`, `*.py[cod]`, `*.egg-info/`

## Go Replacement Paths

- Root test/build entrypoints: `Makefile`
- Root operator entrypoint: `scripts/ops/bigclawctl`
- Root bootstrap helper: `scripts/dev_bootstrap.sh`
- Active Go implementation mainline: `bigclaw-go/`

## Validation

### Diff hygiene

Command:

```bash
git diff --check
```

Result:

```text
passed
```

### Root build path

Command:

```bash
make build
```

Result:

```text
passed
```

### Root operator smoke path

Command:

```bash
bash scripts/ops/bigclawctl github-sync --help >/dev/null && bash scripts/ops/bigclawctl dev-smoke
```

Result:

```text
smoke_ok local
```

### Residual reference sweep

Command:

```bash
rg -n "__pycache__|\\*\\.py\\[cod\\]|\\*\\.egg-info/|pre-commit|\\.pre-commit-config\\.yaml|actions/setup-python|pip install -e|python -m build|pytest-cov|python-version" .gitignore README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh Makefile --glob '!**/.git/**'
```

Result:

```text
no output
```

### Full root test path

Command:

```bash
make test
```

Result:

```text
failed in pre-existing bigclaw-go/internal/regression checks:
- TestBundleFollowUpIndexDocsStayAligned
- TestLiveShadowRuntimeDocsStayAligned
- TestRollbackDocsStayAligned
```

## Git

- Branch: `BIG-GO-15`
- Commits:
  - `bbf8133b47d7b4ff618c497b38ee5d50bc22cc75` `Sweep packaging tooling residuals batch B`
  - `dd8f238cafbd2941fb3d05cd1d2f50e15549dd36` `Drop root python ignore residuals`
- Push target: `origin/BIG-GO-15`

## Blocker

- `make test` is red on the current `origin/main` baseline because
  `bigclaw-go/internal/regression` contains unrelated live-shadow/doc-alignment
  failures outside this issue's scope.
