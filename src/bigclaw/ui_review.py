from dataclasses import dataclass, field
from html import escape
from pathlib import Path
from typing import Dict, List


@dataclass(frozen=True)
class ReviewObjective:
    objective_id: str
    title: str
    persona: str
    outcome: str
    success_signal: str
    priority: str = "P1"
    dependencies: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "objective_id": self.objective_id,
            "title": self.title,
            "persona": self.persona,
            "outcome": self.outcome,
            "success_signal": self.success_signal,
            "priority": self.priority,
            "dependencies": list(self.dependencies),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewObjective":
        return cls(
            objective_id=str(data["objective_id"]),
            title=str(data["title"]),
            persona=str(data["persona"]),
            outcome=str(data["outcome"]),
            success_signal=str(data["success_signal"]),
            priority=str(data.get("priority", "P1")),
            dependencies=[str(item) for item in data.get("dependencies", [])],
        )


@dataclass(frozen=True)
class WireframeSurface:
    surface_id: str
    name: str
    device: str
    entry_point: str
    primary_blocks: List[str] = field(default_factory=list)
    review_notes: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_id": self.surface_id,
            "name": self.name,
            "device": self.device,
            "entry_point": self.entry_point,
            "primary_blocks": list(self.primary_blocks),
            "review_notes": list(self.review_notes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WireframeSurface":
        return cls(
            surface_id=str(data["surface_id"]),
            name=str(data["name"]),
            device=str(data["device"]),
            entry_point=str(data["entry_point"]),
            primary_blocks=[str(item) for item in data.get("primary_blocks", [])],
            review_notes=[str(item) for item in data.get("review_notes", [])],
        )


@dataclass(frozen=True)
class InteractionFlow:
    flow_id: str
    name: str
    trigger: str
    system_response: str
    states: List[str] = field(default_factory=list)
    exceptions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "flow_id": self.flow_id,
            "name": self.name,
            "trigger": self.trigger,
            "system_response": self.system_response,
            "states": list(self.states),
            "exceptions": list(self.exceptions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InteractionFlow":
        return cls(
            flow_id=str(data["flow_id"]),
            name=str(data["name"]),
            trigger=str(data["trigger"]),
            system_response=str(data["system_response"]),
            states=[str(item) for item in data.get("states", [])],
            exceptions=[str(item) for item in data.get("exceptions", [])],
        )


@dataclass(frozen=True)
class OpenQuestion:
    question_id: str
    theme: str
    question: str
    owner: str
    impact: str
    status: str = "open"

    def to_dict(self) -> Dict[str, object]:
        return {
            "question_id": self.question_id,
            "theme": self.theme,
            "question": self.question,
            "owner": self.owner,
            "impact": self.impact,
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "OpenQuestion":
        return cls(
            question_id=str(data["question_id"]),
            theme=str(data["theme"]),
            question=str(data["question"]),
            owner=str(data["owner"]),
            impact=str(data["impact"]),
            status=str(data.get("status", "open")),
        )


@dataclass(frozen=True)
class ReviewerChecklistItem:
    item_id: str
    surface_id: str
    prompt: str
    owner: str
    status: str = "todo"
    evidence_links: List[str] = field(default_factory=list)
    notes: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "item_id": self.item_id,
            "surface_id": self.surface_id,
            "prompt": self.prompt,
            "owner": self.owner,
            "status": self.status,
            "evidence_links": list(self.evidence_links),
            "notes": self.notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewerChecklistItem":
        return cls(
            item_id=str(data["item_id"]),
            surface_id=str(data["surface_id"]),
            prompt=str(data["prompt"]),
            owner=str(data["owner"]),
            status=str(data.get("status", "todo")),
            evidence_links=[str(item) for item in data.get("evidence_links", [])],
            notes=str(data.get("notes", "")),
        )


@dataclass(frozen=True)
class ReviewDecision:
    decision_id: str
    surface_id: str
    owner: str
    summary: str
    rationale: str
    status: str = "proposed"
    follow_up: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "decision_id": self.decision_id,
            "surface_id": self.surface_id,
            "owner": self.owner,
            "summary": self.summary,
            "rationale": self.rationale,
            "status": self.status,
            "follow_up": self.follow_up,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewDecision":
        return cls(
            decision_id=str(data["decision_id"]),
            surface_id=str(data["surface_id"]),
            owner=str(data["owner"]),
            summary=str(data["summary"]),
            rationale=str(data["rationale"]),
            status=str(data.get("status", "proposed")),
            follow_up=str(data.get("follow_up", "")),
        )


@dataclass(frozen=True)
class UIReviewPackArtifacts:
    root_dir: str
    markdown_path: str
    html_path: str
    decision_log_path: str


@dataclass
class UIReviewPack:
    issue_id: str
    title: str
    version: str
    objectives: List[ReviewObjective] = field(default_factory=list)
    wireframes: List[WireframeSurface] = field(default_factory=list)
    interactions: List[InteractionFlow] = field(default_factory=list)
    open_questions: List[OpenQuestion] = field(default_factory=list)
    reviewer_checklist: List[ReviewerChecklistItem] = field(default_factory=list)
    requires_reviewer_checklist: bool = False
    decision_log: List[ReviewDecision] = field(default_factory=list)
    requires_decision_log: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "version": self.version,
            "objectives": [objective.to_dict() for objective in self.objectives],
            "wireframes": [wireframe.to_dict() for wireframe in self.wireframes],
            "interactions": [interaction.to_dict() for interaction in self.interactions],
            "open_questions": [question.to_dict() for question in self.open_questions],
            "reviewer_checklist": [item.to_dict() for item in self.reviewer_checklist],
            "requires_reviewer_checklist": self.requires_reviewer_checklist,
            "decision_log": [decision.to_dict() for decision in self.decision_log],
            "requires_decision_log": self.requires_decision_log,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIReviewPack":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            objectives=[ReviewObjective.from_dict(item) for item in data.get("objectives", [])],
            wireframes=[WireframeSurface.from_dict(item) for item in data.get("wireframes", [])],
            interactions=[InteractionFlow.from_dict(item) for item in data.get("interactions", [])],
            open_questions=[OpenQuestion.from_dict(item) for item in data.get("open_questions", [])],
            reviewer_checklist=[ReviewerChecklistItem.from_dict(item) for item in data.get("reviewer_checklist", [])],
            requires_reviewer_checklist=bool(data.get("requires_reviewer_checklist", False)),
            decision_log=[ReviewDecision.from_dict(item) for item in data.get("decision_log", [])],
            requires_decision_log=bool(data.get("requires_decision_log", False)),
        )


@dataclass(frozen=True)
class UIReviewPackAudit:
    ready: bool
    objective_count: int
    wireframe_count: int
    interaction_count: int
    open_question_count: int
    checklist_count: int = 0
    decision_count: int = 0
    missing_sections: List[str] = field(default_factory=list)
    objectives_missing_signals: List[str] = field(default_factory=list)
    wireframes_missing_blocks: List[str] = field(default_factory=list)
    interactions_missing_states: List[str] = field(default_factory=list)
    unresolved_question_ids: List[str] = field(default_factory=list)
    wireframes_missing_checklists: List[str] = field(default_factory=list)
    orphan_checklist_surfaces: List[str] = field(default_factory=list)
    checklist_items_missing_evidence: List[str] = field(default_factory=list)
    wireframes_missing_decisions: List[str] = field(default_factory=list)
    orphan_decision_surfaces: List[str] = field(default_factory=list)
    unresolved_decision_ids: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: objectives={self.objective_count} "
            f"wireframes={self.wireframe_count} "
            f"interactions={self.interaction_count} "
            f"open_questions={self.open_question_count} "
            f"checklist={self.checklist_count} "
            f"decisions={self.decision_count}"
        )


class UIReviewPackAuditor:
    def audit(self, pack: UIReviewPack) -> UIReviewPackAudit:
        missing_sections = []
        if not pack.objectives:
            missing_sections.append("objectives")
        if not pack.wireframes:
            missing_sections.append("wireframes")
        if not pack.interactions:
            missing_sections.append("interactions")
        if not pack.open_questions:
            missing_sections.append("open_questions")

        objectives_missing_signals = [
            objective.objective_id
            for objective in pack.objectives
            if not objective.success_signal.strip()
        ]
        wireframes_missing_blocks = [
            wireframe.surface_id
            for wireframe in pack.wireframes
            if not wireframe.primary_blocks
        ]
        interactions_missing_states = [
            interaction.flow_id
            for interaction in pack.interactions
            if not interaction.states
        ]
        unresolved_question_ids = [
            question.question_id
            for question in pack.open_questions
            if question.status.lower() != "resolved"
        ]
        wireframe_ids = {wireframe.surface_id for wireframe in pack.wireframes}
        checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
        for item in pack.reviewer_checklist:
            checklist_by_surface.setdefault(item.surface_id, []).append(item)
        wireframes_missing_checklists = []
        orphan_checklist_surfaces = []
        checklist_items_missing_evidence = []
        if pack.requires_reviewer_checklist:
            wireframes_missing_checklists = sorted(
                wireframe.surface_id for wireframe in pack.wireframes if wireframe.surface_id not in checklist_by_surface
            )
            orphan_checklist_surfaces = sorted(
                surface_id for surface_id in checklist_by_surface if surface_id not in wireframe_ids
            )
            checklist_items_missing_evidence = sorted(
                item.item_id for item in pack.reviewer_checklist if not item.evidence_links
            )
        decision_by_surface: Dict[str, List[ReviewDecision]] = {}
        for decision in pack.decision_log:
            decision_by_surface.setdefault(decision.surface_id, []).append(decision)
        wireframes_missing_decisions = []
        orphan_decision_surfaces = []
        unresolved_decision_ids = []
        if pack.requires_decision_log:
            wireframes_missing_decisions = sorted(
                wireframe.surface_id for wireframe in pack.wireframes if wireframe.surface_id not in decision_by_surface
            )
            orphan_decision_surfaces = sorted(
                surface_id for surface_id in decision_by_surface if surface_id not in wireframe_ids
            )
            unresolved_decision_ids = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.status.lower() not in {"accepted", "approved", "resolved", "deferred"}
            )
        ready = not (
            missing_sections
            or objectives_missing_signals
            or wireframes_missing_blocks
            or interactions_missing_states
            or wireframes_missing_checklists
            or orphan_checklist_surfaces
            or checklist_items_missing_evidence
            or wireframes_missing_decisions
            or orphan_decision_surfaces
        )
        return UIReviewPackAudit(
            ready=ready,
            objective_count=len(pack.objectives),
            wireframe_count=len(pack.wireframes),
            interaction_count=len(pack.interactions),
            open_question_count=len(pack.open_questions),
            checklist_count=len(pack.reviewer_checklist),
            decision_count=len(pack.decision_log),
            missing_sections=missing_sections,
            objectives_missing_signals=objectives_missing_signals,
            wireframes_missing_blocks=wireframes_missing_blocks,
            interactions_missing_states=interactions_missing_states,
            unresolved_question_ids=unresolved_question_ids,
            wireframes_missing_checklists=wireframes_missing_checklists,
            orphan_checklist_surfaces=orphan_checklist_surfaces,
            checklist_items_missing_evidence=checklist_items_missing_evidence,
            wireframes_missing_decisions=wireframes_missing_decisions,
            orphan_decision_surfaces=orphan_decision_surfaces,
            unresolved_decision_ids=unresolved_decision_ids,
        )


def render_ui_review_pack_report(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:
    lines = [
        "# UI Review Pack",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Audit: {audit.summary}",
        "",
        "## Objectives",
    ]
    for objective in pack.objectives:
        lines.append(
            "- "
            f"{objective.objective_id}: {objective.title} "
            f"persona={objective.persona} priority={objective.priority}"
        )
        lines.append(
            "  "
            f"outcome={objective.outcome} success_signal={objective.success_signal} "
            f"dependencies={','.join(objective.dependencies) or 'none'}"
        )

    lines.append("")
    lines.append("## Wireframes")
    for wireframe in pack.wireframes:
        lines.append(
            "- "
            f"{wireframe.surface_id}: {wireframe.name} "
            f"device={wireframe.device} entry={wireframe.entry_point}"
        )
        lines.append(
            "  "
            f"blocks={','.join(wireframe.primary_blocks) or 'none'} "
            f"review_notes={','.join(wireframe.review_notes) or 'none'}"
        )

    lines.append("")
    lines.append("## Interactions")
    for interaction in pack.interactions:
        lines.append(
            "- "
            f"{interaction.flow_id}: {interaction.name} trigger={interaction.trigger}"
        )
        lines.append(
            "  "
            f"response={interaction.system_response} "
            f"states={','.join(interaction.states) or 'none'} "
            f"exceptions={','.join(interaction.exceptions) or 'none'}"
        )

    lines.append("")
    lines.append("## Open Questions")
    for question in pack.open_questions:
        lines.append(
            "- "
            f"{question.question_id}: {question.theme} owner={question.owner} "
            f"status={question.status}"
        )
        lines.append("  " f"question={question.question} impact={question.impact}")

    lines.append("")
    lines.append("## Reviewer Checklist")
    for item in pack.reviewer_checklist:
        lines.append(
            "- "
            f"{item.item_id}: surface={item.surface_id} owner={item.owner} status={item.status}"
        )
        lines.append(
            "  "
            f"prompt={item.prompt} evidence={','.join(item.evidence_links) or 'none'} notes={item.notes or 'none'}"
        )
    if not pack.reviewer_checklist:
        lines.append("- none")

    lines.append("")
    lines.append("## Decision Log")
    for decision in pack.decision_log:
        lines.append(
            "- "
            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"
        )
        lines.append(
            "  "
            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or 'none'}"
        )
    if not pack.decision_log:
        lines.append("- none")

    lines.extend(
        [
            "",
            "## Audit Findings",
            f"- Missing sections: {', '.join(audit.missing_sections) or 'none'}",
            f"- Objectives missing success signals: {', '.join(audit.objectives_missing_signals) or 'none'}",
            f"- Wireframes missing blocks: {', '.join(audit.wireframes_missing_blocks) or 'none'}",
            f"- Interactions missing states: {', '.join(audit.interactions_missing_states) or 'none'}",
            f"- Unresolved questions: {', '.join(audit.unresolved_question_ids) or 'none'}",
            f"- Wireframes missing checklist coverage: {', '.join(audit.wireframes_missing_checklists) or 'none'}",
            f"- Orphan checklist surfaces: {', '.join(audit.orphan_checklist_surfaces) or 'none'}",
            f"- Checklist items missing evidence: {', '.join(audit.checklist_items_missing_evidence) or 'none'}",
            f"- Wireframes missing decision coverage: {', '.join(audit.wireframes_missing_decisions) or 'none'}",
            f"- Orphan decision surfaces: {', '.join(audit.orphan_decision_surfaces) or 'none'}",
            f"- Unresolved decision ids: {', '.join(audit.unresolved_decision_ids) or 'none'}",
        ]
    )
    return "\n".join(lines)


def build_big_4204_review_pack() -> UIReviewPack:
    return UIReviewPack(
        issue_id="BIG-4204",
        title="UI评审包输出",
        version="v4.0-design-sprint",
        requires_reviewer_checklist=True,
        requires_decision_log=True,
        objectives=[
            ReviewObjective(
                objective_id="obj-overview-decision",
                title="Validate the executive overview narrative and drill-down posture",
                persona="VP Eng",
                outcome="Leadership can confirm the overview page balances KPI density with investigation entry points.",
                success_signal="Reviewers agree the overview supports release, risk, and queue drill-down without extra walkthroughs.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-132"],
            ),
            ReviewObjective(
                objective_id="obj-queue-governance",
                title="Confirm queue control actions and approval posture",
                persona="Platform Admin",
                outcome="Operators can assess batch approvals, audit visibility, and denial paths from one frame.",
                success_signal="The queue frame clearly shows allowed actions, denied roles, and audit expectations.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-131", "OPE-132"],
            ),
            ReviewObjective(
                objective_id="obj-run-detail-investigation",
                title="Validate replay and audit investigation flow",
                persona="Eng Lead",
                outcome="Run detail reviewers can trace evidence, replay context, and escalation actions in one surface.",
                success_signal="The run-detail frame makes failure replay and escalation decisions reviewable without hidden dependencies.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-72", "OPE-73"],
            ),
            ReviewObjective(
                objective_id="obj-triage-handoff",
                title="Confirm triage and cross-team handoff readiness",
                persona="Cross-Team Operator",
                outcome="Reviewers can evaluate assignment, handoff, and queue-state transitions as one operator journey.",
                success_signal="The triage frame exposes action states, owner switches, and handoff exceptions explicitly.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-76", "OPE-79", "OPE-132"],
            ),
        ],
        wireframes=[
            WireframeSurface(
                surface_id="wf-overview",
                name="Overview command deck",
                device="desktop",
                entry_point="/overview",
                primary_blocks=["top bar", "kpi strip", "risk panel", "drill-down table"],
                review_notes=["Confirm metric density and executive scan path.", "Check alert prominence versus weekly summary card."],
            ),
            WireframeSurface(
                surface_id="wf-queue",
                name="Queue control center",
                device="desktop",
                entry_point="/queue",
                primary_blocks=["approval queue", "selection toolbar", "filters", "audit rail"],
                review_notes=["Validate batch-approve CTA hierarchy.", "Review denied-role behavior for non-operator personas."],
            ),
            WireframeSurface(
                surface_id="wf-run-detail",
                name="Run detail and replay",
                device="desktop",
                entry_point="/runs/detail",
                primary_blocks=["timeline", "artifact drawer", "replay controls", "audit notes"],
                review_notes=["Check replay mode discoverability.", "Ensure escalation path is visible next to audit evidence."],
            ),
            WireframeSurface(
                surface_id="wf-triage",
                name="Triage and handoff board",
                device="desktop",
                entry_point="/triage",
                primary_blocks=["severity lanes", "bulk actions", "handoff panel", "ownership history"],
                review_notes=["Validate cross-team operator workflow.", "Confirm exception path for denied escalation."],
            ),
        ],
        interactions=[
            InteractionFlow(
                flow_id="flow-overview-drilldown",
                name="Overview to investigation drill-down",
                trigger="VP Eng selects a KPI card or blocker cluster on the overview page",
                system_response="The console pivots into the matching queue or run-detail slice while preserving context filters.",
                states=["default", "focus", "handoff-ready"],
                exceptions=["Warn when the requested slice is permission-denied.", "Show fallback summary when no matching runs exist."],
            ),
            InteractionFlow(
                flow_id="flow-queue-bulk-approval",
                name="Queue batch approval review",
                trigger="Platform Admin selects multiple tasks and opens the bulk approval toolbar",
                system_response="The queue shows approval scope, audit consequence, and denied-role messaging before submit.",
                states=["default", "selection", "confirming", "success"],
                exceptions=["Disable submit when tasks cross unauthorized scopes.", "Route to audit timeline when approval policy changes mid-flow."],
            ),
            InteractionFlow(
                flow_id="flow-run-replay",
                name="Run replay with evidence audit",
                trigger="Eng Lead switches replay mode on a failed run",
                system_response="The page updates the timeline, artifacts, and escalation controls while keeping the audit trail visible.",
                states=["default", "replay", "compare", "escalated"],
                exceptions=["Show replay-unavailable state for incomplete artifacts.", "Require escalation reason before handoff."],
            ),
            InteractionFlow(
                flow_id="flow-triage-handoff",
                name="Triage ownership reassignment and handoff",
                trigger="Cross-Team Operator bulk-assigns a finding set or opens the handoff panel",
                system_response="The triage board updates owner, workflow, and handoff evidence in one acknowledgement step.",
                states=["default", "selected", "handoff", "completed"],
                exceptions=["Block handoff when reviewer coverage is incomplete.", "Record denied-role attempt in the audit summary."],
            ),
        ],
        open_questions=[
            OpenQuestion(
                question_id="oq-role-density",
                theme="role-matrix",
                question="Should VP Eng see queue batch controls in read-only form or be routed to a summary-only state?",
                owner="product-experience",
                impact="Changes denial-path copy, button placement, and review criteria for queue and triage pages.",
            ),
            OpenQuestion(
                question_id="oq-alert-priority",
                theme="information-architecture",
                question="Should regression alerts outrank approval alerts in the top bar for the design sprint prototype?",
                owner="engineering-operations",
                impact="Affects alert hierarchy and the scan path used in the overview and triage reviews.",
            ),
            OpenQuestion(
                question_id="oq-handoff-evidence",
                theme="handoff",
                question="How much ownership history must stay visible before the run-detail and triage pages collapse older audit entries?",
                owner="orchestration-office",
                impact="Shapes the default density of the audit rail and the threshold for the review-ready packet.",
            ),
        ],
        reviewer_checklist=[
            ReviewerChecklistItem(
                item_id="chk-overview-kpi-scan",
                surface_id="wf-overview",
                prompt="Verify the KPI strip still supports one-screen executive scanning before drill-down.",
                owner="VP Eng",
                status="ready",
                evidence_links=["wf-overview", "flow-overview-drilldown"],
                notes="Use the overview card hierarchy as the primary decision frame.",
            ),
            ReviewerChecklistItem(
                item_id="chk-overview-alert-hierarchy",
                surface_id="wf-overview",
                prompt="Confirm alert priority is readable when approvals and regressions compete for attention.",
                owner="engineering-operations",
                status="open",
                evidence_links=["wf-overview", "oq-alert-priority"],
            ),
            ReviewerChecklistItem(
                item_id="chk-queue-batch-approval",
                surface_id="wf-queue",
                prompt="Check that batch approval clearly communicates scope, denial paths, and audit consequences.",
                owner="Platform Admin",
                status="ready",
                evidence_links=["wf-queue", "flow-queue-bulk-approval"],
            ),
            ReviewerChecklistItem(
                item_id="chk-queue-role-density",
                surface_id="wf-queue",
                prompt="Validate whether VP Eng should get a summary-only queue variant instead of operator controls.",
                owner="product-experience",
                status="open",
                evidence_links=["wf-queue", "oq-role-density"],
            ),
            ReviewerChecklistItem(
                item_id="chk-run-replay-context",
                surface_id="wf-run-detail",
                prompt="Ensure replay, compare, and escalation states remain distinguishable without narration.",
                owner="Eng Lead",
                status="ready",
                evidence_links=["wf-run-detail", "flow-run-replay"],
            ),
            ReviewerChecklistItem(
                item_id="chk-run-audit-density",
                surface_id="wf-run-detail",
                prompt="Confirm the audit rail retains enough ownership history before collapsing older entries.",
                owner="orchestration-office",
                status="open",
                evidence_links=["wf-run-detail", "oq-handoff-evidence"],
            ),
            ReviewerChecklistItem(
                item_id="chk-triage-handoff-clarity",
                surface_id="wf-triage",
                prompt="Check that cross-team handoff consequences are explicit before ownership changes commit.",
                owner="Cross-Team Operator",
                status="ready",
                evidence_links=["wf-triage", "flow-triage-handoff"],
            ),
            ReviewerChecklistItem(
                item_id="chk-triage-bulk-assign",
                surface_id="wf-triage",
                prompt="Validate bulk assignment visibility without burying the audit context.",
                owner="Platform Admin",
                status="ready",
                evidence_links=["wf-triage", "flow-triage-handoff"],
            ),
        ],
        decision_log=[
            ReviewDecision(
                decision_id="dec-overview-alert-stack",
                surface_id="wf-overview",
                owner="product-experience",
                summary="Keep approval and regression alerts in one stacked priority rail.",
                rationale="Reviewers need one comparison lane before jumping into queue or triage surfaces.",
                status="accepted",
            ),
            ReviewDecision(
                decision_id="dec-queue-vp-summary",
                surface_id="wf-queue",
                owner="VP Eng",
                summary="Route VP Eng to a summary-first queue state instead of operator controls.",
                rationale="The VP Eng persona needs queue visibility without accidental action affordances.",
                status="proposed",
                follow_up="Resolve after the next design critique with policy owners.",
            ),
            ReviewDecision(
                decision_id="dec-run-detail-audit-rail",
                surface_id="wf-run-detail",
                owner="Eng Lead",
                summary="Keep audit evidence visible beside replay controls in all replay states.",
                rationale="Replay decisions are inseparable from audit context and escalation evidence.",
                status="accepted",
            ),
            ReviewDecision(
                decision_id="dec-triage-handoff-density",
                surface_id="wf-triage",
                owner="Cross-Team Operator",
                summary="Preserve ownership history in the triage rail until handoff is acknowledged.",
                rationale="Operators need a stable handoff trail before collapsing older events.",
                status="accepted",
            ),
        ],
    )



def render_ui_review_decision_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Decision Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Decisions: {len(pack.decision_log)}",
        "",
        "## Decisions",
    ]
    for decision in pack.decision_log:
        lines.append(
            "- "
            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"
        )
        lines.append(
            "  "
            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or 'none'}"
        )
    if not pack.decision_log:
        lines.append("- none")
    return "\n".join(lines)



def render_ui_review_pack_html(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:
    objective_html = "".join(
        f"<li><strong>{escape(objective.objective_id)}</strong> · {escape(objective.title)} · persona={escape(objective.persona)} · priority={escape(objective.priority)}<br /><span>{escape(objective.success_signal)}</span></li>"
        for objective in pack.objectives
    ) or "<li>none</li>"
    wireframe_html = "".join(
        f"<li><strong>{escape(wireframe.surface_id)}</strong> · {escape(wireframe.name)} · entry={escape(wireframe.entry_point)}<br /><span>blocks={escape(', '.join(wireframe.primary_blocks) if wireframe.primary_blocks else 'none')}</span></li>"
        for wireframe in pack.wireframes
    ) or "<li>none</li>"
    interaction_html = "".join(
        f"<li><strong>{escape(interaction.flow_id)}</strong> · {escape(interaction.name)}<br /><span>states={escape(', '.join(interaction.states) if interaction.states else 'none')}</span></li>"
        for interaction in pack.interactions
    ) or "<li>none</li>"
    question_html = "".join(
        f"<li><strong>{escape(question.question_id)}</strong> · {escape(question.theme)} · owner={escape(question.owner)} · status={escape(question.status)}<br /><span>{escape(question.question)}</span></li>"
        for question in pack.open_questions
    ) or "<li>none</li>"
    checklist_html = "".join(
        f"<li><strong>{escape(item.item_id)}</strong> · surface={escape(item.surface_id)} · owner={escape(item.owner)} · status={escape(item.status)}<br /><span>{escape(item.prompt)}</span><br /><span>evidence={escape(', '.join(item.evidence_links) if item.evidence_links else 'none')}</span></li>"
        for item in pack.reviewer_checklist
    ) or "<li>none</li>"
    decision_html = "".join(
        f"<li><strong>{escape(decision.decision_id)}</strong> · surface={escape(decision.surface_id)} · owner={escape(decision.owner)} · status={escape(decision.status)}<br /><span>{escape(decision.summary)}</span><br /><span>follow_up={escape(decision.follow_up or 'none')}</span></li>"
        for decision in pack.decision_log
    ) or "<li>none</li>"
    return f"""<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{escape(pack.title)}</title>
    <style>
      body {{ font-family: 'Inter', 'Segoe UI', sans-serif; margin: 40px auto; max-width: 960px; color: #17202a; line-height: 1.6; }}
      h1, h2 {{ color: #0f172a; }}
      .meta {{ color: #52606d; font-size: 0.95rem; }}
      .surface {{ margin-top: 24px; padding: 16px 18px; border: 1px solid #d9e2ec; border-radius: 12px; background: #f8fafc; }}
      ul {{ padding-left: 20px; }}
      .summary {{ padding: 18px 20px; background: #eff6ff; border-left: 4px solid #2563eb; }}
    </style>
  </head>
  <body>
    <header>
      <p class="meta">{escape(pack.issue_id)} · {escape(pack.version)}</p>
      <h1>{escape(pack.title)}</h1>
      <p class="meta">Audit: {escape(audit.summary)}</p>
    </header>
    <section class="summary">
      <h2>Readiness</h2>
      <p>Missing checklist coverage: {escape(', '.join(audit.wireframes_missing_checklists) if audit.wireframes_missing_checklists else 'none')}</p>
      <p>Missing decision coverage: {escape(', '.join(audit.wireframes_missing_decisions) if audit.wireframes_missing_decisions else 'none')}</p>
      <p>Unresolved decisions: {escape(', '.join(audit.unresolved_decision_ids) if audit.unresolved_decision_ids else 'none')}</p>
    </section>
    <section class="surface"><h2>Objectives</h2><ul>{objective_html}</ul></section>
    <section class="surface"><h2>Wireframes</h2><ul>{wireframe_html}</ul></section>
    <section class="surface"><h2>Interactions</h2><ul>{interaction_html}</ul></section>
    <section class="surface"><h2>Open Questions</h2><ul>{question_html}</ul></section>
    <section class="surface"><h2>Reviewer Checklist</h2><ul>{checklist_html}</ul></section>
    <section class="surface"><h2>Decision Log</h2><ul>{decision_html}</ul></section>
  </body>
</html>
"""



def write_ui_review_pack_bundle(root_dir: str, pack: UIReviewPack) -> UIReviewPackArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    slug = pack.issue_id.lower().replace(" ", "-")
    markdown_path = str(base / f"{slug}-review-pack.md")
    html_path = str(base / f"{slug}-review-pack.html")
    decision_log_path = str(base / f"{slug}-decision-log.md")
    audit = UIReviewPackAuditor().audit(pack)
    Path(markdown_path).write_text(render_ui_review_pack_report(pack, audit))
    Path(html_path).write_text(render_ui_review_pack_html(pack, audit))
    Path(decision_log_path).write_text(render_ui_review_decision_log(pack))
    return UIReviewPackArtifacts(
        root_dir=str(base),
        markdown_path=markdown_path,
        html_path=html_path,
        decision_log_path=decision_log_path,
    )
