from dataclasses import dataclass, field
from typing import Dict, List


FOUNDATION_CATEGORIES = ("color", "spacing", "typography", "motion", "radius")
COMPONENT_READINESS_ORDER = {"draft": 0, "alpha": 1, "beta": 2, "stable": 3}
REQUIRED_INTERACTION_STATES = {"default", "hover", "disabled"}


@dataclass(frozen=True)
class DesignToken:
    name: str
    category: str
    value: str
    semantic_role: str = ""
    theme: str = "core"

    def to_dict(self) -> Dict[str, str]:
        return {
            "name": self.name,
            "category": self.category,
            "value": self.value,
            "semantic_role": self.semantic_role,
            "theme": self.theme,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "DesignToken":
        return cls(
            name=data["name"],
            category=data["category"],
            value=data["value"],
            semantic_role=data.get("semantic_role", ""),
            theme=data.get("theme", "core"),
        )


@dataclass
class ComponentVariant:
    name: str
    tokens: List[str] = field(default_factory=list)
    states: List[str] = field(default_factory=list)
    usage_notes: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "tokens": list(self.tokens),
            "states": list(self.states),
            "usage_notes": self.usage_notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ComponentVariant":
        return cls(
            name=str(data["name"]),
            tokens=[str(token) for token in data.get("tokens", [])],
            states=[str(state) for state in data.get("states", [])],
            usage_notes=str(data.get("usage_notes", "")),
        )


@dataclass
class ComponentSpec:
    name: str
    readiness: str = "draft"
    slots: List[str] = field(default_factory=list)
    variants: List[ComponentVariant] = field(default_factory=list)
    accessibility_requirements: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def token_names(self) -> List[str]:
        names: List[str] = []
        for variant in self.variants:
            for token in variant.tokens:
                if token not in names:
                    names.append(token)
        return names

    @property
    def state_coverage(self) -> List[str]:
        coverage: List[str] = []
        for variant in self.variants:
            for state in variant.states:
                if state not in coverage:
                    coverage.append(state)
        return coverage

    @property
    def missing_required_states(self) -> List[str]:
        return sorted(REQUIRED_INTERACTION_STATES.difference(self.state_coverage))

    @property
    def release_ready(self) -> bool:
        return (
            COMPONENT_READINESS_ORDER.get(self.readiness, -1) >= COMPONENT_READINESS_ORDER["beta"]
            and self.documentation_complete
            and bool(self.accessibility_requirements)
            and not self.missing_required_states
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "readiness": self.readiness,
            "slots": list(self.slots),
            "variants": [variant.to_dict() for variant in self.variants],
            "accessibility_requirements": list(self.accessibility_requirements),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ComponentSpec":
        return cls(
            name=str(data["name"]),
            readiness=str(data.get("readiness", "draft")),
            slots=[str(slot) for slot in data.get("slots", [])],
            variants=[ComponentVariant.from_dict(variant) for variant in data.get("variants", [])],
            accessibility_requirements=[
                str(requirement) for requirement in data.get("accessibility_requirements", [])
            ],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class DesignSystemAudit:
    system_name: str
    version: str
    token_counts: Dict[str, int]
    component_count: int
    release_ready_components: List[str] = field(default_factory=list)
    components_missing_docs: List[str] = field(default_factory=list)
    components_missing_accessibility: List[str] = field(default_factory=list)
    components_missing_states: List[str] = field(default_factory=list)
    undefined_token_refs: Dict[str, List[str]] = field(default_factory=dict)
    token_orphans: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.component_count == 0:
            return 0.0
        ready = len(self.release_ready_components)
        penalties = (
            len(self.components_missing_docs)
            + len(self.components_missing_accessibility)
            + len(self.components_missing_states)
        )
        score = max(0.0, ((ready * 100) - (penalties * 10)) / self.component_count)
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "system_name": self.system_name,
            "version": self.version,
            "token_counts": dict(self.token_counts),
            "component_count": self.component_count,
            "release_ready_components": list(self.release_ready_components),
            "components_missing_docs": list(self.components_missing_docs),
            "components_missing_accessibility": list(self.components_missing_accessibility),
            "components_missing_states": list(self.components_missing_states),
            "undefined_token_refs": {name: list(tokens) for name, tokens in self.undefined_token_refs.items()},
            "token_orphans": list(self.token_orphans),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DesignSystemAudit":
        return cls(
            system_name=str(data["system_name"]),
            version=str(data["version"]),
            token_counts={str(name): int(count) for name, count in dict(data.get("token_counts", {})).items()},
            component_count=int(data.get("component_count", 0)),
            release_ready_components=[str(name) for name in data.get("release_ready_components", [])],
            components_missing_docs=[str(name) for name in data.get("components_missing_docs", [])],
            components_missing_accessibility=[
                str(name) for name in data.get("components_missing_accessibility", [])
            ],
            components_missing_states=[str(name) for name in data.get("components_missing_states", [])],
            undefined_token_refs={
                str(name): [str(token) for token in tokens]
                for name, tokens in dict(data.get("undefined_token_refs", {})).items()
            },
            token_orphans=[str(token) for token in data.get("token_orphans", [])],
        )


@dataclass
class CommandAction:
    id: str
    title: str
    section: str
    shortcut: str = ""

    def to_dict(self) -> Dict[str, str]:
        return {
            "id": self.id,
            "title": self.title,
            "section": self.section,
            "shortcut": self.shortcut,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "CommandAction":
        return cls(
            id=data["id"],
            title=data["title"],
            section=data["section"],
            shortcut=data.get("shortcut", ""),
        )


@dataclass
class ConsoleCommandEntry:
    trigger_label: str
    placeholder: str
    shortcut: str
    commands: List[CommandAction] = field(default_factory=list)
    recent_queries_enabled: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "trigger_label": self.trigger_label,
            "placeholder": self.placeholder,
            "shortcut": self.shortcut,
            "commands": [command.to_dict() for command in self.commands],
            "recent_queries_enabled": self.recent_queries_enabled,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleCommandEntry":
        return cls(
            trigger_label=str(data["trigger_label"]),
            placeholder=str(data["placeholder"]),
            shortcut=str(data["shortcut"]),
            commands=[CommandAction.from_dict(command) for command in data.get("commands", [])],
            recent_queries_enabled=bool(data.get("recent_queries_enabled", False)),
        )


@dataclass
class ConsoleTopBar:
    name: str
    search_placeholder: str
    environment_options: List[str] = field(default_factory=list)
    time_range_options: List[str] = field(default_factory=list)
    alert_channels: List[str] = field(default_factory=list)
    command_entry: ConsoleCommandEntry = field(
        default_factory=lambda: ConsoleCommandEntry(trigger_label="", placeholder="", shortcut="")
    )
    documentation_complete: bool = False
    accessibility_requirements: List[str] = field(default_factory=list)

    @property
    def has_global_search(self) -> bool:
        return bool(self.search_placeholder.strip())

    @property
    def has_environment_switch(self) -> bool:
        return len(self.environment_options) >= 2

    @property
    def has_time_range_switch(self) -> bool:
        return len(self.time_range_options) >= 2

    @property
    def has_alert_entry(self) -> bool:
        return bool(self.alert_channels)

    @property
    def has_command_shell(self) -> bool:
        return bool(self.command_entry.trigger_label.strip()) and bool(self.command_entry.commands)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "search_placeholder": self.search_placeholder,
            "environment_options": list(self.environment_options),
            "time_range_options": list(self.time_range_options),
            "alert_channels": list(self.alert_channels),
            "command_entry": self.command_entry.to_dict(),
            "documentation_complete": self.documentation_complete,
            "accessibility_requirements": list(self.accessibility_requirements),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleTopBar":
        return cls(
            name=str(data["name"]),
            search_placeholder=str(data.get("search_placeholder", "")),
            environment_options=[str(option) for option in data.get("environment_options", [])],
            time_range_options=[str(option) for option in data.get("time_range_options", [])],
            alert_channels=[str(channel) for channel in data.get("alert_channels", [])],
            command_entry=ConsoleCommandEntry.from_dict(dict(data.get("command_entry", {}))),
            documentation_complete=bool(data.get("documentation_complete", False)),
            accessibility_requirements=[
                str(requirement) for requirement in data.get("accessibility_requirements", [])
            ],
        )


@dataclass
class ConsoleTopBarAudit:
    name: str
    missing_capabilities: List[str] = field(default_factory=list)
    documentation_complete: bool = False
    accessibility_complete: bool = False
    command_shortcut_supported: bool = False
    command_count: int = 0

    @property
    def release_ready(self) -> bool:
        return (
            not self.missing_capabilities
            and self.documentation_complete
            and self.accessibility_complete
            and self.command_shortcut_supported
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "missing_capabilities": list(self.missing_capabilities),
            "documentation_complete": self.documentation_complete,
            "accessibility_complete": self.accessibility_complete,
            "command_shortcut_supported": self.command_shortcut_supported,
            "command_count": self.command_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleTopBarAudit":
        return cls(
            name=str(data["name"]),
            missing_capabilities=[str(item) for item in data.get("missing_capabilities", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
            accessibility_complete=bool(data.get("accessibility_complete", False)),
            command_shortcut_supported=bool(data.get("command_shortcut_supported", False)),
            command_count=int(data.get("command_count", 0)),
        )


@dataclass
class DesignSystem:
    name: str
    version: str
    tokens: List[DesignToken] = field(default_factory=list)
    components: List[ComponentSpec] = field(default_factory=list)

    @property
    def token_counts(self) -> Dict[str, int]:
        counts = {category: 0 for category in FOUNDATION_CATEGORIES}
        for token in self.tokens:
            counts[token.category] = counts.get(token.category, 0) + 1
        return counts

    @property
    def token_index(self) -> Dict[str, DesignToken]:
        return {token.name: token for token in self.tokens}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "tokens": [token.to_dict() for token in self.tokens],
            "components": [component.to_dict() for component in self.components],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DesignSystem":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            tokens=[DesignToken.from_dict(token) for token in data.get("tokens", [])],
            components=[ComponentSpec.from_dict(component) for component in data.get("components", [])],
        )


class ComponentLibrary:
    def audit(self, system: DesignSystem) -> DesignSystemAudit:
        used_tokens = set()
        release_ready_components: List[str] = []
        components_missing_docs: List[str] = []
        components_missing_accessibility: List[str] = []
        components_missing_states: List[str] = []
        undefined_token_refs: Dict[str, List[str]] = {}
        token_index = system.token_index

        for component in system.components:
            used_tokens.update(component.token_names)
            missing_tokens = sorted(token for token in component.token_names if token not in token_index)
            if missing_tokens:
                undefined_token_refs[component.name] = missing_tokens
            if component.release_ready and not missing_tokens:
                release_ready_components.append(component.name)
            if not component.documentation_complete:
                components_missing_docs.append(component.name)
            if not component.accessibility_requirements:
                components_missing_accessibility.append(component.name)
            if component.missing_required_states:
                components_missing_states.append(component.name)

        token_orphans = sorted(token.name for token in system.tokens if token.name not in used_tokens)
        return DesignSystemAudit(
            system_name=system.name,
            version=system.version,
            token_counts=system.token_counts,
            component_count=len(system.components),
            release_ready_components=sorted(release_ready_components),
            components_missing_docs=sorted(components_missing_docs),
            components_missing_accessibility=sorted(components_missing_accessibility),
            components_missing_states=sorted(components_missing_states),
            undefined_token_refs=undefined_token_refs,
            token_orphans=token_orphans,
        )


class ConsoleChromeLibrary:
    REQUIRED_SHORTCUTS = {"cmd+k", "ctrl+k"}
    REQUIRED_ACCESSIBILITY = {"keyboard-navigation", "screen-reader-label", "focus-visible"}

    def audit_top_bar(self, top_bar: ConsoleTopBar) -> ConsoleTopBarAudit:
        missing_capabilities: List[str] = []
        if not top_bar.has_global_search:
            missing_capabilities.append("global-search")
        if not top_bar.has_time_range_switch:
            missing_capabilities.append("time-range-switch")
        if not top_bar.has_environment_switch:
            missing_capabilities.append("environment-switch")
        if not top_bar.has_alert_entry:
            missing_capabilities.append("alert-entry")
        if not top_bar.has_command_shell:
            missing_capabilities.append("command-shell")

        normalized_shortcuts = {
            item.strip().lower().replace(" ", "")
            for item in top_bar.command_entry.shortcut.split("/")
            if item.strip()
        }
        accessibility_complete = self.REQUIRED_ACCESSIBILITY.issubset(set(top_bar.accessibility_requirements))
        return ConsoleTopBarAudit(
            name=top_bar.name,
            missing_capabilities=missing_capabilities,
            documentation_complete=top_bar.documentation_complete,
            accessibility_complete=accessibility_complete,
            command_shortcut_supported=self.REQUIRED_SHORTCUTS.issubset(normalized_shortcuts),
            command_count=len(top_bar.command_entry.commands),
        )


def render_design_system_report(system: DesignSystem, audit: DesignSystemAudit) -> str:
    lines = [
        "# Design System Report",
        "",
        f"- Name: {system.name}",
        f"- Version: {system.version}",
        f"- Components: {audit.component_count}",
        f"- Release Ready Components: {len(audit.release_ready_components)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Token Foundations",
        "",
    ]

    for category, count in audit.token_counts.items():
        lines.append(f"- {category}: {count}")

    lines.extend(["", "## Component Status", ""])
    if system.components:
        for component in system.components:
            states = ", ".join(component.state_coverage) or "none"
            missing_states = ", ".join(component.missing_required_states) or "none"
            undefined_tokens = ", ".join(audit.undefined_token_refs.get(component.name, [])) or "none"
            lines.append(
                f"- {component.name}: readiness={component.readiness} docs={component.documentation_complete} "
                f"a11y={bool(component.accessibility_requirements)} states={states} missing_states={missing_states} "
                f"undefined_tokens={undefined_tokens}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        f"- Missing docs: {', '.join(audit.components_missing_docs) if audit.components_missing_docs else 'none'}"
    )
    lines.append(
        "- Missing accessibility: "
        f"{', '.join(audit.components_missing_accessibility) if audit.components_missing_accessibility else 'none'}"
    )
    lines.append(
        f"- Missing interaction states: {', '.join(audit.components_missing_states) if audit.components_missing_states else 'none'}"
    )
    if audit.undefined_token_refs:
        undefined_refs = "; ".join(
            f"{component}={', '.join(tokens)}" for component, tokens in sorted(audit.undefined_token_refs.items())
        )
    else:
        undefined_refs = "none"
    lines.append(f"- Undefined token refs: {undefined_refs}")
    lines.append(f"- Orphan tokens: {', '.join(audit.token_orphans) if audit.token_orphans else 'none'}")
    return "\n".join(lines) + "\n"


def render_console_top_bar_report(top_bar: ConsoleTopBar, audit: ConsoleTopBarAudit) -> str:
    lines = [
        "# Console Top Bar Report",
        "",
        f"- Name: {top_bar.name}",
        f"- Global Search: {top_bar.has_global_search}",
        f"- Environment Switch: {', '.join(top_bar.environment_options) if top_bar.environment_options else 'none'}",
        f"- Time Range Switch: {', '.join(top_bar.time_range_options) if top_bar.time_range_options else 'none'}",
        f"- Alert Entry: {', '.join(top_bar.alert_channels) if top_bar.alert_channels else 'none'}",
        f"- Command Trigger: {top_bar.command_entry.trigger_label or 'none'}",
        f"- Command Shortcut: {top_bar.command_entry.shortcut or 'none'}",
        f"- Command Count: {audit.command_count}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Command Palette",
        "",
    ]
    if top_bar.command_entry.commands:
        for command in top_bar.command_entry.commands:
            shortcut = command.shortcut or "none"
            lines.append(f"- {command.id}: {command.title} [{command.section}] shortcut={shortcut}")
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        f"- Missing capabilities: {', '.join(audit.missing_capabilities) if audit.missing_capabilities else 'none'}"
    )
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    lines.append(f"- Accessibility complete: {audit.accessibility_complete}")
    lines.append(f"- Cmd/Ctrl+K supported: {audit.command_shortcut_supported}")
    return "\n".join(lines) + "\n"
