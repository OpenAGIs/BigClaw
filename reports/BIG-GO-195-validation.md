# BIG-GO-195 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-195`

Title: `Residual tooling Python sweep O`

This lane audits the residual tooling and build-helper surface that remains
checked in after the Python-to-Go migration: repo-root shell helpers,
`bigclaw-go/scripts`, and the retired root build-helper manifests.

The checked-out workspace was already at a zero-Python baseline for physical
`.py` files and for the retired root build-helper manifests `setup.py` and
`pyproject.toml`. The delivered work therefore hardens that state with a
lane-specific regression guard and execution evidence rather than deleting
in-branch Python tooling.

## Residual Tooling Inventory

- Repository-wide physical `.py`, `setup.py`, and `pyproject.toml` files: `none`
- `scripts/*.py`: `none`
- `bigclaw-go/scripts/*.py`: `none`
- `setup.py`: `none`
- `pyproject.toml`: `none`

## Active Replacement Paths

- Root posture guidance: `README.md`
- Tool migration plan: `docs/go-cli-script-migration-plan.md`
- Root bootstrap helper: `scripts/dev_bootstrap.sh`
- Root operator entrypoint: `scripts/ops/bigclawctl`
- Root issue shim: `scripts/ops/bigclaw-issue`
- Root panel shim: `scripts/ops/bigclaw-panel`
- Root symphony shim: `scripts/ops/bigclaw-symphony`
- Go CLI implementation: `bigclaw-go/cmd/bigclawctl/main.go`
- Root residual regression guard: `bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
- Script migration regression guard: `bigclaw-go/internal/regression/big_go_1160_script_migration_test.go`
- Benchmark wrapper: `bigclaw-go/scripts/benchmark/run_suite.sh`
- E2E wrapper: `bigclaw-go/scripts/e2e/run_all.sh`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195 -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO195(ToolingRepositoryHasNoPythonFiles|ResidualToolingDirectoriesStayPythonFree|RetiredBuildHelpersRemainAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `python3 -m json.tool /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/local-issues.json >/dev/null`

## Validation Results

### Repository tooling inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195 -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sort
```

Result:

```text
none
```

### Residual tooling directory inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort
```

Result:

```text
none
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO195(ToolingRepositoryHasNoPythonFiles|ResidualToolingDirectoriesStayPythonFree|RetiredBuildHelpersRemainAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.493s
```

### Tracker JSON shape

Command:

```bash
python3 -m json.tool /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/local-issues.json >/dev/null
```

Result:

```text
exit 0
```

## Validation Rerun

Date: 2026-04-11

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195 -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sort` -> `none`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort` -> `none`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO195(ToolingRepositoryHasNoPythonFiles|ResidualToolingDirectoriesStayPythonFree|RetiredBuildHelpersRemainAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'` -> `ok   bigclaw-go/internal/regression 0.198s`
- `python3 -m json.tool /Users/openagi/code/bigclaw-workspaces/BIG-GO-195/local-issues.json >/dev/null` -> `exit 0`

## Git

- Branch: `BIG-GO-195`
- Baseline commit before lane changes: `0e46f22b26d520f2669723ff41ed13e9b8ef251d`
- Push target: `origin/BIG-GO-195`

## Residual Risk

- The checkout was already Python-free for the scoped tooling surface, so
  `BIG-GO-195` hardens the baseline and records exact validation evidence but
  does not reduce the physical file count below zero.
