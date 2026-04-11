Issue: BIG-GO-259

Plan:
- Sweep the repository for residual Python artifacts, including hidden files, nested paths, shebang-based scripts, Python-specific config, and documentation references.
- Remove any overlooked Python files or references that conflict with the issue scope while keeping changes limited to this cleanup task.
- Run targeted validation to confirm no residual Python files remain and that the affected build/test paths still pass.
- Commit the scoped changes and push the branch to the remote.

Acceptance:
- Hidden, nested, or overlooked Python files relevant to this issue are removed from the repository.
- Residual Python-specific references introduced by those files are removed or updated if needed.
- Validation commands and results are recorded exactly.
- Changes remain scoped to BIG-GO-259.

Validation:
- `find . -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name '*.ipynb' \\)`
- `rg -n "python|python3|\\.py\\b|#!/usr/bin/env python|#!/usr/bin/python" .`
- Targeted repo checks/tests based on files changed.

Validation Results:
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-259 -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \) -print | sort` -> no output
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/.github /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/.githooks /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/.symphony /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/scripts/ops /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go/docs/reports /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go/docs/reports/broker-failover-stub-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go/docs/reports/live-shadow-runs /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go/docs/reports/live-validation-runs -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \) 2>/dev/null | sort` -> no output
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-259 -path '*/.git' -prune -o -type f -perm -u+x -print | while IFS= read -r f; do first=$(LC_ALL=C sed -n '1p' "$f" 2>/dev/null || true); case "$first" in '#!'*python*) printf '%s\n' "$f";; esac; done | sort` -> no output
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-259/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO259(RepositoryHasNoPythonLikeFiles|AuxiliaryResidualDirectoriesStayPythonFree|RetainedNativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'` -> `ok  	bigclaw-go/internal/regression	5.596s`
