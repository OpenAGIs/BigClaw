## Codex Workpad

### Issue

- `BIG-GO-910` - 并行收口与main合并计划

### Plan

- [x] Audit the existing Go-mainline cutover handoff, parallel follow-up indexes, and local tracker notes to extract the first nine prerequisite slices, their merge state, and current validation evidence.
- [x] Add a repo-native merge-plan handoff document that defines the executable migration plan, first implementation tranche, validation/regression surface, branch and PR strategy, and explicit risks for merging the prior parallel slices to `main`.
- [x] Add a targeted regression test that locks the merge-plan document to the required merge path, validation commands, and risk language.
- [x] Run the targeted validation commands for the new regression coverage and the touched docs/tests.
- [x] Commit the scoped changes and push the branch to `origin`.

### Acceptance Criteria

- [x] The repo contains an executable handoff document for `BIG-GO-910` that names the prior nine prerequisite slices, the first implementation batch, the main-merge path, validation commands, regression surface, and merge risks.
- [x] The merge-plan content is covered by a targeted Go regression test under `bigclaw-go/internal/regression`.
- [x] Exact validation commands and results are recorded from this workspace after the changes land.
- [x] The final branch state is committed and pushed to `origin`.

### Validation

- [x] `cd bigclaw-go && go test ./internal/regression -run TestGoMainlineMergePlanDoc -count=1`
- [x] `cd bigclaw-go && go test ./internal/regression -count=1`
- [x] `bash scripts/ops/bigclawctl github-sync status --json`

### Notes

- Scope is limited to the merge-planning handoff for the completed Go-mainline cutover slices and the minimum regression needed to keep that handoff from drifting.
- The repo-visible first nine prerequisite slices are treated as `BIG-GOM-301` through `BIG-GOM-309`, with `BIG-GOM-310` acting as the closeout handoff slice.
- Validation results:
  - `cd bigclaw-go && go test ./internal/regression -run TestGoMainlineMergePlanDoc -count=1` -> `ok  	bigclaw-go/internal/regression	0.484s`
  - `cd bigclaw-go && go test ./internal/regression -count=1` -> `ok  	bigclaw-go/internal/regression	0.285s`
  - `bash scripts/ops/bigclawctl github-sync status --json` -> `status: ok`, `branch: symphony/BIG-GO-910`, `synced: true`, `dirty: false`, `local_sha=remote_sha=af0a38726e7a0853dce1ae61937e92a393c1ec61`
