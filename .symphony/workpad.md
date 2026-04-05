# BIG-GO-1472 Workpad

## Plan

1. Baseline the repository's current Python footprint and residual Python test/bootstrap dependency files.
2. Remove or rewrite any canonical repo guidance that still instructs operators to rely on Python bootstrap assets.
3. Add focused Go regression coverage that proves the repository remains in a Go-only state with no tracked `.py`, `conftest.py`, `pytest.ini`, `pyproject.toml`, `tox.ini`, or Python requirements files.
4. Run targeted validation commands and record the exact commands and outcomes.
5. Commit the scoped changes and push `BIG-GO-1472` to `origin`.

## Acceptance

- Repository reality remains at zero physical `.py` files and adds an automated guard so future regressions fail fast.
- Canonical bootstrap guidance no longer tells users to create Python bootstrap compatibility files.
- The change log explicitly states what was migrated or deleted and which Go-owned path replaces the retired Python bootstrap guidance.
- Validation captures exact commands and results proving the repository moved closer to an enforceable Go-only posture.

## Validation

- `find . -type f -name '*.py' | sort`
- `find . -maxdepth 3 \( -name 'pytest.ini' -o -name 'conftest.py' -o -name 'pyproject.toml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name '.python-version' \) | sort`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBigGo1472'`
- `cd bigclaw-go && go test ./internal/regression`
