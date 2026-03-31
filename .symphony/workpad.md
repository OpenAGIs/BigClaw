Issue: BIG-GO-1030

Plan
- Merge `tests/test_parallel_validation_bundle.py` and `tests/test_validation_bundle_continuation_policy_gate.py` into the existing bundle/script regression file `tests/test_live_shadow_bundle.py`.
- Remove the two standalone bundle-policy test files once their assertions live in the surviving test module.
- Re-run targeted script-bundle pytest coverage, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `tests/test_parallel_validation_bundle.py` and `tests/test_validation_bundle_continuation_policy_gate.py` are removed from the tree.
- Validation-bundle export and continuation-policy gate coverage still passes from the merged surviving file.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_live_shadow_bundle.py -q`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
