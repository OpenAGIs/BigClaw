# BIG-GO-165 Python Asset Sweep

Date: 2026-04-09

## Scope

Issue: `BIG-GO-165`

Title: `Residual tooling Python sweep L`

This lane locks down the repo-root tooling posture after the Python helper and
build-helper removal work. The checked-out workspace already carries no tracked
Python tooling under `scripts`, no Python ops wrappers under `scripts/ops`, and
no root Python build metadata files.

## Remaining Tooling Python Inventory

- Tracked `scripts/*.py`: `none`
- Tracked `scripts/ops/*.py`: `none`
- Tracked `setup.py`: `none`
- Tracked `pyproject.toml`: `none`
- Physical repository matches for `*.py`, `setup.py`, or `pyproject.toml`: `none`

## Supported Replacement Paths

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/internal/githubsync/sync.go`
- `bigclaw-go/internal/refill/queue.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`

## Validation Commands

- `git ls-files 'scripts/*.py' 'scripts/ops/*.py' 'setup.py' 'pyproject.toml' | sort`
- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sed 's#^./##' | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO165(ToolingPythonPathsRemainAbsent|GoToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`

## Validation Results

### Tracked tooling inventory

Command:

```bash
git ls-files 'scripts/*.py' 'scripts/ops/*.py' 'setup.py' 'pyproject.toml' | sort
```

Result:

```text
none
```

### Physical repository inventory

Command:

```bash
find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \) -print | sed 's#^./##' | sort
```

Result:

```text
none
```

### Targeted regression guard

Command:

```bash
cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO165(ToolingPythonPathsRemainAbsent|GoToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.190s
```

## Notes

- This lane is intentionally evidence-heavy rather than deletion-heavy because
  the checked-out branch already reflects the retired Python tooling posture.
- The regression guard is scoped to repo-root tooling and build-helper paths so
  the issue does not broaden into unrelated Python-removal surfaces.
