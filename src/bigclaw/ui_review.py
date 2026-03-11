from dataclasses import dataclass, field
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


@dataclass
class UIReviewPack:
    issue_id: str
    title: str
    version: str
    objectives: List[ReviewObjective] = field(default_factory=list)
    wireframes: List[WireframeSurface] = field(default_factory=list)
    interactions: List[InteractionFlow] = field(default_factory=list)
    open_questions: List[OpenQuestion] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "version": self.version,
            "objectives": [objective.to_dict() for objective in self.objectives],
            "wireframes": [wireframe.to_dict() for wireframe in self.wireframes],
            "interactions": [interaction.to_dict() for interaction in self.interactions],
            "open_questions": [question.to_dict() for question in self.open_questions],
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
        )


@dataclass(frozen=True)
class UIReviewPackAudit:
    ready: bool
    objective_count: int
    wireframe_count: int
    interaction_count: int
    open_question_count: int
    missing_sections: List[str] = field(default_factory=list)
    objectives_missing_signals: List[str] = field(default_factory=list)
    wireframes_missing_blocks: List[str] = field(default_factory=list)
    interactions_missing_states: List[str] = field(default_factory=list)
    unresolved_question_ids: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: objectives={self.objective_count} "
            f"wireframes={self.wireframe_count} "
            f"interactions={self.interaction_count} "
            f"open_questions={self.open_question_count}"
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
        ready = not (
            missing_sections
            or objectives_missing_signals
            or wireframes_missing_blocks
            or interactions_missing_states
        )
        return UIReviewPackAudit(
            ready=ready,
            objective_count=len(pack.objectives),
            wireframe_count=len(pack.wireframes),
            interaction_count=len(pack.interactions),
            open_question_count=len(pack.open_questions),
            missing_sections=missing_sections,
            objectives_missing_signals=objectives_missing_signals,
            wireframes_missing_blocks=wireframes_missing_blocks,
            interactions_missing_states=interactions_missing_states,
            unresolved_question_ids=unresolved_question_ids,
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

    lines.extend(
        [
            "",
            "## Audit Findings",
            f"- Missing sections: {', '.join(audit.missing_sections) or 'none'}",
            f"- Objectives missing success signals: {', '.join(audit.objectives_missing_signals) or 'none'}",
            f"- Wireframes missing blocks: {', '.join(audit.wireframes_missing_blocks) or 'none'}",
            f"- Interactions missing states: {', '.join(audit.interactions_missing_states) or 'none'}",
            f"- Unresolved questions: {', '.join(audit.unresolved_question_ids) or 'none'}",
        ]
    )
    return "\n".join(lines)
