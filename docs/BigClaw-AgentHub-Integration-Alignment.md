# BigClaw AgentHub Integration PRD ↔ Issue Plan Alignment (Recheck)

Date: 2026-03-12

## Why this exists
The original Issue Plan and PRD are directionally consistent, but differ in **issue numbering/packaging granularity** and some **acceptance wording**. This document reconciles them into one executable baseline.

## Key inconsistencies found

1. **Identifier drift**
   - Issue plan examples referenced `OPE-140~154`.
   - Active Linear execution used `OPE-160~174` in project `BigClaw AgentHub Integration`.
   - Resolution: treat `OPE-160~174` as canonical execution IDs.

2. **Scope wording drift (same intent, different grouping)**
   - PRD FR sections are capability-oriented (`FR-1`..`FR-8`).
   - Issue plan is implementation-step oriented (`BIG-4501`..`BIG-4512`).
   - Resolution: establish explicit FR→Issue mapping (below).

3. **MVP gate wording mismatch**
   - PRD asks for sidecar-first MVP + evidence-first closure.
   - Issue plan split this across multiple issues; some acceptance text omitted explicit FR references.
   - Resolution: add canonical acceptance checklist aligned to FR and evidence policy.

## Canonical FR → Linear mapping (final)

- **FR-1 RepoSpace** → `OPE-161 / BIG-4501`
- **FR-2 Commit DAG & diff gateway** → `OPE-162 / BIG-4502`
- **FR-3 TaskRun ↔ commit binding** → `OPE-163 / BIG-4503`
- **FR-4 Repo board discussion** → `OPE-164 / BIG-4504`
- **FR-5 Run Detail repo evidence** → `OPE-167 / BIG-4506`
- **FR-6 Audit & acceptance repo evidence** → `OPE-172 / BIG-4510` + `OPE-163 / BIG-4503`
- **FR-7 Repo metrics/reporting** → `OPE-168 / BIG-4507` + `OPE-169 / BIG-4508`
- **FR-8 Permission/quota/rate limit** → `OPE-172 / BIG-4510` + `OPE-173 / BIG-4511`

Supporting integration:
- Collaboration aggregation: `OPE-165 / BIG-4505`
- Approval/triage lineage-aware evidence: `OPE-170 / BIG-4509`
- Pilot gate and rollout scorecard: `OPE-174 / BIG-4512`

## Canonical MVP acceptance checklist

1. A `TaskRun` can persist one-or-more commit links with role semantics (`source/candidate/closeout/accepted`).
2. Run Detail can render repo evidence without live gateway fetch at render time.
3. Repo board post/reply can target task/run/commit and merge into collaboration narrative.
4. Audit contract validates required repo fields for repo actions and accepted-commit decisions.
5. Governance controls can block/degrade out-of-policy actions deterministically.
6. Weekly and narrative exports include repo evidence section in markdown/text/html.

## Validation command baseline

- Incremental:
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_registry.py tests/test_repo_gateway.py tests/test_repo_links.py`
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_reports.py`
  - `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py tests/test_repo_triage.py tests/test_operations.py tests/test_repo_rollout.py`
  - `cd bigclaw-go && go test ./internal/api ./internal/repo`
- Full:
  - `PYTHONPATH=src python3 -m pytest -q`

## Final execution note
This alignment file is now the single source of truth for PRD-to-issue traceability and release acceptance in the AgentHub integration stream.
