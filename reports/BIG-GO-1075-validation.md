# BIG-GO-1075 Validation

## Scope

Converted the repository git auto-sync hooks away from the shell wrapper hop and onto the canonical Go entrypoint:

- `.githooks/post-commit`
- `.githooks/post-rewrite`
- `bigclaw-go/internal/githubsync/sync.go`

## Baseline

- Previous hook content at `d36a1c700480054955f46d8a02e4c25cf80d094b` called:
  - `bash scripts/ops/bigclawctl github-sync sync --json --allow-dirty`
- Current hook content calls:
  - `cd "$repo_root/bigclaw-go" && go run ./cmd/bigclawctl github-sync sync --json --allow-dirty --repo "$repo_root"`

## Validation Commands

1. `find . -name '*.py' | wc -l`
   - Result: `43`
2. `cd bigclaw-go && go test ./internal/githubsync ./cmd/bigclawctl`
   - Result:
     - `ok  	bigclaw-go/internal/githubsync	5.045s`
     - `ok  	bigclaw-go/cmd/bigclawctl	5.432s`
3. `bash .githooks/post-commit`
   - Result: JSON payload with `status: ok`, `branch: main`, `synced: true`, `pushed: true`, `dirty: true`
4. `bash .githooks/post-rewrite`
   - Result: JSON payload with `status: ok`, `branch: main`, `synced: true`, `pushed: true`, `dirty: true`
5. `bash scripts/ops/bigclawctl github-sync status --json`
   - First successful result during implementation:
     - JSON payload with `status: ok`, `branch: main`, `synced: true`, `pushed: true`, `dirty: true`
   - Follow-up rerun result while recording closeout:
     - JSON error payload: `fatal: unable to access 'https://github.com/OpenAGIs/BigClaw.git/': LibreSSL SSL_connect: SSL_ERROR_SYSCALL in connection to github.com:443`
6. `cd bigclaw-go && go test ./internal/githubsync ./cmd/bigclawctl`
   - Rerun result while recording closeout:
     - `ok  	bigclaw-go/internal/githubsync	2.908s`
     - `ok  	bigclaw-go/cmd/bigclawctl	2.185s`

## Python Count Impact

- Baseline tree count before this slice: `43`
- Tree count after this slice: `43`
- Net `.py` delta for this issue: `0`

This slice reduced the live default Python-adjacent execution path for git auto-sync hooks but did not remove a physical `.py` asset because the Python shim had already been retired in an earlier lane.
