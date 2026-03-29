package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonReportsContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "reports_contract.py")
	script := `import json
import tempfile
import sys
from pathlib import Path
from typing import List, Optional

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.collaboration import CollaborationComment, DecisionNote, build_collaboration_thread
from bigclaw.reports import (
    ConsoleAction,
    BillingEntitlementsPage,
    BillingRunCharge,
    DocumentationArtifact,
    FinalDeliveryChecklist,
    LaunchChecklistItem,
    NarrativeSection,
    OrchestrationCanvas,
    OrchestrationPortfolio,
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    ReportStudio,
    SharedViewContext,
    SharedViewFilter,
    build_auto_triage_center,
    build_billing_entitlements_page,
    build_billing_entitlements_page_from_ledger,
    build_final_delivery_checklist,
    build_orchestration_canvas,
    build_orchestration_canvas_from_ledger_entry,
    build_orchestration_portfolio,
    build_orchestration_portfolio_from_ledger,
    build_launch_checklist,
    build_takeover_queue_from_ledger,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_billing_entitlements_page,
    render_billing_entitlements_report,
    render_final_delivery_checklist_report,
    render_orchestration_canvas,
    render_orchestration_overview_page,
    render_orchestration_portfolio_report,
    render_issue_validation_report,
    render_launch_checklist_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_report_studio_html,
    render_report_studio_plain_text,
    render_report_studio_report,
    render_shared_view_context,
    render_takeover_queue_report,
    validation_report_exists,
    write_report,
    write_report_studio_bundle,
    TriageFeedbackRecord,
)
from bigclaw.observability import TaskRun
from bigclaw.models import Task
from bigclaw.orchestration import DepartmentHandoff, HandoffRequest, OrchestrationPlan, OrchestrationPolicyDecision


def make_shared_view(
    result_count: int,
    *,
    loading: bool = False,
    errors: Optional[List[str]] = None,
    partial_data: Optional[List[str]] = None,
) -> SharedViewContext:
    return SharedViewContext(
        filters=[
            SharedViewFilter(label="Team", value="engineering"),
            SharedViewFilter(label="Window", value="2026-03-10"),
        ],
        result_count=result_count,
        loading=loading,
        errors=errors or [],
        partial_data=partial_data or [],
        last_updated="2026-03-11T09:00:00Z",
    )


with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    content = render_issue_validation_report("BIG-101", "v0.1", "sandbox", "pass")
    out = td / "report.md"
    write_report(str(out), content)
    render_write_report = {
        "exists": out.exists(),
        "has_issue": "BIG-101" in out.read_text(),
        "has_pass": "pass" in out.read_text(),
    }

enabled = ConsoleAction("retry", "Retry", "run-1")
disabled = ConsoleAction("pause", "Pause", "run-1", enabled=False, reason="already completed")

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    studio = ReportStudio(
        name="Executive Weekly Narrative",
        issue_id="OPE-112",
        audience="executive",
        period="2026-W11",
        summary="Delivery recovered after approval bottlenecks were cleared in the second half of the week.",
        sections=[
            NarrativeSection(
                heading="What changed",
                body="Approval queue depth fell from 5 to 1 after moving browser-heavy runs onto the shared operations lane.",
                evidence=["queue-control-center", "weekly-operations"],
                callouts=["SLA risk contained", "No new regressions opened"],
            ),
            NarrativeSection(
                heading="What needs attention",
                body="Security takeover requests still cluster around data-export tasks and need a dedicated reviewer window.",
                evidence=["takeover-queue"],
                callouts=["Review staffing before Friday close"],
            ),
        ],
        action_items=["Publish the markdown export to leadership", "Review security handoff staffing"],
        source_reports=["reports/weekly-operations.md", "reports/takeover-queue.md"],
    )
    markdown = render_report_studio_report(studio)
    plain_text = render_report_studio_plain_text(studio)
    html = render_report_studio_html(studio)
    artifacts = write_report_studio_bundle(str(td / "studio"), studio)
    report_studio = {
        "ready": studio.ready,
        "recommendation": studio.recommendation,
        "markdown": "# Report Studio" in markdown and "### What changed" in markdown,
        "plain": "Recommendation: publish" in plain_text,
        "html": "<h1>Executive Weekly Narrative</h1>" in html,
        "artifacts": Path(artifacts.markdown_path).exists() and Path(artifacts.html_path).exists() and Path(artifacts.text_path).exists(),
        "slug": "executive-weekly-narrative.md" in artifacts.markdown_path,
    }

draft_studio = ReportStudio(
    name="Draft Narrative",
    issue_id="OPE-112",
    audience="operations",
    period="2026-W11",
    summary="",
    sections=[NarrativeSection(heading="Open risks", body="")],
)

scorecard = PilotScorecard(
    issue_id="OPE-60",
    customer="Design Partner A",
    period="2026-Q2",
    metrics=[
        PilotMetric(name="Automation coverage", baseline=35, current=82, target=80, unit="%"),
        PilotMetric(name="Manual review time", baseline=12, current=4, target=5, unit="h", higher_is_better=False),
    ],
    monthly_benefit=12000,
    monthly_cost=2500,
    implementation_cost=18000,
    benchmark_score=96,
    benchmark_passed=True,
)
scorecard_content = render_pilot_scorecard(scorecard)

negative_scorecard = PilotScorecard(
    issue_id="OPE-60",
    customer="Design Partner B",
    period="2026-Q2",
    metrics=[PilotMetric(name="Backlog aging", baseline=5, current=7, target=4, unit="d", higher_is_better=False)],
    monthly_benefit=1000,
    monthly_cost=3000,
    implementation_cost=12000,
    benchmark_passed=False,
)

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    report_path = td / "validation.md"
    missing_decision = evaluate_issue_closure("BIG-602", str(report_path))
    write_report(str(report_path), "# Validation\n\nfailed")
    failed_decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=False)
    write_report(str(report_path), render_issue_validation_report("BIG-602", "v0.1", "sandbox", "pass"))
    success_decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=True)
    closure_basics = {
        "missing_allowed": missing_decision.allowed,
        "missing_reason": missing_decision.reason,
        "missing_exists": validation_report_exists(str(td / "missing.md")),
        "failed_allowed": failed_decision.allowed,
        "failed_reason": failed_decision.reason,
        "success_allowed": success_decision.allowed,
        "success_reason": success_decision.reason,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runbook = td / "runbook.md"
    faq = td / "faq.md"
    write_report(str(runbook), "# Runbook\n\nready")
    checklist = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="faq", path=str(faq)),
        ],
        items=[
            LaunchChecklistItem(name="Operations handoff", evidence=["runbook"]),
            LaunchChecklistItem(name="Support handoff", evidence=["faq"]),
        ],
    )
    report = render_launch_checklist_report(checklist)
    launch_checklist = {
        "documentation_status": checklist.documentation_status,
        "completed_items": checklist.completed_items,
        "missing_documentation": checklist.missing_documentation,
        "ready": checklist.ready,
        "report": "runbook: available=True" in report and "faq: available=False" in report and "Support handoff: completed=False evidence=faq" in report,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    validation_bundle = td / "validation-bundle.md"
    release_notes = td / "release-notes.md"
    write_report(str(validation_bundle), "# Validation Bundle\n\nready")
    checklist = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[
            DocumentationArtifact(name="validation-bundle", path=str(validation_bundle)),
            DocumentationArtifact(name="release-notes", path=str(release_notes)),
        ],
        recommended_documentation=[
            DocumentationArtifact(name="runbook", path=str(td / "runbook.md")),
            DocumentationArtifact(name="faq", path=str(td / "faq.md")),
        ],
    )
    report = render_final_delivery_checklist_report(checklist)
    final_delivery = {
        "required_output_status": checklist.required_output_status,
        "recommended_status": checklist.recommended_documentation_status,
        "generated_required": checklist.generated_required_outputs,
        "generated_recommended": checklist.generated_recommended_documentation,
        "missing_required": checklist.missing_required_outputs,
        "missing_recommended": checklist.missing_recommended_documentation,
        "ready": checklist.ready,
        "report": "Required Outputs Generated: 1/2" in report and "Recommended Docs Generated: 0/2" in report and "release-notes: available=False" in report and "runbook: available=False" in report,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    report_path = td / "validation.md"
    runbook = td / "runbook.md"
    write_report(str(report_path), render_issue_validation_report("BIG-1003", "v0.2", "staging", "pass"))
    write_report(str(runbook), "# Runbook\n\nready")
    launch_incomplete = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="launch-faq", path=str(td / "launch-faq.md")),
        ],
        items=[LaunchChecklistItem(name="Launch comms", evidence=["runbook", "launch-faq"])],
    )
    launch_blocked = evaluate_issue_closure("BIG-1003", str(report_path), validation_passed=True, launch_checklist=launch_incomplete)

    final_incomplete = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[DocumentationArtifact(name="validation-bundle", path=str(td / "validation-bundle.md"))],
        recommended_documentation=[DocumentationArtifact(name="runbook", path=str(td / "runbook.md"))],
    )
    final_blocked = evaluate_issue_closure("BIG-4702", str(report_path), validation_passed=True, final_delivery_checklist=final_incomplete)

    validation_bundle = td / "validation-bundle-ready.md"
    release_notes = td / "release-notes-ready.md"
    write_report(str(validation_bundle), "# Validation Bundle\n\nready")
    write_report(str(release_notes), "# Release Notes\n\nready")
    final_ready = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[
            DocumentationArtifact(name="validation-bundle", path=str(validation_bundle)),
            DocumentationArtifact(name="release-notes", path=str(release_notes)),
        ],
        recommended_documentation=[DocumentationArtifact(name="runbook", path=str(td / "runbook.md"))],
    )
    final_allowed = evaluate_issue_closure("BIG-4702", str(report_path), validation_passed=True, final_delivery_checklist=final_ready)

    faq = td / "launch-faq.md"
    write_report(str(faq), "# FAQ\n\nready")
    launch_ready = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="launch-faq", path=str(faq)),
        ],
        items=[LaunchChecklistItem(name="Launch comms", evidence=["runbook", "launch-faq"])],
    )
    launch_allowed = evaluate_issue_closure("BIG-1003", str(report_path), validation_passed=True, launch_checklist=launch_ready)
    closure_checklists = {
        "launch_blocked": launch_blocked.allowed,
        "launch_blocked_reason": launch_blocked.reason,
        "final_blocked": final_blocked.allowed,
        "final_blocked_reason": final_blocked.reason,
        "final_allowed": final_allowed.allowed,
        "final_allowed_reason": final_allowed.reason,
        "launch_allowed": launch_allowed.allowed,
        "launch_allowed_reason": launch_allowed.reason,
    }

portfolio = PilotPortfolio(
    name="Design Partners",
    period="2026-H1",
    scorecards=[
        PilotScorecard(
            issue_id="OPE-60",
            customer="Partner A",
            period="2026-Q2",
            metrics=[PilotMetric(name="Coverage", baseline=40, current=85, target=80, unit="%")],
            monthly_benefit=15000,
            monthly_cost=3000,
            implementation_cost=18000,
            benchmark_score=97,
            benchmark_passed=True,
        ),
        PilotScorecard(
            issue_id="OPE-61",
            customer="Partner B",
            period="2026-Q2",
            metrics=[PilotMetric(name="Cycle time", baseline=12, current=7, target=5, unit="h", higher_is_better=False)],
            monthly_benefit=9000,
            monthly_cost=2500,
            implementation_cost=12000,
            benchmark_score=88,
            benchmark_passed=True,
        ),
    ],
)
pilot_portfolio_report = render_pilot_portfolio_report(portfolio)

view = SharedViewContext(
    filters=[SharedViewFilter(label="Team", value="ops")],
    result_count=4,
    collaboration=build_collaboration_thread(
        "dashboard",
        "ops-overview",
        comments=[
            CollaborationComment(
                comment_id="dashboard-comment-1",
                author="pm",
                body="Please review blocker copy with @ops and @eng.",
                mentions=["ops", "eng"],
                anchor="blockers",
            )
        ],
        decisions=[
            DecisionNote(
                decision_id="dashboard-decision-1",
                author="ops",
                outcome="approved",
                summary="Keep the blocker module visible for managers.",
                mentions=["pm"],
                follow_up="Recheck after next data refresh.",
            )
        ],
    ),
)
shared_view_content = "\n".join(render_shared_view_context(view))

approval_task = Task(task_id="OPE-76-risk", source="linear", title="Prod approval", description="")
approval_run = TaskRun.from_task(approval_task, run_id="run-risk", medium="vm")
approval_run.trace("scheduler.decide", "pending")
approval_run.audit("scheduler.decision", "scheduler", "pending", reason="requires approval for high-risk task")
approval_run.finalize("needs-approval", "requires approval for high-risk task")

failed_task = Task(task_id="OPE-76-browser", source="linear", title="Replay browser task", description="")
failed_run = TaskRun.from_task(failed_task, run_id="run-browser", medium="browser")
failed_run.trace("runtime.execute", "failed")
failed_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
failed_run.finalize("failed", "browser session crashed")

healthy_task = Task(task_id="OPE-76-ok", source="linear", title="Healthy run", description="")
healthy_run = TaskRun.from_task(healthy_task, run_id="run-ok", medium="docker")
healthy_run.trace("scheduler.decide", "ok")
healthy_run.audit("scheduler.decision", "scheduler", "approved", reason="default low risk path")
healthy_run.finalize("approved", "default low risk path")

triage_center = build_auto_triage_center([healthy_run, approval_run, failed_run], name="Engineering Ops", period="2026-03-10")
triage_report = render_auto_triage_center_report(triage_center, total_runs=3)

risk_task = Task(task_id="OPE-94-risk", source="linear", title="Prod approval", description="")
risk_run = TaskRun.from_task(risk_task, run_id="run-risk", medium="vm")
risk_run.audit("scheduler.decision", "scheduler", "pending", reason="requires approval for high-risk task")
risk_run.finalize("needs-approval", "requires approval for high-risk task")
partial_triage = render_auto_triage_center_report(
    build_auto_triage_center([risk_run], name="Engineering Ops", period="2026-03-10"),
    total_runs=1,
    view=make_shared_view(1, partial_data=["Replay ledger data is still backfilling."]),
)

failed_browser_task = Task(task_id="OPE-100-browser-a", source="linear", title="Browser replay failure", description="")
failed_browser_run = TaskRun.from_task(failed_browser_task, run_id="run-browser-a", medium="browser")
failed_browser_run.trace("runtime.execute", "failed")
failed_browser_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
failed_browser_run.finalize("failed", "browser session crashed")

similar_browser_task = Task(task_id="OPE-100-browser-b", source="linear", title="Browser replay failure", description="")
similar_browser_run = TaskRun.from_task(similar_browser_task, run_id="run-browser-b", medium="browser")
similar_browser_run.trace("runtime.execute", "failed")
similar_browser_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
similar_browser_run.finalize("failed", "browser session crashed")

approval_task2 = Task(task_id="OPE-100-security", source="linear", title="Security approval", description="")
approval_run2 = TaskRun.from_task(approval_task2, run_id="run-security", medium="vm")
approval_run2.trace("scheduler.decide", "pending")
approval_run2.audit("scheduler.decision", "scheduler", "pending", reason="requires approval for high-risk task")
approval_run2.finalize("needs-approval", "requires approval for high-risk task")

feedback = [
    TriageFeedbackRecord(run_id="run-browser-a", action="replay run and inspect tool failures", decision="accepted", actor="ops-lead", notes="matched previous recovery path"),
    TriageFeedbackRecord(run_id="run-security", action="request approval and queue security review", decision="rejected", actor="sec-reviewer", notes="approval already in flight"),
]
feedback_center = build_auto_triage_center([failed_browser_run, similar_browser_run, approval_run2], name="Auto Triage Center", period="2026-03-11", feedback=feedback)
feedback_report = render_auto_triage_center_report(feedback_center, total_runs=3)
browser_item = next(item for item in feedback_center.inbox if item.run_id == "run-browser-a")
approval_item = next(item for item in feedback_center.inbox if item.run_id == "run-security")

entries = [
    {
        "run_id": "run-sec",
        "task_id": "OPE-66-sec",
        "source": "linear",
        "summary": "requires approval for high-risk task",
        "audits": [
            {
                "action": "orchestration.handoff",
                "outcome": "pending",
                "details": {
                    "target_team": "security",
                    "reason": "requires approval for high-risk task",
                    "required_approvals": ["security-review"],
                },
            }
        ],
    },
    {
        "run_id": "run-ops",
        "task_id": "OPE-66-ops",
        "source": "linear",
        "summary": "premium tier required for advanced cross-department orchestration",
        "audits": [
            {
                "action": "orchestration.handoff",
                "outcome": "pending",
                "details": {
                    "target_team": "operations",
                    "reason": "premium tier required for advanced cross-department orchestration",
                    "required_approvals": ["ops-manager"],
                },
            }
        ],
    },
    {
        "run_id": "run-ok",
        "task_id": "OPE-66-ok",
        "source": "linear",
        "summary": "default low risk path",
        "audits": [{"action": "scheduler.decision", "outcome": "approved", "details": {"reason": "default low risk path"}}],
    },
]
takeover_queue = build_takeover_queue_from_ledger(entries, name="Cross-Team Takeovers", period="2026-03-10")
takeover_report = render_takeover_queue_report(takeover_queue, total_runs=3)
takeover_error = render_takeover_queue_report(build_takeover_queue_from_ledger([], name="Cross-Team Takeovers", period="2026-03-10"), total_runs=0, view=make_shared_view(0, errors=["Takeover approvals service timed out."]))

canvas_task = Task(task_id="OPE-66-canvas", source="linear", title="Canvas run", description="")
canvas_run = TaskRun.from_task(canvas_task, run_id="run-canvas", medium="browser")
canvas_run.audit("tool.invoke", "worker", "success", tool="browser")
plan = OrchestrationPlan(
    task_id="OPE-66-canvas",
    collaboration_mode="tier-limited",
    handoffs=[
        DepartmentHandoff("operations", "coordinate"),
        DepartmentHandoff("engineering", "execute", required_tools=["browser"]),
    ],
)
policy = OrchestrationPolicyDecision(
    tier="standard",
    upgrade_required=True,
    reason="premium tier required for advanced cross-department orchestration",
    blocked_departments=["customer-success"],
    entitlement_status="upgrade-required",
    billing_model="standard-blocked",
    estimated_cost_usd=7.0,
    included_usage_units=2,
    overage_usage_units=1,
    overage_cost_usd=4.0,
)
handoff = HandoffRequest(target_team="operations", reason=policy.reason, required_approvals=["ops-manager"])
canvas = build_orchestration_canvas(canvas_run, plan, policy, handoff)
canvas_report = render_orchestration_canvas(canvas)

flow_entry = {
    "run_id": "run-flow-1",
    "task_id": "OPE-113",
    "audits": [
        {"action": "orchestration.plan", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:00:00Z", "details": {"collaboration_mode": "cross-functional", "departments": ["operations", "engineering"], "approvals": []}},
        {"action": "orchestration.policy", "actor": "scheduler", "outcome": "enabled", "timestamp": "2026-03-11T11:01:00Z", "details": {"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included"}},
        {"action": "collaboration.comment", "actor": "ops-lead", "outcome": "recorded", "timestamp": "2026-03-11T11:02:00Z", "details": {"surface": "flow", "comment_id": "flow-comment-1", "body": "Route @eng once the dashboard note is resolved.", "mentions": ["eng"], "anchor": "handoff-lane", "status": "open"}},
        {"action": "collaboration.decision", "actor": "eng-manager", "outcome": "accepted", "timestamp": "2026-03-11T11:03:00Z", "details": {"surface": "flow", "decision_id": "flow-decision-1", "summary": "Engineering owns the next flow handoff.", "mentions": ["ops-lead"], "related_comment_ids": ["flow-comment-1"], "follow_up": "Post in the shared channel after deploy."}},
    ],
}
flow_canvas = build_orchestration_canvas_from_ledger_entry(flow_entry)
flow_report = render_orchestration_canvas(flow_canvas)

portfolio_canvases = [
    OrchestrationCanvas(task_id="OPE-66-a", run_id="run-a", collaboration_mode="cross-functional", departments=["operations", "engineering", "security"], tier="premium", entitlement_status="included", billing_model="premium-included", estimated_cost_usd=4.5, included_usage_units=3, handoff_team="security", handoff_status="pending"),
    OrchestrationCanvas(task_id="OPE-66-b", run_id="run-b", collaboration_mode="tier-limited", departments=["operations", "engineering"], tier="standard", upgrade_required=True, entitlement_status="upgrade-required", billing_model="standard-blocked", estimated_cost_usd=7.0, included_usage_units=2, overage_usage_units=1, overage_cost_usd=4.0, blocked_departments=["customer-success"], handoff_team="operations", handoff_status="pending"),
]
portfolio_queue = build_takeover_queue_from_ledger(
    [
        {"run_id": "run-a", "task_id": "OPE-66-a", "source": "linear", "audits": [{"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "security", "reason": "risk", "required_approvals": ["security-review"]}}]},
        {"run_id": "run-b", "task_id": "OPE-66-b", "source": "linear", "audits": [{"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "operations", "reason": "entitlement", "required_approvals": ["ops-manager"]}}]},
    ],
    name="Cross-Team Takeovers",
    period="2026-03-10",
)
portfolio2 = build_orchestration_portfolio(portfolio_canvases, name="Cross-Team Portfolio", period="2026-03-10", takeover_queue=portfolio_queue)
portfolio_report = render_orchestration_portfolio_report(portfolio2)
portfolio_empty = render_orchestration_portfolio_report(build_orchestration_portfolio([], name="Cross-Team Portfolio", period="2026-03-10"), view=make_shared_view(0))

overview_page = render_orchestration_overview_page(
    OrchestrationPortfolio(
        name="Cross-Team Portfolio",
        period="2026-03-10",
        canvases=[OrchestrationCanvas(task_id="OPE-66-a", run_id="run-a", collaboration_mode="cross-functional", departments=["operations", "engineering"], tier="premium", entitlement_status="included", billing_model="premium-included", estimated_cost_usd=3.0, handoff_team="security")],
        takeover_queue=build_takeover_queue_from_ledger(
            [{"run_id": "run-a", "task_id": "OPE-66-a", "source": "linear", "audits": [{"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "security", "reason": "risk", "required_approvals": ["security-review"]}}]}],
            name="Cross-Team Takeovers",
            period="2026-03-10",
        ),
    )
)

ledger_canvas = build_orchestration_canvas_from_ledger_entry(
    {
        "run_id": "run-ledger",
        "task_id": "OPE-66-ledger",
        "audits": [
            {"action": "orchestration.plan", "outcome": "ready", "details": {"collaboration_mode": "tier-limited", "departments": ["operations", "engineering"], "approvals": ["security-review"]}},
            {"action": "orchestration.policy", "outcome": "upgrade-required", "details": {"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": ["security", "customer-success"]}},
            {"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "operations", "reason": "premium tier required for advanced cross-department orchestration"}},
            {"action": "tool.invoke", "outcome": "success", "details": {"tool": "browser"}},
        ],
    }
)

portfolio_from_ledger = build_orchestration_portfolio_from_ledger(
    [
        {
            "run_id": "run-a",
            "task_id": "OPE-66-a",
            "audits": [
                {"action": "orchestration.plan", "outcome": "ready", "details": {"collaboration_mode": "cross-functional", "departments": ["operations", "engineering", "security"], "approvals": ["security-review"]}},
                {"action": "orchestration.policy", "outcome": "enabled", "details": {"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []}},
                {"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "security", "reason": "approval required", "required_approvals": ["security-review"]}},
            ],
        },
        {
            "run_id": "run-b",
            "task_id": "OPE-66-b",
            "audits": [
                {"action": "orchestration.plan", "outcome": "ready", "details": {"collaboration_mode": "tier-limited", "departments": ["operations", "engineering"], "approvals": []}},
                {"action": "orchestration.policy", "outcome": "upgrade-required", "details": {"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": ["customer-success"]}},
                {"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "operations", "reason": "entitlement gap", "required_approvals": ["ops-manager"]}},
            ],
        },
    ],
    name="Ledger Portfolio",
    period="2026-03-10",
)

billing_page = build_billing_entitlements_page(
    OrchestrationPortfolio(
        name="Revenue Ops",
        period="2026-03",
        canvases=[
            OrchestrationCanvas(task_id="OPE-104-a", run_id="run-billing-a", collaboration_mode="cross-functional", departments=["operations", "engineering", "security"], tier="premium", entitlement_status="included", billing_model="premium-included", estimated_cost_usd=4.5, included_usage_units=3, handoff_team="security"),
            OrchestrationCanvas(task_id="OPE-104-b", run_id="run-billing-b", collaboration_mode="tier-limited", departments=["operations", "engineering"], tier="standard", upgrade_required=True, entitlement_status="upgrade-required", billing_model="standard-blocked", estimated_cost_usd=7.0, included_usage_units=2, overage_usage_units=1, overage_cost_usd=4.0, blocked_departments=["customer-success"], handoff_team="operations"),
        ],
    ),
    workspace_name="OpenAGI Revenue Cloud",
    plan_name="Standard",
    billing_period="2026-03",
)
billing_report = render_billing_entitlements_report(billing_page)

html_billing = render_billing_entitlements_page(
    BillingEntitlementsPage(
        workspace_name="OpenAGI Revenue Cloud",
        plan_name="Premium",
        billing_period="2026-03",
        charges=[BillingRunCharge(run_id="run-billing-a", task_id="OPE-104-a", entitlement_status="included", billing_model="premium-included", estimated_cost_usd=4.5, included_usage_units=3, recommendation="review-security-takeover")],
    )
)

ledger_billing = build_billing_entitlements_page_from_ledger(
    [
        {
            "run_id": "run-ledger-a",
            "task_id": "OPE-104-a",
            "audits": [
                {"action": "orchestration.plan", "outcome": "ready", "details": {"collaboration_mode": "cross-functional", "departments": ["operations", "engineering", "security"], "approvals": ["security-review"]}},
                {"action": "orchestration.policy", "outcome": "enabled", "details": {"tier": "premium", "entitlement_status": "included", "billing_model": "premium-included", "estimated_cost_usd": 4.5, "included_usage_units": 3, "blocked_departments": []}},
            ],
        },
        {
            "run_id": "run-ledger-b",
            "task_id": "OPE-104-b",
            "audits": [
                {"action": "orchestration.plan", "outcome": "ready", "details": {"collaboration_mode": "tier-limited", "departments": ["operations", "engineering"], "approvals": []}},
                {"action": "orchestration.policy", "outcome": "upgrade-required", "details": {"tier": "standard", "entitlement_status": "upgrade-required", "billing_model": "standard-blocked", "estimated_cost_usd": 7.0, "included_usage_units": 2, "overage_usage_units": 1, "overage_cost_usd": 4.0, "blocked_departments": ["customer-success"]}},
                {"action": "orchestration.handoff", "outcome": "pending", "details": {"target_team": "operations", "reason": "entitlement gap", "required_approvals": ["ops-manager"]}},
            ],
        },
    ],
    workspace_name="OpenAGI Revenue Cloud",
    plan_name="Standard",
    billing_period="2026-03",
)

feedback_record = TriageFeedbackRecord(run_id="run-1", action="classify", decision="accepted", actor="ops")
validation_content = render_issue_validation_report("BIG-900", "v1", "repo", "pass")
timestamp_line = next(line for line in validation_content.splitlines() if line.startswith("- 生成时间:"))
timestamp_value = timestamp_line.split(": ", 1)[1]

print(json.dumps({
    "render_write_report": render_write_report,
    "console_action": {"enabled": enabled.state, "disabled": disabled.state},
    "report_studio": report_studio,
    "draft_studio": {"ready": draft_studio.ready, "recommendation": draft_studio.recommendation},
    "pilot_scorecard": {
        "metrics_met": scorecard.metrics_met,
        "recommendation": scorecard.recommendation,
        "payback_months": scorecard.payback_months,
        "content": "Annualized ROI: 200.0%" in scorecard_content and "Recommendation: go" in scorecard_content and "Benchmark Score: 96" in scorecard_content and "Automation coverage" in scorecard_content,
        "negative_value": negative_scorecard.monthly_net_value,
        "negative_payback": negative_scorecard.payback_months,
        "negative_recommendation": negative_scorecard.recommendation,
    },
    "closure_basics": closure_basics,
    "launch_checklist": launch_checklist,
    "final_delivery": final_delivery,
    "closure_checklists": closure_checklists,
    "pilot_portfolio": {
        "net_value": portfolio.total_monthly_net_value,
        "average_roi": portfolio.average_roi,
        "counts": portfolio.recommendation_counts,
        "recommendation": portfolio.recommendation,
        "report": "Recommendation Mix: go=1 iterate=1 hold=0" in pilot_portfolio_report and "Partner A: recommendation=go" in pilot_portfolio_report and "Partner B: recommendation=iterate" in pilot_portfolio_report,
    },
    "shared_view": {
        "collaboration": "## Collaboration" in shared_view_content and "Surface: dashboard" in shared_view_content and "Please review blocker copy with @ops and @eng." in shared_view_content and "Keep the blocker module visible for managers." in shared_view_content,
    },
    "triage_center": {
        "flagged_runs": triage_center.flagged_runs,
        "inbox_size": triage_center.inbox_size,
        "severity_counts": triage_center.severity_counts,
        "owner_counts": triage_center.owner_counts,
        "recommendation": triage_center.recommendation,
        "findings": [finding.run_id for finding in triage_center.findings],
        "inbox": [item.run_id for item in triage_center.inbox],
        "suggestion_label": triage_center.inbox[0].suggestions[0].label,
        "suggestion_confidence": triage_center.inbox[0].suggestions[0].confidence,
        "next_actions": [triage_center.findings[0].next_action, triage_center.findings[1].next_action],
        "action_states": [triage_center.findings[0].actions[4].enabled, triage_center.findings[1].actions[4].enabled, triage_center.findings[1].actions[6].enabled],
        "report": "Flagged Runs: 2" in triage_report and "Inbox Size: 2" in triage_report and "Severity Mix: critical=1 high=1 medium=0" in triage_report and "Feedback Loop: accepted=0 rejected=0 pending=2" in triage_report and "run-browser: severity=critical owner=engineering status=failed" in triage_report and "run-risk: severity=high owner=security status=needs-approval" in triage_report and "actions=Drill Down [drill-down]" in triage_report and "Retry [retry] state=disabled target=run-risk reason=retry available after owner review" in triage_report,
        "partial": "## View State" in partial_triage and "- State: partial-data" in partial_triage and "Replay ledger data is still backfilling." in partial_triage,
    },
    "triage_feedback": {
        "feedback_counts": feedback_center.feedback_counts,
        "browser_status": browser_item.suggestions[0].feedback_status,
        "approval_status": approval_item.suggestions[0].feedback_status,
        "related_run": browser_item.suggestions[0].evidence[0].related_run_id,
        "score": browser_item.suggestions[0].evidence[0].score,
        "report": "## Inbox" in feedback_report and "run-browser-a: severity=critical owner=engineering status=failed" in feedback_report and "similar=run-browser-b:" in feedback_report and "Feedback Loop: accepted=1 rejected=1 pending=1" in feedback_report,
    },
    "takeover_queue": {
        "pending_requests": takeover_queue.pending_requests,
        "team_counts": takeover_queue.team_counts,
        "approval_count": takeover_queue.approval_count,
        "recommendation": takeover_queue.recommendation,
        "requests": [request.run_id for request in takeover_queue.requests],
        "action_enabled": takeover_queue.requests[0].actions[3].enabled,
        "action_disabled": takeover_queue.requests[1].actions[3].enabled,
        "report": "Pending Requests: 2" in takeover_report and "Team Mix: operations=1 security=1" in takeover_report and "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review" in takeover_report and "run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager" in takeover_report and "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated" in takeover_report,
        "error": "- State: error" in takeover_error and "- Summary: Unable to load data for the current filters." in takeover_error and "Takeover approvals service timed out." in takeover_error,
    },
    "orchestration_canvas": {
        "recommendation": canvas.recommendation,
        "active_tools": canvas.active_tools,
        "actions": [canvas.actions[3].enabled, canvas.actions[4].enabled],
        "report": "# Orchestration Canvas" in canvas_report and "- Tier: standard" in canvas_report and "- Entitlement Status: upgrade-required" in canvas_report and "- Billing Model: standard-blocked" in canvas_report and "- Estimated Cost (USD): 7.00" in canvas_report and "- Handoff Team: operations" in canvas_report and "- Recommendation: resolve-entitlement-gap" in canvas_report and "Escalate [escalate] state=enabled target=run-canvas" in canvas_report,
        "flow": flow_canvas.recommendation == "resolve-flow-comments" and "## Collaboration" in flow_report and "Route @eng once the dashboard note is resolved." in flow_report and "Engineering owns the next flow handoff." in flow_report,
    },
    "orchestration_portfolio": {
        "total_runs": portfolio2.total_runs,
        "modes": portfolio2.collaboration_modes,
        "tiers": portfolio2.tier_counts,
        "entitlements": portfolio2.entitlement_counts,
        "billing": portfolio2.billing_model_counts,
        "estimated_cost": portfolio2.total_estimated_cost_usd,
        "overage_cost": portfolio2.total_overage_cost_usd,
        "upgrade_required": portfolio2.upgrade_required_count,
        "active_handoffs": portfolio2.active_handoffs,
        "recommendation": portfolio2.recommendation,
        "report": "# Orchestration Portfolio Report" in portfolio_report and "- Collaboration Mix: cross-functional=1 tier-limited=1" in portfolio_report and "- Tier Mix: premium=1 standard=1" in portfolio_report and "- Entitlement Mix: included=1 upgrade-required=1" in portfolio_report and "- Billing Models: premium-included=1 standard-blocked=1" in portfolio_report and "- Estimated Cost (USD): 11.50" in portfolio_report and "- Overage Cost (USD): 4.00" in portfolio_report and "- Takeover Queue: pending=2 recommendation=expedite-security-review" in portfolio_report and "- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=False handoff=security" in portfolio_report and "actions=Drill Down [drill-down]" in portfolio_report,
        "empty": "- State: empty" in portfolio_empty and "- Summary: No records match the current filters." in portfolio_empty and "## Filters" in portfolio_empty,
        "overview_page": "<title>Orchestration Overview" in overview_page and "Cross-Team Portfolio" in overview_page and "review-security-takeover" in overview_page and "Estimated Cost" in overview_page and "premium-included" in overview_page and "pending=1 recommendation=expedite-security-review" in overview_page and "run-a" in overview_page and "actions=Drill Down [drill-down]" in overview_page,
    },
    "ledger_orchestration": {
        "canvas": {
            "run_id": ledger_canvas.run_id,
            "mode": ledger_canvas.collaboration_mode,
            "departments": ledger_canvas.departments,
            "approvals": ledger_canvas.required_approvals,
            "tier": ledger_canvas.tier,
            "upgrade_required": ledger_canvas.upgrade_required,
            "entitlement": ledger_canvas.entitlement_status,
            "billing_model": ledger_canvas.billing_model,
            "estimated_cost": ledger_canvas.estimated_cost_usd,
            "included": ledger_canvas.included_usage_units,
            "overage_units": ledger_canvas.overage_usage_units,
            "overage_cost": ledger_canvas.overage_cost_usd,
            "blocked_departments": ledger_canvas.blocked_departments,
            "handoff_team": ledger_canvas.handoff_team,
            "active_tools": ledger_canvas.active_tools,
            "actions": [ledger_canvas.actions[3].enabled, ledger_canvas.actions[4].enabled],
        },
        "portfolio": {
            "total_runs": portfolio_from_ledger.total_runs,
            "modes": portfolio_from_ledger.collaboration_modes,
            "tiers": portfolio_from_ledger.tier_counts,
            "entitlements": portfolio_from_ledger.entitlement_counts,
            "estimated_cost": portfolio_from_ledger.total_estimated_cost_usd,
            "pending_requests": portfolio_from_ledger.takeover_queue.pending_requests if portfolio_from_ledger.takeover_queue else None,
            "recommendation": portfolio_from_ledger.recommendation,
        },
    },
    "billing": {
        "page": {
            "run_count": billing_page.run_count,
            "included_units": billing_page.total_included_usage_units,
            "overage_units": billing_page.total_overage_usage_units,
            "estimated_cost": billing_page.total_estimated_cost_usd,
            "overage_cost": billing_page.total_overage_cost_usd,
            "upgrade_required": billing_page.upgrade_required_count,
            "entitlement_counts": billing_page.entitlement_counts,
            "billing_counts": billing_page.billing_model_counts,
            "blocked_capabilities": billing_page.blocked_capabilities,
            "recommendation": billing_page.recommendation,
            "report": "# Billing & Entitlements Report" in billing_report and "- Workspace: OpenAGI Revenue Cloud" in billing_report and "- Overage Cost (USD): 4.00" in billing_report and "- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked" in billing_report,
        },
        "html": "<title>Billing & Entitlements" in html_billing and "OpenAGI Revenue Cloud" in html_billing and "Premium plan for 2026-03" in html_billing and "Charge Feed" in html_billing and "premium-included" in html_billing,
        "ledger": {
            "run_count": ledger_billing.run_count,
            "recommendation": ledger_billing.recommendation,
            "overage_cost": ledger_billing.total_overage_cost_usd,
            "blocked_capabilities": ledger_billing.charges[1].blocked_capabilities,
            "handoff_team": ledger_billing.charges[1].handoff_team,
        },
    },
    "timestamps": {
        "feedback_z": feedback_record.timestamp.endswith("Z"),
        "feedback_tz": __import__("datetime").datetime.fromisoformat(feedback_record.timestamp.replace("Z", "+00:00")).utcoffset().total_seconds(),
        "validation_z": timestamp_value.endswith("Z"),
        "validation_tz": __import__("datetime").datetime.fromisoformat(timestamp_value.replace("Z", "+00:00")).utcoffset().total_seconds(),
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write reports contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run reports contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		RenderWriteReport map[string]bool `json:"render_write_report"`
		ConsoleAction     struct {
			Enabled  string `json:"enabled"`
			Disabled string `json:"disabled"`
		} `json:"console_action"`
		ReportStudio map[string]any `json:"report_studio"`
		DraftStudio  struct {
			Ready          bool   `json:"ready"`
			Recommendation string `json:"recommendation"`
		} `json:"draft_studio"`
		PilotScorecard struct {
			MetricsMet             int     `json:"metrics_met"`
			Recommendation         string  `json:"recommendation"`
			PaybackMonths          float64 `json:"payback_months"`
			Content                bool    `json:"content"`
			NegativeValue          int     `json:"negative_value"`
			NegativePayback        any     `json:"negative_payback"`
			NegativeRecommendation string  `json:"negative_recommendation"`
		} `json:"pilot_scorecard"`
		ClosureBasics struct {
			MissingAllowed bool   `json:"missing_allowed"`
			MissingReason  string `json:"missing_reason"`
			MissingExists  bool   `json:"missing_exists"`
			FailedAllowed  bool   `json:"failed_allowed"`
			FailedReason   string `json:"failed_reason"`
			SuccessAllowed bool   `json:"success_allowed"`
			SuccessReason  string `json:"success_reason"`
		} `json:"closure_basics"`
		LaunchChecklist struct {
			DocumentationStatus  map[string]bool `json:"documentation_status"`
			CompletedItems       int             `json:"completed_items"`
			MissingDocumentation []string        `json:"missing_documentation"`
			Ready                bool            `json:"ready"`
			Report               bool            `json:"report"`
		} `json:"launch_checklist"`
		FinalDelivery struct {
			RequiredOutputStatus map[string]bool `json:"required_output_status"`
			RecommendedStatus    map[string]bool `json:"recommended_status"`
			GeneratedRequired    int             `json:"generated_required"`
			GeneratedRecommended int             `json:"generated_recommended"`
			MissingRequired      []string        `json:"missing_required"`
			MissingRecommended   []string        `json:"missing_recommended"`
			Ready                bool            `json:"ready"`
			Report               bool            `json:"report"`
		} `json:"final_delivery"`
		ClosureChecklists struct {
			LaunchBlocked       bool   `json:"launch_blocked"`
			LaunchBlockedReason string `json:"launch_blocked_reason"`
			FinalBlocked        bool   `json:"final_blocked"`
			FinalBlockedReason  string `json:"final_blocked_reason"`
			FinalAllowed        bool   `json:"final_allowed"`
			FinalAllowedReason  string `json:"final_allowed_reason"`
			LaunchAllowed       bool   `json:"launch_allowed"`
			LaunchAllowedReason string `json:"launch_allowed_reason"`
		} `json:"closure_checklists"`
		PilotPortfolio struct {
			NetValue       int            `json:"net_value"`
			AverageROI     float64        `json:"average_roi"`
			Counts         map[string]int `json:"counts"`
			Recommendation string         `json:"recommendation"`
			Report         bool           `json:"report"`
		} `json:"pilot_portfolio"`
		SharedView struct {
			Collaboration bool `json:"collaboration"`
		} `json:"shared_view"`
		TriageCenter struct {
			FlaggedRuns          int            `json:"flagged_runs"`
			InboxSize            int            `json:"inbox_size"`
			SeverityCounts       map[string]int `json:"severity_counts"`
			OwnerCounts          map[string]int `json:"owner_counts"`
			Recommendation       string         `json:"recommendation"`
			Findings             []string       `json:"findings"`
			Inbox                []string       `json:"inbox"`
			SuggestionLabel      string         `json:"suggestion_label"`
			SuggestionConfidence float64        `json:"suggestion_confidence"`
			NextActions          []string       `json:"next_actions"`
			ActionStates         []bool         `json:"action_states"`
			Report               bool           `json:"report"`
			Partial              bool           `json:"partial"`
		} `json:"triage_center"`
		TriageFeedback struct {
			FeedbackCounts map[string]int `json:"feedback_counts"`
			BrowserStatus  string         `json:"browser_status"`
			ApprovalStatus string         `json:"approval_status"`
			RelatedRun     string         `json:"related_run"`
			Score          float64        `json:"score"`
			Report         bool           `json:"report"`
		} `json:"triage_feedback"`
		TakeoverQueue struct {
			PendingRequests int            `json:"pending_requests"`
			TeamCounts      map[string]int `json:"team_counts"`
			ApprovalCount   int            `json:"approval_count"`
			Recommendation  string         `json:"recommendation"`
			Requests        []string       `json:"requests"`
			ActionEnabled   bool           `json:"action_enabled"`
			ActionDisabled  bool           `json:"action_disabled"`
			Report          bool           `json:"report"`
			Error           bool           `json:"error"`
		} `json:"takeover_queue"`
		OrchestrationCanvas struct {
			Recommendation string   `json:"recommendation"`
			ActiveTools    []string `json:"active_tools"`
			Actions        []bool   `json:"actions"`
			Report         bool     `json:"report"`
			Flow           bool     `json:"flow"`
		} `json:"orchestration_canvas"`
		OrchestrationPortfolio struct {
			TotalRuns       int            `json:"total_runs"`
			Modes           map[string]int `json:"modes"`
			Tiers           map[string]int `json:"tiers"`
			Entitlements    map[string]int `json:"entitlements"`
			Billing         map[string]int `json:"billing"`
			EstimatedCost   float64        `json:"estimated_cost"`
			OverageCost     float64        `json:"overage_cost"`
			UpgradeRequired int            `json:"upgrade_required"`
			ActiveHandoffs  int            `json:"active_handoffs"`
			Recommendation  string         `json:"recommendation"`
			Report          bool           `json:"report"`
			Empty           bool           `json:"empty"`
			OverviewPage    bool           `json:"overview_page"`
		} `json:"orchestration_portfolio"`
		LedgerOrchestration struct {
			Canvas struct {
				RunID              string   `json:"run_id"`
				Mode               string   `json:"mode"`
				Departments        []string `json:"departments"`
				Approvals          []string `json:"approvals"`
				Tier               string   `json:"tier"`
				UpgradeRequired    bool     `json:"upgrade_required"`
				Entitlement        string   `json:"entitlement"`
				BillingModel       string   `json:"billing_model"`
				EstimatedCost      float64  `json:"estimated_cost"`
				Included           int      `json:"included"`
				OverageUnits       int      `json:"overage_units"`
				OverageCost        float64  `json:"overage_cost"`
				BlockedDepartments []string `json:"blocked_departments"`
				HandoffTeam        string   `json:"handoff_team"`
				ActiveTools        []string `json:"active_tools"`
				Actions            []bool   `json:"actions"`
			} `json:"canvas"`
			Portfolio struct {
				TotalRuns       int            `json:"total_runs"`
				Modes           map[string]int `json:"modes"`
				Tiers           map[string]int `json:"tiers"`
				Entitlements    map[string]int `json:"entitlements"`
				EstimatedCost   float64        `json:"estimated_cost"`
				PendingRequests int            `json:"pending_requests"`
				Recommendation  string         `json:"recommendation"`
			} `json:"portfolio"`
		} `json:"ledger_orchestration"`
		Billing struct {
			Page struct {
				RunCount            int            `json:"run_count"`
				IncludedUnits       int            `json:"included_units"`
				OverageUnits        int            `json:"overage_units"`
				EstimatedCost       float64        `json:"estimated_cost"`
				OverageCost         float64        `json:"overage_cost"`
				UpgradeRequired     int            `json:"upgrade_required"`
				EntitlementCounts   map[string]int `json:"entitlement_counts"`
				BillingCounts       map[string]int `json:"billing_counts"`
				BlockedCapabilities []string       `json:"blocked_capabilities"`
				Recommendation      string         `json:"recommendation"`
				Report              bool           `json:"report"`
			} `json:"page"`
			HTML   bool `json:"html"`
			Ledger struct {
				RunCount            int      `json:"run_count"`
				Recommendation      string   `json:"recommendation"`
				OverageCost         float64  `json:"overage_cost"`
				BlockedCapabilities []string `json:"blocked_capabilities"`
				HandoffTeam         string   `json:"handoff_team"`
			} `json:"ledger"`
		} `json:"billing"`
		Timestamps struct {
			FeedbackZ    bool    `json:"feedback_z"`
			FeedbackTZ   float64 `json:"feedback_tz"`
			ValidationZ  bool    `json:"validation_z"`
			ValidationTZ float64 `json:"validation_tz"`
		} `json:"timestamps"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode reports contract output: %v\n%s", err, string(output))
	}

	for name, ok := range decoded.RenderWriteReport {
		if !ok {
			t.Fatalf("expected render/write report check %s to pass", name)
		}
	}
	if decoded.ConsoleAction.Enabled != "enabled" || decoded.ConsoleAction.Disabled != "disabled" {
		t.Fatalf("unexpected console action payload: %+v", decoded.ConsoleAction)
	}
	if ready, _ := decoded.ReportStudio["ready"].(bool); !ready {
		t.Fatalf("expected report studio ready, got %+v", decoded.ReportStudio)
	}
	if decoded.DraftStudio.Ready || decoded.DraftStudio.Recommendation != "draft" {
		t.Fatalf("unexpected draft studio payload: %+v", decoded.DraftStudio)
	}
	if decoded.PilotScorecard.MetricsMet != 2 || decoded.PilotScorecard.Recommendation != "go" || decoded.PilotScorecard.PaybackMonths != 1.9 || !decoded.PilotScorecard.Content || decoded.PilotScorecard.NegativeValue != -2000 || decoded.PilotScorecard.NegativePayback != nil || decoded.PilotScorecard.NegativeRecommendation != "hold" {
		t.Fatalf("unexpected pilot scorecard payload: %+v", decoded.PilotScorecard)
	}
	if decoded.ClosureBasics.MissingAllowed || decoded.ClosureBasics.MissingReason != "validation report required before closing issue" || decoded.ClosureBasics.MissingExists || decoded.ClosureBasics.FailedAllowed || decoded.ClosureBasics.FailedReason != "validation failed; issue must remain open" || !decoded.ClosureBasics.SuccessAllowed || decoded.ClosureBasics.SuccessReason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected closure basics payload: %+v", decoded.ClosureBasics)
	}
	if !decoded.LaunchChecklist.DocumentationStatus["runbook"] || decoded.LaunchChecklist.DocumentationStatus["faq"] || decoded.LaunchChecklist.CompletedItems != 1 || len(decoded.LaunchChecklist.MissingDocumentation) != 1 || decoded.LaunchChecklist.MissingDocumentation[0] != "faq" || decoded.LaunchChecklist.Ready || !decoded.LaunchChecklist.Report {
		t.Fatalf("unexpected launch checklist payload: %+v", decoded.LaunchChecklist)
	}
	if !decoded.FinalDelivery.RequiredOutputStatus["validation-bundle"] || decoded.FinalDelivery.RequiredOutputStatus["release-notes"] || decoded.FinalDelivery.GeneratedRequired != 1 || decoded.FinalDelivery.GeneratedRecommended != 0 || len(decoded.FinalDelivery.MissingRequired) != 1 || decoded.FinalDelivery.MissingRequired[0] != "release-notes" || len(decoded.FinalDelivery.MissingRecommended) != 2 || decoded.FinalDelivery.Ready || !decoded.FinalDelivery.Report {
		t.Fatalf("unexpected final delivery payload: %+v", decoded.FinalDelivery)
	}
	if decoded.ClosureChecklists.LaunchBlocked || decoded.ClosureChecklists.LaunchBlockedReason != "launch checklist incomplete; linked documentation missing or empty" || decoded.ClosureChecklists.FinalBlocked || decoded.ClosureChecklists.FinalBlockedReason != "final delivery checklist incomplete; required outputs missing" || !decoded.ClosureChecklists.FinalAllowed || decoded.ClosureChecklists.FinalAllowedReason != "validation report and final delivery checklist requirements satisfied; issue can be closed" || !decoded.ClosureChecklists.LaunchAllowed || decoded.ClosureChecklists.LaunchAllowedReason != "validation report and launch checklist requirements satisfied; issue can be closed" {
		t.Fatalf("unexpected closure checklists payload: %+v", decoded.ClosureChecklists)
	}
	if decoded.PilotPortfolio.NetValue != 18500 || decoded.PilotPortfolio.AverageROI != 195.2 || decoded.PilotPortfolio.Counts["go"] != 1 || decoded.PilotPortfolio.Counts["iterate"] != 1 || decoded.PilotPortfolio.Counts["hold"] != 0 || decoded.PilotPortfolio.Recommendation != "continue" || !decoded.PilotPortfolio.Report {
		t.Fatalf("unexpected pilot portfolio payload: %+v", decoded.PilotPortfolio)
	}
	if !decoded.SharedView.Collaboration {
		t.Fatal("expected shared view collaboration rendering to pass")
	}
	if decoded.TriageCenter.FlaggedRuns != 2 || decoded.TriageCenter.InboxSize != 2 || decoded.TriageCenter.SeverityCounts["critical"] != 1 || decoded.TriageCenter.SeverityCounts["high"] != 1 || decoded.TriageCenter.OwnerCounts["security"] != 1 || decoded.TriageCenter.OwnerCounts["engineering"] != 1 || decoded.TriageCenter.Recommendation != "immediate-attention" || len(decoded.TriageCenter.Findings) != 2 || decoded.TriageCenter.Findings[0] != "run-browser" || decoded.TriageCenter.Findings[1] != "run-risk" || len(decoded.TriageCenter.Inbox) != 2 || decoded.TriageCenter.SuggestionLabel != "replay candidate" || decoded.TriageCenter.SuggestionConfidence < 0.55 || decoded.TriageCenter.NextActions[0] != "replay run and inspect tool failures" || decoded.TriageCenter.NextActions[1] != "request approval and queue security review" || !decoded.TriageCenter.ActionStates[0] || decoded.TriageCenter.ActionStates[1] || decoded.TriageCenter.ActionStates[2] || !decoded.TriageCenter.Report || !decoded.TriageCenter.Partial {
		t.Fatalf("unexpected triage center payload: %+v", decoded.TriageCenter)
	}
	if decoded.TriageFeedback.FeedbackCounts["accepted"] != 1 || decoded.TriageFeedback.FeedbackCounts["rejected"] != 1 || decoded.TriageFeedback.FeedbackCounts["pending"] != 1 || decoded.TriageFeedback.BrowserStatus != "accepted" || decoded.TriageFeedback.ApprovalStatus != "rejected" || decoded.TriageFeedback.RelatedRun != "run-browser-b" || decoded.TriageFeedback.Score < 0.8 || !decoded.TriageFeedback.Report {
		t.Fatalf("unexpected triage feedback payload: %+v", decoded.TriageFeedback)
	}
	if decoded.TakeoverQueue.PendingRequests != 2 || decoded.TakeoverQueue.TeamCounts["operations"] != 1 || decoded.TakeoverQueue.TeamCounts["security"] != 1 || decoded.TakeoverQueue.ApprovalCount != 2 || decoded.TakeoverQueue.Recommendation != "expedite-security-review" || len(decoded.TakeoverQueue.Requests) != 2 || decoded.TakeoverQueue.Requests[0] != "run-ops" || decoded.TakeoverQueue.Requests[1] != "run-sec" || !decoded.TakeoverQueue.ActionEnabled || decoded.TakeoverQueue.ActionDisabled || !decoded.TakeoverQueue.Report || !decoded.TakeoverQueue.Error {
		t.Fatalf("unexpected takeover queue payload: %+v", decoded.TakeoverQueue)
	}
	if decoded.OrchestrationCanvas.Recommendation != "resolve-entitlement-gap" || len(decoded.OrchestrationCanvas.ActiveTools) != 1 || decoded.OrchestrationCanvas.ActiveTools[0] != "browser" || !decoded.OrchestrationCanvas.Actions[0] || decoded.OrchestrationCanvas.Actions[1] || !decoded.OrchestrationCanvas.Report || !decoded.OrchestrationCanvas.Flow {
		t.Fatalf("unexpected orchestration canvas payload: %+v", decoded.OrchestrationCanvas)
	}
	if decoded.OrchestrationPortfolio.TotalRuns != 2 || decoded.OrchestrationPortfolio.Modes["cross-functional"] != 1 || decoded.OrchestrationPortfolio.Modes["tier-limited"] != 1 || decoded.OrchestrationPortfolio.Tiers["premium"] != 1 || decoded.OrchestrationPortfolio.Tiers["standard"] != 1 || decoded.OrchestrationPortfolio.Entitlements["included"] != 1 || decoded.OrchestrationPortfolio.Entitlements["upgrade-required"] != 1 || decoded.OrchestrationPortfolio.Billing["premium-included"] != 1 || decoded.OrchestrationPortfolio.Billing["standard-blocked"] != 1 || decoded.OrchestrationPortfolio.EstimatedCost != 11.5 || decoded.OrchestrationPortfolio.OverageCost != 4 || decoded.OrchestrationPortfolio.UpgradeRequired != 1 || decoded.OrchestrationPortfolio.ActiveHandoffs != 2 || decoded.OrchestrationPortfolio.Recommendation != "stabilize-security-takeovers" || !decoded.OrchestrationPortfolio.Report || !decoded.OrchestrationPortfolio.Empty || !decoded.OrchestrationPortfolio.OverviewPage {
		t.Fatalf("unexpected orchestration portfolio payload: %+v", decoded.OrchestrationPortfolio)
	}
	if decoded.LedgerOrchestration.Canvas.RunID != "run-ledger" || decoded.LedgerOrchestration.Canvas.Mode != "tier-limited" || len(decoded.LedgerOrchestration.Canvas.Departments) != 2 || decoded.LedgerOrchestration.Canvas.Approvals[0] != "security-review" || decoded.LedgerOrchestration.Canvas.Tier != "standard" || !decoded.LedgerOrchestration.Canvas.UpgradeRequired || decoded.LedgerOrchestration.Canvas.Entitlement != "upgrade-required" || decoded.LedgerOrchestration.Canvas.BillingModel != "standard-blocked" || decoded.LedgerOrchestration.Canvas.EstimatedCost != 7 || decoded.LedgerOrchestration.Canvas.Included != 2 || decoded.LedgerOrchestration.Canvas.OverageUnits != 1 || decoded.LedgerOrchestration.Canvas.OverageCost != 4 || len(decoded.LedgerOrchestration.Canvas.BlockedDepartments) != 2 || decoded.LedgerOrchestration.Canvas.HandoffTeam != "operations" || len(decoded.LedgerOrchestration.Canvas.ActiveTools) != 1 || decoded.LedgerOrchestration.Canvas.ActiveTools[0] != "browser" || !decoded.LedgerOrchestration.Canvas.Actions[0] || decoded.LedgerOrchestration.Canvas.Actions[1] {
		t.Fatalf("unexpected ledger canvas payload: %+v", decoded.LedgerOrchestration.Canvas)
	}
	if decoded.LedgerOrchestration.Portfolio.TotalRuns != 2 || decoded.LedgerOrchestration.Portfolio.Modes["cross-functional"] != 1 || decoded.LedgerOrchestration.Portfolio.Modes["tier-limited"] != 1 || decoded.LedgerOrchestration.Portfolio.Tiers["premium"] != 1 || decoded.LedgerOrchestration.Portfolio.Tiers["standard"] != 1 || decoded.LedgerOrchestration.Portfolio.Entitlements["included"] != 1 || decoded.LedgerOrchestration.Portfolio.Entitlements["upgrade-required"] != 1 || decoded.LedgerOrchestration.Portfolio.EstimatedCost != 11.5 || decoded.LedgerOrchestration.Portfolio.PendingRequests != 2 || decoded.LedgerOrchestration.Portfolio.Recommendation != "stabilize-security-takeovers" {
		t.Fatalf("unexpected ledger portfolio payload: %+v", decoded.LedgerOrchestration.Portfolio)
	}
	if decoded.Billing.Page.RunCount != 2 || decoded.Billing.Page.IncludedUnits != 5 || decoded.Billing.Page.OverageUnits != 1 || decoded.Billing.Page.EstimatedCost != 11.5 || decoded.Billing.Page.OverageCost != 4 || decoded.Billing.Page.UpgradeRequired != 1 || decoded.Billing.Page.EntitlementCounts["included"] != 1 || decoded.Billing.Page.EntitlementCounts["upgrade-required"] != 1 || decoded.Billing.Page.BillingCounts["premium-included"] != 1 || decoded.Billing.Page.BillingCounts["standard-blocked"] != 1 || len(decoded.Billing.Page.BlockedCapabilities) != 1 || decoded.Billing.Page.BlockedCapabilities[0] != "customer-success" || decoded.Billing.Page.Recommendation != "resolve-plan-gaps" || !decoded.Billing.Page.Report || !decoded.Billing.HTML || decoded.Billing.Ledger.RunCount != 2 || decoded.Billing.Ledger.Recommendation != "resolve-plan-gaps" || decoded.Billing.Ledger.OverageCost != 4 || len(decoded.Billing.Ledger.BlockedCapabilities) != 1 || decoded.Billing.Ledger.BlockedCapabilities[0] != "customer-success" || decoded.Billing.Ledger.HandoffTeam != "operations" {
		t.Fatalf("unexpected billing payload: %+v", decoded.Billing)
	}
	if !decoded.Timestamps.FeedbackZ || decoded.Timestamps.FeedbackTZ != 0 || !decoded.Timestamps.ValidationZ || decoded.Timestamps.ValidationTZ != 0 {
		t.Fatalf("unexpected timestamp payload: %+v", decoded.Timestamps)
	}
}
