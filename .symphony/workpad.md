# BIG-GO-1093

## Plan
- inspect the remaining `src/bigclaw/*.py` package entrypoints and confirm the smallest Go-only replacement slice that reduces the residual Python count without breaking the retained shim scripts
- add a Go-native `bigclawctl repo-sync-audit` command backed by the existing Go observability renderer so the frozen `python -m bigclaw repo-sync-audit` path has a direct replacement
- update docs and regression/compile-check coverage to point at the Go command instead of the removed Python package entrypoint
- delete the obsolete Python package entrypoint and warning helper once the Go replacement and references are in place
- run targeted validation, record exact commands and results here, then commit and push the issue branch

## Acceptance
- `bigclawctl` exposes a Go-native repo sync audit rendering command that accepts the legacy JSON payload and writes the markdown report
- `src/bigclaw/__main__.py` and `src/bigclaw/deprecation.py` are removed from the repository
- frozen-compatibility checks and docs no longer require or mention `python -m bigclaw` as an active path
- the repository `src/bigclaw/*.py` count decreases from the pre-change baseline of `19`

## Validation
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression ./internal/observability`
- `tmpdir=$(mktemp -d) && cat >"$tmpdir/audit.json" <<'EOF' ... EOF && bash scripts/ops/bigclawctl repo-sync-audit --input "$tmpdir/audit.json" --output "$tmpdir/report.md" && test -s "$tmpdir/report.md"`
- `rg -n "python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/deprecation\\.py" README.md docs bigclaw-go scripts -g '!bigclaw-go/docs/reports/**' -g '!**/*_test.go'`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | wc -l`

## Validation Results
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression ./internal/observability` -> `ok   bigclaw-go/cmd/bigclawctl 4.385s`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/internal/regression 1.243s`; `ok   bigclaw-go/internal/observability (cached)`
- `tmpdir=$(mktemp -d) && cat >"$tmpdir/audit.json" <<'EOF' ... EOF && bash scripts/ops/bigclawctl repo-sync-audit --input "$tmpdir/audit.json" --output "$tmpdir/report.md" && test -s "$tmpdir/report.md"` -> exit `0`; wrote a markdown report beginning with `# Repo Sync Audit`, `- Status: dirty`, and `- PR Number: 188`
- `rg -n "python -m bigclaw|src/bigclaw/__main__\\.py|src/bigclaw/deprecation\\.py" README.md docs bigclaw-go scripts -g '!bigclaw-go/docs/reports/**' -g '!**/*_test.go'` -> exit `1` with no matches
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; reported only `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1093/src/bigclaw/legacy_shim.py` in the frozen shim list with `status: ok`
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | wc -l` -> `17`, down from the pre-change baseline `19`
