# BIG-GO-1480 Python Reality Audit

## Baseline

- Audit date: 2026-04-06
- Baseline command:

```bash
find . -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' \) | sed 's#^./##' | sort | wc -l
```

- Baseline count: `138`

## Deleted in this sweep

- `src/bigclaw/*.py` (`50` files)
  - Delete condition: root Python control-plane/runtime package was legacy-only and duplicated Go-mainline ownership already carried under `bigclaw-go/internal/*`.
  - Replacement Go ownership: domain, queue, scheduler, workflow, repo, reporting, governance, orchestration, and operator surfaces under `bigclaw-go/internal/*`.
- `tests/*.py` (`57` files)
  - Delete condition: these tests only covered the removed root Python package and root Python migration shims.
  - Replacement Go ownership: package-local Go tests under `bigclaw-go/internal/*` and `bigclaw-go/cmd/*`.
- `scripts/create_issues.py`
  - Delete condition: bootstrap issue creation was legacy Python-only and outside the active Go CLI path.
  - Replacement Go ownership: repo operations now route through `scripts/ops/bigclawctl`.
- `scripts/dev_smoke.py`
  - Delete condition: this was only a deprecated Python smoke path for the retired root package.
  - Replacement Go ownership: `cd bigclaw-go && go test ./...` plus `go run ./cmd/bigclawd`.
- `scripts/ops/*.py` compatibility wrappers (`5` files)
  - Delete condition: these were shell wrappers stored under `.py` names only for migration compatibility.
  - Replacement Go ownership: `scripts/ops/bigclawctl`, `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and `scripts/ops/bigclaw-symphony`.
- `setup.py`
  - Delete condition: setuptools compatibility shim for the removed root Python package.

## Post-sweep status

- Expected remaining Python count: `23`
- Remaining Python ownership: report generators and validation helpers under `bigclaw-go/scripts/*`
- Net reduction: `115` files removed

## Validation commands

```bash
find . -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' \) | sed 's#^./##' | sort | wc -l
git ls-files | rg '\.(py|pyi|pyw)$' | wc -l
cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap ./internal/githubsync ./internal/refill
cd bigclaw-go && go test ./...
```
