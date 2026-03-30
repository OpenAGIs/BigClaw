Issue: BIG-GO-1020

Plan
- Inspect repository-level Python residue and pick a narrow slice that lowers the `.py` file count without changing core product behavior.
- Port `bigclaw-go/scripts/e2e/mixed_workload_matrix.py` to a Go-native command plus shell wrapper, then remove the Python entrypoint.
- Port `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` to a Go-native command plus shell wrapper, then remove the Python entrypoint.
- Replace the five `scripts/ops/*.py` operator compatibility shims with shell wrappers that dispatch into `scripts/ops/bigclawctl`, preserving the existing wrapper behavior for `github-sync`, `refill`, and workspace commands.
- Replace additional thin Python trampolines when they only forward into Go automation entrypoints and can be retired without touching non-wrapper benchmark/report logic.
- Replace small Python-only verification files with equivalent Go regression tests when the underlying Python generator remains active but the `.py` test file itself is not required.
- Prefer end-to-end or function-level Go shims that execute the remaining Python generators with fixed inputs, so the `.py` verification count falls without changing the checked-in Python generator behavior.
- When the Python test validates multiple derived summaries from one generator, collapse that coverage into one Go regression test that shells into Python once and asserts the returned JSON payloads.
- Keep these Go replacements focused on stable contract points from checked-in evidence or deterministic synthetic inputs, so they reduce `.py` count without introducing fragile cross-language harnessing.
- Use the same deterministic-tempdir pattern for shell harness tests when their behavior can be covered from Go with stub executables and temporary files.
- Where a remaining Python generator uses newer type syntax, prefer a no-behavior-change compatibility import over abandoning the file-count reduction.
- For small active benchmark/e2e orchestration scripts, prefer a direct Go port plus a shell wrapper when the logic is already just subprocess orchestration and JSON assembly.
- Update the minimal operator-facing docs that still advertise those Python wrapper paths so the repository no longer points users at deleted `.py` entrypoints.
- Run targeted validation on the new shell wrappers and repo counts, then commit and push the scoped change.

Acceptance
- Repository Python file count decreases through direct removal of repo-level `.py` assets.
- The removed Python assets are replaced by working repo-native wrappers or equivalent documented entrypoints.
- Changes stay scoped to the wrapper migration slice and directly related docs/workpad updates.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find . -type f -name '*.py' | wc -l`
- `find . -type f -name '*.go' | wc -l`
- `find . -maxdepth 2 \\( -name 'pyproject.toml' -o -name 'setup.py' \\) | sort`
- `bash scripts/ops/bigclaw-github-sync status --json`
- `bash scripts/ops/bigclaw-refill-queue --help`
- `bash scripts/ops/symphony-workspace-bootstrap --help`
- `bash scripts/ops/symphony-workspace-validate --help`
- `BIGCLAW_BOOTSTRAP_REPO_URL=git@github.com:OpenAGIs/BigClaw.git BIGCLAW_BOOTSTRAP_CACHE_KEY=openagis-bigclaw bash scripts/ops/bigclaw-workspace-bootstrap --help`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/regression -run ValidationBundleContinuationPolicyGate`
- `cd bigclaw-go && go test ./internal/regression -run RunAllScript`
- `bash bigclaw-go/scripts/e2e/validation-bundle-continuation-policy-gate --help`
- `cd bigclaw-go && go test ./internal/regression -run MixedWorkloadMatrix`
- `bash bigclaw-go/scripts/e2e/mixed-workload-matrix --help`
- `git diff --stat`
- `git status --short`
