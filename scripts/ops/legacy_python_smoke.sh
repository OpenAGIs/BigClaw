#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)

PYTHONPATH="$repo_root/src" python3 - <<'PY'
from bigclaw.console_ia import ConsoleInteractionAuditor, build_big_4203_console_interaction_draft
from bigclaw.operations import OperationsAnalytics
from bigclaw.planning import build_big_4701_execution_plan, build_v3_candidate_backlog
from bigclaw.ui_review import UIReviewPackAuditor, build_big_4204_review_pack

backlog = build_v3_candidate_backlog()
assert backlog.ranked_candidates[0].candidate_id == "candidate-ops-hardening"

plan = build_big_4701_execution_plan()
assert plan.plan_id == "BIG-4701"
plan.validate()

draft = build_big_4203_console_interaction_draft()
assert ConsoleInteractionAuditor().audit(draft).release_ready

pack = build_big_4204_review_pack()
assert UIReviewPackAuditor().audit(pack).ready

snapshot = OperationsAnalytics().summarize_runs([])
assert snapshot.total_runs == 0
assert snapshot.approval_queue_depth == 0

print("legacy_python_smoke_ok")
PY
