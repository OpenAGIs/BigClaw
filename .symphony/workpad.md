## BIG-GO-246

### Plan
1. Replace lingering Python-era support-doc wording in active bootstrap/operator guidance with Go-first wording.
2. Keep the sweep scoped to active support assets, not historical migration reports.
3. Run targeted search and bootstrap-package validation to confirm the touched guidance is clean and still aligned with the Go implementation.
4. Commit the scoped documentation sweep and push `BIG-GO-246` to `origin`.

### Acceptance
- Active support docs touched by this lane no longer present the swept Python helper wording as current guidance.
- Historical migration reports remain untouched.
- Targeted validation commands and exact results are recorded.
- The final change set is committed and pushed on `BIG-GO-246`.

### Validation
- `rg -n "root workspace Python helpers|Python asset status|Python compatibility shims" README.md docs/symphony-repo-bootstrap-template.md`
- `git diff --check`
- `cd bigclaw-go && go test ./internal/bootstrap`

### Validation Results
- `rg -n "root workspace Python helpers|Python asset status|Python compatibility shims" README.md docs/symphony-repo-bootstrap-template.md`
  - exit `1`
  - result: no matches
- `git diff --check`
  - exit `0`
  - result: no whitespace or conflict-marker errors
- `cd bigclaw-go && go test ./internal/bootstrap`
  - exit `0`
  - result: `ok  	bigclaw-go/internal/bootstrap	2.551s`
