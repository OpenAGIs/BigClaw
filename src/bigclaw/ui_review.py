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
class ReviewRoleAssignment:
    assignment_id: str
    surface_id: str
    role: str
    responsibilities: List[str] = field(default_factory=list)
    checklist_item_ids: List[str] = field(default_factory=list)
    decision_ids: List[str] = field(default_factory=list)
    status: str = "planned"

    def to_dict(self) -> Dict[str, object]:
        return {
            "assignment_id": self.assignment_id,
            "surface_id": self.surface_id,
            "role": self.role,
            "responsibilities": list(self.responsibilities),
            "checklist_item_ids": list(self.checklist_item_ids),
            "decision_ids": list(self.decision_ids),
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewRoleAssignment":
        return cls(
            assignment_id=str(data["assignment_id"]),
            surface_id=str(data["surface_id"]),
            role=str(data["role"]),
            responsibilities=[str(item) for item in data.get("responsibilities", [])],
            checklist_item_ids=[str(item) for item in data.get("checklist_item_ids", [])],
            decision_ids=[str(item) for item in data.get("decision_ids", [])],
            status=str(data.get("status", "planned")),
        )


@dataclass(frozen=True)
class ReviewSignoff:
    signoff_id: str
    assignment_id: str
    surface_id: str
    role: str
    status: str = "pending"
    required: bool = True
    evidence_links: List[str] = field(default_factory=list)
    notes: str = ""
    waiver_owner: str = ""
    waiver_reason: str = ""
    requested_at: str = ""
    due_at: str = ""
    escalation_owner: str = ""
    sla_status: str = "on-track"
    reminder_owner: str = ""
    reminder_channel: str = ""
    last_reminder_at: str = ""
    next_reminder_at: str = ""
    reminder_cadence: str = ""
    reminder_status: str = "scheduled"

    def to_dict(self) -> Dict[str, object]:
        return {
            "signoff_id": self.signoff_id,
            "assignment_id": self.assignment_id,
            "surface_id": self.surface_id,
            "role": self.role,
            "status": self.status,
            "required": self.required,
            "evidence_links": list(self.evidence_links),
            "notes": self.notes,
            "waiver_owner": self.waiver_owner,
            "waiver_reason": self.waiver_reason,
            "requested_at": self.requested_at,
            "due_at": self.due_at,
            "escalation_owner": self.escalation_owner,
            "sla_status": self.sla_status,
            "reminder_owner": self.reminder_owner,
            "reminder_channel": self.reminder_channel,
            "last_reminder_at": self.last_reminder_at,
            "next_reminder_at": self.next_reminder_at,
            "reminder_cadence": self.reminder_cadence,
            "reminder_status": self.reminder_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewSignoff":
        return cls(
            signoff_id=str(data["signoff_id"]),
            assignment_id=str(data["assignment_id"]),
            surface_id=str(data["surface_id"]),
            role=str(data["role"]),
            status=str(data.get("status", "pending")),
            required=bool(data.get("required", True)),
            evidence_links=[str(item) for item in data.get("evidence_links", [])],
            notes=str(data.get("notes", "")),
            waiver_owner=str(data.get("waiver_owner", "")),
            waiver_reason=str(data.get("waiver_reason", "")),
            requested_at=str(data.get("requested_at", "")),
            due_at=str(data.get("due_at", "")),
            escalation_owner=str(data.get("escalation_owner", "")),
            sla_status=str(data.get("sla_status", "on-track")),
            reminder_owner=str(data.get("reminder_owner", "")),
            reminder_channel=str(data.get("reminder_channel", "")),
            last_reminder_at=str(data.get("last_reminder_at", "")),
            next_reminder_at=str(data.get("next_reminder_at", "")),
            reminder_cadence=str(data.get("reminder_cadence", "")),
            reminder_status=str(data.get("reminder_status", "scheduled")),
        )


@dataclass(frozen=True)
class ReviewBlocker:
    blocker_id: str
    surface_id: str
    signoff_id: str
    owner: str
    summary: str
    status: str = "open"
    severity: str = "medium"
    escalation_owner: str = ""
    next_action: str = ""
    freeze_exception: bool = False
    freeze_owner: str = ""
    freeze_until: str = ""
    freeze_reason: str = ""
    freeze_approved_by: str = ""
    freeze_approved_at: str = ""
    freeze_renewal_owner: str = ""
    freeze_renewal_by: str = ""
    freeze_renewal_status: str = "not-needed"

    def to_dict(self) -> Dict[str, object]:
        return {
            "blocker_id": self.blocker_id,
            "surface_id": self.surface_id,
            "signoff_id": self.signoff_id,
            "owner": self.owner,
            "summary": self.summary,
            "status": self.status,
            "severity": self.severity,
            "escalation_owner": self.escalation_owner,
            "next_action": self.next_action,
            "freeze_exception": self.freeze_exception,
            "freeze_owner": self.freeze_owner,
            "freeze_until": self.freeze_until,
            "freeze_reason": self.freeze_reason,
            "freeze_approved_by": self.freeze_approved_by,
            "freeze_approved_at": self.freeze_approved_at,
            "freeze_renewal_owner": self.freeze_renewal_owner,
            "freeze_renewal_by": self.freeze_renewal_by,
            "freeze_renewal_status": self.freeze_renewal_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlocker":
        return cls(
            blocker_id=str(data["blocker_id"]),
            surface_id=str(data["surface_id"]),
            signoff_id=str(data["signoff_id"]),
            owner=str(data["owner"]),
            summary=str(data["summary"]),
            status=str(data.get("status", "open")),
            severity=str(data.get("severity", "medium")),
            escalation_owner=str(data.get("escalation_owner", "")),
            next_action=str(data.get("next_action", "")),
            freeze_exception=bool(data.get("freeze_exception", False)),
            freeze_owner=str(data.get("freeze_owner", "")),
            freeze_until=str(data.get("freeze_until", "")),
            freeze_reason=str(data.get("freeze_reason", "")),
            freeze_approved_by=str(data.get("freeze_approved_by", "")),
            freeze_approved_at=str(data.get("freeze_approved_at", "")),
            freeze_renewal_owner=str(data.get("freeze_renewal_owner", "")),
            freeze_renewal_by=str(data.get("freeze_renewal_by", "")),
            freeze_renewal_status=str(data.get("freeze_renewal_status", "not-needed")),
        )


@dataclass(frozen=True)
class ReviewBlockerEvent:
    event_id: str
    blocker_id: str
    actor: str
    status: str
    summary: str
    timestamp: str
    next_action: str = ""
    handoff_from: str = ""
    handoff_to: str = ""
    channel: str = ""
    artifact_ref: str = ""
    ack_owner: str = ""
    ack_at: str = ""
    ack_status: str = "pending"

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_id": self.event_id,
            "blocker_id": self.blocker_id,
            "actor": self.actor,
            "status": self.status,
            "summary": self.summary,
            "timestamp": self.timestamp,
            "next_action": self.next_action,
            "handoff_from": self.handoff_from,
            "handoff_to": self.handoff_to,
            "channel": self.channel,
            "artifact_ref": self.artifact_ref,
            "ack_owner": self.ack_owner,
            "ack_at": self.ack_at,
            "ack_status": self.ack_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlockerEvent":
        return cls(
            event_id=str(data["event_id"]),
            blocker_id=str(data["blocker_id"]),
            actor=str(data["actor"]),
            status=str(data["status"]),
            summary=str(data["summary"]),
            timestamp=str(data["timestamp"]),
            next_action=str(data.get("next_action", "")),
            handoff_from=str(data.get("handoff_from", "")),
            handoff_to=str(data.get("handoff_to", "")),
            channel=str(data.get("channel", "")),
            artifact_ref=str(data.get("artifact_ref", "")),
            ack_owner=str(data.get("ack_owner", "")),
            ack_at=str(data.get("ack_at", "")),
            ack_status=str(data.get("ack_status", "pending")),
        )


@dataclass(frozen=True)
class UIReviewPackArtifacts:
    root_dir: str
    markdown_path: str
    html_path: str
    decision_log_path: str
    review_summary_board_path: str
    objective_coverage_board_path: str
    persona_readiness_board_path: str
    wireframe_readiness_board_path: str
    interaction_coverage_board_path: str
    open_question_tracker_path: str
    checklist_traceability_board_path: str
    decision_followup_tracker_path: str
    role_matrix_path: str
    role_coverage_board_path: str
    signoff_dependency_board_path: str
    signoff_log_path: str
    signoff_sla_dashboard_path: str
    signoff_reminder_queue_path: str
    reminder_cadence_board_path: str
    signoff_breach_board_path: str
    escalation_dashboard_path: str
    escalation_handoff_ledger_path: str
    handoff_ack_ledger_path: str
    owner_escalation_digest_path: str
    owner_workload_board_path: str
    blocker_log_path: str
    blocker_timeline_path: str
    freeze_exception_board_path: str
    freeze_approval_trail_path: str
    freeze_renewal_tracker_path: str
    exception_log_path: str
    exception_matrix_path: str
    audit_density_board_path: str
    owner_review_queue_path: str
    blocker_timeline_summary_path: str


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
    role_matrix: List[ReviewRoleAssignment] = field(default_factory=list)
    requires_role_matrix: bool = False
    signoff_log: List[ReviewSignoff] = field(default_factory=list)
    requires_signoff_log: bool = False
    blocker_log: List[ReviewBlocker] = field(default_factory=list)
    requires_blocker_log: bool = False
    blocker_timeline: List[ReviewBlockerEvent] = field(default_factory=list)
    requires_blocker_timeline: bool = False

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
            "role_matrix": [assignment.to_dict() for assignment in self.role_matrix],
            "requires_role_matrix": self.requires_role_matrix,
            "signoff_log": [signoff.to_dict() for signoff in self.signoff_log],
            "requires_signoff_log": self.requires_signoff_log,
            "blocker_log": [blocker.to_dict() for blocker in self.blocker_log],
            "requires_blocker_log": self.requires_blocker_log,
            "blocker_timeline": [event.to_dict() for event in self.blocker_timeline],
            "requires_blocker_timeline": self.requires_blocker_timeline,
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
            role_matrix=[ReviewRoleAssignment.from_dict(item) for item in data.get("role_matrix", [])],
            requires_role_matrix=bool(data.get("requires_role_matrix", False)),
            signoff_log=[ReviewSignoff.from_dict(item) for item in data.get("signoff_log", [])],
            requires_signoff_log=bool(data.get("requires_signoff_log", False)),
            blocker_log=[ReviewBlocker.from_dict(item) for item in data.get("blocker_log", [])],
            requires_blocker_log=bool(data.get("requires_blocker_log", False)),
            blocker_timeline=[ReviewBlockerEvent.from_dict(item) for item in data.get("blocker_timeline", [])],
            requires_blocker_timeline=bool(data.get("requires_blocker_timeline", False)),
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
    role_assignment_count: int = 0
    signoff_count: int = 0
    blocker_count: int = 0
    blocker_timeline_count: int = 0
    missing_sections: List[str] = field(default_factory=list)
    objectives_missing_signals: List[str] = field(default_factory=list)
    wireframes_missing_blocks: List[str] = field(default_factory=list)
    interactions_missing_states: List[str] = field(default_factory=list)
    unresolved_question_ids: List[str] = field(default_factory=list)
    wireframes_missing_checklists: List[str] = field(default_factory=list)
    orphan_checklist_surfaces: List[str] = field(default_factory=list)
    checklist_items_missing_evidence: List[str] = field(default_factory=list)
    checklist_items_missing_role_links: List[str] = field(default_factory=list)
    wireframes_missing_decisions: List[str] = field(default_factory=list)
    orphan_decision_surfaces: List[str] = field(default_factory=list)
    unresolved_decision_ids: List[str] = field(default_factory=list)
    unresolved_decisions_missing_follow_ups: List[str] = field(default_factory=list)
    wireframes_missing_role_assignments: List[str] = field(default_factory=list)
    orphan_role_assignment_surfaces: List[str] = field(default_factory=list)
    role_assignments_missing_responsibilities: List[str] = field(default_factory=list)
    role_assignments_missing_checklist_links: List[str] = field(default_factory=list)
    role_assignments_missing_decision_links: List[str] = field(default_factory=list)
    decisions_missing_role_links: List[str] = field(default_factory=list)
    wireframes_missing_signoffs: List[str] = field(default_factory=list)
    orphan_signoff_surfaces: List[str] = field(default_factory=list)
    signoffs_missing_assignments: List[str] = field(default_factory=list)
    signoffs_missing_evidence: List[str] = field(default_factory=list)
    signoffs_missing_requested_dates: List[str] = field(default_factory=list)
    signoffs_missing_due_dates: List[str] = field(default_factory=list)
    signoffs_missing_escalation_owners: List[str] = field(default_factory=list)
    signoffs_missing_reminder_owners: List[str] = field(default_factory=list)
    signoffs_missing_next_reminders: List[str] = field(default_factory=list)
    signoffs_missing_reminder_cadence: List[str] = field(default_factory=list)
    signoffs_with_breached_sla: List[str] = field(default_factory=list)
    waived_signoffs_missing_metadata: List[str] = field(default_factory=list)
    unresolved_required_signoff_ids: List[str] = field(default_factory=list)
    blockers_missing_signoff_links: List[str] = field(default_factory=list)
    blockers_missing_escalation_owners: List[str] = field(default_factory=list)
    blockers_missing_next_actions: List[str] = field(default_factory=list)
    freeze_exceptions_missing_owners: List[str] = field(default_factory=list)
    freeze_exceptions_missing_until: List[str] = field(default_factory=list)
    freeze_exceptions_missing_approvers: List[str] = field(default_factory=list)
    freeze_exceptions_missing_approval_dates: List[str] = field(default_factory=list)
    freeze_exceptions_missing_renewal_owners: List[str] = field(default_factory=list)
    freeze_exceptions_missing_renewal_dates: List[str] = field(default_factory=list)
    blockers_missing_timeline_events: List[str] = field(default_factory=list)
    closed_blockers_missing_resolution_events: List[str] = field(default_factory=list)
    orphan_blocker_surfaces: List[str] = field(default_factory=list)
    orphan_blocker_timeline_blocker_ids: List[str] = field(default_factory=list)
    handoff_events_missing_targets: List[str] = field(default_factory=list)
    handoff_events_missing_artifacts: List[str] = field(default_factory=list)
    handoff_events_missing_ack_owners: List[str] = field(default_factory=list)
    handoff_events_missing_ack_dates: List[str] = field(default_factory=list)
    unresolved_required_signoffs_without_blockers: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: objectives={self.objective_count} "
            f"wireframes={self.wireframe_count} "
            f"interactions={self.interaction_count} "
            f"open_questions={self.open_question_count} "
            f"checklist={self.checklist_count} "
            f"decisions={self.decision_count} "
            f"role_assignments={self.role_assignment_count} "
            f"signoffs={self.signoff_count} "
            f"blockers={self.blocker_count} "
            f"timeline_events={self.blocker_timeline_count}"
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
        checklist_items_missing_role_links = []
        if pack.requires_reviewer_checklist:
            wireframes_missing_checklists = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in checklist_by_surface
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
        unresolved_decisions_missing_follow_ups = []
        if pack.requires_decision_log:
            wireframes_missing_decisions = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in decision_by_surface
            )
            orphan_decision_surfaces = sorted(
                surface_id for surface_id in decision_by_surface if surface_id not in wireframe_ids
            )
            unresolved_decision_ids = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}
            )
            unresolved_decisions_missing_follow_ups = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}
                and not decision.follow_up.strip()
            )

        checklist_item_ids = {item.item_id for item in pack.reviewer_checklist}
        decision_ids = {decision.decision_id for decision in pack.decision_log}
        assignment_ids = {assignment.assignment_id for assignment in pack.role_matrix}
        role_assignments_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
        for assignment in pack.role_matrix:
            role_assignments_by_surface.setdefault(assignment.surface_id, []).append(assignment)
        wireframes_missing_role_assignments = []
        orphan_role_assignment_surfaces = []
        role_assignments_missing_responsibilities = []
        role_assignments_missing_checklist_links = []
        role_assignments_missing_decision_links = []
        decisions_missing_role_links = []
        if pack.requires_role_matrix:
            wireframes_missing_role_assignments = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in role_assignments_by_surface
            )
            orphan_role_assignment_surfaces = sorted(
                surface_id
                for surface_id in role_assignments_by_surface
                if surface_id not in wireframe_ids
            )
            role_assignments_missing_responsibilities = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.responsibilities
            )
            role_assignments_missing_checklist_links = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.checklist_item_ids
                or any(item_id not in checklist_item_ids for item_id in assignment.checklist_item_ids)
            )
            role_assignments_missing_decision_links = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.decision_ids
                or any(decision_id not in decision_ids for decision_id in assignment.decision_ids)
            )
            role_linked_checklist_ids = {
                item_id
                for assignment in pack.role_matrix
                for item_id in assignment.checklist_item_ids
            }
            role_linked_decision_ids = {
                decision_id
                for assignment in pack.role_matrix
                for decision_id in assignment.decision_ids
            }
            checklist_items_missing_role_links = sorted(
                item.item_id
                for item in pack.reviewer_checklist
                if item.item_id not in role_linked_checklist_ids
            )
            decisions_missing_role_links = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.decision_id not in role_linked_decision_ids
            )

        signoffs_by_surface: Dict[str, List[ReviewSignoff]] = {}
        for signoff in pack.signoff_log:
            signoffs_by_surface.setdefault(signoff.surface_id, []).append(signoff)
        wireframes_missing_signoffs = []
        orphan_signoff_surfaces = []
        signoffs_missing_assignments = []
        signoffs_missing_evidence = []
        signoffs_missing_requested_dates = []
        signoffs_missing_due_dates = []
        signoffs_missing_escalation_owners = []
        signoffs_missing_reminder_owners = []
        signoffs_missing_next_reminders = []
        signoffs_missing_reminder_cadence = []
        signoffs_with_breached_sla = []
        waived_signoffs_missing_metadata = []
        unresolved_required_signoff_ids = []
        if pack.requires_signoff_log:
            wireframes_missing_signoffs = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in signoffs_by_surface
            )
            orphan_signoff_surfaces = sorted(
                surface_id for surface_id in signoffs_by_surface if surface_id not in wireframe_ids
            )
            signoffs_missing_assignments = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.assignment_id not in assignment_ids
            )
            signoffs_missing_evidence = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.status.lower() != "waived" and not signoff.evidence_links
            )
            signoffs_missing_requested_dates = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.requested_at.strip()
            )
            signoffs_missing_due_dates = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.due_at.strip()
            )
            signoffs_missing_escalation_owners = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.escalation_owner.strip()
            )
            unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
            signoffs_missing_reminder_owners = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.reminder_owner.strip()
            )
            signoffs_missing_next_reminders = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.next_reminder_at.strip()
            )
            signoffs_missing_reminder_cadence = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.reminder_cadence.strip()
            )
            signoffs_with_breached_sla = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.sla_status.lower() == "breached"
                and signoff.status.lower() not in {"approved", "accepted", "resolved"}
            )
            waived_signoffs_missing_metadata = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.status.lower() == "waived"
                and (not signoff.waiver_owner.strip() or not signoff.waiver_reason.strip())
            )
            unresolved_required_signoff_ids = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
            )

        blocker_by_signoff: Dict[str, List[ReviewBlocker]] = {}
        blocker_surfaces = set()
        for blocker in pack.blocker_log:
            blocker_surfaces.add(blocker.surface_id)
            blocker_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
        blockers_missing_signoff_links = []
        blockers_missing_escalation_owners = []
        blockers_missing_next_actions = []
        freeze_exceptions_missing_owners = []
        freeze_exceptions_missing_until = []
        freeze_exceptions_missing_approvers = []
        freeze_exceptions_missing_approval_dates = []
        freeze_exceptions_missing_renewal_owners = []
        freeze_exceptions_missing_renewal_dates = []
        orphan_blocker_surfaces = []
        unresolved_required_signoffs_without_blockers = []
        if pack.requires_blocker_log:
            signoff_ids = {signoff.signoff_id for signoff in pack.signoff_log}
            blockers_missing_signoff_links = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if blocker.signoff_id not in signoff_ids
            )
            blockers_missing_escalation_owners = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if not blocker.escalation_owner.strip()
            )
            blockers_missing_next_actions = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if not blocker.next_action.strip()
            )
            freeze_exceptions_missing_owners = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_owner.strip()
            )
            freeze_exceptions_missing_until = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_until.strip()
            )
            freeze_exceptions_missing_approvers = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_approved_by.strip()
            )
            freeze_exceptions_missing_approval_dates = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_approved_at.strip()
            )
            freeze_exceptions_missing_renewal_owners = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_renewal_owner.strip()
            )
            freeze_exceptions_missing_renewal_dates = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_renewal_by.strip()
            )
            orphan_blocker_surfaces = sorted(
                surface_id for surface_id in blocker_surfaces if surface_id not in wireframe_ids
            )
            unresolved_required_signoffs_without_blockers = sorted(
                signoff_id
                for signoff_id in unresolved_required_signoff_ids
                if signoff_id not in blocker_by_signoff
            )

        blocker_timeline_by_blocker: Dict[str, List[ReviewBlockerEvent]] = {}
        for event in pack.blocker_timeline:
            blocker_timeline_by_blocker.setdefault(event.blocker_id, []).append(event)
        blockers_missing_timeline_events = []
        closed_blockers_missing_resolution_events = []
        orphan_blocker_timeline_blocker_ids = []
        handoff_events_missing_targets = []
        handoff_events_missing_artifacts = []
        handoff_events_missing_ack_owners = []
        handoff_events_missing_ack_dates = []
        if pack.requires_blocker_timeline:
            blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
            orphan_blocker_timeline_blocker_ids = sorted(
                blocker_id
                for blocker_id in blocker_timeline_by_blocker
                if blocker_id not in blocker_ids
            )
            blockers_missing_timeline_events = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.status.lower() not in {"resolved", "closed"}
                and blocker.blocker_id not in blocker_timeline_by_blocker
            )
            closed_blockers_missing_resolution_events = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.status.lower() in {"resolved", "closed"}
                and not any(
                    event.status.lower() in {"resolved", "closed"}
                    for event in blocker_timeline_by_blocker.get(blocker.blocker_id, [])
                )
            )
            handoff_statuses = {"escalated", "handoff", "reassigned"}
            handoff_events_missing_targets = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.handoff_to.strip()
            )
            handoff_events_missing_artifacts = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.artifact_ref.strip()
            )
            handoff_events_missing_ack_owners = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.ack_owner.strip()
            )
            handoff_events_missing_ack_dates = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.ack_at.strip()
            )

        ready = not (
            missing_sections
            or objectives_missing_signals
            or wireframes_missing_blocks
            or interactions_missing_states
            or wireframes_missing_checklists
            or orphan_checklist_surfaces
            or checklist_items_missing_evidence
            or checklist_items_missing_role_links
            or wireframes_missing_decisions
            or orphan_decision_surfaces
            or unresolved_decisions_missing_follow_ups
            or wireframes_missing_role_assignments
            or orphan_role_assignment_surfaces
            or role_assignments_missing_responsibilities
            or role_assignments_missing_checklist_links
            or role_assignments_missing_decision_links
            or decisions_missing_role_links
            or wireframes_missing_signoffs
            or orphan_signoff_surfaces
            or signoffs_missing_assignments
            or signoffs_missing_evidence
            or signoffs_missing_requested_dates
            or signoffs_missing_due_dates
            or signoffs_missing_escalation_owners
            or signoffs_missing_reminder_owners
            or signoffs_missing_next_reminders
            or signoffs_missing_reminder_cadence
            or waived_signoffs_missing_metadata
            or blockers_missing_signoff_links
            or blockers_missing_escalation_owners
            or blockers_missing_next_actions
            or freeze_exceptions_missing_owners
            or freeze_exceptions_missing_until
            or freeze_exceptions_missing_approvers
            or freeze_exceptions_missing_approval_dates
            or freeze_exceptions_missing_renewal_owners
            or freeze_exceptions_missing_renewal_dates
            or blockers_missing_timeline_events
            or closed_blockers_missing_resolution_events
            or orphan_blocker_surfaces
            or orphan_blocker_timeline_blocker_ids
            or handoff_events_missing_targets
            or handoff_events_missing_artifacts
            or handoff_events_missing_ack_owners
            or handoff_events_missing_ack_dates
            or unresolved_required_signoffs_without_blockers
        )
        return UIReviewPackAudit(
            ready=ready,
            objective_count=len(pack.objectives),
            wireframe_count=len(pack.wireframes),
            interaction_count=len(pack.interactions),
            open_question_count=len(pack.open_questions),
            checklist_count=len(pack.reviewer_checklist),
            decision_count=len(pack.decision_log),
            role_assignment_count=len(pack.role_matrix),
            signoff_count=len(pack.signoff_log),
            blocker_count=len(pack.blocker_log),
            blocker_timeline_count=len(pack.blocker_timeline),
            missing_sections=missing_sections,
            objectives_missing_signals=objectives_missing_signals,
            wireframes_missing_blocks=wireframes_missing_blocks,
            interactions_missing_states=interactions_missing_states,
            unresolved_question_ids=unresolved_question_ids,
            wireframes_missing_checklists=wireframes_missing_checklists,
            orphan_checklist_surfaces=orphan_checklist_surfaces,
            checklist_items_missing_evidence=checklist_items_missing_evidence,
            checklist_items_missing_role_links=checklist_items_missing_role_links,
            wireframes_missing_decisions=wireframes_missing_decisions,
            orphan_decision_surfaces=orphan_decision_surfaces,
            unresolved_decision_ids=unresolved_decision_ids,
            unresolved_decisions_missing_follow_ups=unresolved_decisions_missing_follow_ups,
            wireframes_missing_role_assignments=wireframes_missing_role_assignments,
            orphan_role_assignment_surfaces=orphan_role_assignment_surfaces,
            role_assignments_missing_responsibilities=role_assignments_missing_responsibilities,
            role_assignments_missing_checklist_links=role_assignments_missing_checklist_links,
            role_assignments_missing_decision_links=role_assignments_missing_decision_links,
            decisions_missing_role_links=decisions_missing_role_links,
            wireframes_missing_signoffs=wireframes_missing_signoffs,
            orphan_signoff_surfaces=orphan_signoff_surfaces,
            signoffs_missing_assignments=signoffs_missing_assignments,
            signoffs_missing_evidence=signoffs_missing_evidence,
            signoffs_missing_requested_dates=signoffs_missing_requested_dates,
            signoffs_missing_due_dates=signoffs_missing_due_dates,
            signoffs_missing_escalation_owners=signoffs_missing_escalation_owners,
            signoffs_missing_reminder_owners=signoffs_missing_reminder_owners,
            signoffs_missing_next_reminders=signoffs_missing_next_reminders,
            signoffs_missing_reminder_cadence=signoffs_missing_reminder_cadence,
            signoffs_with_breached_sla=signoffs_with_breached_sla,
            waived_signoffs_missing_metadata=waived_signoffs_missing_metadata,
            unresolved_required_signoff_ids=unresolved_required_signoff_ids,
            blockers_missing_signoff_links=blockers_missing_signoff_links,
            blockers_missing_escalation_owners=blockers_missing_escalation_owners,
            blockers_missing_next_actions=blockers_missing_next_actions,
            freeze_exceptions_missing_owners=freeze_exceptions_missing_owners,
            freeze_exceptions_missing_until=freeze_exceptions_missing_until,
            freeze_exceptions_missing_approvers=freeze_exceptions_missing_approvers,
            freeze_exceptions_missing_approval_dates=freeze_exceptions_missing_approval_dates,
            freeze_exceptions_missing_renewal_owners=freeze_exceptions_missing_renewal_owners,
            freeze_exceptions_missing_renewal_dates=freeze_exceptions_missing_renewal_dates,
            blockers_missing_timeline_events=blockers_missing_timeline_events,
            closed_blockers_missing_resolution_events=closed_blockers_missing_resolution_events,
            orphan_blocker_surfaces=orphan_blocker_surfaces,
            orphan_blocker_timeline_blocker_ids=orphan_blocker_timeline_blocker_ids,
            handoff_events_missing_targets=handoff_events_missing_targets,
            handoff_events_missing_artifacts=handoff_events_missing_artifacts,
            handoff_events_missing_ack_owners=handoff_events_missing_ack_owners,
            handoff_events_missing_ack_dates=handoff_events_missing_ack_dates,
            unresolved_required_signoffs_without_blockers=unresolved_required_signoffs_without_blockers,
        )


def _build_blocker_timeline_index(pack: UIReviewPack) -> Dict[str, List[ReviewBlockerEvent]]:
    timeline_index: Dict[str, List[ReviewBlockerEvent]] = {}
    for event in sorted(pack.blocker_timeline, key=lambda item: (item.timestamp, item.event_id)):
        timeline_index.setdefault(event.blocker_id, []).append(event)
    return timeline_index


def _build_review_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        if signoff.status.lower() not in {"waived", "deferred"}:
            continue
        entries.append(
            {
                "exception_id": f"exc-{signoff.signoff_id}",
                "category": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "owner": signoff.waiver_owner or signoff.role,
                "status": signoff.status,
                "severity": "none",
                "summary": signoff.waiver_reason or signoff.notes or "none",
                "evidence": ",".join(signoff.evidence_links) or "none",
                "latest_event": "none",
                "next_action": signoff.notes or signoff.waiver_reason or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "exception_id": f"exc-{blocker.blocker_id}",
                "category": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "owner": blocker.owner,
                "status": blocker.status,
                "severity": blocker.severity,
                "summary": blocker.summary,
                "evidence": blocker.escalation_owner or "none",
                "latest_event": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["surface_id"], item["category"], item["source_id"]),
    )


def _build_freeze_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        if signoff.status.lower() not in {"waived", "deferred"}:
            continue
        entries.append(
            {
                "entry_id": f"freeze-{signoff.signoff_id}",
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "owner": signoff.waiver_owner or signoff.role,
                "status": signoff.status,
                "window": "none",
                "summary": signoff.waiver_reason or signoff.notes or "none",
                "evidence": ",".join(signoff.evidence_links) or "none",
                "next_action": signoff.notes or signoff.waiver_reason or "none",
            }
        )
    for blocker in pack.blocker_log:
        if not blocker.freeze_exception:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "entry_id": f"freeze-{blocker.blocker_id}",
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "owner": blocker.freeze_owner or blocker.owner,
                "status": blocker.status,
                "window": blocker.freeze_until or "none",
                "summary": blocker.freeze_reason or blocker.summary,
                "evidence": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["surface_id"], item["item_type"], item["source_id"]),
    )


def _build_signoff_breach_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index: Dict[str, List[str]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        blocker_index.setdefault(blocker.signoff_id, []).append(blocker.blocker_id)
    entries = [
        {
            "entry_id": f"breach-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "requested_at": signoff.requested_at or "none",
            "due_at": signoff.due_at or "none",
            "escalation_owner": signoff.escalation_owner or "none",
            "linked_blockers": ",".join(sorted(blocker_index.get(signoff.signoff_id, []))) or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.sla_status.lower() in {"at-risk", "breached"}
        and signoff.status.lower() not in {"approved", "accepted", "resolved", "waived", "deferred"}
    ]
    return sorted(
        entries,
        key=lambda item: (item["due_at"], item["sla_status"], item["escalation_owner"], item["signoff_id"]),
    )


def _build_escalation_handoff_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}
    handoff_statuses = {"escalated", "handoff", "reassigned"}
    entries: List[Dict[str, str]] = []
    for event in pack.blocker_timeline:
        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():
            continue
        blocker = blocker_index.get(event.blocker_id)
        entries.append(
            {
                "ledger_id": f"handoff-{event.event_id}",
                "event_id": event.event_id,
                "blocker_id": event.blocker_id,
                "surface_id": blocker.surface_id if blocker else "none",
                "actor": event.actor,
                "status": event.status,
                "handoff_from": event.handoff_from or (blocker.owner if blocker else "none"),
                "handoff_to": event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none",
                "channel": event.channel or "none",
                "artifact_ref": event.artifact_ref or "none",
                "timestamp": event.timestamp,
                "summary": event.summary,
                "next_action": event.next_action or "none",
            }
        )
    return sorted(entries, key=lambda item: (item["timestamp"], item["event_id"]))


def _build_handoff_ack_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}
    handoff_statuses = {"escalated", "handoff", "reassigned"}
    entries: List[Dict[str, str]] = []
    for event in pack.blocker_timeline:
        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():
            continue
        blocker = blocker_index.get(event.blocker_id)
        fallback_owner = event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none"
        entries.append(
            {
                "entry_id": f"ack-{event.event_id}",
                "event_id": event.event_id,
                "blocker_id": event.blocker_id,
                "surface_id": blocker.surface_id if blocker else "none",
                "actor": event.actor,
                "status": event.status,
                "handoff_to": event.handoff_to or fallback_owner,
                "ack_owner": event.ack_owner or fallback_owner,
                "ack_status": event.ack_status or "pending",
                "ack_at": event.ack_at or "none",
                "channel": event.channel or "none",
                "artifact_ref": event.artifact_ref or "none",
                "summary": event.summary,
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["ack_status"], item["ack_owner"], item["event_id"]),
    )


def _build_signoff_reminder_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    entries = [
        {
            "entry_id": f"rem-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "reminder_owner": signoff.reminder_owner or "none",
            "reminder_channel": signoff.reminder_channel or "none",
            "last_reminder_at": signoff.last_reminder_at or "none",
            "next_reminder_at": signoff.next_reminder_at or "none",
            "due_at": signoff.due_at or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.required and signoff.status.lower() not in unresolved_statuses
    ]
    return sorted(
        entries,
        key=lambda item: (item["next_reminder_at"], item["reminder_owner"], item["signoff_id"]),
    )


def _build_reminder_cadence_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    entries = [
        {
            "entry_id": f"cad-rem-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "reminder_owner": signoff.reminder_owner or "none",
            "reminder_cadence": signoff.reminder_cadence or "none",
            "reminder_status": signoff.reminder_status or "scheduled",
            "last_reminder_at": signoff.last_reminder_at or "none",
            "next_reminder_at": signoff.next_reminder_at or "none",
            "due_at": signoff.due_at or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.required and signoff.status.lower() not in unresolved_statuses
    ]
    return sorted(
        entries,
        key=lambda item: (item["reminder_cadence"], item["reminder_status"], item["signoff_id"]),
    )


def _build_freeze_approval_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for blocker in pack.blocker_log:
        if not blocker.freeze_exception:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "entry_id": f"freeze-approval-{blocker.blocker_id}",
                "blocker_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "freeze_owner": blocker.freeze_owner or blocker.owner,
                "freeze_until": blocker.freeze_until or "none",
                "freeze_approved_by": blocker.freeze_approved_by or "none",
                "freeze_approved_at": blocker.freeze_approved_at or "none",
                "summary": blocker.freeze_reason or blocker.summary,
                "latest_event": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["freeze_approved_at"], item["freeze_until"], item["blocker_id"]),
    )


def _build_freeze_renewal_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries = [
        {
            "entry_id": f"renew-{blocker.blocker_id}",
            "blocker_id": blocker.blocker_id,
            "surface_id": blocker.surface_id,
            "status": blocker.status,
            "freeze_owner": blocker.freeze_owner or blocker.owner,
            "freeze_until": blocker.freeze_until or "none",
            "renewal_owner": blocker.freeze_renewal_owner or "none",
            "renewal_by": blocker.freeze_renewal_by or "none",
            "renewal_status": blocker.freeze_renewal_status or "not-needed",
            "freeze_approved_by": blocker.freeze_approved_by or "none",
            "summary": blocker.freeze_reason or blocker.summary,
            "next_action": blocker.next_action or "none",
        }
        for blocker in pack.blocker_log
        if blocker.freeze_exception
    ]
    return sorted(
        entries,
        key=lambda item: (item["renewal_by"], item["renewal_owner"], item["blocker_id"]),
    )


def _build_owner_escalation_digest_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for entry in _build_escalation_dashboard_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['escalation_id']}",
                "owner": entry["escalation_owner"],
                "item_type": entry["item_type"],
                "source_id": entry["source_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["priority"],
            }
        )
    for entry in _build_signoff_reminder_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['entry_id']}",
                "owner": entry["reminder_owner"],
                "item_type": "reminder",
                "source_id": entry["signoff_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["next_reminder_at"],
            }
        )
    for entry in _build_freeze_approval_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['entry_id']}",
                "owner": entry["freeze_approved_by"],
                "item_type": "freeze",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["freeze_until"],
            }
        )
    for entry in _build_escalation_handoff_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['ledger_id']}",
                "owner": entry["handoff_to"],
                "item_type": "handoff",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["timestamp"],
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),
    )


def _build_owner_review_queue_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    signoff_ready_statuses = {"approved", "accepted", "resolved"}
    blocker_done_statuses = {"resolved", "closed"}

    for item in pack.reviewer_checklist:
        if item.status.lower() in checklist_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{item.item_id}",
                "owner": item.owner,
                "item_type": "checklist",
                "source_id": item.item_id,
                "surface_id": item.surface_id,
                "status": item.status,
                "summary": item.prompt,
                "next_action": item.notes or ",".join(item.evidence_links) or "none",
            }
        )
    for decision in pack.decision_log:
        if decision.status.lower() in decision_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{decision.decision_id}",
                "owner": decision.owner,
                "item_type": "decision",
                "source_id": decision.decision_id,
                "surface_id": decision.surface_id,
                "status": decision.status,
                "summary": decision.summary,
                "next_action": decision.follow_up or decision.rationale,
            }
        )
    for signoff in pack.signoff_log:
        if signoff.status.lower() in signoff_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{signoff.signoff_id}",
                "owner": signoff.waiver_owner or signoff.role,
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "status": signoff.status,
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
                "next_action": signoff.waiver_reason or signoff.notes or signoff.due_at or ",".join(signoff.evidence_links) or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in blocker_done_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{blocker.blocker_id}",
                "owner": blocker.owner,
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "summary": blocker.summary,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),
    )


def _build_checklist_traceability_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignments_by_item: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        for item_id in assignment.checklist_item_ids:
            assignments_by_item.setdefault(item_id, []).append(assignment)
    entries: List[Dict[str, str]] = []
    for item in pack.reviewer_checklist:
        linked_assignments = assignments_by_item.get(item.item_id, [])
        linked_decisions = sorted(
            {decision_id for assignment in linked_assignments for decision_id in assignment.decision_ids}
        )
        entries.append(
            {
                "entry_id": f"trace-{item.item_id}",
                "item_id": item.item_id,
                "surface_id": item.surface_id,
                "owner": item.owner,
                "status": item.status,
                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",
                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",
                "linked_decisions": ",".join(linked_decisions) or "none",
                "evidence": ",".join(item.evidence_links) or "none",
                "summary": item.notes or item.prompt,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["item_id"]))


def _build_decision_followup_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignments_by_decision: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        for decision_id in assignment.decision_ids:
            assignments_by_decision.setdefault(decision_id, []).append(assignment)
    entries: List[Dict[str, str]] = []
    for decision in pack.decision_log:
        linked_assignments = assignments_by_decision.get(decision.decision_id, [])
        linked_checklist_ids = sorted(
            {item_id for assignment in linked_assignments for item_id in assignment.checklist_item_ids}
        )
        entries.append(
            {
                "entry_id": f"follow-{decision.decision_id}",
                "decision_id": decision.decision_id,
                "surface_id": decision.surface_id,
                "owner": decision.owner,
                "status": decision.status,
                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",
                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",
                "linked_checklists": ",".join(linked_checklist_ids) or "none",
                "follow_up": decision.follow_up or "none",
                "summary": decision.summary,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["decision_id"]))


def _build_objective_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignments_by_role.setdefault(assignment.role, []).append(assignment)
    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    checklist_index = {item.item_id: item for item in pack.reviewer_checklist}
    decision_index = {decision.decision_id: decision for decision in pack.decision_log}
    status_priority = {"blocked": 0, "at-risk": 1, "covered": 2}
    entries: List[Dict[str, str]] = []
    for objective in pack.objectives:
        assignments = assignments_by_role.get(objective.persona, [])
        checklist_ids = sorted(
            {item_id for assignment in assignments for item_id in assignment.checklist_item_ids}
        )
        decision_ids = sorted(
            {decision_id for assignment in assignments for decision_id in assignment.decision_ids}
        )
        signoffs = [
            signoff_by_assignment[assignment.assignment_id]
            for assignment in assignments
            if assignment.assignment_id in signoff_by_assignment
        ]
        blocker_ids = sorted(
            {
                blocker.blocker_id
                for signoff in signoffs
                for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
            }
        )
        open_checklist = sum(
            1
            for item_id in checklist_ids
            if checklist_index[item_id].status.lower() not in checklist_ready_statuses
        )
        open_decisions = sum(
            1
            for decision_id in decision_ids
            if decision_index[decision_id].status.lower() not in decision_ready_statuses
        )
        open_assignments = sum(
            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses
        )
        open_signoffs = sum(
            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses
        )
        coverage_status = (
            "blocked"
            if blocker_ids
            else "at-risk"
            if open_checklist or open_decisions or open_assignments or open_signoffs
            else "covered"
        )
        entries.append(
            {
                "entry_id": f"objcov-{objective.objective_id}",
                "objective_id": objective.objective_id,
                "persona": objective.persona,
                "priority": objective.priority,
                "coverage_status": coverage_status,
                "dependency_count": str(len(objective.dependencies)),
                "dependency_ids": ",".join(objective.dependencies) or "none",
                "surface_ids": ",".join(sorted({assignment.surface_id for assignment in assignments})) or "none",
                "assignment_ids": ",".join(assignment.assignment_id for assignment in assignments) or "none",
                "checklist_ids": ",".join(checklist_ids) or "none",
                "decision_ids": ",".join(decision_ids) or "none",
                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",
                "blocker_ids": ",".join(blocker_ids) or "none",
                "summary": objective.success_signal or objective.outcome,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["coverage_status"]], item["persona"], item["objective_id"]),
    )


def _build_wireframe_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    blocker_done_statuses = {"resolved", "closed"}
    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        checklist_by_surface.setdefault(item.surface_id, []).append(item)
    decision_by_surface: Dict[str, List[ReviewDecision]] = {}
    for decision in pack.decision_log:
        decision_by_surface.setdefault(decision.surface_id, []).append(decision)
    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)
    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}
    for signoff in pack.signoff_log:
        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)
    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)
    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}
    entries: List[Dict[str, str]] = []
    for wireframe in pack.wireframes:
        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])
        decisions = decision_by_surface.get(wireframe.surface_id, [])
        assignments = assignment_by_surface.get(wireframe.surface_id, [])
        signoffs = signoff_by_surface.get(wireframe.surface_id, [])
        blockers = [
            blocker
            for blocker in blocker_by_surface.get(wireframe.surface_id, [])
            if blocker.status.lower() not in blocker_done_statuses
        ]
        checklist_open = sum(
            1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses
        )
        decisions_open = sum(
            1 for decision in decisions if decision.status.lower() not in decision_ready_statuses
        )
        assignments_open = sum(
            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses
        )
        signoffs_open = sum(
            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses
        )
        blockers_open = len(blockers)
        open_total = (
            checklist_open + decisions_open + assignments_open + signoffs_open + blockers_open
        )
        readiness_status = (
            "blocked"
            if blockers_open
            else "at-risk"
            if checklist_open or decisions_open or assignments_open or signoffs_open
            else "ready"
        )
        entries.append(
            {
                "entry_id": f"wire-{wireframe.surface_id}",
                "surface_id": wireframe.surface_id,
                "device": wireframe.device,
                "entry_point": wireframe.entry_point,
                "readiness_status": readiness_status,
                "open_total": str(open_total),
                "checklist_open": str(checklist_open),
                "decisions_open": str(decisions_open),
                "assignments_open": str(assignments_open),
                "signoffs_open": str(signoffs_open),
                "blockers_open": str(blockers_open),
                "block_count": str(len(wireframe.primary_blocks)),
                "note_count": str(len(wireframe.review_notes)),
                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",
                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",
                "summary": wireframe.name,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["readiness_status"]], item["surface_id"]),
    )


def _build_open_question_tracker_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for question in pack.open_questions:
        linked_items = [
            item for item in pack.reviewer_checklist if question.question_id in item.evidence_links
        ]
        flow_ids = sorted(
            {
                evidence_link
                for item in linked_items
                for evidence_link in item.evidence_links
                if evidence_link.startswith("flow-")
            }
        )
        entries.append(
            {
                "entry_id": f"qtrack-{question.question_id}",
                "question_id": question.question_id,
                "owner": question.owner,
                "theme": question.theme,
                "status": question.status,
                "link_status": "linked" if linked_items else "orphan",
                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",
                "checklist_ids": ",".join(item.item_id for item in linked_items) or "none",
                "flow_ids": ",".join(flow_ids) or "none",
                "summary": question.question,
                "impact": question.impact,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["question_id"]))


def _build_interaction_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    checklist_by_flow: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        for evidence_link in item.evidence_links:
            if evidence_link.startswith("flow-"):
                checklist_by_flow.setdefault(evidence_link, []).append(item)
    status_priority = {"missing": 0, "watch": 1, "covered": 2}
    entries: List[Dict[str, str]] = []
    for interaction in pack.interactions:
        linked_items = checklist_by_flow.get(interaction.flow_id, [])
        checklist_ids = list(dict.fromkeys(item.item_id for item in linked_items))
        open_checklist_ids = list(
            dict.fromkeys(
                item.item_id
                for item in linked_items
                if item.status.lower() not in checklist_ready_statuses
            )
        )
        coverage_status = (
            "missing"
            if not checklist_ids
            else "watch"
            if open_checklist_ids
            else "covered"
        )
        entries.append(
            {
                "entry_id": f"intcov-{interaction.flow_id}",
                "flow_id": interaction.flow_id,
                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",
                "owners": ",".join(sorted({item.owner for item in linked_items})) or "none",
                "checklist_ids": ",".join(checklist_ids) or "none",
                "open_checklist_ids": ",".join(open_checklist_ids) or "none",
                "coverage_status": coverage_status,
                "state_count": str(len(interaction.states)),
                "exception_count": str(len(interaction.exceptions)),
                "summary": interaction.trigger,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["coverage_status"]], item["flow_id"]),
    )


def _build_persona_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    objective_entries = _build_objective_coverage_entries(pack)
    objective_entries_by_persona: Dict[str, List[Dict[str, str]]] = {}
    for entry in objective_entries:
        objective_entries_by_persona.setdefault(entry["persona"], []).append(entry)
    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignments_by_role.setdefault(assignment.role, []).append(assignment)
    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    questions_by_owner: Dict[str, List[OpenQuestion]] = {}
    for question in pack.open_questions:
        questions_by_owner.setdefault(question.owner, []).append(question)
    queue_entries = _build_owner_review_queue_entries(pack)
    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}
    entries: List[Dict[str, str]] = []
    for persona, persona_objectives in objective_entries_by_persona.items():
        surface_ids = sorted(
            {
                surface_id
                for entry in persona_objectives
                for surface_id in entry["surface_ids"].split(",")
                if surface_id and surface_id != "none"
            }
        )
        assignments = assignments_by_role.get(persona, [])
        signoffs = [
            signoff_by_assignment[assignment.assignment_id]
            for assignment in assignments
            if assignment.assignment_id in signoff_by_assignment
        ]
        blockers = [
            blocker
            for signoff in signoffs
            for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
        ]
        blocker_ids = sorted({blocker.blocker_id for blocker in blockers})
        questions = questions_by_owner.get(persona, [])
        queue_items = [
            entry
            for entry in queue_entries
            if entry["owner"] == persona
            and (not surface_ids or entry["surface_id"] in surface_ids)
        ]
        objective_statuses = {entry["coverage_status"] for entry in persona_objectives}
        readiness = (
            "blocked"
            if "blocked" in objective_statuses or blocker_ids
            else "at-risk"
            if "at-risk" in objective_statuses or questions or queue_items
            else "ready"
        )
        entries.append(
            {
                "entry_id": f"persona-{persona.lower().replace(' ', '-')}",
                "persona": persona,
                "readiness": readiness,
                "objective_count": str(len(persona_objectives)),
                "assignment_count": str(len(assignments)),
                "signoff_count": str(len(signoffs)),
                "question_count": str(len(questions)),
                "queue_count": str(len(queue_items)),
                "blocker_count": str(len(blocker_ids)),
                "objective_ids": ",".join(
                    sorted(entry["objective_id"] for entry in persona_objectives)
                )
                or "none",
                "surface_ids": ",".join(surface_ids) or "none",
                "queue_ids": ",".join(sorted(entry["queue_id"] for entry in queue_items)) or "none",
                "blocker_ids": ",".join(blocker_ids) or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["readiness"]], item["persona"]),
    )


def _build_review_summary_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    objective_entries = _build_objective_coverage_entries(pack)
    objective_status_counts: Dict[str, int] = {}
    for entry in objective_entries:
        objective_status_counts[entry["coverage_status"]] = (
            objective_status_counts.get(entry["coverage_status"], 0) + 1
        )

    persona_entries = _build_persona_readiness_entries(pack)
    persona_status_counts: Dict[str, int] = {}
    for entry in persona_entries:
        persona_status_counts[entry["readiness"]] = persona_status_counts.get(entry["readiness"], 0) + 1

    wireframe_entries = _build_wireframe_readiness_entries(pack)
    wireframe_status_counts: Dict[str, int] = {}
    for entry in wireframe_entries:
        wireframe_status_counts[entry["readiness_status"]] = (
            wireframe_status_counts.get(entry["readiness_status"], 0) + 1
        )

    interaction_entries = _build_interaction_coverage_entries(pack)
    interaction_status_counts: Dict[str, int] = {}
    for entry in interaction_entries:
        interaction_status_counts[entry["coverage_status"]] = (
            interaction_status_counts.get(entry["coverage_status"], 0) + 1
        )

    question_entries = _build_open_question_tracker_entries(pack)
    question_link_counts: Dict[str, int] = {}
    question_owners = {entry["owner"] for entry in question_entries}
    for entry in question_entries:
        question_link_counts[entry["link_status"]] = question_link_counts.get(entry["link_status"], 0) + 1

    action_entries = _build_owner_workload_entries(pack)
    action_lane_counts: Dict[str, int] = {}
    for entry in action_entries:
        action_lane_counts[entry["lane"]] = action_lane_counts.get(entry["lane"], 0) + 1

    return [
        {
            "entry_id": "summary-objectives",
            "category": "objectives",
            "total": str(len(objective_entries)),
            "metrics": (
                f"blocked={objective_status_counts.get('blocked', 0)} "
                f"at-risk={objective_status_counts.get('at-risk', 0)} "
                f"covered={objective_status_counts.get('covered', 0)}"
            ),
        },
        {
            "entry_id": "summary-personas",
            "category": "personas",
            "total": str(len(persona_entries)),
            "metrics": (
                f"blocked={persona_status_counts.get('blocked', 0)} "
                f"at-risk={persona_status_counts.get('at-risk', 0)} "
                f"ready={persona_status_counts.get('ready', 0)}"
            ),
        },
        {
            "entry_id": "summary-wireframes",
            "category": "wireframes",
            "total": str(len(wireframe_entries)),
            "metrics": (
                f"blocked={wireframe_status_counts.get('blocked', 0)} "
                f"at-risk={wireframe_status_counts.get('at-risk', 0)} "
                f"ready={wireframe_status_counts.get('ready', 0)}"
            ),
        },
        {
            "entry_id": "summary-interactions",
            "category": "interactions",
            "total": str(len(interaction_entries)),
            "metrics": (
                f"covered={interaction_status_counts.get('covered', 0)} "
                f"watch={interaction_status_counts.get('watch', 0)} "
                f"missing={interaction_status_counts.get('missing', 0)}"
            ),
        },
        {
            "entry_id": "summary-questions",
            "category": "questions",
            "total": str(len(question_entries)),
            "metrics": (
                f"linked={question_link_counts.get('linked', 0)} "
                f"orphan={question_link_counts.get('orphan', 0)} "
                f"owners={len(question_owners)}"
            ),
        },
        {
            "entry_id": "summary-actions",
            "category": "actions",
            "total": str(len(action_entries)),
            "metrics": (
                f"queue={action_lane_counts.get('queue', 0)} "
                f"reminder={action_lane_counts.get('reminder', 0)} "
                f"renewal={action_lane_counts.get('renewal', 0)}"
            ),
        },
    ]


def _build_role_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    signoffs_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    entries = [
        {
            "entry_id": f"cover-{assignment.assignment_id}",
            "assignment_id": assignment.assignment_id,
            "surface_id": assignment.surface_id,
            "role": assignment.role,
            "status": assignment.status,
            "responsibility_count": str(len(assignment.responsibilities)),
            "checklist_count": str(len(assignment.checklist_item_ids)),
            "decision_count": str(len(assignment.decision_ids)),
            "signoff_id": signoffs_by_assignment.get(assignment.assignment_id).signoff_id if assignment.assignment_id in signoffs_by_assignment else "none",
            "signoff_status": signoffs_by_assignment.get(assignment.assignment_id).status if assignment.assignment_id in signoffs_by_assignment else "none",
            "summary": ",".join(assignment.responsibilities) or "none",
        }
        for assignment in pack.role_matrix
    ]
    return sorted(entries, key=lambda item: (item["surface_id"], item["status"], item["assignment_id"]))


def _build_owner_workload_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for entry in _build_owner_review_queue_entries(pack):
        entries.append(
            {
                "entry_id": f"load-{entry['queue_id']}",
                "owner": entry["owner"],
                "item_type": entry["item_type"],
                "source_id": entry["source_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "lane": "queue",
                "detail": entry["next_action"],
                "summary": entry["summary"],
            }
        )
    for entry in _build_signoff_reminder_entries(pack):
        entries.append(
            {
                "entry_id": f"load-{entry['entry_id']}",
                "owner": entry["reminder_owner"],
                "item_type": "reminder",
                "source_id": entry["signoff_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "lane": "reminder",
                "detail": entry["next_reminder_at"],
                "summary": entry["summary"],
            }
        )
    for entry in _build_freeze_renewal_entries(pack):
        if entry["renewal_status"] == "not-needed":
            continue
        entries.append(
            {
                "entry_id": f"load-{entry['entry_id']}",
                "owner": entry["renewal_owner"],
                "item_type": "renewal",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["renewal_status"],
                "lane": "renewal",
                "detail": entry["renewal_by"],
                "summary": entry["summary"],
            }
        )
    return sorted(entries, key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]))


def _build_signoff_dependency_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignment_by_id = {assignment.assignment_id: assignment for assignment in pack.role_matrix}
    timeline_index = _build_blocker_timeline_index(pack)
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        assignment = assignment_by_id.get(signoff.assignment_id)
        blockers = unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
        latest_event = None
        for blocker in blockers:
            events = timeline_index.get(blocker.blocker_id, [])
            if events:
                candidate = events[-1]
                if latest_event is None or (candidate.timestamp, candidate.event_id) > (latest_event.timestamp, latest_event.event_id):
                    latest_event = candidate
        latest_label = (
            f"{latest_event.event_id}/{latest_event.status}/{latest_event.actor}@{latest_event.timestamp}"
            if latest_event
            else "none"
        )
        entries.append(
            {
                "entry_id": f"dep-{signoff.signoff_id}",
                "signoff_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "role": signoff.role,
                "status": signoff.status,
                "assignment_id": signoff.assignment_id,
                "dependency_status": "blocked" if blockers else "clear",
                "checklist_ids": ",".join(assignment.checklist_item_ids) if assignment else "none",
                "decision_ids": ",".join(assignment.decision_ids) if assignment else "none",
                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",
                "blocker_owners": ",".join(sorted({blocker.owner for blocker in blockers})) or "none",
                "latest_blocker_event": latest_label,
                "sla_status": signoff.sla_status,
                "due_at": signoff.due_at or "none",
                "reminder_cadence": signoff.reminder_cadence or "none",
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
            }
        )
    return sorted(entries, key=lambda item: (item["dependency_status"], item["due_at"], item["signoff_id"]))


def _build_audit_density_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    blocker_done_statuses = {"resolved", "closed"}
    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        checklist_by_surface.setdefault(item.surface_id, []).append(item)
    decision_by_surface: Dict[str, List[ReviewDecision]] = {}
    for decision in pack.decision_log:
        decision_by_surface.setdefault(decision.surface_id, []).append(decision)
    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)
    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}
    for signoff in pack.signoff_log:
        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)
    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}
    blocker_surface_by_id: Dict[str, str] = {}
    for blocker in pack.blocker_log:
        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)
        blocker_surface_by_id[blocker.blocker_id] = blocker.surface_id
    timeline_by_surface: Dict[str, List[ReviewBlockerEvent]] = {}
    for event in pack.blocker_timeline:
        surface_id = blocker_surface_by_id.get(event.blocker_id, "none")
        timeline_by_surface.setdefault(surface_id, []).append(event)
    entries: List[Dict[str, str]] = []
    for wireframe in pack.wireframes:
        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])
        decisions = decision_by_surface.get(wireframe.surface_id, [])
        assignments = assignment_by_surface.get(wireframe.surface_id, [])
        signoffs = signoff_by_surface.get(wireframe.surface_id, [])
        blockers = blocker_by_surface.get(wireframe.surface_id, [])
        timeline_events = timeline_by_surface.get(wireframe.surface_id, [])
        open_total = (
            sum(1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses)
            + sum(1 for decision in decisions if decision.status.lower() not in decision_ready_statuses)
            + sum(1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses)
            + sum(1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses)
            + sum(1 for blocker in blockers if blocker.status.lower() not in blocker_done_statuses)
        )
        artifact_total = len(checklist_items) + len(decisions) + len(assignments) + len(signoffs) + len(blockers) + len(timeline_events)
        load_band = "dense" if open_total >= 4 else "active" if open_total >= 2 else "light"
        entries.append(
            {
                "entry_id": f"density-{wireframe.surface_id}",
                "surface_id": wireframe.surface_id,
                "artifact_total": str(artifact_total),
                "open_total": str(open_total),
                "load_band": load_band,
                "block_count": str(len(wireframe.primary_blocks)),
                "note_count": str(len(wireframe.review_notes)),
                "checklist_count": str(len(checklist_items)),
                "decision_count": str(len(decisions)),
                "assignment_count": str(len(assignments)),
                "signoff_count": str(len(signoffs)),
                "blocker_count": str(len(blockers)),
                "timeline_count": str(len(timeline_events)),
            }
        )
    return sorted(entries, key=lambda item: (-int(item["open_total"]), item["surface_id"]))


def _build_signoff_sla_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries = [
        {
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "requested_at": signoff.requested_at or "none",
            "due_at": signoff.due_at or "none",
            "escalation_owner": signoff.escalation_owner or "none",
            "required": "yes" if signoff.required else "no",
            "evidence": ",".join(signoff.evidence_links) or "none",
        }
        for signoff in pack.signoff_log
    ]
    return sorted(entries, key=lambda item: (item["due_at"], item["sla_status"], item["signoff_id"]))


def _build_escalation_dashboard_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    signoff_done_statuses = {"approved", "accepted", "resolved"}
    blocker_done_statuses = {"resolved", "closed"}
    for signoff in pack.signoff_log:
        if signoff.status.lower() in signoff_done_statuses:
            continue
        entries.append(
            {
                "escalation_id": f"esc-{signoff.signoff_id}",
                "escalation_owner": signoff.escalation_owner or "none",
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "status": signoff.status,
                "priority": signoff.sla_status,
                "current_owner": signoff.role,
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
                "due_at": signoff.due_at or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in blocker_done_statuses:
            continue
        entries.append(
            {
                "escalation_id": f"esc-{blocker.blocker_id}",
                "escalation_owner": blocker.escalation_owner or "none",
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "priority": blocker.severity,
                "current_owner": blocker.owner,
                "summary": blocker.summary,
                "due_at": "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["escalation_owner"], item["item_type"], item["surface_id"], item["source_id"]),
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
            f"{objective.objective_id}: {objective.title} persona={objective.persona} priority={objective.priority}"
        )
        lines.append(
            "  "
            f"outcome={objective.outcome} success_signal={objective.success_signal} dependencies={','.join(objective.dependencies) or 'none'}"
        )

    review_summary_entries = _build_review_summary_entries(pack)
    lines.append("")
    lines.append("## Review Summary Board")
    lines.append(f"- Categories: {len(review_summary_entries)}")
    lines.append("")
    lines.append("### Entries")
    for entry in review_summary_entries:
        lines.append(
            f"- {entry['entry_id']}: category={entry['category']} total={entry['total']} {entry['metrics']}"
        )
    if not review_summary_entries:
        lines.append("- none")

    objective_coverage_entries = _build_objective_coverage_entries(pack)
    objective_persona_counts: Dict[str, int] = {}
    objective_status_counts: Dict[str, int] = {}
    for entry in objective_coverage_entries:
        objective_persona_counts[entry['persona']] = objective_persona_counts.get(entry['persona'], 0) + 1
        objective_status_counts[entry['coverage_status']] = objective_status_counts.get(entry['coverage_status'], 0) + 1

    lines.append("")
    lines.append("## Objective Coverage Board")
    lines.append(f"- Objectives: {len(objective_coverage_entries)}")
    lines.append(f"- Personas: {len(objective_persona_counts)}")
    lines.append("")
    lines.append("### By Coverage Status")
    for status, count in sorted(objective_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not objective_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Persona")
    for persona, count in sorted(objective_persona_counts.items()):
        lines.append(f"- {persona}: {count}")
    if not objective_persona_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in objective_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: objective={entry['objective_id']} persona={entry['persona']} priority={entry['priority']} coverage={entry['coverage_status']} dependencies={entry['dependency_count']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  dependency_ids={entry['dependency_ids']} assignments={entry['assignment_ids']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} summary={entry['summary']}"
        )
    if not objective_coverage_entries:
        lines.append("- none")

    persona_readiness_entries = _build_persona_readiness_entries(pack)
    persona_readiness_counts: Dict[str, int] = {}
    for entry in persona_readiness_entries:
        persona_readiness_counts[entry['readiness']] = persona_readiness_counts.get(entry['readiness'], 0) + 1

    lines.append("")
    lines.append("## Persona Readiness Board")
    lines.append(f"- Personas: {len(persona_readiness_entries)}")
    lines.append(f"- Objectives: {len(pack.objectives)}")
    lines.append("")
    lines.append("### By Readiness")
    for readiness, count in sorted(persona_readiness_counts.items()):
        lines.append(f"- {readiness}: {count}")
    if not persona_readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in persona_readiness_entries:
        lines.append(
            f"- {entry['entry_id']}: persona={entry['persona']} readiness={entry['readiness']} objectives={entry['objective_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} open_questions={entry['question_count']} queue_items={entry['queue_count']} blockers={entry['blocker_count']}"
        )
        lines.append(
            f"  objective_ids={entry['objective_ids']} surfaces={entry['surface_ids']} queue_ids={entry['queue_ids']} blocker_ids={entry['blocker_ids']}"
        )
    if not persona_readiness_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Wireframes")
    for wireframe in pack.wireframes:
        lines.append(
            "- "
            f"{wireframe.surface_id}: {wireframe.name} device={wireframe.device} entry={wireframe.entry_point}"
        )
        lines.append(
            "  "
            f"blocks={','.join(wireframe.primary_blocks) or 'none'} review_notes={','.join(wireframe.review_notes) or 'none'}"
        )

    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)
    wireframe_readiness_counts: Dict[str, int] = {}
    wireframe_device_counts: Dict[str, int] = {}
    for entry in wireframe_readiness_entries:
        wireframe_readiness_counts[entry['readiness_status']] = wireframe_readiness_counts.get(entry['readiness_status'], 0) + 1
        wireframe_device_counts[entry['device']] = wireframe_device_counts.get(entry['device'], 0) + 1

    lines.append("")
    lines.append("## Wireframe Readiness Board")
    lines.append(f"- Wireframes: {len(wireframe_readiness_entries)}")
    lines.append(f"- Devices: {len(wireframe_device_counts)}")
    lines.append("")
    lines.append("### By Readiness")
    for status, count in sorted(wireframe_readiness_counts.items()):
        lines.append(f"- {status}: {count}")
    if not wireframe_readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Device")
    for device, count in sorted(wireframe_device_counts.items()):
        lines.append(f"- {device}: {count}")
    if not wireframe_device_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in wireframe_readiness_entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} device={entry['device']} readiness={entry['readiness_status']} open_total={entry['open_total']} entry={entry['entry_point']}"
        )
        lines.append(
            f"  checklist_open={entry['checklist_open']} decisions_open={entry['decisions_open']} assignments_open={entry['assignments_open']} signoffs_open={entry['signoffs_open']} blockers_open={entry['blockers_open']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} blocks={entry['block_count']} notes={entry['note_count']} summary={entry['summary']}"
        )
    if not wireframe_readiness_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Interactions")
    for interaction in pack.interactions:
        lines.append(
            "- "
            f"{interaction.flow_id}: {interaction.name} trigger={interaction.trigger}"
        )
        lines.append(
            "  "
            f"response={interaction.system_response} states={','.join(interaction.states) or 'none'} exceptions={','.join(interaction.exceptions) or 'none'}"
        )

    interaction_coverage_entries = _build_interaction_coverage_entries(pack)
    interaction_coverage_counts: Dict[str, int] = {}
    interaction_surface_counts: Dict[str, int] = {}
    for entry in interaction_coverage_entries:
        interaction_coverage_counts[entry['coverage_status']] = interaction_coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1

    lines.append("")
    lines.append("## Interaction Coverage Board")
    lines.append(f"- Interactions: {len(interaction_coverage_entries)}")
    lines.append(f"- Surfaces: {len(interaction_surface_counts)}")
    lines.append("")
    lines.append("### By Coverage Status")
    for status, count in sorted(interaction_coverage_counts.items()):
        lines.append(f"- {status}: {count}")
    if not interaction_coverage_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, count in sorted(interaction_surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not interaction_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in interaction_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: flow={entry['flow_id']} surfaces={entry['surface_ids']} owners={entry['owners']} coverage={entry['coverage_status']} states={entry['state_count']} exceptions={entry['exception_count']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} open_checklist={entry['open_checklist_ids']} trigger={entry['summary']}"
        )
    if not interaction_coverage_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Open Questions")
    for question in pack.open_questions:
        lines.append(
            "- "
            f"{question.question_id}: {question.theme} owner={question.owner} status={question.status}"
        )
        lines.append("  " f"question={question.question} impact={question.impact}")

    open_question_entries = _build_open_question_tracker_entries(pack)
    open_question_owner_counts: Dict[str, int] = {}
    open_question_theme_counts: Dict[str, int] = {}
    for entry in open_question_entries:
        open_question_owner_counts[entry['owner']] = open_question_owner_counts.get(entry['owner'], 0) + 1
        open_question_theme_counts[entry['theme']] = open_question_theme_counts.get(entry['theme'], 0) + 1

    lines.append("")
    lines.append("## Open Question Tracker")
    lines.append(f"- Questions: {len(open_question_entries)}")
    lines.append(f"- Owners: {len(open_question_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(open_question_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not open_question_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Theme")
    for theme, count in sorted(open_question_theme_counts.items()):
        lines.append(f"- {theme}: {count}")
    if not open_question_theme_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in open_question_entries:
        lines.append(
            f"- {entry['entry_id']}: question={entry['question_id']} owner={entry['owner']} theme={entry['theme']} status={entry['status']} link_status={entry['link_status']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} flows={entry['flow_ids']} impact={entry['impact']} prompt={entry['summary']}"
        )
    if not open_question_entries:
        lines.append("- none")

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

    lines.append("")
    lines.append("## Role Matrix")
    for assignment in pack.role_matrix:
        lines.append(
            "- "
            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"
        )
        lines.append(
            "  "
            f"responsibilities={','.join(assignment.responsibilities) or 'none'} checklist={','.join(assignment.checklist_item_ids) or 'none'} decisions={','.join(assignment.decision_ids) or 'none'}"
        )
    if not pack.role_matrix:
        lines.append("- none")

    checklist_trace_entries = _build_checklist_traceability_entries(pack)
    checklist_trace_owner_counts: Dict[str, int] = {}
    checklist_trace_status_counts: Dict[str, int] = {}
    for entry in checklist_trace_entries:
        checklist_trace_owner_counts[entry['owner']] = checklist_trace_owner_counts.get(entry['owner'], 0) + 1
        checklist_trace_status_counts[entry['status']] = checklist_trace_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Checklist Traceability Board")
    lines.append(f"- Checklist items: {len(checklist_trace_entries)}")
    lines.append(f"- Owners: {len(checklist_trace_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(checklist_trace_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not checklist_trace_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(checklist_trace_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not checklist_trace_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in checklist_trace_entries:
        lines.append(
            f"- {entry['entry_id']}: item={entry['item_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_decisions={entry['linked_decisions']} evidence={entry['evidence']} summary={entry['summary']}"
        )
    if not checklist_trace_entries:
        lines.append("- none")

    decision_followup_entries = _build_decision_followup_entries(pack)
    decision_followup_owner_counts: Dict[str, int] = {}
    decision_followup_status_counts: Dict[str, int] = {}
    for entry in decision_followup_entries:
        decision_followup_owner_counts[entry['owner']] = decision_followup_owner_counts.get(entry['owner'], 0) + 1
        decision_followup_status_counts[entry['status']] = decision_followup_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Decision Follow-up Tracker")
    lines.append(f"- Decisions: {len(decision_followup_entries)}")
    lines.append(f"- Owners: {len(decision_followup_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(decision_followup_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not decision_followup_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(decision_followup_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not decision_followup_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in decision_followup_entries:
        lines.append(
            f"- {entry['entry_id']}: decision={entry['decision_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_checklists={entry['linked_checklists']} follow_up={entry['follow_up']} summary={entry['summary']}"
        )
    if not decision_followup_entries:
        lines.append("- none")

    role_coverage_entries = _build_role_coverage_entries(pack)
    role_coverage_surface_counts: Dict[str, int] = {}
    role_coverage_status_counts: Dict[str, int] = {}
    for entry in role_coverage_entries:
        role_coverage_surface_counts[entry['surface_id']] = role_coverage_surface_counts.get(entry['surface_id'], 0) + 1
        role_coverage_status_counts[entry['status']] = role_coverage_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Role Coverage Board")
    lines.append(f"- Assignments: {len(role_coverage_entries)}")
    lines.append(f"- Surfaces: {len(role_coverage_surface_counts)}")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, count in sorted(role_coverage_surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not role_coverage_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(role_coverage_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not role_coverage_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in role_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: assignment={entry['assignment_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} responsibilities={entry['responsibility_count']} checklist={entry['checklist_count']} decisions={entry['decision_count']}"
        )
        lines.append(
            f"  signoff={entry['signoff_id']} signoff_status={entry['signoff_status']} summary={entry['summary']}"
        )
    if not role_coverage_entries:
        lines.append("- none")

    signoff_dependency_entries = _build_signoff_dependency_entries(pack)
    dependency_counts: Dict[str, int] = {}
    dependency_sla_counts: Dict[str, int] = {}
    for entry in signoff_dependency_entries:
        dependency_counts[entry['dependency_status']] = dependency_counts.get(entry['dependency_status'], 0) + 1
        dependency_sla_counts[entry['sla_status']] = dependency_sla_counts.get(entry['sla_status'], 0) + 1

    lines.append("")
    lines.append("## Signoff Dependency Board")
    lines.append(f"- Sign-offs: {len(signoff_dependency_entries)}")
    lines.append(f"- Dependency states: {len(dependency_counts)}")
    lines.append("")
    lines.append("### By Dependency Status")
    for status, count in sorted(dependency_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By SLA State")
    for status, count in sorted(dependency_sla_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_sla_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in signoff_dependency_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} dependency_status={entry['dependency_status']} blockers={entry['blocker_ids']}"
        )
        lines.append(
            f"  assignment={entry['assignment_id']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} latest_blocker_event={entry['latest_blocker_event']} sla={entry['sla_status']} due_at={entry['due_at']} cadence={entry['reminder_cadence']} summary={entry['summary']}"
        )
    if not signoff_dependency_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Sign-off Log")
    for signoff in pack.signoff_log:
        lines.append(
            "- "
            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"
        )
        lines.append(
            "  "
            f"required={'yes' if signoff.required else 'no'} evidence={','.join(signoff.evidence_links) or 'none'} notes={signoff.notes or 'none'} waiver_owner={signoff.waiver_owner or 'none'} waiver_reason={signoff.waiver_reason or 'none'} requested_at={signoff.requested_at or 'none'} due_at={signoff.due_at or 'none'} escalation_owner={signoff.escalation_owner or 'none'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or 'none'} reminder_channel={signoff.reminder_channel or 'none'} last_reminder_at={signoff.last_reminder_at or 'none'} next_reminder_at={signoff.next_reminder_at or 'none'}"
        )
    if not pack.signoff_log:
        lines.append("- none")

    signoff_sla_entries = _build_signoff_sla_entries(pack)
    sla_state_counts: Dict[str, int] = {}
    sla_owner_counts: Dict[str, int] = {}
    for entry in signoff_sla_entries:
        sla_state_counts[entry['sla_status']] = sla_state_counts.get(entry['sla_status'], 0) + 1
        sla_owner_counts[entry['escalation_owner']] = sla_owner_counts.get(entry['escalation_owner'], 0) + 1

    lines.append("")
    lines.append("## Sign-off SLA Dashboard")
    lines.append(f"- Sign-offs: {len(signoff_sla_entries)}")
    lines.append(f"- Escalation owners: {len(sla_owner_counts)}")
    lines.append("")
    lines.append("### SLA States")
    for sla_status, count in sorted(sla_state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not sla_state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalation Owners")
    for owner, count in sorted(sla_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not sla_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Sign-offs")
    for entry in signoff_sla_entries:
        lines.append(
            f"- {entry['signoff_id']}: role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} requested_at={entry['requested_at']} due_at={entry['due_at']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(f"  required={entry['required']} evidence={entry['evidence']}")
    if not signoff_sla_entries:
        lines.append("- none")

    signoff_reminder_entries = _build_signoff_reminder_entries(pack)
    reminder_owner_counts: Dict[str, int] = {}
    reminder_channel_counts: Dict[str, int] = {}
    for entry in signoff_reminder_entries:
        reminder_owner_counts[entry['reminder_owner']] = reminder_owner_counts.get(entry['reminder_owner'], 0) + 1
        reminder_channel_counts[entry['reminder_channel']] = reminder_channel_counts.get(entry['reminder_channel'], 0) + 1

    lines.append("")
    lines.append("## Sign-off Reminder Queue")
    lines.append(f"- Reminders: {len(signoff_reminder_entries)}")
    lines.append(f"- Owners: {len(reminder_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(reminder_owner_counts.items()):
        lines.append(f"- {owner}: reminders={count}")
    if not reminder_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Channel")
    for channel, count in sorted(reminder_channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not reminder_channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in signoff_reminder_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} owner={entry['reminder_owner']} channel={entry['reminder_channel']}"
        )
        lines.append(
            f"  last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not signoff_reminder_entries:
        lines.append("- none")

    reminder_cadence_entries = _build_reminder_cadence_entries(pack)
    reminder_cadence_counts: Dict[str, int] = {}
    reminder_status_counts: Dict[str, int] = {}
    for entry in reminder_cadence_entries:
        reminder_cadence_counts[entry['reminder_cadence']] = reminder_cadence_counts.get(entry['reminder_cadence'], 0) + 1
        reminder_status_counts[entry['reminder_status']] = reminder_status_counts.get(entry['reminder_status'], 0) + 1

    lines.append("")
    lines.append("## Reminder Cadence Board")
    lines.append(f"- Items: {len(reminder_cadence_entries)}")
    lines.append(f"- Cadences: {len(reminder_cadence_counts)}")
    lines.append("")
    lines.append("### By Cadence")
    for cadence, count in sorted(reminder_cadence_counts.items()):
        lines.append(f"- {cadence}: {count}")
    if not reminder_cadence_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(reminder_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not reminder_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in reminder_cadence_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} cadence={entry['reminder_cadence']} status={entry['reminder_status']} owner={entry['reminder_owner']}"
        )
        lines.append(
            f"  sla={entry['sla_status']} last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not reminder_cadence_entries:
        lines.append("- none")

    signoff_breach_entries = _build_signoff_breach_entries(pack)
    breach_state_counts: Dict[str, int] = {}
    breach_owner_counts: Dict[str, int] = {}
    for entry in signoff_breach_entries:
        breach_state_counts[entry['sla_status']] = breach_state_counts.get(entry['sla_status'], 0) + 1
        breach_owner_counts[entry['escalation_owner']] = breach_owner_counts.get(entry['escalation_owner'], 0) + 1

    lines.append("")
    lines.append("## Sign-off Breach Board")
    lines.append(f"- Breach items: {len(signoff_breach_entries)}")
    lines.append(f"- Escalation owners: {len(breach_owner_counts)}")
    lines.append("")
    lines.append("### SLA States")
    for sla_status, count in sorted(breach_state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not breach_state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalation Owners")
    for owner, count in sorted(breach_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not breach_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in signoff_breach_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(
            f"  requested_at={entry['requested_at']} due_at={entry['due_at']} linked_blockers={entry['linked_blockers']} summary={entry['summary']}"
        )
    if not signoff_breach_entries:
        lines.append("- none")

    escalation_entries = _build_escalation_dashboard_entries(pack)
    escalation_owner_counts: Dict[str, Dict[str, int]] = {}
    escalation_status_counts: Dict[str, Dict[str, int]] = {}
    for entry in escalation_entries:
        owner_counts = escalation_owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_counts[entry['item_type']] += 1
        owner_counts['total'] += 1
        status_counts = escalation_status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_counts[entry['item_type']] += 1
        status_counts['total'] += 1

    lines.append("")
    lines.append("## Escalation Dashboard")
    lines.append(f"- Items: {len(escalation_entries)}")
    lines.append(f"- Escalation owners: {len(escalation_owner_counts)}")
    lines.append("")
    lines.append("### By Escalation Owner")
    for owner, counts in sorted(escalation_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not escalation_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, counts in sorted(escalation_status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not escalation_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalations")
    for entry in escalation_entries:
        lines.append(
            f"- {entry['escalation_id']}: owner={entry['escalation_owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} priority={entry['priority']} current_owner={entry['current_owner']}"
        )
        lines.append(f"  summary={entry['summary']} due_at={entry['due_at']}")
    if not escalation_entries:
        lines.append("- none")

    escalation_handoff_entries = _build_escalation_handoff_entries(pack)
    handoff_status_counts: Dict[str, int] = {}
    handoff_channel_counts: Dict[str, int] = {}
    for entry in escalation_handoff_entries:
        handoff_status_counts[entry['status']] = handoff_status_counts.get(entry['status'], 0) + 1
        handoff_channel_counts[entry['channel']] = handoff_channel_counts.get(entry['channel'], 0) + 1

    lines.append("")
    lines.append("## Escalation Handoff Ledger")
    lines.append(f"- Handoffs: {len(escalation_handoff_entries)}")
    lines.append(f"- Channels: {len(handoff_channel_counts)}")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(handoff_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not handoff_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Channel")
    for channel, count in sorted(handoff_channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not handoff_channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in escalation_handoff_entries:
        lines.append(
            f"- {entry['ledger_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} actor={entry['actor']} status={entry['status']} at={entry['timestamp']}"
        )
        lines.append(
            f"  from={entry['handoff_from']} to={entry['handoff_to']} channel={entry['channel']} artifact={entry['artifact_ref']} next_action={entry['next_action']}"
        )
    if not escalation_handoff_entries:
        lines.append("- none")

    handoff_ack_entries = _build_handoff_ack_entries(pack)
    handoff_ack_owner_counts: Dict[str, int] = {}
    handoff_ack_status_counts: Dict[str, int] = {}
    for entry in handoff_ack_entries:
        handoff_ack_owner_counts[entry['ack_owner']] = handoff_ack_owner_counts.get(entry['ack_owner'], 0) + 1
        handoff_ack_status_counts[entry['ack_status']] = handoff_ack_status_counts.get(entry['ack_status'], 0) + 1

    lines.append("")
    lines.append("## Handoff Ack Ledger")
    lines.append(f"- Ack items: {len(handoff_ack_entries)}")
    lines.append(f"- Ack owners: {len(handoff_ack_owner_counts)}")
    lines.append("")
    lines.append("### By Ack Owner")
    for owner, count in sorted(handoff_ack_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not handoff_ack_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Ack Status")
    for status, count in sorted(handoff_ack_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not handoff_ack_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in handoff_ack_entries:
        lines.append(
            f"- {entry['entry_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} handoff_to={entry['handoff_to']} ack_owner={entry['ack_owner']} ack_status={entry['ack_status']} ack_at={entry['ack_at']}"
        )
        lines.append(
            f"  actor={entry['actor']} status={entry['status']} channel={entry['channel']} artifact={entry['artifact_ref']} summary={entry['summary']}"
        )
    if not handoff_ack_entries:
        lines.append("- none")

    owner_digest_entries = _build_owner_escalation_digest_entries(pack)
    owner_digest_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_digest_entries:
        counts = owner_digest_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'signoff': 0, 'reminder': 0, 'freeze': 0, 'handoff': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1

    lines.append("")
    lines.append("## Owner Escalation Digest")
    lines.append(f"- Owners: {len(owner_digest_counts)}")
    lines.append(f"- Items: {len(owner_digest_entries)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_digest_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} reminders={counts['reminder']} freezes={counts['freeze']} handoffs={counts['handoff']} total={counts['total']}"
        )
    if not owner_digest_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_digest_entries:
        lines.append(
            f"- {entry['digest_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} detail={entry['detail']}")
    if not owner_digest_entries:
        lines.append("- none")

    owner_workload_entries = _build_owner_workload_entries(pack)
    owner_workload_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_workload_entries:
        counts = owner_workload_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'checklist': 0, 'decision': 0, 'signoff': 0, 'reminder': 0, 'renewal': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1

    lines.append("")
    lines.append("## Owner Workload Board")
    lines.append(f"- Owners: {len(owner_workload_counts)}")
    lines.append(f"- Items: {len(owner_workload_entries)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_workload_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} reminders={counts['reminder']} renewals={counts['renewal']} total={counts['total']}"
        )
    if not owner_workload_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_workload_entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} lane={entry['lane']}"
        )
        lines.append(f"  detail={entry['detail']} summary={entry['summary']}")
    if not owner_workload_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Blocker Log")
    for blocker in pack.blocker_log:
        lines.append(
            "- "
            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"
        )
        lines.append(
            "  "
            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or 'none'} next_action={blocker.next_action or 'none'} freeze_owner={blocker.freeze_owner or 'none'} freeze_until={blocker.freeze_until or 'none'} freeze_approved_by={blocker.freeze_approved_by or 'none'} freeze_approved_at={blocker.freeze_approved_at or 'none'}"
        )
    if not pack.blocker_log:
        lines.append("- none")

    lines.append("")
    lines.append("## Blocker Timeline")
    for event in pack.blocker_timeline:
        lines.append(
            "- "
            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"
        )
        lines.append(
            "  "
            f"summary={event.summary} next_action={event.next_action or 'none'}"
        )
    if not pack.blocker_timeline:
        lines.append("- none")

    exception_entries = _build_review_exception_entries(pack)
    timeline_index = _build_blocker_timeline_index(pack)

    lines.append("")
    lines.append("## Review Exceptions")
    for entry in exception_entries:
        lines.append(
            f"- {entry['exception_id']}: type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")

    freeze_entries = _build_freeze_exception_entries(pack)
    freeze_owner_counts: Dict[str, Dict[str, int]] = {}
    freeze_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in freeze_entries:
        owner_counts = freeze_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_counts[entry["item_type"]] += 1
        owner_counts["total"] += 1
        surface_counts = freeze_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_counts[entry["item_type"]] += 1
        surface_counts["total"] += 1

    lines.append("")
    lines.append("## Review Freeze Exception Board")
    lines.append(f"- Exceptions: {len(freeze_entries)}")
    lines.append(f"- Owners: {len(freeze_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, counts in sorted(freeze_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not freeze_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, counts in sorted(freeze_surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not freeze_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} window={entry['window']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} next_action={entry['next_action']}"
        )
    if not freeze_entries:
        lines.append("- none")

    freeze_approval_entries = _build_freeze_approval_entries(pack)
    freeze_approval_owner_counts: Dict[str, int] = {}
    freeze_approval_status_counts: Dict[str, int] = {}
    for entry in freeze_approval_entries:
        freeze_approval_owner_counts[entry['freeze_approved_by']] = freeze_approval_owner_counts.get(entry['freeze_approved_by'], 0) + 1
        freeze_approval_status_counts[entry['status']] = freeze_approval_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Freeze Approval Trail")
    lines.append(f"- Approvals: {len(freeze_approval_entries)}")
    lines.append(f"- Approvers: {len(freeze_approval_owner_counts)}")
    lines.append("")
    lines.append("### By Approver")
    for owner, count in sorted(freeze_approval_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not freeze_approval_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(freeze_approval_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not freeze_approval_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_approval_entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} owner={entry['freeze_owner']} approved_by={entry['freeze_approved_by']} approved_at={entry['freeze_approved_at']} window={entry['freeze_until']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not freeze_approval_entries:
        lines.append("- none")

    freeze_renewal_entries = _build_freeze_renewal_entries(pack)
    freeze_renewal_owner_counts: Dict[str, int] = {}
    freeze_renewal_status_counts: Dict[str, int] = {}
    for entry in freeze_renewal_entries:
        freeze_renewal_owner_counts[entry['renewal_owner']] = freeze_renewal_owner_counts.get(entry['renewal_owner'], 0) + 1
        freeze_renewal_status_counts[entry['renewal_status']] = freeze_renewal_status_counts.get(entry['renewal_status'], 0) + 1

    lines.append("")
    lines.append("## Freeze Renewal Tracker")
    lines.append(f"- Renewal items: {len(freeze_renewal_entries)}")
    lines.append(f"- Renewal owners: {len(freeze_renewal_owner_counts)}")
    lines.append("")
    lines.append("### By Renewal Owner")
    for owner, count in sorted(freeze_renewal_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not freeze_renewal_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Renewal Status")
    for status, count in sorted(freeze_renewal_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not freeze_renewal_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_renewal_entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} renewal_owner={entry['renewal_owner']} renewal_by={entry['renewal_by']} renewal_status={entry['renewal_status']}"
        )
        lines.append(
            f"  freeze_owner={entry['freeze_owner']} freeze_until={entry['freeze_until']} approved_by={entry['freeze_approved_by']} summary={entry['summary']} next_action={entry['next_action']}"
        )
    if not freeze_renewal_entries:
        lines.append("- none")

    exception_owner_counts: Dict[str, Dict[str, int]] = {}
    exception_status_counts: Dict[str, Dict[str, int]] = {}
    exception_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_counts = exception_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_counts[entry["category"]] += 1
        owner_counts["total"] += 1
        status_counts = exception_status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_counts[entry["category"]] += 1
        status_counts["total"] += 1
        surface_counts = exception_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_counts[entry["category"]] += 1
        surface_counts["total"] += 1

    lines.append("")
    lines.append("## Review Exception Matrix")
    lines.append(f"- Exceptions: {len(exception_entries)}")
    lines.append(f"- Owners: {len(exception_owner_counts)}")
    lines.append(f"- Surfaces: {len(exception_surface_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, counts in sorted(exception_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, counts in sorted(exception_status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, counts in sorted(exception_surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_surface_counts:
        lines.append("- none")

    audit_density_entries = _build_audit_density_entries(pack)
    audit_density_band_counts: Dict[str, int] = {}
    for entry in audit_density_entries:
        audit_density_band_counts[entry['load_band']] = audit_density_band_counts.get(entry['load_band'], 0) + 1

    lines.append("")
    lines.append("## Audit Density Board")
    lines.append(f"- Surfaces: {len(audit_density_entries)}")
    lines.append(f"- Load bands: {len(audit_density_band_counts)}")
    lines.append("")
    lines.append("### By Load Band")
    for band, count in sorted(audit_density_band_counts.items()):
        lines.append(f"- {band}: {count}")
    if not audit_density_band_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in audit_density_entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} artifact_total={entry['artifact_total']} open_total={entry['open_total']} band={entry['load_band']}"
        )
        lines.append(
            f"  checklist={entry['checklist_count']} decisions={entry['decision_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} blockers={entry['blocker_count']} timeline={entry['timeline_count']} blocks={entry['block_count']} notes={entry['note_count']}"
        )
    if not audit_density_entries:
        lines.append("- none")

    owner_review_queue = _build_owner_review_queue_entries(pack)
    owner_queue_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_review_queue:
        counts = owner_queue_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1

    lines.append("")
    lines.append("## Owner Review Queue")
    lines.append(f"- Owners: {len(owner_queue_counts)}")
    lines.append(f"- Queue items: {len(owner_review_queue)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_queue_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_queue_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_review_queue:
        lines.append(
            f"- {entry['queue_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} next_action={entry['next_action']}")
    if not owner_review_queue:
        lines.append("- none")

    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )

    lines.append("")
    lines.append("## Blocker Timeline Summary")
    lines.append(f"- Total events: {len(pack.blocker_timeline)}")
    lines.append(f"- Blockers with timeline: {len(timeline_index)}")
    lines.append(f"- Orphan timeline blockers: {','.join(orphan_timeline_ids) or 'none'}")
    lines.append("")
    lines.append("### Events by Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Events by Actor")
    for actor, count in sorted(actor_counts.items()):
        lines.append(f"- {actor}: {count}")
    if not actor_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Latest Blocker Events")
    for blocker in pack.blocker_log:
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        if latest is None:
            lines.append(f"- {blocker.blocker_id}: latest=none")
            continue
        lines.append(
            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"
        )
    if not pack.blocker_log:
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
            f"- Checklist items missing role links: {', '.join(audit.checklist_items_missing_role_links) or 'none'}",
            f"- Wireframes missing decision coverage: {', '.join(audit.wireframes_missing_decisions) or 'none'}",
            f"- Orphan decision surfaces: {', '.join(audit.orphan_decision_surfaces) or 'none'}",
            f"- Unresolved decision ids: {', '.join(audit.unresolved_decision_ids) or 'none'}",
            f"- Unresolved decisions missing follow-ups: {', '.join(audit.unresolved_decisions_missing_follow_ups) or 'none'}",
            f"- Wireframes missing role assignments: {', '.join(audit.wireframes_missing_role_assignments) or 'none'}",
            f"- Orphan role assignment surfaces: {', '.join(audit.orphan_role_assignment_surfaces) or 'none'}",
            f"- Role assignments missing responsibilities: {', '.join(audit.role_assignments_missing_responsibilities) or 'none'}",
            f"- Role assignments missing checklist links: {', '.join(audit.role_assignments_missing_checklist_links) or 'none'}",
            f"- Role assignments missing decision links: {', '.join(audit.role_assignments_missing_decision_links) or 'none'}",
            f"- Decisions missing role links: {', '.join(audit.decisions_missing_role_links) or 'none'}",
            f"- Wireframes missing signoff coverage: {', '.join(audit.wireframes_missing_signoffs) or 'none'}",
            f"- Orphan signoff surfaces: {', '.join(audit.orphan_signoff_surfaces) or 'none'}",
            f"- Signoffs missing role assignments: {', '.join(audit.signoffs_missing_assignments) or 'none'}",
            f"- Signoffs missing evidence: {', '.join(audit.signoffs_missing_evidence) or 'none'}",
            f"- Signoffs missing requested dates: {', '.join(audit.signoffs_missing_requested_dates) or 'none'}",
            f"- Signoffs missing due dates: {', '.join(audit.signoffs_missing_due_dates) or 'none'}",
            f"- Signoffs missing escalation owners: {', '.join(audit.signoffs_missing_escalation_owners) or 'none'}",
            f"- Signoffs missing reminder owners: {', '.join(audit.signoffs_missing_reminder_owners) or 'none'}",
            f"- Signoffs missing next reminders: {', '.join(audit.signoffs_missing_next_reminders) or 'none'}",
            f"- Signoffs missing reminder cadence: {', '.join(audit.signoffs_missing_reminder_cadence) or 'none'}",
            f"- Signoffs with breached SLA: {', '.join(audit.signoffs_with_breached_sla) or 'none'}",
            f"- Waived signoffs missing metadata: {', '.join(audit.waived_signoffs_missing_metadata) or 'none'}",
            f"- Unresolved required signoff ids: {', '.join(audit.unresolved_required_signoff_ids) or 'none'}",
            f"- Blockers missing signoff links: {', '.join(audit.blockers_missing_signoff_links) or 'none'}",
            f"- Blockers missing escalation owners: {', '.join(audit.blockers_missing_escalation_owners) or 'none'}",
            f"- Blockers missing next actions: {', '.join(audit.blockers_missing_next_actions) or 'none'}",
            f"- Freeze exceptions missing owners: {', '.join(audit.freeze_exceptions_missing_owners) or 'none'}",
            f"- Freeze exceptions missing windows: {', '.join(audit.freeze_exceptions_missing_until) or 'none'}",
            f"- Freeze exceptions missing approvers: {', '.join(audit.freeze_exceptions_missing_approvers) or 'none'}",
            f"- Freeze exceptions missing approval dates: {', '.join(audit.freeze_exceptions_missing_approval_dates) or 'none'}",
            f"- Freeze exceptions missing renewal owners: {', '.join(audit.freeze_exceptions_missing_renewal_owners) or 'none'}",
            f"- Freeze exceptions missing renewal dates: {', '.join(audit.freeze_exceptions_missing_renewal_dates) or 'none'}",
            f"- Blockers missing timeline events: {', '.join(audit.blockers_missing_timeline_events) or 'none'}",
            f"- Closed blockers missing resolution events: {', '.join(audit.closed_blockers_missing_resolution_events) or 'none'}",
            f"- Orphan blocker surfaces: {', '.join(audit.orphan_blocker_surfaces) or 'none'}",
            f"- Orphan blocker timeline blocker ids: {', '.join(audit.orphan_blocker_timeline_blocker_ids) or 'none'}",
            f"- Handoff events missing targets: {', '.join(audit.handoff_events_missing_targets) or 'none'}",
            f"- Handoff events missing artifacts: {', '.join(audit.handoff_events_missing_artifacts) or 'none'}",
            f"- Handoff events missing ack owners: {', '.join(audit.handoff_events_missing_ack_owners) or 'none'}",
            f"- Handoff events missing ack dates: {', '.join(audit.handoff_events_missing_ack_dates) or 'none'}",
            f"- Unresolved required signoffs without blockers: {', '.join(audit.unresolved_required_signoffs_without_blockers) or 'none'}",
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
        requires_role_matrix=True,
        requires_signoff_log=True,
        requires_blocker_log=True,
        requires_blocker_timeline=True,
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
                outcome="Operators can assess batch retry, failure attribution, manual takeover, and audit visibility from one frame.",
                success_signal="The queue frame clearly shows retry eligibility, failure ownership, takeover entry points, and denied roles.",
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
                primary_blocks=["failure attribution", "bulk retry toolbar", "filters", "takeover audit rail"],
                review_notes=["Validate bulk-retry CTA hierarchy.", "Review denied-role and manual takeover behavior for non-operator personas."],
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
                flow_id="flow-queue-bulk-retry",
                name="Queue bulk retry and takeover review",
                trigger="Platform Admin selects multiple failed tasks and opens the bulk retry toolbar",
                system_response="The queue shows retry eligibility, failure attribution, takeover blockers, and denied-role messaging before submit.",
                states=["default", "selection", "confirming", "success"],
                exceptions=["Disable submit when tasks cross unauthorized scopes or require manual takeover.", "Route to the takeover audit timeline when ownership policy changes mid-flow."],
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
                question="Should VP Eng see queue bulk-retry and manual takeover controls in read-only form or be routed to a summary-only state?",
                owner="product-experience",
                impact="Changes denial-path copy, bulk-action placement, and review criteria for queue and triage pages.",
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
                item_id="chk-queue-bulk-retry",
                surface_id="wf-queue",
                prompt="Check that bulk retry clearly communicates eligibility, failure attribution, takeover blockers, and audit consequences.",
                owner="Platform Admin",
                status="ready",
                evidence_links=["wf-queue", "flow-queue-bulk-retry"],
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
        role_matrix=[
            ReviewRoleAssignment(
                assignment_id="role-overview-vp-eng",
                surface_id="wf-overview",
                role="VP Eng",
                responsibilities=["approve overview scan path", "validate KPI-to-drilldown narrative"],
                checklist_item_ids=["chk-overview-kpi-scan"],
                decision_ids=["dec-overview-alert-stack"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-overview-eng-ops",
                surface_id="wf-overview",
                role="engineering-operations",
                responsibilities=["review alert priority posture"],
                checklist_item_ids=["chk-overview-alert-hierarchy"],
                decision_ids=["dec-overview-alert-stack"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-queue-platform-admin",
                surface_id="wf-queue",
                role="Platform Admin",
                responsibilities=["validate bulk-retry copy", "confirm permission posture"],
                checklist_item_ids=["chk-queue-bulk-retry"],
                decision_ids=["dec-queue-vp-summary"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-queue-product-experience",
                surface_id="wf-queue",
                role="product-experience",
                responsibilities=["tune summary-only VP variant"],
                checklist_item_ids=["chk-queue-role-density"],
                decision_ids=["dec-queue-vp-summary"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-run-detail-eng-lead",
                surface_id="wf-run-detail",
                role="Eng Lead",
                responsibilities=["approve replay-state clarity", "confirm escalation adjacency"],
                checklist_item_ids=["chk-run-replay-context"],
                decision_ids=["dec-run-detail-audit-rail"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-run-detail-orchestration-office",
                surface_id="wf-run-detail",
                role="orchestration-office",
                responsibilities=["review audit density threshold"],
                checklist_item_ids=["chk-run-audit-density"],
                decision_ids=["dec-run-detail-audit-rail"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-triage-cross-team-operator",
                surface_id="wf-triage",
                role="Cross-Team Operator",
                responsibilities=["approve handoff clarity", "validate ownership transition story"],
                checklist_item_ids=["chk-triage-handoff-clarity"],
                decision_ids=["dec-triage-handoff-density"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-triage-platform-admin",
                surface_id="wf-triage",
                role="Platform Admin",
                responsibilities=["confirm bulk-assign audit visibility"],
                checklist_item_ids=["chk-triage-bulk-assign"],
                decision_ids=["dec-triage-handoff-density"],
                status="ready",
            ),
        ],
        signoff_log=[
            ReviewSignoff(
                signoff_id="sig-overview-vp-eng",
                assignment_id="role-overview-vp-eng",
                surface_id="wf-overview",
                role="VP Eng",
                status="approved",
                evidence_links=["chk-overview-kpi-scan", "dec-overview-alert-stack"],
                notes="Overview narrative approved for design sprint review.",
                requested_at="2026-03-10T09:00:00Z",
                due_at="2026-03-12T18:00:00Z",
                escalation_owner="design-program-manager",
                sla_status="met",
            ),
            ReviewSignoff(
                signoff_id="sig-queue-platform-admin",
                assignment_id="role-queue-platform-admin",
                surface_id="wf-queue",
                role="Platform Admin",
                status="approved",
                evidence_links=["chk-queue-bulk-retry", "dec-queue-vp-summary"],
                notes="Queue control actions meet operator review criteria.",
                requested_at="2026-03-10T11:00:00Z",
                due_at="2026-03-13T18:00:00Z",
                escalation_owner="platform-ops-manager",
                sla_status="met",
            ),
            ReviewSignoff(
                signoff_id="sig-run-detail-eng-lead",
                assignment_id="role-run-detail-eng-lead",
                surface_id="wf-run-detail",
                role="Eng Lead",
                status="pending",
                evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
                notes="Waiting for final replay-state copy review.",
                requested_at="2026-03-12T11:00:00Z",
                due_at="2026-03-15T18:00:00Z",
                escalation_owner="engineering-director",
                sla_status="at-risk",
                reminder_owner="design-program-manager",
                reminder_channel="slack",
                last_reminder_at="2026-03-14T09:45:00Z",
                next_reminder_at="2026-03-15T10:00:00Z",
                reminder_cadence="daily",
                reminder_status="scheduled",
            ),
            ReviewSignoff(
                signoff_id="sig-triage-cross-team-operator",
                assignment_id="role-triage-cross-team-operator",
                surface_id="wf-triage",
                role="Cross-Team Operator",
                status="approved",
                evidence_links=["chk-triage-handoff-clarity", "dec-triage-handoff-density"],
                notes="Cross-team handoff flow approved for prototype review.",
                requested_at="2026-03-11T14:00:00Z",
                due_at="2026-03-13T12:00:00Z",
                escalation_owner="cross-team-program-manager",
                sla_status="met",
            ),
        ],
        blocker_log=[
            ReviewBlocker(
                blocker_id="blk-run-detail-copy-final",
                surface_id="wf-run-detail",
                signoff_id="sig-run-detail-eng-lead",
                owner="product-experience",
                summary="Replay-state copy still needs final wording review before Eng Lead signoff can close.",
                status="open",
                severity="medium",
                escalation_owner="design-program-manager",
                next_action="Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",
                freeze_exception=True,
                freeze_owner="release-director",
                freeze_until="2026-03-18T18:00:00Z",
                freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
                freeze_approved_by="release-director",
                freeze_approved_at="2026-03-14T08:30:00Z",
                freeze_renewal_owner="release-director",
                freeze_renewal_by="2026-03-17T12:00:00Z",
                freeze_renewal_status="review-needed",
            ),
        ],
        blocker_timeline=[
            ReviewBlockerEvent(
                event_id="evt-run-detail-copy-opened",
                blocker_id="blk-run-detail-copy-final",
                actor="product-experience",
                status="opened",
                summary="Captured the final replay-state copy gap during design sprint prep.",
                timestamp="2026-03-13T10:00:00Z",
                next_action="Draft updated replay labels before the Eng Lead review.",
            ),
            ReviewBlockerEvent(
                event_id="evt-run-detail-copy-escalated",
                blocker_id="blk-run-detail-copy-final",
                actor="design-program-manager",
                status="escalated",
                summary="Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
                timestamp="2026-03-14T09:30:00Z",
                next_action="Refresh the run-detail frame annotations after the wording review completes.",
                handoff_from="product-experience",
                handoff_to="Eng Lead",
                channel="design-critique",
                artifact_ref="wf-run-detail#copy-v5",
                ack_owner="Eng Lead",
                ack_at="2026-03-14T10:15:00Z",
                ack_status="acknowledged",
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



def render_ui_review_role_matrix(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Role Matrix",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Assignments: {len(pack.role_matrix)}",
        "",
        "## Assignments",
    ]
    for assignment in pack.role_matrix:
        lines.append(
            "- "
            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"
        )
        lines.append(
            "  "
            f"responsibilities={','.join(assignment.responsibilities) or 'none'} "
            f"checklist={','.join(assignment.checklist_item_ids) or 'none'} "
            f"decisions={','.join(assignment.decision_ids) or 'none'}"
        )
    if not pack.role_matrix:
        lines.append("- none")
    return "\n".join(lines)



def render_ui_review_objective_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_objective_coverage_entries(pack)
    persona_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        persona_counts[entry['persona']] = persona_counts.get(entry['persona'], 0) + 1
        status_counts[entry['coverage_status']] = status_counts.get(entry['coverage_status'], 0) + 1
    lines = [
        "# UI Review Objective Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Objectives: {len(entries)}",
        f"- Personas: {len(persona_counts)}",
        "",
        "## By Coverage Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Persona")
    for persona, count in sorted(persona_counts.items()):
        lines.append(f"- {persona}: {count}")
    if not persona_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: objective={entry['objective_id']} persona={entry['persona']} priority={entry['priority']} coverage={entry['coverage_status']} dependencies={entry['dependency_count']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  dependency_ids={entry['dependency_ids']} assignments={entry['assignment_ids']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_wireframe_readiness_board(pack: UIReviewPack) -> str:
    entries = _build_wireframe_readiness_entries(pack)
    readiness_counts: Dict[str, int] = {}
    device_counts: Dict[str, int] = {}
    for entry in entries:
        readiness_counts[entry['readiness_status']] = readiness_counts.get(entry['readiness_status'], 0) + 1
        device_counts[entry['device']] = device_counts.get(entry['device'], 0) + 1
    lines = [
        "# UI Review Wireframe Readiness Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Wireframes: {len(entries)}",
        f"- Devices: {len(device_counts)}",
        "",
        "## By Readiness",
    ]
    for status, count in sorted(readiness_counts.items()):
        lines.append(f"- {status}: {count}")
    if not readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Device")
    for device, count in sorted(device_counts.items()):
        lines.append(f"- {device}: {count}")
    if not device_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} device={entry['device']} readiness={entry['readiness_status']} open_total={entry['open_total']} entry={entry['entry_point']}"
        )
        lines.append(
            f"  checklist_open={entry['checklist_open']} decisions_open={entry['decisions_open']} assignments_open={entry['assignments_open']} signoffs_open={entry['signoffs_open']} blockers_open={entry['blockers_open']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} blocks={entry['block_count']} notes={entry['note_count']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_open_question_tracker(pack: UIReviewPack) -> str:
    entries = _build_open_question_tracker_entries(pack)
    owner_counts: Dict[str, int] = {}
    theme_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        theme_counts[entry['theme']] = theme_counts.get(entry['theme'], 0) + 1
    lines = [
        "# UI Review Open Question Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Questions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Theme")
    for theme, count in sorted(theme_counts.items()):
        lines.append(f"- {theme}: {count}")
    if not theme_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: question={entry['question_id']} owner={entry['owner']} theme={entry['theme']} status={entry['status']} link_status={entry['link_status']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} flows={entry['flow_ids']} impact={entry['impact']} prompt={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_review_summary_board(pack: UIReviewPack) -> str:
    entries = _build_review_summary_entries(pack)
    lines = [
        "# UI Review Review Summary Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Categories: {len(entries)}",
        "",
        "## Entries",
    ]
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: category={entry['category']} total={entry['total']} {entry['metrics']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_persona_readiness_board(pack: UIReviewPack) -> str:
    entries = _build_persona_readiness_entries(pack)
    readiness_counts: Dict[str, int] = {}
    for entry in entries:
        readiness_counts[entry['readiness']] = readiness_counts.get(entry['readiness'], 0) + 1
    lines = [
        "# UI Review Persona Readiness Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Personas: {len(entries)}",
        f"- Objectives: {len(pack.objectives)}",
        "",
        "## By Readiness",
    ]
    for readiness, count in sorted(readiness_counts.items()):
        lines.append(f"- {readiness}: {count}")
    if not readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: persona={entry['persona']} readiness={entry['readiness']} objectives={entry['objective_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} open_questions={entry['question_count']} queue_items={entry['queue_count']} blockers={entry['blocker_count']}"
        )
        lines.append(
            f"  objective_ids={entry['objective_ids']} surfaces={entry['surface_ids']} queue_ids={entry['queue_ids']} blocker_ids={entry['blocker_ids']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_interaction_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_interaction_coverage_entries(pack)
    coverage_counts: Dict[str, int] = {}
    surface_counts: Dict[str, int] = {}
    for entry in entries:
        coverage_counts[entry['coverage_status']] = coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                surface_counts[surface_id] = surface_counts.get(surface_id, 0) + 1
    lines = [
        "# UI Review Interaction Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Interactions: {len(entries)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Coverage Status",
    ]
    for status, count in sorted(coverage_counts.items()):
        lines.append(f"- {status}: {count}")
    if not coverage_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, count in sorted(surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: flow={entry['flow_id']} surfaces={entry['surface_ids']} owners={entry['owners']} coverage={entry['coverage_status']} states={entry['state_count']} exceptions={entry['exception_count']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} open_checklist={entry['open_checklist_ids']} trigger={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_checklist_traceability_board(pack: UIReviewPack) -> str:
    entries = _build_checklist_traceability_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Checklist Traceability Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Checklist items: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: item={entry['item_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_decisions={entry['linked_decisions']} evidence={entry['evidence']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_decision_followup_tracker(pack: UIReviewPack) -> str:
    entries = _build_decision_followup_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Decision Follow-up Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Decisions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: decision={entry['decision_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_checklists={entry['linked_checklists']} follow_up={entry['follow_up']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_role_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_role_coverage_entries(pack)
    surface_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        surface_counts[entry['surface_id']] = surface_counts.get(entry['surface_id'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Role Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Assignments: {len(entries)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Surface",
    ]
    for surface_id, count in sorted(surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: assignment={entry['assignment_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} responsibilities={entry['responsibility_count']} checklist={entry['checklist_count']} decisions={entry['decision_count']}"
        )
        lines.append(
            f"  signoff={entry['signoff_id']} signoff_status={entry['signoff_status']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_workload_board(pack: UIReviewPack) -> str:
    entries = _build_owner_workload_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "reminder": 0, "renewal": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Workload Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Items: {len(entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} reminders={counts['reminder']} renewals={counts['renewal']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} lane={entry['lane']}"
        )
        lines.append(f"  detail={entry['detail']} summary={entry['summary']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_dependency_board(pack: UIReviewPack) -> str:
    entries = _build_signoff_dependency_entries(pack)
    dependency_counts: Dict[str, int] = {}
    sla_counts: Dict[str, int] = {}
    for entry in entries:
        dependency_counts[entry['dependency_status']] = dependency_counts.get(entry['dependency_status'], 0) + 1
        sla_counts[entry['sla_status']] = sla_counts.get(entry['sla_status'], 0) + 1
    lines = [
        "# UI Review Signoff Dependency Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(entries)}",
        f"- Dependency states: {len(dependency_counts)}",
        "",
        "## By Dependency Status",
    ]
    for status, count in sorted(dependency_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By SLA State")
    for status, count in sorted(sla_counts.items()):
        lines.append(f"- {status}: {count}")
    if not sla_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} dependency_status={entry['dependency_status']} blockers={entry['blocker_ids']}"
        )
        lines.append(
            f"  assignment={entry['assignment_id']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} latest_blocker_event={entry['latest_blocker_event']} sla={entry['sla_status']} due_at={entry['due_at']} cadence={entry['reminder_cadence']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_audit_density_board(pack: UIReviewPack) -> str:
    entries = _build_audit_density_entries(pack)
    band_counts: Dict[str, int] = {}
    for entry in entries:
        band_counts[entry['load_band']] = band_counts.get(entry['load_band'], 0) + 1
    lines = [
        "# UI Review Audit Density Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Surfaces: {len(entries)}",
        f"- Load bands: {len(band_counts)}",
        "",
        "## By Load Band",
    ]
    for band, count in sorted(band_counts.items()):
        lines.append(f"- {band}: {count}")
    if not band_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} artifact_total={entry['artifact_total']} open_total={entry['open_total']} band={entry['load_band']}"
        )
        lines.append(
            f"  checklist={entry['checklist_count']} decisions={entry['decision_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} blockers={entry['blocker_count']} timeline={entry['timeline_count']} blocks={entry['block_count']} notes={entry['note_count']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Sign-off Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(pack.signoff_log)}",
        "",
        "## Sign-offs",
    ]
    for signoff in pack.signoff_log:
        lines.append(
            "- "
            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"
        )
        lines.append(
            "  "
            f"required={'yes' if signoff.required else 'no'} evidence={','.join(signoff.evidence_links) or 'none'} notes={signoff.notes or 'none'} waiver_owner={signoff.waiver_owner or 'none'} waiver_reason={signoff.waiver_reason or 'none'} requested_at={signoff.requested_at or 'none'} due_at={signoff.due_at or 'none'} escalation_owner={signoff.escalation_owner or 'none'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or 'none'} reminder_channel={signoff.reminder_channel or 'none'} last_reminder_at={signoff.last_reminder_at or 'none'} next_reminder_at={signoff.next_reminder_at or 'none'}"
        )
    if not pack.signoff_log:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_sla_dashboard(pack: UIReviewPack) -> str:
    entries = _build_signoff_sla_entries(pack)
    state_counts: Dict[str, int] = {}
    owner_counts: Dict[str, int] = {}
    for entry in entries:
        state_counts[entry['sla_status']] = state_counts.get(entry['sla_status'], 0) + 1
        owner_counts[entry['escalation_owner']] = owner_counts.get(entry['escalation_owner'], 0) + 1
    lines = [
        "# UI Review Sign-off SLA Dashboard",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## SLA States",
    ]
    for sla_status, count in sorted(state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalation Owners")
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Sign-offs")
    for entry in entries:
        lines.append(
            f"- {entry['signoff_id']}: role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} requested_at={entry['requested_at']} due_at={entry['due_at']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(f"  required={entry['required']} evidence={entry['evidence']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_reminder_queue(pack: UIReviewPack) -> str:
    entries = _build_signoff_reminder_entries(pack)
    owner_counts: Dict[str, int] = {}
    channel_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry["reminder_owner"]] = owner_counts.get(entry["reminder_owner"], 0) + 1
        channel_counts[entry["reminder_channel"]] = channel_counts.get(entry["reminder_channel"], 0) + 1
    lines = [
        "# UI Review Sign-off Reminder Queue",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Reminders: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: reminders={count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Channel")
    for channel, count in sorted(channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} owner={entry['reminder_owner']} channel={entry['reminder_channel']}"
        )
        lines.append(
            f"  last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_reminder_cadence_board(pack: UIReviewPack) -> str:
    entries = _build_reminder_cadence_entries(pack)
    cadence_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        cadence_counts[entry["reminder_cadence"]] = cadence_counts.get(entry["reminder_cadence"], 0) + 1
        status_counts[entry["reminder_status"]] = status_counts.get(entry["reminder_status"], 0) + 1
    lines = [
        "# UI Review Reminder Cadence Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Items: {len(entries)}",
        f"- Cadences: {len(cadence_counts)}",
        "",
        "## By Cadence",
    ]
    for cadence, count in sorted(cadence_counts.items()):
        lines.append(f"- {cadence}: {count}")
    if not cadence_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} cadence={entry['reminder_cadence']} status={entry['reminder_status']} owner={entry['reminder_owner']}"
        )
        lines.append(
            f"  sla={entry['sla_status']} last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_escalation_dashboard(pack: UIReviewPack) -> str:
    entries = _build_escalation_dashboard_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    status_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        owner_bucket = owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        status_bucket = status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_bucket[entry['item_type']] += 1
        status_bucket['total'] += 1
    lines = [
        "# UI Review Escalation Dashboard",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Items: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## By Escalation Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, counts in sorted(status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalations")
    for entry in entries:
        lines.append(
            f"- {entry['escalation_id']}: owner={entry['escalation_owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} priority={entry['priority']} current_owner={entry['current_owner']}"
        )
        lines.append(f"  summary={entry['summary']} due_at={entry['due_at']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_breach_board(pack: UIReviewPack) -> str:
    entries = _build_signoff_breach_entries(pack)
    state_counts: Dict[str, int] = {}
    owner_counts: Dict[str, int] = {}
    for entry in entries:
        state_counts[entry['sla_status']] = state_counts.get(entry['sla_status'], 0) + 1
        owner_counts[entry['escalation_owner']] = owner_counts.get(entry['escalation_owner'], 0) + 1
    lines = [
        "# UI Review Sign-off Breach Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Breach items: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## SLA States",
    ]
    for sla_status, count in sorted(state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalation Owners")
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(
            f"  requested_at={entry['requested_at']} due_at={entry['due_at']} linked_blockers={entry['linked_blockers']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_escalation_handoff_ledger(pack: UIReviewPack) -> str:
    entries = _build_escalation_handoff_entries(pack)
    channel_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        channel_counts[entry['channel']] = channel_counts.get(entry['channel'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Escalation Handoff Ledger",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Handoffs: {len(entries)}",
        f"- Channels: {len(channel_counts)}",
        "",
        "## By Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Channel")
    for channel, count in sorted(channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['ledger_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} actor={entry['actor']} status={entry['status']} at={entry['timestamp']}"
        )
        lines.append(
            f"  from={entry['handoff_from']} to={entry['handoff_to']} channel={entry['channel']} artifact={entry['artifact_ref']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_handoff_ack_ledger(pack: UIReviewPack) -> str:
    entries = _build_handoff_ack_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['ack_owner']] = owner_counts.get(entry['ack_owner'], 0) + 1
        status_counts[entry['ack_status']] = status_counts.get(entry['ack_status'], 0) + 1
    lines = [
        "# UI Review Handoff Ack Ledger",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Ack items: {len(entries)}",
        f"- Ack owners: {len(owner_counts)}",
        "",
        "## By Ack Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Ack Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} handoff_to={entry['handoff_to']} ack_owner={entry['ack_owner']} ack_status={entry['ack_status']} ack_at={entry['ack_at']}"
        )
        lines.append(
            f"  actor={entry['actor']} status={entry['status']} channel={entry['channel']} artifact={entry['artifact_ref']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_approval_trail(pack: UIReviewPack) -> str:
    entries = _build_freeze_approval_entries(pack)
    approver_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        approver_counts[entry["freeze_approved_by"]] = approver_counts.get(entry["freeze_approved_by"], 0) + 1
        status_counts[entry["status"]] = status_counts.get(entry["status"], 0) + 1
    lines = [
        "# UI Review Freeze Approval Trail",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Approvals: {len(entries)}",
        f"- Approvers: {len(approver_counts)}",
        "",
        "## By Approver",
    ]
    for owner, count in sorted(approver_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not approver_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} owner={entry['freeze_owner']} approved_by={entry['freeze_approved_by']} approved_at={entry['freeze_approved_at']} window={entry['freeze_until']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_renewal_tracker(pack: UIReviewPack) -> str:
    entries = _build_freeze_renewal_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['renewal_owner']] = owner_counts.get(entry['renewal_owner'], 0) + 1
        status_counts[entry['renewal_status']] = status_counts.get(entry['renewal_status'], 0) + 1
    lines = [
        "# UI Review Freeze Renewal Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Renewal items: {len(entries)}",
        f"- Renewal owners: {len(owner_counts)}",
        "",
        "## By Renewal Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Renewal Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} renewal_owner={entry['renewal_owner']} renewal_by={entry['renewal_by']} renewal_status={entry['renewal_status']}"
        )
        lines.append(
            f"  freeze_owner={entry['freeze_owner']} freeze_until={entry['freeze_until']} approved_by={entry['freeze_approved_by']} summary={entry['summary']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_exception_board(pack: UIReviewPack) -> str:
    entries = _build_freeze_exception_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        owner_bucket = owner_counts.setdefault(entry['owner'], {'blocker': 0, 'signoff': 0, 'total': 0})
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        surface_bucket = surface_counts.setdefault(entry['surface_id'], {'blocker': 0, 'signoff': 0, 'total': 0})
        surface_bucket[entry['item_type']] += 1
        surface_bucket['total'] += 1
    lines = [
        "# UI Review Freeze Exception Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, counts in sorted(surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} window={entry['window']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_escalation_digest(pack: UIReviewPack) -> str:
    entries = _build_owner_escalation_digest_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "signoff": 0, "reminder": 0, "freeze": 0, "handoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Escalation Digest",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Items: {len(entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} reminders={counts['reminder']} freezes={counts['freeze']} handoffs={counts['handoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['digest_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} detail={entry['detail']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Blocker Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Blockers: {len(pack.blocker_log)}",
        "",
        "## Blockers",
    ]
    for blocker in pack.blocker_log:
        lines.append(
            "- "
            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"
        )
        lines.append(
            "  "
            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or 'none'} next_action={blocker.next_action or 'none'} freeze_owner={blocker.freeze_owner or 'none'} freeze_until={blocker.freeze_until or 'none'} freeze_approved_by={blocker.freeze_approved_by or 'none'} freeze_approved_at={blocker.freeze_approved_at or 'none'}"
        )
    if not pack.blocker_log:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_timeline(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Blocker Timeline",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Events: {len(pack.blocker_timeline)}",
        "",
        "## Events",
    ]
    for event in pack.blocker_timeline:
        lines.append(
            "- "
            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"
        )
        lines.append(
            "  "
            f"summary={event.summary} next_action={event.next_action or 'none'}"
        )
    if not pack.blocker_timeline:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_exception_log(pack: UIReviewPack) -> str:
    exception_entries = _build_review_exception_entries(pack)
    lines = [
        "# UI Review Exception Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(exception_entries)}",
        "",
        "## Exceptions",
    ]
    for entry in exception_entries:
        lines.append(
            "- "
            f"{entry['exception_id']}: type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            "  "
            f"summary={entry['summary']} evidence={entry['evidence']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_exception_matrix(pack: UIReviewPack) -> str:
    exception_entries = _build_review_exception_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    status_counts: Dict[str, Dict[str, int]] = {}
    surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_bucket = owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["category"]] += 1
        owner_bucket["total"] += 1
        status_bucket = status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_bucket[entry["category"]] += 1
        status_bucket["total"] += 1
        surface_bucket = surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["category"]] += 1
        surface_bucket["total"] += 1
    lines = [
        "# UI Review Exception Matrix",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(exception_entries)}",
        f"- Owners: {len(owner_counts)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, counts in sorted(status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, counts in sorted(surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in exception_entries:
        lines.append(
            f"- {entry['exception_id']}: owner={entry['owner']} type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_review_queue(pack: UIReviewPack) -> str:
    queue_entries = _build_owner_review_queue_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in queue_entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Review Queue",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Queue items: {len(queue_entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in queue_entries:
        lines.append(
            f"- {entry['queue_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} next_action={entry['next_action']}")
    if not queue_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_timeline_summary(pack: UIReviewPack) -> str:
    timeline_index = _build_blocker_timeline_index(pack)
    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )
    lines = [
        "# UI Review Blocker Timeline Summary",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Events: {len(pack.blocker_timeline)}",
        f"- Blockers with timeline: {len(timeline_index)}",
        f"- Orphan timeline blockers: {','.join(orphan_timeline_ids) or 'none'}",
        "",
        "## Events by Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Events by Actor")
    for actor, count in sorted(actor_counts.items()):
        lines.append(f"- {actor}: {count}")
    if not actor_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Latest Blocker Events")
    for blocker in pack.blocker_log:
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        if latest is None:
            lines.append(f"- {blocker.blocker_id}: latest=none")
            continue
        lines.append(
            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"
        )
    if not pack.blocker_log:
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
    interaction_coverage_entries = _build_interaction_coverage_entries(pack)
    interaction_coverage_counts: Dict[str, int] = {}
    interaction_surface_counts: Dict[str, int] = {}
    for entry in interaction_coverage_entries:
        interaction_coverage_counts[entry['coverage_status']] = interaction_coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1
    interaction_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(interaction_coverage_counts.items())
    ) or "<li>none</li>"
    interaction_coverage_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"
        for surface_id, count in sorted(interaction_surface_counts.items())
    ) or "<li>none</li>"
    interaction_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · flow={escape(entry['flow_id'])} · surfaces={escape(entry['surface_ids'])} · owners={escape(entry['owners'])} · coverage={escape(entry['coverage_status'])}<br /><span>states={escape(entry['state_count'])} · exceptions={escape(entry['exception_count'])}</span><br /><span>checklist={escape(entry['checklist_ids'])} · open_checklist={escape(entry['open_checklist_ids'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in interaction_coverage_entries
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
    role_matrix_html = "".join(
        f"<li><strong>{escape(assignment.assignment_id)}</strong> · surface={escape(assignment.surface_id)} · role={escape(assignment.role)} · status={escape(assignment.status)}<br /><span>responsibilities={escape(', '.join(assignment.responsibilities) if assignment.responsibilities else 'none')}</span><br /><span>checklist={escape(', '.join(assignment.checklist_item_ids) if assignment.checklist_item_ids else 'none')} · decisions={escape(', '.join(assignment.decision_ids) if assignment.decision_ids else 'none')}</span></li>"
        for assignment in pack.role_matrix
    ) or "<li>none</li>"
    objective_coverage_entries = _build_objective_coverage_entries(pack)
    objective_coverage_status_counts: Dict[str, int] = {}
    objective_coverage_persona_counts: Dict[str, int] = {}
    for entry in objective_coverage_entries:
        objective_coverage_status_counts[entry['coverage_status']] = objective_coverage_status_counts.get(entry['coverage_status'], 0) + 1
        objective_coverage_persona_counts[entry['persona']] = objective_coverage_persona_counts.get(entry['persona'], 0) + 1
    objective_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(objective_coverage_status_counts.items())
    ) or "<li>none</li>"
    objective_coverage_persona_html = "".join(
        f"<li><strong>{escape(persona)}</strong> · count={count}</li>"
        for persona, count in sorted(objective_coverage_persona_counts.items())
    ) or "<li>none</li>"
    objective_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · objective={escape(entry['objective_id'])} · persona={escape(entry['persona'])} · priority={escape(entry['priority'])} · coverage={escape(entry['coverage_status'])}<br /><span>dependencies={escape(entry['dependency_count'])} · surfaces={escape(entry['surface_ids'])} · blockers={escape(entry['blocker_ids'])}</span><br /><span>assignments={escape(entry['assignment_ids'])} · checklist={escape(entry['checklist_ids'])} · decisions={escape(entry['decision_ids'])}</span><br /><span>signoffs={escape(entry['signoff_ids'])} · dependency_ids={escape(entry['dependency_ids'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in objective_coverage_entries
    ) or "<li>none</li>"
    review_summary_entries = _build_review_summary_entries(pack)
    review_summary_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · category={escape(entry['category'])} · total={escape(entry['total'])} · {escape(entry['metrics'])}</li>"
        for entry in review_summary_entries
    ) or "<li>none</li>"
    persona_readiness_entries = _build_persona_readiness_entries(pack)
    persona_readiness_counts: Dict[str, int] = {}
    for entry in persona_readiness_entries:
        persona_readiness_counts[entry['readiness']] = persona_readiness_counts.get(entry['readiness'], 0) + 1
    persona_readiness_status_html = "".join(
        f"<li><strong>{escape(readiness)}</strong> · count={count}</li>"
        for readiness, count in sorted(persona_readiness_counts.items())
    ) or "<li>none</li>"
    persona_readiness_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · persona={escape(entry['persona'])} · readiness={escape(entry['readiness'])} · objectives={escape(entry['objective_count'])}<br /><span>assignments={escape(entry['assignment_count'])} · signoffs={escape(entry['signoff_count'])} · open_questions={escape(entry['question_count'])} · queue_items={escape(entry['queue_count'])} · blockers={escape(entry['blocker_count'])}</span><br /><span>objective_ids={escape(entry['objective_ids'])} · surfaces={escape(entry['surface_ids'])}</span><br /><span>queue_ids={escape(entry['queue_ids'])} · blocker_ids={escape(entry['blocker_ids'])}</span></li>"
        for entry in persona_readiness_entries
    ) or "<li>none</li>"
    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)
    wireframe_readiness_counts: Dict[str, int] = {}
    wireframe_device_counts: Dict[str, int] = {}
    for entry in wireframe_readiness_entries:
        wireframe_readiness_counts[entry['readiness_status']] = wireframe_readiness_counts.get(entry['readiness_status'], 0) + 1
        wireframe_device_counts[entry['device']] = wireframe_device_counts.get(entry['device'], 0) + 1
    wireframe_readiness_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(wireframe_readiness_counts.items())
    ) or "<li>none</li>"
    wireframe_device_html = "".join(
        f"<li><strong>{escape(device)}</strong> · count={count}</li>"
        for device, count in sorted(wireframe_device_counts.items())
    ) or "<li>none</li>"
    wireframe_readiness_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · surface={escape(entry['surface_id'])} · device={escape(entry['device'])} · readiness={escape(entry['readiness_status'])} · open_total={escape(entry['open_total'])}<br /><span>entry={escape(entry['entry_point'])} · signoffs={escape(entry['signoff_ids'])} · blockers={escape(entry['blocker_ids'])}</span><br /><span>checklist_open={escape(entry['checklist_open'])} · decisions_open={escape(entry['decisions_open'])} · assignments_open={escape(entry['assignments_open'])}</span><br /><span>signoffs_open={escape(entry['signoffs_open'])} · blockers_open={escape(entry['blockers_open'])} · blocks={escape(entry['block_count'])} · notes={escape(entry['note_count'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in wireframe_readiness_entries
    ) or "<li>none</li>"
    open_question_entries = _build_open_question_tracker_entries(pack)
    open_question_owner_counts: Dict[str, int] = {}
    open_question_theme_counts: Dict[str, int] = {}
    for entry in open_question_entries:
        open_question_owner_counts[entry['owner']] = open_question_owner_counts.get(entry['owner'], 0) + 1
        open_question_theme_counts[entry['theme']] = open_question_theme_counts.get(entry['theme'], 0) + 1
    open_question_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(open_question_owner_counts.items())
    ) or "<li>none</li>"
    open_question_theme_html = "".join(
        f"<li><strong>{escape(theme)}</strong> · count={count}</li>"
        for theme, count in sorted(open_question_theme_counts.items())
    ) or "<li>none</li>"
    open_question_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · question={escape(entry['question_id'])} · owner={escape(entry['owner'])} · theme={escape(entry['theme'])} · status={escape(entry['status'])}<br /><span>link_status={escape(entry['link_status'])} · surfaces={escape(entry['surface_ids'])} · checklist={escape(entry['checklist_ids'])}</span><br /><span>flows={escape(entry['flow_ids'])}</span><br /><span>impact={escape(entry['impact'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in open_question_entries
    ) or "<li>none</li>"
    checklist_trace_entries = _build_checklist_traceability_entries(pack)
    checklist_trace_owner_counts: Dict[str, int] = {}
    checklist_trace_status_counts: Dict[str, int] = {}
    for entry in checklist_trace_entries:
        checklist_trace_owner_counts[entry['owner']] = checklist_trace_owner_counts.get(entry['owner'], 0) + 1
        checklist_trace_status_counts[entry['status']] = checklist_trace_status_counts.get(entry['status'], 0) + 1
    checklist_trace_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(checklist_trace_owner_counts.items())
    ) or "<li>none</li>"
    checklist_trace_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(checklist_trace_status_counts.items())
    ) or "<li>none</li>"
    checklist_trace_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · item={escape(entry['item_id'])} · surface={escape(entry['surface_id'])} · owner={escape(entry['owner'])} · status={escape(entry['status'])}<br /><span>linked_roles={escape(entry['linked_roles'])} · linked_assignments={escape(entry['linked_assignments'])}</span><br /><span>linked_decisions={escape(entry['linked_decisions'])} · evidence={escape(entry['evidence'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in checklist_trace_entries
    ) or "<li>none</li>"
    decision_followup_entries = _build_decision_followup_entries(pack)
    decision_followup_owner_counts: Dict[str, int] = {}
    decision_followup_status_counts: Dict[str, int] = {}
    for entry in decision_followup_entries:
        decision_followup_owner_counts[entry['owner']] = decision_followup_owner_counts.get(entry['owner'], 0) + 1
        decision_followup_status_counts[entry['status']] = decision_followup_status_counts.get(entry['status'], 0) + 1
    decision_followup_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(decision_followup_owner_counts.items())
    ) or "<li>none</li>"
    decision_followup_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(decision_followup_status_counts.items())
    ) or "<li>none</li>"
    decision_followup_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · decision={escape(entry['decision_id'])} · surface={escape(entry['surface_id'])} · owner={escape(entry['owner'])} · status={escape(entry['status'])}<br /><span>linked_roles={escape(entry['linked_roles'])} · linked_assignments={escape(entry['linked_assignments'])}</span><br /><span>linked_checklists={escape(entry['linked_checklists'])} · follow_up={escape(entry['follow_up'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in decision_followup_entries
    ) or "<li>none</li>"
    role_coverage_entries = _build_role_coverage_entries(pack)
    role_coverage_surface_counts: Dict[str, int] = {}
    role_coverage_status_counts: Dict[str, int] = {}
    for entry in role_coverage_entries:
        role_coverage_surface_counts[entry['surface_id']] = role_coverage_surface_counts.get(entry['surface_id'], 0) + 1
        role_coverage_status_counts[entry['status']] = role_coverage_status_counts.get(entry['status'], 0) + 1
    role_coverage_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"
        for surface_id, count in sorted(role_coverage_surface_counts.items())
    ) or "<li>none</li>"
    role_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(role_coverage_status_counts.items())
    ) or "<li>none</li>"
    role_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · assignment={escape(entry['assignment_id'])} · surface={escape(entry['surface_id'])} · role={escape(entry['role'])} · status={escape(entry['status'])}<br /><span>responsibilities={escape(entry['responsibility_count'])} · checklist={escape(entry['checklist_count'])} · decisions={escape(entry['decision_count'])}</span><br /><span>signoff={escape(entry['signoff_id'])} · signoff_status={escape(entry['signoff_status'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in role_coverage_entries
    ) or "<li>none</li>"
    signoff_dependency_entries = _build_signoff_dependency_entries(pack)
    signoff_dependency_status_counts: Dict[str, int] = {}
    signoff_dependency_sla_counts: Dict[str, int] = {}
    for entry in signoff_dependency_entries:
        signoff_dependency_status_counts[entry['dependency_status']] = signoff_dependency_status_counts.get(entry['dependency_status'], 0) + 1
        signoff_dependency_sla_counts[entry['sla_status']] = signoff_dependency_sla_counts.get(entry['sla_status'], 0) + 1
    signoff_dependency_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(signoff_dependency_status_counts.items())
    ) or "<li>none</li>"
    signoff_dependency_sla_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(signoff_dependency_sla_counts.items())
    ) or "<li>none</li>"
    signoff_dependency_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · surface={escape(entry['surface_id'])} · role={escape(entry['role'])} · status={escape(entry['status'])} · dependency_status={escape(entry['dependency_status'])}<br /><span>assignment={escape(entry['assignment_id'])} · checklist={escape(entry['checklist_ids'])} · decisions={escape(entry['decision_ids'])}</span><br /><span>blockers={escape(entry['blocker_ids'])} · latest_blocker_event={escape(entry['latest_blocker_event'])}</span><br /><span>sla={escape(entry['sla_status'])} · due_at={escape(entry['due_at'])} · cadence={escape(entry['reminder_cadence'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in signoff_dependency_entries
    ) or "<li>none</li>"
    signoff_html = "".join(
        f"<li><strong>{escape(signoff.signoff_id)}</strong> · surface={escape(signoff.surface_id)} · role={escape(signoff.role)} · status={escape(signoff.status)}<br /><span>assignment={escape(signoff.assignment_id)} · required={escape('yes' if signoff.required else 'no')}</span><br /><span>evidence={escape(', '.join(signoff.evidence_links) if signoff.evidence_links else 'none')}</span><br /><span>waiver_owner={escape(signoff.waiver_owner or 'none')} · waiver_reason={escape(signoff.waiver_reason or 'none')}</span><br /><span>requested_at={escape(signoff.requested_at or 'none')} · due_at={escape(signoff.due_at or 'none')} · escalation_owner={escape(signoff.escalation_owner or 'none')} · sla_status={escape(signoff.sla_status)}</span></li>"
        for signoff in pack.signoff_log
    ) or "<li>none</li>"
    signoff_sla_entries = _build_signoff_sla_entries(pack)
    signoff_sla_state_counts: Dict[str, int] = {}
    signoff_sla_owner_counts: Dict[str, int] = {}
    for entry in signoff_sla_entries:
        signoff_sla_state_counts[entry['sla_status']] = signoff_sla_state_counts.get(entry['sla_status'], 0) + 1
        signoff_sla_owner_counts[entry['escalation_owner']] = signoff_sla_owner_counts.get(entry['escalation_owner'], 0) + 1
    signoff_sla_state_html = "".join(
        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"
        for sla_status, count in sorted(signoff_sla_state_counts.items())
    ) or "<li>none</li>"
    signoff_sla_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(signoff_sla_owner_counts.items())
    ) or "<li>none</li>"
    signoff_sla_item_html = "".join(
        f"<li><strong>{escape(entry['signoff_id'])}</strong> · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>requested_at={escape(entry['requested_at'])} · due_at={escape(entry['due_at'])} · escalation_owner={escape(entry['escalation_owner'])}</span><br /><span>required={escape(entry['required'])} · evidence={escape(entry['evidence'])}</span></li>"
        for entry in signoff_sla_entries
    ) or "<li>none</li>"
    signoff_reminder_entries = _build_signoff_reminder_entries(pack)
    signoff_reminder_owner_counts: Dict[str, int] = {}
    signoff_reminder_channel_counts: Dict[str, int] = {}
    for entry in signoff_reminder_entries:
        signoff_reminder_owner_counts[entry['reminder_owner']] = signoff_reminder_owner_counts.get(entry['reminder_owner'], 0) + 1
        signoff_reminder_channel_counts[entry['reminder_channel']] = signoff_reminder_channel_counts.get(entry['reminder_channel'], 0) + 1
    signoff_reminder_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · reminders={count}</li>"
        for owner, count in sorted(signoff_reminder_owner_counts.items())
    ) or "<li>none</li>"
    signoff_reminder_channel_html = "".join(
        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"
        for channel, count in sorted(signoff_reminder_channel_counts.items())
    ) or "<li>none</li>"
    signoff_reminder_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>owner={escape(entry['reminder_owner'])} · channel={escape(entry['reminder_channel'])}</span><br /><span>last_reminder_at={escape(entry['last_reminder_at'])} · next_reminder_at={escape(entry['next_reminder_at'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in signoff_reminder_entries
    ) or "<li>none</li>"
    reminder_cadence_entries = _build_reminder_cadence_entries(pack)
    reminder_cadence_counts: Dict[str, int] = {}
    reminder_status_counts: Dict[str, int] = {}
    for entry in reminder_cadence_entries:
        reminder_cadence_counts[entry['reminder_cadence']] = reminder_cadence_counts.get(entry['reminder_cadence'], 0) + 1
        reminder_status_counts[entry['reminder_status']] = reminder_status_counts.get(entry['reminder_status'], 0) + 1
    reminder_cadence_owner_html = "".join(
        f"<li><strong>{escape(cadence)}</strong> · count={count}</li>"
        for cadence, count in sorted(reminder_cadence_counts.items())
    ) or "<li>none</li>"
    reminder_cadence_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(reminder_status_counts.items())
    ) or "<li>none</li>"
    reminder_cadence_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · cadence={escape(entry['reminder_cadence'])} · status={escape(entry['reminder_status'])}<br /><span>owner={escape(entry['reminder_owner'])} · sla={escape(entry['sla_status'])}</span><br /><span>last_reminder_at={escape(entry['last_reminder_at'])} · next_reminder_at={escape(entry['next_reminder_at'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in reminder_cadence_entries
    ) or "<li>none</li>"
    signoff_breach_entries = _build_signoff_breach_entries(pack)
    signoff_breach_state_counts: Dict[str, int] = {}
    signoff_breach_owner_counts: Dict[str, int] = {}
    for entry in signoff_breach_entries:
        signoff_breach_state_counts[entry['sla_status']] = signoff_breach_state_counts.get(entry['sla_status'], 0) + 1
        signoff_breach_owner_counts[entry['escalation_owner']] = signoff_breach_owner_counts.get(entry['escalation_owner'], 0) + 1
    signoff_breach_state_html = "".join(
        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"
        for sla_status, count in sorted(signoff_breach_state_counts.items())
    ) or "<li>none</li>"
    signoff_breach_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(signoff_breach_owner_counts.items())
    ) or "<li>none</li>"
    signoff_breach_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>requested_at={escape(entry['requested_at'])} · due_at={escape(entry['due_at'])} · escalation_owner={escape(entry['escalation_owner'])}</span><br /><span>linked_blockers={escape(entry['linked_blockers'])} · summary={escape(entry['summary'])}</span></li>"
        for entry in signoff_breach_entries
    ) or "<li>none</li>"
    escalation_entries = _build_escalation_dashboard_entries(pack)
    escalation_owner_counts: Dict[str, Dict[str, int]] = {}
    escalation_status_counts: Dict[str, Dict[str, int]] = {}
    for entry in escalation_entries:
        owner_bucket = escalation_owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        status_bucket = escalation_status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_bucket[entry['item_type']] += 1
        status_bucket['total'] += 1
    escalation_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(escalation_owner_counts.items())
    ) or "<li>none</li>"
    escalation_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for status, counts in sorted(escalation_status_counts.items())
    ) or "<li>none</li>"
    escalation_item_html = "".join(
        f"<li><strong>{escape(entry['escalation_id'])}</strong> · owner={escape(entry['escalation_owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · priority={escape(entry['priority'])}<br /><span>current_owner={escape(entry['current_owner'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in escalation_entries
    ) or "<li>none</li>"
    escalation_handoff_entries = _build_escalation_handoff_entries(pack)
    escalation_handoff_status_counts: Dict[str, int] = {}
    escalation_handoff_channel_counts: Dict[str, int] = {}
    for entry in escalation_handoff_entries:
        escalation_handoff_status_counts[entry['status']] = escalation_handoff_status_counts.get(entry['status'], 0) + 1
        escalation_handoff_channel_counts[entry['channel']] = escalation_handoff_channel_counts.get(entry['channel'], 0) + 1
    escalation_handoff_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(escalation_handoff_status_counts.items())
    ) or "<li>none</li>"
    escalation_handoff_channel_html = "".join(
        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"
        for channel, count in sorted(escalation_handoff_channel_counts.items())
    ) or "<li>none</li>"
    escalation_handoff_item_html = "".join(
        f"<li><strong>{escape(entry['ledger_id'])}</strong> · event={escape(entry['event_id'])} · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · actor={escape(entry['actor'])} · status={escape(entry['status'])}<br /><span>from={escape(entry['handoff_from'])} · to={escape(entry['handoff_to'])} · channel={escape(entry['channel'])}</span><br /><span>artifact={escape(entry['artifact_ref'])} · next_action={escape(entry['next_action'])} · at={escape(entry['timestamp'])}</span></li>"
        for entry in escalation_handoff_entries
    ) or "<li>none</li>"
    handoff_ack_entries = _build_handoff_ack_entries(pack)
    handoff_ack_owner_counts: Dict[str, int] = {}
    handoff_ack_status_counts: Dict[str, int] = {}
    for entry in handoff_ack_entries:
        handoff_ack_owner_counts[entry['ack_owner']] = handoff_ack_owner_counts.get(entry['ack_owner'], 0) + 1
        handoff_ack_status_counts[entry['ack_status']] = handoff_ack_status_counts.get(entry['ack_status'], 0) + 1
    handoff_ack_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(handoff_ack_owner_counts.items())
    ) or "<li>none</li>"
    handoff_ack_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(handoff_ack_status_counts.items())
    ) or "<li>none</li>"
    handoff_ack_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · event={escape(entry['event_id'])} · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · handoff_to={escape(entry['handoff_to'])}<br /><span>ack_owner={escape(entry['ack_owner'])} · ack_status={escape(entry['ack_status'])} · ack_at={escape(entry['ack_at'])}</span><br /><span>actor={escape(entry['actor'])} · channel={escape(entry['channel'])} · artifact={escape(entry['artifact_ref'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in handoff_ack_entries
    ) or "<li>none</li>"
    owner_escalation_entries = _build_owner_escalation_digest_entries(pack)
    owner_escalation_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_escalation_entries:
        counts = owner_escalation_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'signoff': 0, 'reminder': 0, 'freeze': 0, 'handoff': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1
    owner_escalation_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · reminders={counts['reminder']} · freezes={counts['freeze']} · handoffs={counts['handoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_escalation_counts.items())
    ) or "<li>none</li>"
    owner_escalation_item_html = "".join(
        f"<li><strong>{escape(entry['digest_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>{escape(entry['summary'])}</span><br /><span>detail={escape(entry['detail'])}</span></li>"
        for entry in owner_escalation_entries
    ) or "<li>none</li>"
    owner_workload_entries = _build_owner_workload_entries(pack)
    owner_workload_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_workload_entries:
        counts = owner_workload_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'checklist': 0, 'decision': 0, 'signoff': 0, 'reminder': 0, 'renewal': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1
    owner_workload_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · checklist={counts['checklist']} · decisions={counts['decision']} · signoffs={counts['signoff']} · reminders={counts['reminder']} · renewals={counts['renewal']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_workload_counts.items())
    ) or "<li>none</li>"
    owner_workload_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · lane={escape(entry['lane'])}<br /><span>{escape(entry['summary'])}</span><br /><span>detail={escape(entry['detail'])}</span></li>"
        for entry in owner_workload_entries
    ) or "<li>none</li>"
    blocker_html = "".join(
        f"<li><strong>{escape(blocker.blocker_id)}</strong> · surface={escape(blocker.surface_id)} · signoff={escape(blocker.signoff_id)} · owner={escape(blocker.owner)} · status={escape(blocker.status)} · severity={escape(blocker.severity)}<br /><span>{escape(blocker.summary)}</span><br /><span>escalation_owner={escape(blocker.escalation_owner or 'none')} · next_action={escape(blocker.next_action or 'none')}</span></li>"
        for blocker in pack.blocker_log
    ) or "<li>none</li>"
    blocker_timeline_html = "".join(
        f"<li><strong>{escape(event.event_id)}</strong> · blocker={escape(event.blocker_id)} · actor={escape(event.actor)} · status={escape(event.status)}<br /><span>timestamp={escape(event.timestamp)}</span><br /><span>{escape(event.summary)}</span><br /><span>next_action={escape(event.next_action or 'none')}</span></li>"
        for event in pack.blocker_timeline
    ) or "<li>none</li>"
    timeline_index = _build_blocker_timeline_index(pack)
    exception_entries = _build_review_exception_entries(pack)
    exception_html = "".join(
        f"<li><strong>{escape(entry['exception_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['category'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · severity={escape(entry['severity'])}<br /><span>{escape(entry['summary'])}</span><br /><span>latest_event={escape(entry['latest_event'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in exception_entries
    ) or "<li>none</li>"
    exception_owner_counts: Dict[str, Dict[str, int]] = {}
    exception_status_counts: Dict[str, Dict[str, int]] = {}
    exception_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_bucket = exception_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["category"]] += 1
        owner_bucket["total"] += 1
        status_bucket = exception_status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_bucket[entry["category"]] += 1
        status_bucket["total"] += 1
        surface_bucket = exception_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["category"]] += 1
        surface_bucket["total"] += 1
    exception_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(exception_owner_counts.items())
    ) or "<li>none</li>"
    exception_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for status, counts in sorted(exception_status_counts.items())
    ) or "<li>none</li>"
    exception_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for surface_id, counts in sorted(exception_surface_counts.items())
    ) or "<li>none</li>"
    audit_density_entries = _build_audit_density_entries(pack)
    audit_density_band_counts: Dict[str, int] = {}
    for entry in audit_density_entries:
        audit_density_band_counts[entry['load_band']] = audit_density_band_counts.get(entry['load_band'], 0) + 1
    audit_density_band_html = "".join(
        f"<li><strong>{escape(band)}</strong> · count={count}</li>"
        for band, count in sorted(audit_density_band_counts.items())
    ) or "<li>none</li>"
    audit_density_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · surface={escape(entry['surface_id'])} · artifact_total={escape(entry['artifact_total'])} · open_total={escape(entry['open_total'])} · band={escape(entry['load_band'])}<br /><span>checklist={escape(entry['checklist_count'])} · decisions={escape(entry['decision_count'])} · assignments={escape(entry['assignment_count'])}</span><br /><span>signoffs={escape(entry['signoff_count'])} · blockers={escape(entry['blocker_count'])} · timeline={escape(entry['timeline_count'])}</span><br /><span>blocks={escape(entry['block_count'])} · notes={escape(entry['note_count'])}</span></li>"
        for entry in audit_density_entries
    ) or "<li>none</li>"
    freeze_entries = _build_freeze_exception_entries(pack)
    freeze_owner_counts: Dict[str, Dict[str, int]] = {}
    freeze_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in freeze_entries:
        owner_bucket = freeze_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["item_type"]] += 1
        owner_bucket["total"] += 1
        surface_bucket = freeze_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["item_type"]] += 1
        surface_bucket["total"] += 1
    freeze_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(freeze_owner_counts.items())
    ) or "<li>none</li>"
    freeze_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for surface_id, counts in sorted(freeze_surface_counts.items())
    ) or "<li>none</li>"
    freeze_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · window={escape(entry['window'])}<br /><span>{escape(entry['summary'])}</span><br /><span>evidence={escape(entry['evidence'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_entries
    ) or "<li>none</li>"
    freeze_approval_entries = _build_freeze_approval_entries(pack)
    freeze_approval_owner_counts: Dict[str, int] = {}
    freeze_approval_status_counts: Dict[str, int] = {}
    for entry in freeze_approval_entries:
        freeze_approval_owner_counts[entry['freeze_approved_by']] = freeze_approval_owner_counts.get(entry['freeze_approved_by'], 0) + 1
        freeze_approval_status_counts[entry['status']] = freeze_approval_status_counts.get(entry['status'], 0) + 1
    freeze_approval_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(freeze_approval_owner_counts.items())
    ) or "<li>none</li>"
    freeze_approval_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(freeze_approval_status_counts.items())
    ) or "<li>none</li>"
    freeze_approval_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>owner={escape(entry['freeze_owner'])} · approved_by={escape(entry['freeze_approved_by'])} · approved_at={escape(entry['freeze_approved_at'])} · window={escape(entry['freeze_until'])}</span><br /><span>{escape(entry['summary'])}</span><br /><span>latest_event={escape(entry['latest_event'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_approval_entries
    ) or "<li>none</li>"
    freeze_renewal_entries = _build_freeze_renewal_entries(pack)
    freeze_renewal_owner_counts: Dict[str, int] = {}
    freeze_renewal_status_counts: Dict[str, int] = {}
    for entry in freeze_renewal_entries:
        freeze_renewal_owner_counts[entry['renewal_owner']] = freeze_renewal_owner_counts.get(entry['renewal_owner'], 0) + 1
        freeze_renewal_status_counts[entry['renewal_status']] = freeze_renewal_status_counts.get(entry['renewal_status'], 0) + 1
    freeze_renewal_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(freeze_renewal_owner_counts.items())
    ) or "<li>none</li>"
    freeze_renewal_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(freeze_renewal_status_counts.items())
    ) or "<li>none</li>"
    freeze_renewal_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>renewal_owner={escape(entry['renewal_owner'])} · renewal_by={escape(entry['renewal_by'])} · renewal_status={escape(entry['renewal_status'])}</span><br /><span>freeze_owner={escape(entry['freeze_owner'])} · freeze_until={escape(entry['freeze_until'])} · approved_by={escape(entry['freeze_approved_by'])}</span><br /><span>{escape(entry['summary'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_renewal_entries
    ) or "<li>none</li>"
    owner_review_queue = _build_owner_review_queue_entries(pack)
    owner_queue_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_review_queue:
        counts = owner_queue_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    owner_queue_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · checklist={counts['checklist']} · decisions={counts['decision']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_queue_counts.items())
    ) or "<li>none</li>"
    owner_queue_item_html = "".join(
        f"<li><strong>{escape(entry['queue_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>{escape(entry['summary'])}</span><br /><span>next_action={escape(entry['next_action'])}</span></li>"
        for entry in owner_review_queue
    ) or "<li>none</li>"
    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    status_summary_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(status_counts.items())
    ) or "<li>none</li>"
    actor_summary_html = "".join(
        f"<li><strong>{escape(actor)}</strong> · count={count}</li>"
        for actor, count in sorted(actor_counts.items())
    ) or "<li>none</li>"
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )
    latest_blocker_html = "".join(
        (
            f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest={escape(timeline_index[blocker.blocker_id][-1].event_id)} · actor={escape(timeline_index[blocker.blocker_id][-1].actor)} · status={escape(timeline_index[blocker.blocker_id][-1].status)} · timestamp={escape(timeline_index[blocker.blocker_id][-1].timestamp)}</li>"
            if blocker.blocker_id in timeline_index
            else f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest=none</li>"
        )
        for blocker in pack.blocker_log
    ) or "<li>none</li>"
    orphan_timeline_html = "".join(
        f"<li><strong>{escape(blocker_id)}</strong></li>"
        for blocker_id in orphan_timeline_ids
    ) or "<li>none</li>"
    return f'''<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{escape(pack.issue_id)} UI Review Pack</title>
    <style>
      body {{ font-family: Arial, sans-serif; margin: 32px; color: #0f172a; }}
      header {{ margin-bottom: 24px; }}
      h1 {{ margin-bottom: 4px; }}
      .meta {{ color: #475569; font-size: 0.95rem; }}
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
      <p>Checklist items missing role links: {escape(', '.join(audit.checklist_items_missing_role_links) if audit.checklist_items_missing_role_links else 'none')}</p>
      <p>Missing decision coverage: {escape(', '.join(audit.wireframes_missing_decisions) if audit.wireframes_missing_decisions else 'none')}</p>
      <p>Unresolved decisions missing follow-ups: {escape(', '.join(audit.unresolved_decisions_missing_follow_ups) if audit.unresolved_decisions_missing_follow_ups else 'none')}</p>
      <p>Missing role assignments: {escape(', '.join(audit.wireframes_missing_role_assignments) if audit.wireframes_missing_role_assignments else 'none')}</p>
      <p>Missing signoff coverage: {escape(', '.join(audit.wireframes_missing_signoffs) if audit.wireframes_missing_signoffs else 'none')}</p>
      <p>Decisions missing role links: {escape(', '.join(audit.decisions_missing_role_links) if audit.decisions_missing_role_links else 'none')}</p>
      <p>Missing blocker coverage: {escape(', '.join(audit.unresolved_required_signoffs_without_blockers) if audit.unresolved_required_signoffs_without_blockers else 'none')}</p>
      <p>Missing signoff requested dates: {escape(', '.join(audit.signoffs_missing_requested_dates) if audit.signoffs_missing_requested_dates else 'none')}</p>
      <p>Missing signoff due dates: {escape(', '.join(audit.signoffs_missing_due_dates) if audit.signoffs_missing_due_dates else 'none')}</p>
      <p>Missing signoff escalation owners: {escape(', '.join(audit.signoffs_missing_escalation_owners) if audit.signoffs_missing_escalation_owners else 'none')}</p>
      <p>Missing signoff reminder owners: {escape(', '.join(audit.signoffs_missing_reminder_owners) if audit.signoffs_missing_reminder_owners else 'none')}</p>
      <p>Missing signoff next reminders: {escape(', '.join(audit.signoffs_missing_next_reminders) if audit.signoffs_missing_next_reminders else 'none')}</p>
      <p>Missing signoff reminder cadence: {escape(', '.join(audit.signoffs_missing_reminder_cadence) if audit.signoffs_missing_reminder_cadence else 'none')}</p>
      <p>Breached signoff SLA: {escape(', '.join(audit.signoffs_with_breached_sla) if audit.signoffs_with_breached_sla else 'none')}</p>
      <p>Missing waiver metadata: {escape(', '.join(audit.waived_signoffs_missing_metadata) if audit.waived_signoffs_missing_metadata else 'none')}</p>
      <p>Missing blocker timeline: {escape(', '.join(audit.blockers_missing_timeline_events) if audit.blockers_missing_timeline_events else 'none')}</p>
      <p>Closed blockers missing resolution events: {escape(', '.join(audit.closed_blockers_missing_resolution_events) if audit.closed_blockers_missing_resolution_events else 'none')}</p>
      <p>Freeze exceptions missing owners: {escape(', '.join(audit.freeze_exceptions_missing_owners) if audit.freeze_exceptions_missing_owners else 'none')}</p>
      <p>Freeze exceptions missing windows: {escape(', '.join(audit.freeze_exceptions_missing_until) if audit.freeze_exceptions_missing_until else 'none')}</p>
      <p>Freeze exceptions missing approvers: {escape(', '.join(audit.freeze_exceptions_missing_approvers) if audit.freeze_exceptions_missing_approvers else 'none')}</p>
      <p>Freeze exceptions missing approval dates: {escape(', '.join(audit.freeze_exceptions_missing_approval_dates) if audit.freeze_exceptions_missing_approval_dates else 'none')}</p>
      <p>Freeze exceptions missing renewal owners: {escape(', '.join(audit.freeze_exceptions_missing_renewal_owners) if audit.freeze_exceptions_missing_renewal_owners else 'none')}</p>
      <p>Freeze exceptions missing renewal dates: {escape(', '.join(audit.freeze_exceptions_missing_renewal_dates) if audit.freeze_exceptions_missing_renewal_dates else 'none')}</p>
      <p>Orphan blocker timeline ids: {escape(', '.join(audit.orphan_blocker_timeline_blocker_ids) if audit.orphan_blocker_timeline_blocker_ids else 'none')}</p>
      <p>Handoff events missing targets: {escape(', '.join(audit.handoff_events_missing_targets) if audit.handoff_events_missing_targets else 'none')}</p>
      <p>Handoff events missing artifacts: {escape(', '.join(audit.handoff_events_missing_artifacts) if audit.handoff_events_missing_artifacts else 'none')}</p>
      <p>Handoff events missing ack owners: {escape(', '.join(audit.handoff_events_missing_ack_owners) if audit.handoff_events_missing_ack_owners else 'none')}</p>
      <p>Handoff events missing ack dates: {escape(', '.join(audit.handoff_events_missing_ack_dates) if audit.handoff_events_missing_ack_dates else 'none')}</p>
      <p>Unresolved decisions: {escape(', '.join(audit.unresolved_decision_ids) if audit.unresolved_decision_ids else 'none')}</p>
      <p>Unresolved required signoffs: {escape(', '.join(audit.unresolved_required_signoff_ids) if audit.unresolved_required_signoff_ids else 'none')}</p>
    </section>
    <section class="surface"><h2>Objectives</h2><ul>{objective_html}</ul></section>
    <section class="surface"><h2>Review Summary Board</h2><h3>Entries</h3><ul>{review_summary_item_html}</ul></section>
    <section class="surface"><h2>Objective Coverage Board</h2><h3>By Coverage Status</h3><ul>{objective_coverage_status_html}</ul><h3>By Persona</h3><ul>{objective_coverage_persona_html}</ul><h3>Entries</h3><ul>{objective_coverage_item_html}</ul></section>
    <section class="surface"><h2>Persona Readiness Board</h2><h3>By Readiness</h3><ul>{persona_readiness_status_html}</ul><h3>Entries</h3><ul>{persona_readiness_item_html}</ul></section>
    <section class="surface"><h2>Wireframes</h2><ul>{wireframe_html}</ul></section>
    <section class="surface"><h2>Wireframe Readiness Board</h2><h3>By Readiness</h3><ul>{wireframe_readiness_status_html}</ul><h3>By Device</h3><ul>{wireframe_device_html}</ul><h3>Entries</h3><ul>{wireframe_readiness_item_html}</ul></section>
    <section class="surface"><h2>Interactions</h2><ul>{interaction_html}</ul></section>
    <section class="surface"><h2>Interaction Coverage Board</h2><h3>By Coverage Status</h3><ul>{interaction_coverage_status_html}</ul><h3>By Surface</h3><ul>{interaction_coverage_surface_html}</ul><h3>Entries</h3><ul>{interaction_coverage_item_html}</ul></section>
    <section class="surface"><h2>Open Questions</h2><ul>{question_html}</ul></section>
    <section class="surface"><h2>Open Question Tracker</h2><h3>By Owner</h3><ul>{open_question_owner_html}</ul><h3>By Theme</h3><ul>{open_question_theme_html}</ul><h3>Entries</h3><ul>{open_question_item_html}</ul></section>
    <section class="surface"><h2>Reviewer Checklist</h2><ul>{checklist_html}</ul></section>
    <section class="surface"><h2>Decision Log</h2><ul>{decision_html}</ul></section>
    <section class="surface"><h2>Role Matrix</h2><ul>{role_matrix_html}</ul></section>
    <section class="surface"><h2>Checklist Traceability Board</h2><h3>By Owner</h3><ul>{checklist_trace_owner_html}</ul><h3>By Status</h3><ul>{checklist_trace_status_html}</ul><h3>Entries</h3><ul>{checklist_trace_item_html}</ul></section>
    <section class="surface"><h2>Decision Follow-up Tracker</h2><h3>By Owner</h3><ul>{decision_followup_owner_html}</ul><h3>By Status</h3><ul>{decision_followup_status_html}</ul><h3>Entries</h3><ul>{decision_followup_item_html}</ul></section>
    <section class="surface"><h2>Role Coverage Board</h2><h3>By Surface</h3><ul>{role_coverage_surface_html}</ul><h3>By Status</h3><ul>{role_coverage_status_html}</ul><h3>Entries</h3><ul>{role_coverage_item_html}</ul></section>
    <section class="surface"><h2>Signoff Dependency Board</h2><h3>By Dependency Status</h3><ul>{signoff_dependency_status_html}</ul><h3>By SLA State</h3><ul>{signoff_dependency_sla_html}</ul><h3>Entries</h3><ul>{signoff_dependency_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Log</h2><ul>{signoff_html}</ul></section>
    <section class="surface"><h2>Sign-off SLA Dashboard</h2><h3>SLA States</h3><ul>{signoff_sla_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_sla_owner_html}</ul><h3>Sign-offs</h3><ul>{signoff_sla_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Reminder Queue</h2><h3>By Owner</h3><ul>{signoff_reminder_owner_html}</ul><h3>By Channel</h3><ul>{signoff_reminder_channel_html}</ul><h3>Items</h3><ul>{signoff_reminder_item_html}</ul></section>
    <section class="surface"><h2>Reminder Cadence Board</h2><h3>By Cadence</h3><ul>{reminder_cadence_owner_html}</ul><h3>By Status</h3><ul>{reminder_cadence_status_html}</ul><h3>Items</h3><ul>{reminder_cadence_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Breach Board</h2><h3>SLA States</h3><ul>{signoff_breach_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_breach_owner_html}</ul><h3>Items</h3><ul>{signoff_breach_item_html}</ul></section>
    <section class="surface"><h2>Escalation Dashboard</h2><h3>By Escalation Owner</h3><ul>{escalation_owner_html}</ul><h3>By Status</h3><ul>{escalation_status_html}</ul><h3>Escalations</h3><ul>{escalation_item_html}</ul></section>
    <section class="surface"><h2>Escalation Handoff Ledger</h2><h3>By Status</h3><ul>{escalation_handoff_status_html}</ul><h3>By Channel</h3><ul>{escalation_handoff_channel_html}</ul><h3>Entries</h3><ul>{escalation_handoff_item_html}</ul></section>
    <section class="surface"><h2>Handoff Ack Ledger</h2><h3>By Ack Owner</h3><ul>{handoff_ack_owner_html}</ul><h3>By Ack Status</h3><ul>{handoff_ack_status_html}</ul><h3>Entries</h3><ul>{handoff_ack_item_html}</ul></section>
    <section class="surface"><h2>Owner Escalation Digest</h2><h3>Owners</h3><ul>{owner_escalation_owner_html}</ul><h3>Items</h3><ul>{owner_escalation_item_html}</ul></section>
    <section class="surface"><h2>Owner Workload Board</h2><h3>Owners</h3><ul>{owner_workload_owner_html}</ul><h3>Items</h3><ul>{owner_workload_item_html}</ul></section>
    <section class="surface"><h2>Blocker Log</h2><ul>{blocker_html}</ul></section>
    <section class="surface"><h2>Blocker Timeline</h2><ul>{blocker_timeline_html}</ul></section>
    <section class="surface"><h2>Review Freeze Exception Board</h2><h3>By Owner</h3><ul>{freeze_owner_html}</ul><h3>By Surface</h3><ul>{freeze_surface_html}</ul><h3>Entries</h3><ul>{freeze_item_html}</ul></section>
    <section class="surface"><h2>Freeze Approval Trail</h2><h3>By Approver</h3><ul>{freeze_approval_owner_html}</ul><h3>By Status</h3><ul>{freeze_approval_status_html}</ul><h3>Entries</h3><ul>{freeze_approval_item_html}</ul></section>
    <section class="surface"><h2>Freeze Renewal Tracker</h2><h3>By Renewal Owner</h3><ul>{freeze_renewal_owner_html}</ul><h3>By Renewal Status</h3><ul>{freeze_renewal_status_html}</ul><h3>Entries</h3><ul>{freeze_renewal_item_html}</ul></section>
    <section class="surface"><h2>Review Exceptions</h2><ul>{exception_html}</ul></section>
    <section class="surface"><h2>Review Exception Matrix</h2><h3>By Owner</h3><ul>{exception_owner_html}</ul><h3>By Status</h3><ul>{exception_status_html}</ul><h3>By Surface</h3><ul>{exception_surface_html}</ul></section>
    <section class="surface"><h2>Audit Density Board</h2><h3>By Load Band</h3><ul>{audit_density_band_html}</ul><h3>Entries</h3><ul>{audit_density_item_html}</ul></section>
    <section class="surface"><h2>Owner Review Queue</h2><h3>Owners</h3><ul>{owner_queue_owner_html}</ul><h3>Items</h3><ul>{owner_queue_item_html}</ul></section>
    <section class="surface"><h2>Blocker Timeline Summary</h2><h3>Events by Status</h3><ul>{status_summary_html}</ul><h3>Events by Actor</h3><ul>{actor_summary_html}</ul><h3>Latest Blocker Events</h3><ul>{latest_blocker_html}</ul><h3>Orphan Timeline Blockers</h3><ul>{orphan_timeline_html}</ul></section>
  </body>
</html>
'''


def write_ui_review_pack_bundle(root_dir: str, pack: UIReviewPack) -> UIReviewPackArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    slug = pack.issue_id.lower().replace(" ", "-")
    markdown_path = str(base / f"{slug}-review-pack.md")
    html_path = str(base / f"{slug}-review-pack.html")
    decision_log_path = str(base / f"{slug}-decision-log.md")
    review_summary_board_path = str(base / f"{slug}-review-summary-board.md")
    objective_coverage_board_path = str(base / f"{slug}-objective-coverage-board.md")
    persona_readiness_board_path = str(base / f"{slug}-persona-readiness-board.md")
    wireframe_readiness_board_path = str(base / f"{slug}-wireframe-readiness-board.md")
    interaction_coverage_board_path = str(base / f"{slug}-interaction-coverage-board.md")
    open_question_tracker_path = str(base / f"{slug}-open-question-tracker.md")
    checklist_traceability_board_path = str(base / f"{slug}-checklist-traceability-board.md")
    decision_followup_tracker_path = str(base / f"{slug}-decision-followup-tracker.md")
    role_matrix_path = str(base / f"{slug}-role-matrix.md")
    role_coverage_board_path = str(base / f"{slug}-role-coverage-board.md")
    signoff_dependency_board_path = str(base / f"{slug}-signoff-dependency-board.md")
    signoff_log_path = str(base / f"{slug}-signoff-log.md")
    signoff_sla_dashboard_path = str(base / f"{slug}-signoff-sla-dashboard.md")
    signoff_reminder_queue_path = str(base / f"{slug}-signoff-reminder-queue.md")
    reminder_cadence_board_path = str(base / f"{slug}-reminder-cadence-board.md")
    signoff_breach_board_path = str(base / f"{slug}-signoff-breach-board.md")
    escalation_dashboard_path = str(base / f"{slug}-escalation-dashboard.md")
    escalation_handoff_ledger_path = str(base / f"{slug}-escalation-handoff-ledger.md")
    handoff_ack_ledger_path = str(base / f"{slug}-handoff-ack-ledger.md")
    owner_escalation_digest_path = str(base / f"{slug}-owner-escalation-digest.md")
    owner_workload_board_path = str(base / f"{slug}-owner-workload-board.md")
    blocker_log_path = str(base / f"{slug}-blocker-log.md")
    blocker_timeline_path = str(base / f"{slug}-blocker-timeline.md")
    freeze_exception_board_path = str(base / f"{slug}-freeze-exception-board.md")
    freeze_approval_trail_path = str(base / f"{slug}-freeze-approval-trail.md")
    freeze_renewal_tracker_path = str(base / f"{slug}-freeze-renewal-tracker.md")
    exception_log_path = str(base / f"{slug}-exception-log.md")
    exception_matrix_path = str(base / f"{slug}-exception-matrix.md")
    audit_density_board_path = str(base / f"{slug}-audit-density-board.md")
    owner_review_queue_path = str(base / f"{slug}-owner-review-queue.md")
    blocker_timeline_summary_path = str(base / f"{slug}-blocker-timeline-summary.md")
    audit = UIReviewPackAuditor().audit(pack)
    Path(markdown_path).write_text(render_ui_review_pack_report(pack, audit))
    Path(html_path).write_text(render_ui_review_pack_html(pack, audit))
    Path(decision_log_path).write_text(render_ui_review_decision_log(pack))
    Path(review_summary_board_path).write_text(render_ui_review_review_summary_board(pack))
    Path(objective_coverage_board_path).write_text(render_ui_review_objective_coverage_board(pack))
    Path(persona_readiness_board_path).write_text(render_ui_review_persona_readiness_board(pack))
    Path(wireframe_readiness_board_path).write_text(render_ui_review_wireframe_readiness_board(pack))
    Path(interaction_coverage_board_path).write_text(render_ui_review_interaction_coverage_board(pack))
    Path(open_question_tracker_path).write_text(render_ui_review_open_question_tracker(pack))
    Path(checklist_traceability_board_path).write_text(render_ui_review_checklist_traceability_board(pack))
    Path(decision_followup_tracker_path).write_text(render_ui_review_decision_followup_tracker(pack))
    Path(role_matrix_path).write_text(render_ui_review_role_matrix(pack))
    Path(role_coverage_board_path).write_text(render_ui_review_role_coverage_board(pack))
    Path(signoff_dependency_board_path).write_text(render_ui_review_signoff_dependency_board(pack))
    Path(signoff_log_path).write_text(render_ui_review_signoff_log(pack))
    Path(signoff_sla_dashboard_path).write_text(render_ui_review_signoff_sla_dashboard(pack))
    Path(signoff_reminder_queue_path).write_text(render_ui_review_signoff_reminder_queue(pack))
    Path(reminder_cadence_board_path).write_text(render_ui_review_reminder_cadence_board(pack))
    Path(signoff_breach_board_path).write_text(render_ui_review_signoff_breach_board(pack))
    Path(escalation_dashboard_path).write_text(render_ui_review_escalation_dashboard(pack))
    Path(escalation_handoff_ledger_path).write_text(render_ui_review_escalation_handoff_ledger(pack))
    Path(handoff_ack_ledger_path).write_text(render_ui_review_handoff_ack_ledger(pack))
    Path(owner_escalation_digest_path).write_text(render_ui_review_owner_escalation_digest(pack))
    Path(owner_workload_board_path).write_text(render_ui_review_owner_workload_board(pack))
    Path(blocker_log_path).write_text(render_ui_review_blocker_log(pack))
    Path(blocker_timeline_path).write_text(render_ui_review_blocker_timeline(pack))
    Path(freeze_exception_board_path).write_text(render_ui_review_freeze_exception_board(pack))
    Path(freeze_approval_trail_path).write_text(render_ui_review_freeze_approval_trail(pack))
    Path(freeze_renewal_tracker_path).write_text(render_ui_review_freeze_renewal_tracker(pack))
    Path(exception_log_path).write_text(render_ui_review_exception_log(pack))
    Path(exception_matrix_path).write_text(render_ui_review_exception_matrix(pack))
    Path(audit_density_board_path).write_text(render_ui_review_audit_density_board(pack))
    Path(owner_review_queue_path).write_text(render_ui_review_owner_review_queue(pack))
    Path(blocker_timeline_summary_path).write_text(render_ui_review_blocker_timeline_summary(pack))
    return UIReviewPackArtifacts(
        root_dir=str(base),
        markdown_path=markdown_path,
        html_path=html_path,
        decision_log_path=decision_log_path,
        review_summary_board_path=review_summary_board_path,
        objective_coverage_board_path=objective_coverage_board_path,
        persona_readiness_board_path=persona_readiness_board_path,
        wireframe_readiness_board_path=wireframe_readiness_board_path,
        interaction_coverage_board_path=interaction_coverage_board_path,
        open_question_tracker_path=open_question_tracker_path,
        checklist_traceability_board_path=checklist_traceability_board_path,
        decision_followup_tracker_path=decision_followup_tracker_path,
        role_matrix_path=role_matrix_path,
        role_coverage_board_path=role_coverage_board_path,
        signoff_dependency_board_path=signoff_dependency_board_path,
        signoff_log_path=signoff_log_path,
        signoff_sla_dashboard_path=signoff_sla_dashboard_path,
        signoff_reminder_queue_path=signoff_reminder_queue_path,
        reminder_cadence_board_path=reminder_cadence_board_path,
        signoff_breach_board_path=signoff_breach_board_path,
        escalation_dashboard_path=escalation_dashboard_path,
        escalation_handoff_ledger_path=escalation_handoff_ledger_path,
        handoff_ack_ledger_path=handoff_ack_ledger_path,
        owner_escalation_digest_path=owner_escalation_digest_path,
        owner_workload_board_path=owner_workload_board_path,
        blocker_log_path=blocker_log_path,
        blocker_timeline_path=blocker_timeline_path,
        freeze_exception_board_path=freeze_exception_board_path,
        freeze_approval_trail_path=freeze_approval_trail_path,
        freeze_renewal_tracker_path=freeze_renewal_tracker_path,
        exception_log_path=exception_log_path,
        exception_matrix_path=exception_matrix_path,
        audit_density_board_path=audit_density_board_path,
        owner_review_queue_path=owner_review_queue_path,
        blocker_timeline_summary_path=blocker_timeline_summary_path,
    )
