Issue: BIG-GO-1020

Plan
- Inspect repository-level Python residue and keep the scope on the thinnest removable `.py` assets.
- Replace five small root Python unit tests with one Go regression test that shells into `python3` and asserts the same stable `src/bigclaw` contracts.
- Remove `tests/test_validation_policy.py`, `tests/test_repo_governance.py`, `tests/test_repo_board.py`, `tests/test_repo_links.py`, and `tests/test_repo_triage.py` once the Go regression coverage is in place.
- Run targeted regression and repo-count validation, then commit and push the scoped change.

Acceptance
- Repository Python file count decreases through direct removal of repo-level `.py` assets.
- The removed Python tests are replaced by a working Go regression test that exercises the same Python contract points.
- Changes stay scoped to this repo-level test-migration slice and the related workpad update.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `printf 'py files: '; rg --files -g '*.py' | wc -l`
- `printf 'go files: '; rg --files -g '*.go' | wc -l`
- `printf 'pyproject.toml: '; rg --files -g 'pyproject.toml' | wc -l`
- `printf 'setup.py: '; rg --files -g 'setup.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run PythonRepoContractMigration`
- `git diff --stat`
- `git status --short`
