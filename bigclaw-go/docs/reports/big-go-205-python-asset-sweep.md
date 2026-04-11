# BIG-GO-205 Python Asset Sweep

## Scope

`BIG-GO-205` covers the last checked-in repo-root developer tooling surface
that still depended on Python-oriented hook infrastructure even after the
repository had already reached a physical `.py` file count of zero.

This lane is intentionally narrow: it removes the root `.pre-commit` config and
refreshes the documented hygiene path so the retained helper surface is fully
Go/shell-native.

## Residual Tooling Baseline

Repository-wide Python file count: `0`.

Audited repo-root tooling state after this sweep:

- `.pre-commit-config.yaml`: absent
- `README.md` no longer documents `pre-commit run --all-files`
- `README.md` now points repository hygiene at `git diff --check` and `bash scripts/ops/bigclawctl github-sync --help >/dev/null`

## Retained Go Or Shell Helper Surface

The root helper surface retained by this lane remains:

- `Makefile`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `bigclaw-go/cmd/bigclawctl/main.go`

## Why This Sweep Is Safe

The deleted root config only provided Python-based developer hook wiring and no
longer matched the repository’s Go-only runtime and shell helper posture. The
replacement hygiene commands are already supported in-tree and exercise the
same practical operator checks without reintroducing Python tooling.

## Validation Commands And Results

- `test ! -e .pre-commit-config.yaml`
  Result: exit status `0`; the residual Python tooling config is absent.
- `rg -n "pre-commit|ruff" README.md`
  Result: no output; the root README no longer references the removed
  Python-based tooling.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO205(ResidualPythonToolingConfigStaysAbsent|RootGoHelperSurfaceRemainsAvailable|LaneReportCapturesToolingSweep)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.169s`
