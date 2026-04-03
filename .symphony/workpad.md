## BIG-GO-1169

### Plan
- Confirm the lane's candidate Python files are no longer present and identify the current Go or shell replacement surfaces.
- Add an issue-scoped regression guard so this Python sweep cannot silently regress.
- Update the migration plan document to record that the BIG-GO-1169 sweep targets are already retired and covered by repo checks.
- Run targeted validation covering the new regression and the documented repo-level Python count.

### Acceptance
- Candidate Python assets for this lane are confirmed absent.
- A targeted regression check protects the absence of the BIG-GO-1169 Python sweep surface.
- Go or shell replacement/compatibility paths are documented for the retired surfaces.
- `find . -name '*.py' | wc -l` remains at `0`.

### Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1169|RootScriptResidualSweep|BIGGO1160)'`
- `git diff --stat`

### Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'Test(BIGGO1169|RootScriptResidualSweep|BIGGO1160)'` -> `ok  	bigclaw-go/internal/regression	0.824s`
