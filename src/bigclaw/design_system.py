from dataclasses import dataclass, field
from typing import Dict, List, Optional


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


def _normalize_route_path(path: str) -> str:
    stripped = path.strip("/")
    return f"/{stripped}" if stripped else "/"


@dataclass(frozen=True)
class NavigationRoute:
    path: str
    screen_id: str
    title: str
    nav_node_id: str = ""
    layout: str = "workspace"

    def __post_init__(self) -> None:
        object.__setattr__(self, "path", _normalize_route_path(self.path))

    def to_dict(self) -> Dict[str, str]:
        return {
            "path": self.path,
            "screen_id": self.screen_id,
            "title": self.title,
            "nav_node_id": self.nav_node_id,
            "layout": self.layout,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "NavigationRoute":
        return cls(
            path=data["path"],
            screen_id=data["screen_id"],
            title=data["title"],
            nav_node_id=data.get("nav_node_id", ""),
            layout=data.get("layout", "workspace"),
        )


@dataclass
class NavigationNode:
    node_id: str
    title: str
    segment: str
    screen_id: str = ""
    children: List["NavigationNode"] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "node_id": self.node_id,
            "title": self.title,
            "segment": self.segment,
            "screen_id": self.screen_id,
            "children": [child.to_dict() for child in self.children],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "NavigationNode":
        return cls(
            node_id=str(data["node_id"]),
            title=str(data["title"]),
            segment=str(data.get("segment", "")),
            screen_id=str(data.get("screen_id", "")),
            children=[cls.from_dict(child) for child in data.get("children", [])],
        )


@dataclass(frozen=True)
class NavigationEntry:
    node_id: str
    title: str
    path: str
    depth: int
    parent_id: str = ""
    screen_id: str = ""


@dataclass
class InformationArchitectureAudit:
    total_navigation_nodes: int
    total_routes: int
    duplicate_routes: List[str] = field(default_factory=list)
    missing_route_nodes: Dict[str, str] = field(default_factory=dict)
    secondary_nav_gaps: Dict[str, List[str]] = field(default_factory=dict)
    orphan_routes: List[str] = field(default_factory=list)

    @property
    def healthy(self) -> bool:
        return not (
            self.duplicate_routes
            or self.missing_route_nodes
            or self.secondary_nav_gaps
            or self.orphan_routes
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "total_navigation_nodes": self.total_navigation_nodes,
            "total_routes": self.total_routes,
            "duplicate_routes": list(self.duplicate_routes),
            "missing_route_nodes": dict(self.missing_route_nodes),
            "secondary_nav_gaps": {
                section: list(paths) for section, paths in self.secondary_nav_gaps.items()
            },
            "orphan_routes": list(self.orphan_routes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitectureAudit":
        return cls(
            total_navigation_nodes=int(data.get("total_navigation_nodes", 0)),
            total_routes=int(data.get("total_routes", 0)),
            duplicate_routes=[str(path) for path in data.get("duplicate_routes", [])],
            missing_route_nodes={
                str(node_id): str(path) for node_id, path in dict(data.get("missing_route_nodes", {})).items()
            },
            secondary_nav_gaps={
                str(section): [str(path) for path in paths]
                for section, paths in dict(data.get("secondary_nav_gaps", {})).items()
            },
            orphan_routes=[str(path) for path in data.get("orphan_routes", [])],
        )


@dataclass
class InformationArchitecture:
    global_nav: List[NavigationNode] = field(default_factory=list)
    routes: List[NavigationRoute] = field(default_factory=list)

    @property
    def route_index(self) -> Dict[str, NavigationRoute]:
        index: Dict[str, NavigationRoute] = {}
        for route in self.routes:
            if route.path not in index:
                index[route.path] = route
        return index

    @property
    def navigation_entries(self) -> List[NavigationEntry]:
        entries: List[NavigationEntry] = []
        for node in self.global_nav:
            entries.extend(self._flatten_node(node=node, parent_path="", depth=0, parent_id=""))
        return entries

    def resolve_route(self, path: str) -> Optional[NavigationRoute]:
        return self.route_index.get(_normalize_route_path(path))

    def audit(self) -> InformationArchitectureAudit:
        entries = self.navigation_entries
        route_counts: Dict[str, int] = {}
        for route in self.routes:
            route_counts[route.path] = route_counts.get(route.path, 0) + 1

        duplicate_routes = sorted(path for path, count in route_counts.items() if count > 1)
        route_index = self.route_index
        missing_route_nodes = {
            entry.node_id: entry.path
            for entry in entries
            if entry.path not in route_index
        }

        secondary_nav_gaps: Dict[str, List[str]] = {}
        for root in self.global_nav:
            gaps = sorted(self._missing_paths_for_descendants(root, parent_path=""))
            if gaps:
                secondary_nav_gaps[root.title] = gaps

        nav_paths = {entry.path for entry in entries}
        orphan_routes = sorted(route.path for route in self.routes if route.path not in nav_paths)

        return InformationArchitectureAudit(
            total_navigation_nodes=len(entries),
            total_routes=len(self.routes),
            duplicate_routes=duplicate_routes,
            missing_route_nodes=missing_route_nodes,
            secondary_nav_gaps=secondary_nav_gaps,
            orphan_routes=orphan_routes,
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "global_nav": [node.to_dict() for node in self.global_nav],
            "routes": [route.to_dict() for route in self.routes],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitecture":
        return cls(
            global_nav=[NavigationNode.from_dict(node) for node in data.get("global_nav", [])],
            routes=[NavigationRoute.from_dict(route) for route in data.get("routes", [])],
        )

    def _flatten_node(
        self,
        node: NavigationNode,
        parent_path: str,
        depth: int,
        parent_id: str,
    ) -> List[NavigationEntry]:
        path = self._join_path(parent_path, node.segment)
        entries = [
            NavigationEntry(
                node_id=node.node_id,
                title=node.title,
                path=path,
                depth=depth,
                parent_id=parent_id,
                screen_id=node.screen_id,
            )
        ]
        for child in node.children:
            entries.extend(self._flatten_node(child, parent_path=path, depth=depth + 1, parent_id=node.node_id))
        return entries

    def _missing_paths_for_descendants(self, node: NavigationNode, parent_path: str) -> List[str]:
        path = self._join_path(parent_path, node.segment)
        missing: List[str] = []
        if node.children and path not in self.route_index:
            missing.append(path)
        for child in node.children:
            missing.extend(self._missing_paths_for_descendants(child, parent_path=path))
        return missing

    @staticmethod
    def _join_path(parent_path: str, segment: str) -> str:
        base = _normalize_route_path(parent_path)
        part = segment.strip("/")
        if not part:
            return base
        if base == "/":
            return f"/{part}"
        return f"{base}/{part}"


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


REQUIRED_CONSOLE_SURFACE_STATES = {"default", "loading", "empty", "error"}


@dataclass(frozen=True)
class NavigationItem:
    name: str
    route: str
    section: str
    icon: str = ""
    badge_count: int = 0

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "route": self.route,
            "section": self.section,
            "icon": self.icon,
            "badge_count": self.badge_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "NavigationItem":
        return cls(
            name=str(data["name"]),
            route=str(data["route"]),
            section=str(data["section"]),
            icon=str(data.get("icon", "")),
            badge_count=int(data.get("badge_count", 0)),
        )


@dataclass(frozen=True)
class GlobalAction:
    action_id: str
    label: str
    placement: str
    requires_selection: bool = False
    intent: str = "default"

    def to_dict(self) -> Dict[str, object]:
        return {
            "action_id": self.action_id,
            "label": self.label,
            "placement": self.placement,
            "requires_selection": self.requires_selection,
            "intent": self.intent,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "GlobalAction":
        return cls(
            action_id=str(data["action_id"]),
            label=str(data["label"]),
            placement=str(data["placement"]),
            requires_selection=bool(data.get("requires_selection", False)),
            intent=str(data.get("intent", "default")),
        )


@dataclass(frozen=True)
class FilterDefinition:
    name: str
    field: str
    control: str
    options: List[str] = field(default_factory=list)
    default_value: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field": self.field,
            "control": self.control,
            "options": list(self.options),
            "default_value": self.default_value,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "FilterDefinition":
        return cls(
            name=str(data["name"]),
            field=str(data["field"]),
            control=str(data["control"]),
            options=[str(option) for option in data.get("options", [])],
            default_value=str(data.get("default_value", "")),
        )


@dataclass(frozen=True)
class SurfaceState:
    name: str
    message: str = ""
    allowed_actions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "message": self.message,
            "allowed_actions": list(self.allowed_actions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceState":
        return cls(
            name=str(data["name"]),
            message=str(data.get("message", "")),
            allowed_actions=[str(action_id) for action_id in data.get("allowed_actions", [])],
        )


@dataclass
class ConsoleSurface:
    name: str
    route: str
    navigation_section: str
    top_bar_actions: List[GlobalAction] = field(default_factory=list)
    filters: List[FilterDefinition] = field(default_factory=list)
    states: List[SurfaceState] = field(default_factory=list)
    supports_bulk_actions: bool = False

    @property
    def action_ids(self) -> List[str]:
        return [action.action_id for action in self.top_bar_actions]

    @property
    def state_names(self) -> List[str]:
        return [state.name for state in self.states]

    @property
    def missing_required_states(self) -> List[str]:
        return sorted(REQUIRED_CONSOLE_SURFACE_STATES.difference(self.state_names))

    @property
    def unresolved_state_actions(self) -> Dict[str, List[str]]:
        available = set(self.action_ids)
        unresolved: Dict[str, List[str]] = {}
        for state in self.states:
            missing = sorted(action_id for action_id in state.allowed_actions if action_id not in available)
            if missing:
                unresolved[state.name] = missing
        return unresolved

    @property
    def states_missing_actions(self) -> List[str]:
        missing: List[str] = []
        for state in self.states:
            if state.name != "default" and not state.allowed_actions:
                missing.append(state.name)
        return missing

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "route": self.route,
            "navigation_section": self.navigation_section,
            "top_bar_actions": [action.to_dict() for action in self.top_bar_actions],
            "filters": [surface_filter.to_dict() for surface_filter in self.filters],
            "states": [state.to_dict() for state in self.states],
            "supports_bulk_actions": self.supports_bulk_actions,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleSurface":
        return cls(
            name=str(data["name"]),
            route=str(data["route"]),
            navigation_section=str(data["navigation_section"]),
            top_bar_actions=[GlobalAction.from_dict(item) for item in data.get("top_bar_actions", [])],
            filters=[FilterDefinition.from_dict(item) for item in data.get("filters", [])],
            states=[SurfaceState.from_dict(item) for item in data.get("states", [])],
            supports_bulk_actions=bool(data.get("supports_bulk_actions", False)),
        )


@dataclass
class ConsoleIA:
    name: str
    version: str
    navigation: List[NavigationItem] = field(default_factory=list)
    surfaces: List[ConsoleSurface] = field(default_factory=list)
    top_bar: ConsoleTopBar = field(default_factory=lambda: ConsoleTopBar(name="", search_placeholder=""))

    @property
    def route_index(self) -> Dict[str, ConsoleSurface]:
        return {surface.route: surface for surface in self.surfaces}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "navigation": [item.to_dict() for item in self.navigation],
            "surfaces": [surface.to_dict() for surface in self.surfaces],
            "top_bar": self.top_bar.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleIA":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            navigation=[NavigationItem.from_dict(item) for item in data.get("navigation", [])],
            surfaces=[ConsoleSurface.from_dict(item) for item in data.get("surfaces", [])],
            top_bar=ConsoleTopBar.from_dict(dict(data.get("top_bar", {}))),
        )


@dataclass(frozen=True)
class SurfacePermissionRule:
    allowed_roles: List[str] = field(default_factory=list)
    denied_roles: List[str] = field(default_factory=list)
    audit_event: str = ""

    @property
    def missing_coverage(self) -> List[str]:
        missing: List[str] = []
        if not self.allowed_roles:
            missing.append("allowed-roles")
        if not self.denied_roles:
            missing.append("denied-roles")
        if not self.audit_event.strip():
            missing.append("audit-event")
        return missing

    @property
    def complete(self) -> bool:
        return not self.missing_coverage

    def to_dict(self) -> Dict[str, object]:
        return {
            "allowed_roles": list(self.allowed_roles),
            "denied_roles": list(self.denied_roles),
            "audit_event": self.audit_event,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfacePermissionRule":
        return cls(
            allowed_roles=[str(role) for role in data.get("allowed_roles", [])],
            denied_roles=[str(role) for role in data.get("denied_roles", [])],
            audit_event=str(data.get("audit_event", "")),
        )


@dataclass
class SurfaceInteractionContract:
    surface_name: str
    required_action_ids: List[str] = field(default_factory=list)
    requires_filters: bool = True
    requires_batch_actions: bool = False
    required_states: List[str] = field(default_factory=lambda: sorted(REQUIRED_CONSOLE_SURFACE_STATES))
    permission_rule: SurfacePermissionRule = field(default_factory=SurfacePermissionRule)
    primary_persona: str = ""
    linked_wireframe_id: str = ""
    review_focus_areas: List[str] = field(default_factory=list)
    decision_prompts: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_name": self.surface_name,
            "required_action_ids": list(self.required_action_ids),
            "requires_filters": self.requires_filters,
            "requires_batch_actions": self.requires_batch_actions,
            "required_states": list(self.required_states),
            "permission_rule": self.permission_rule.to_dict(),
            "primary_persona": self.primary_persona,
            "linked_wireframe_id": self.linked_wireframe_id,
            "review_focus_areas": list(self.review_focus_areas),
            "decision_prompts": list(self.decision_prompts),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceInteractionContract":
        return cls(
            surface_name=str(data["surface_name"]),
            required_action_ids=[str(action_id) for action_id in data.get("required_action_ids", [])],
            requires_filters=bool(data.get("requires_filters", True)),
            requires_batch_actions=bool(data.get("requires_batch_actions", False)),
            required_states=[
                str(state) for state in data.get("required_states", sorted(REQUIRED_CONSOLE_SURFACE_STATES))
            ],
            permission_rule=SurfacePermissionRule.from_dict(dict(data.get("permission_rule", {}))),
            primary_persona=str(data.get("primary_persona", "")),
            linked_wireframe_id=str(data.get("linked_wireframe_id", "")),
            review_focus_areas=[str(item) for item in data.get("review_focus_areas", [])],
            decision_prompts=[str(item) for item in data.get("decision_prompts", [])],
        )


@dataclass
class ConsoleInteractionDraft:
    name: str
    version: str
    architecture: ConsoleIA
    contracts: List[SurfaceInteractionContract] = field(default_factory=list)
    required_roles: List[str] = field(default_factory=list)
    requires_frame_contracts: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "architecture": self.architecture.to_dict(),
            "contracts": [contract.to_dict() for contract in self.contracts],
            "required_roles": list(self.required_roles),
            "requires_frame_contracts": self.requires_frame_contracts,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleInteractionDraft":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            architecture=ConsoleIA.from_dict(dict(data["architecture"])),
            contracts=[SurfaceInteractionContract.from_dict(item) for item in data.get("contracts", [])],
            required_roles=[str(role) for role in data.get("required_roles", [])],
            requires_frame_contracts=bool(data.get("requires_frame_contracts", False)),
        )


@dataclass
class ConsoleInteractionAudit:
    name: str
    version: str
    contract_count: int
    missing_surfaces: List[str] = field(default_factory=list)
    surfaces_missing_filters: List[str] = field(default_factory=list)
    surfaces_missing_actions: Dict[str, List[str]] = field(default_factory=dict)
    surfaces_missing_batch_actions: List[str] = field(default_factory=list)
    surfaces_missing_states: Dict[str, List[str]] = field(default_factory=dict)
    permission_gaps: Dict[str, List[str]] = field(default_factory=dict)
    uncovered_roles: List[str] = field(default_factory=list)
    surfaces_missing_primary_personas: List[str] = field(default_factory=list)
    surfaces_missing_wireframe_links: List[str] = field(default_factory=list)
    surfaces_missing_review_focus: List[str] = field(default_factory=list)
    surfaces_missing_decision_prompts: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.contract_count == 0:
            return 0.0
        penalties = (
            len(self.missing_surfaces)
            + len(self.surfaces_missing_filters)
            + len(self.surfaces_missing_actions)
            + len(self.surfaces_missing_batch_actions)
            + len(self.surfaces_missing_states)
            + len(self.permission_gaps)
            + len(self.uncovered_roles)
            + len(self.surfaces_missing_primary_personas)
            + len(self.surfaces_missing_wireframe_links)
            + len(self.surfaces_missing_review_focus)
            + len(self.surfaces_missing_decision_prompts)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.contract_count))
        return round(score, 1)

    @property
    def release_ready(self) -> bool:
        return (
            not self.missing_surfaces
            and not self.surfaces_missing_filters
            and not self.surfaces_missing_actions
            and not self.surfaces_missing_batch_actions
            and not self.surfaces_missing_states
            and not self.permission_gaps
            and not self.uncovered_roles
            and not self.surfaces_missing_primary_personas
            and not self.surfaces_missing_wireframe_links
            and not self.surfaces_missing_review_focus
            and not self.surfaces_missing_decision_prompts
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "contract_count": self.contract_count,
            "missing_surfaces": list(self.missing_surfaces),
            "surfaces_missing_filters": list(self.surfaces_missing_filters),
            "surfaces_missing_actions": {
                name: list(actions) for name, actions in self.surfaces_missing_actions.items()
            },
            "surfaces_missing_batch_actions": list(self.surfaces_missing_batch_actions),
            "surfaces_missing_states": {
                name: list(states) for name, states in self.surfaces_missing_states.items()
            },
            "permission_gaps": {name: list(gaps) for name, gaps in self.permission_gaps.items()},
            "uncovered_roles": list(self.uncovered_roles),
            "surfaces_missing_primary_personas": list(self.surfaces_missing_primary_personas),
            "surfaces_missing_wireframe_links": list(self.surfaces_missing_wireframe_links),
            "surfaces_missing_review_focus": list(self.surfaces_missing_review_focus),
            "surfaces_missing_decision_prompts": list(self.surfaces_missing_decision_prompts),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleInteractionAudit":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            contract_count=int(data.get("contract_count", 0)),
            missing_surfaces=[str(name) for name in data.get("missing_surfaces", [])],
            surfaces_missing_filters=[str(name) for name in data.get("surfaces_missing_filters", [])],
            surfaces_missing_actions={
                str(name): [str(action_id) for action_id in actions]
                for name, actions in dict(data.get("surfaces_missing_actions", {})).items()
            },
            surfaces_missing_batch_actions=[str(name) for name in data.get("surfaces_missing_batch_actions", [])],
            surfaces_missing_states={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("surfaces_missing_states", {})).items()
            },
            permission_gaps={
                str(name): [str(gap) for gap in gaps]
                for name, gaps in dict(data.get("permission_gaps", {})).items()
            },
            uncovered_roles=[str(role) for role in data.get("uncovered_roles", [])],
            surfaces_missing_primary_personas=[
                str(name) for name in data.get("surfaces_missing_primary_personas", [])
            ],
            surfaces_missing_wireframe_links=[str(name) for name in data.get("surfaces_missing_wireframe_links", [])],
            surfaces_missing_review_focus=[str(name) for name in data.get("surfaces_missing_review_focus", [])],
            surfaces_missing_decision_prompts=[
                str(name) for name in data.get("surfaces_missing_decision_prompts", [])
            ],
        )


@dataclass
class ConsoleIAAudit:
    system_name: str
    version: str
    surface_count: int
    navigation_count: int
    top_bar_audit: ConsoleTopBarAudit = field(default_factory=lambda: ConsoleTopBarAudit(name=""))
    surfaces_missing_filters: List[str] = field(default_factory=list)
    surfaces_missing_actions: List[str] = field(default_factory=list)
    surfaces_missing_states: Dict[str, List[str]] = field(default_factory=dict)
    states_missing_actions: Dict[str, List[str]] = field(default_factory=dict)
    unresolved_state_actions: Dict[str, Dict[str, List[str]]] = field(default_factory=dict)
    orphan_navigation_routes: List[str] = field(default_factory=list)
    unnavigable_surfaces: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.surface_count == 0:
            return 0.0
        penalties = (
            (0 if self.top_bar_audit.release_ready else 1)
            + len(self.surfaces_missing_filters)
            + len(self.surfaces_missing_actions)
            + len(self.surfaces_missing_states)
            + len(self.states_missing_actions)
            + len(self.unresolved_state_actions)
            + len(self.orphan_navigation_routes)
            + len(self.unnavigable_surfaces)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.surface_count))
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "system_name": self.system_name,
            "version": self.version,
            "surface_count": self.surface_count,
            "navigation_count": self.navigation_count,
            "top_bar_audit": self.top_bar_audit.to_dict(),
            "surfaces_missing_filters": list(self.surfaces_missing_filters),
            "surfaces_missing_actions": list(self.surfaces_missing_actions),
            "surfaces_missing_states": {
                name: list(states) for name, states in self.surfaces_missing_states.items()
            },
            "states_missing_actions": {
                name: list(states) for name, states in self.states_missing_actions.items()
            },
            "unresolved_state_actions": {
                name: {state: list(actions) for state, actions in states.items()}
                for name, states in self.unresolved_state_actions.items()
            },
            "orphan_navigation_routes": list(self.orphan_navigation_routes),
            "unnavigable_surfaces": list(self.unnavigable_surfaces),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleIAAudit":
        return cls(
            system_name=str(data["system_name"]),
            version=str(data["version"]),
            surface_count=int(data.get("surface_count", 0)),
            navigation_count=int(data.get("navigation_count", 0)),
            top_bar_audit=ConsoleTopBarAudit.from_dict(dict(data.get("top_bar_audit", {}))),
            surfaces_missing_filters=[str(name) for name in data.get("surfaces_missing_filters", [])],
            surfaces_missing_actions=[str(name) for name in data.get("surfaces_missing_actions", [])],
            surfaces_missing_states={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("surfaces_missing_states", {})).items()
            },
            states_missing_actions={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("states_missing_actions", {})).items()
            },
            unresolved_state_actions={
                str(name): {
                    str(state): [str(action_id) for action_id in actions]
                    for state, actions in dict(states).items()
                }
                for name, states in dict(data.get("unresolved_state_actions", {})).items()
            },
            orphan_navigation_routes=[str(route) for route in data.get("orphan_navigation_routes", [])],
            unnavigable_surfaces=[str(name) for name in data.get("unnavigable_surfaces", [])],
        )


class ConsoleIAAuditor:
    def audit(self, architecture: ConsoleIA) -> ConsoleIAAudit:
        top_bar_audit = ConsoleChromeLibrary().audit_top_bar(architecture.top_bar)
        route_index = architecture.route_index
        navigation_routes = {item.route for item in architecture.navigation}
        surfaces_missing_filters: List[str] = []
        surfaces_missing_actions: List[str] = []
        surfaces_missing_states: Dict[str, List[str]] = {}
        states_missing_actions: Dict[str, List[str]] = {}
        unresolved_state_actions: Dict[str, Dict[str, List[str]]] = {}

        for surface in architecture.surfaces:
            if not surface.filters:
                surfaces_missing_filters.append(surface.name)
            if not surface.top_bar_actions:
                surfaces_missing_actions.append(surface.name)
            if surface.missing_required_states:
                surfaces_missing_states[surface.name] = surface.missing_required_states
            if surface.states_missing_actions:
                states_missing_actions[surface.name] = surface.states_missing_actions
            if surface.unresolved_state_actions:
                unresolved_state_actions[surface.name] = surface.unresolved_state_actions

        orphan_navigation_routes = sorted(route for route in navigation_routes if route not in route_index)
        unnavigable_surfaces = sorted(
            surface.name for surface in architecture.surfaces if surface.route not in navigation_routes
        )

        return ConsoleIAAudit(
            system_name=architecture.name,
            version=architecture.version,
            surface_count=len(architecture.surfaces),
            navigation_count=len(architecture.navigation),
            top_bar_audit=top_bar_audit,
            surfaces_missing_filters=sorted(surfaces_missing_filters),
            surfaces_missing_actions=sorted(surfaces_missing_actions),
            surfaces_missing_states=dict(sorted(surfaces_missing_states.items())),
            states_missing_actions=dict(sorted(states_missing_actions.items())),
            unresolved_state_actions=dict(sorted(unresolved_state_actions.items())),
            orphan_navigation_routes=orphan_navigation_routes,
            unnavigable_surfaces=unnavigable_surfaces,
        )


class ConsoleInteractionAuditor:
    def audit(self, draft: ConsoleInteractionDraft) -> ConsoleInteractionAudit:
        route_index = draft.architecture.route_index
        missing_surfaces: List[str] = []
        surfaces_missing_filters: List[str] = []
        surfaces_missing_actions: Dict[str, List[str]] = {}
        surfaces_missing_batch_actions: List[str] = []
        surfaces_missing_states: Dict[str, List[str]] = {}
        permission_gaps: Dict[str, List[str]] = {}
        referenced_roles = set()
        surfaces_missing_primary_personas: List[str] = []
        surfaces_missing_wireframe_links: List[str] = []
        surfaces_missing_review_focus: List[str] = []
        surfaces_missing_decision_prompts: List[str] = []

        for contract in draft.contracts:
            surface: Optional[ConsoleSurface] = route_index.get(contract.surface_name)
            if surface is None:
                surface = next(
                    (candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name),
                    None,
                )
            if surface is None:
                missing_surfaces.append(contract.surface_name)
                continue

            if contract.requires_filters and not surface.filters:
                surfaces_missing_filters.append(contract.surface_name)

            available_action_ids = set(surface.action_ids)
            missing_action_ids = sorted(
                action_id for action_id in contract.required_action_ids if action_id not in available_action_ids
            )
            if missing_action_ids:
                surfaces_missing_actions[contract.surface_name] = missing_action_ids

            if contract.requires_batch_actions and not any(
                action.requires_selection for action in surface.top_bar_actions
            ):
                surfaces_missing_batch_actions.append(contract.surface_name)

            missing_state_ids = sorted(
                state_name for state_name in contract.required_states if state_name not in surface.state_names
            )
            if missing_state_ids:
                surfaces_missing_states[contract.surface_name] = missing_state_ids

            referenced_roles.update(contract.permission_rule.allowed_roles)
            referenced_roles.update(contract.permission_rule.denied_roles)
            if contract.permission_rule.missing_coverage:
                permission_gaps[contract.surface_name] = contract.permission_rule.missing_coverage

            if draft.requires_frame_contracts:
                if not contract.primary_persona.strip():
                    surfaces_missing_primary_personas.append(contract.surface_name)
                if not contract.linked_wireframe_id.strip():
                    surfaces_missing_wireframe_links.append(contract.surface_name)
                if not contract.review_focus_areas:
                    surfaces_missing_review_focus.append(contract.surface_name)
                if not contract.decision_prompts:
                    surfaces_missing_decision_prompts.append(contract.surface_name)

        uncovered_roles = sorted(role for role in draft.required_roles if role not in referenced_roles)

        return ConsoleInteractionAudit(
            name=draft.name,
            version=draft.version,
            contract_count=len(draft.contracts),
            missing_surfaces=sorted(missing_surfaces),
            surfaces_missing_filters=sorted(surfaces_missing_filters),
            surfaces_missing_actions=dict(sorted(surfaces_missing_actions.items())),
            surfaces_missing_batch_actions=sorted(surfaces_missing_batch_actions),
            surfaces_missing_states=dict(sorted(surfaces_missing_states.items())),
            permission_gaps=dict(sorted(permission_gaps.items())),
            uncovered_roles=uncovered_roles,
            surfaces_missing_primary_personas=sorted(surfaces_missing_primary_personas),
            surfaces_missing_wireframe_links=sorted(surfaces_missing_wireframe_links),
            surfaces_missing_review_focus=sorted(surfaces_missing_review_focus),
            surfaces_missing_decision_prompts=sorted(surfaces_missing_decision_prompts),
        )


def render_console_ia_report(architecture: ConsoleIA, audit: ConsoleIAAudit) -> str:
    lines = [
        "# Console Information Architecture Report",
        "",
        f"- Name: {architecture.name}",
        f"- Version: {architecture.version}",
        f"- Navigation Items: {audit.navigation_count}",
        f"- Surfaces: {audit.surface_count}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Global Header",
        "",
        f"- Name: {architecture.top_bar.name or 'none'}",
        f"- Release Ready: {audit.top_bar_audit.release_ready}",
        "- Missing capabilities: "
        f"{', '.join(audit.top_bar_audit.missing_capabilities) if audit.top_bar_audit.missing_capabilities else 'none'}",
        f"- Command Count: {audit.top_bar_audit.command_count}",
        f"- Cmd/Ctrl+K supported: {audit.top_bar_audit.command_shortcut_supported}",
        "",
        "## Navigation",
        "",
    ]

    if architecture.navigation:
        for item in architecture.navigation:
            lines.append(
                f"- {item.section} / {item.name}: route={item.route} badge={item.badge_count} icon={item.icon or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Surface Coverage", ""])
    if architecture.surfaces:
        for surface in architecture.surfaces:
            filters = ", ".join(surface_filter.name for surface_filter in surface.filters) or "none"
            actions = ", ".join(action.label for action in surface.top_bar_actions) or "none"
            states = ", ".join(surface.state_names) or "none"
            missing_states = ", ".join(surface.missing_required_states) or "none"
            unresolved = audit.unresolved_state_actions.get(surface.name, {})
            if unresolved:
                unresolved_text = "; ".join(
                    f"{state}={', '.join(action_ids)}" for state, action_ids in sorted(unresolved.items())
                )
            else:
                unresolved_text = "none"
            state_actions_missing = ", ".join(audit.states_missing_actions.get(surface.name, [])) or "none"
            lines.append(
                f"- {surface.name}: route={surface.route} filters={filters} actions={actions} states={states} "
                f"missing_states={missing_states} states_without_actions={state_actions_missing} "
                f"unresolved_state_actions={unresolved_text}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        "- Surfaces missing filters: "
        f"{', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}"
    )
    lines.append(
        "- Surfaces missing top-bar actions: "
        f"{', '.join(audit.surfaces_missing_actions) if audit.surfaces_missing_actions else 'none'}"
    )
    if audit.surfaces_missing_states:
        missing_states_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())
        )
    else:
        missing_states_text = "none"
    lines.append(f"- Surfaces missing required states: {missing_states_text}")
    if audit.states_missing_actions:
        states_without_actions_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.states_missing_actions.items())
        )
    else:
        states_without_actions_text = "none"
    lines.append(f"- States without recovery actions: {states_without_actions_text}")
    if audit.unresolved_state_actions:
        unresolved_text = "; ".join(
            f"{name}="
            + ", ".join(f"{state}:{'/'.join(actions)}" for state, actions in sorted(states.items()))
            for name, states in sorted(audit.unresolved_state_actions.items())
        )
    else:
        unresolved_text = "none"
    lines.append(f"- Undefined state actions: {unresolved_text}")
    lines.append(
        "- Orphan navigation routes: "
        f"{', '.join(audit.orphan_navigation_routes) if audit.orphan_navigation_routes else 'none'}"
    )
    lines.append(
        f"- Unnavigable surfaces: {', '.join(audit.unnavigable_surfaces) if audit.unnavigable_surfaces else 'none'}"
    )
    return "\n".join(lines) + "\n"


def render_console_interaction_report(
    draft: ConsoleInteractionDraft,
    audit: ConsoleInteractionAudit,
) -> str:
    route_index = draft.architecture.route_index
    lines = [
        "# Console Interaction Draft Report",
        "",
        f"- Name: {draft.name}",
        f"- Version: {draft.version}",
        f"- Critical Pages: {len(draft.contracts)}",
        f"- Required Roles: {', '.join(draft.required_roles) if draft.required_roles else 'none'}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Page Coverage",
        "",
    ]

    if draft.contracts:
        for contract in draft.contracts:
            surface = route_index.get(contract.surface_name)
            if surface is None:
                surface = next(
                    (candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name),
                    None,
                )
            if surface is None:
                lines.append(f"- {contract.surface_name}: missing surface definition")
                continue

            required_actions = ", ".join(contract.required_action_ids) or "none"
            available_actions = ", ".join(surface.action_ids) or "none"
            batch_mode = "required" if contract.requires_batch_actions else "optional"
            permission_state = "complete" if contract.permission_rule.complete else "incomplete"
            lines.append(
                f"- {contract.surface_name}: route={surface.route} required_actions={required_actions} "
                f"available_actions={available_actions} filters={len(surface.filters)} "
                f"states={', '.join(surface.state_names) or 'none'} batch={batch_mode} "
                f"permissions={permission_state}"
            )
            lines.append(
                "  "
                f"persona={contract.primary_persona or 'none'} "
                f"wireframe={contract.linked_wireframe_id or 'none'} "
                f"review_focus={','.join(contract.review_focus_areas) or 'none'} "
                f"decision_prompts={','.join(contract.decision_prompts) or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Missing surfaces: {', '.join(audit.missing_surfaces) if audit.missing_surfaces else 'none'}")
    lines.append(
        f"- Pages missing filters: {', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}"
    )
    if audit.surfaces_missing_actions:
        action_gap_text = "; ".join(
            f"{name}={', '.join(actions)}" for name, actions in sorted(audit.surfaces_missing_actions.items())
        )
    else:
        action_gap_text = "none"
    lines.append(f"- Pages missing actions: {action_gap_text}")
    lines.append(
        "- Pages missing batch actions: "
        f"{', '.join(audit.surfaces_missing_batch_actions) if audit.surfaces_missing_batch_actions else 'none'}"
    )
    if audit.surfaces_missing_states:
        state_gap_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())
        )
    else:
        state_gap_text = "none"
    lines.append(f"- Pages missing states: {state_gap_text}")
    if audit.permission_gaps:
        permission_gap_text = "; ".join(
            f"{name}={', '.join(gaps)}" for name, gaps in sorted(audit.permission_gaps.items())
        )
    else:
        permission_gap_text = "none"
    lines.append(f"- Permission gaps: {permission_gap_text}")
    lines.append(f"- Uncovered roles: {', '.join(audit.uncovered_roles) if audit.uncovered_roles else 'none'}")
    lines.append(
        "- Pages missing personas: "
        f"{', '.join(audit.surfaces_missing_primary_personas) if audit.surfaces_missing_primary_personas else 'none'}"
    )
    lines.append(
        "- Pages missing wireframe links: "
        f"{', '.join(audit.surfaces_missing_wireframe_links) if audit.surfaces_missing_wireframe_links else 'none'}"
    )
    lines.append(
        "- Pages missing review focus: "
        f"{', '.join(audit.surfaces_missing_review_focus) if audit.surfaces_missing_review_focus else 'none'}"
    )
    lines.append(
        "- Pages missing decision prompts: "
        f"{', '.join(audit.surfaces_missing_decision_prompts) if audit.surfaces_missing_decision_prompts else 'none'}"
    )
    return "\n".join(lines) + "\n"


def build_big_4203_console_interaction_draft() -> ConsoleInteractionDraft:
    return ConsoleInteractionDraft(
        name="BIG-4203 Four Critical Pages",
        version="v4.0-design-sprint",
        required_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
        requires_frame_contracts=True,
        architecture=ConsoleIA(
            name="BigClaw Console IA",
            version="v4.0-design-sprint",
            top_bar=ConsoleTopBar(
                name="BigClaw Global Header",
                search_placeholder="Search runs, queues, prompts, and commands",
                environment_options=["Production", "Staging", "Shadow"],
                time_range_options=["24h", "7d", "30d"],
                alert_channels=["approvals", "sla", "regressions"],
                documentation_complete=True,
                accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
                command_entry=ConsoleCommandEntry(
                    trigger_label="Command Menu",
                    placeholder="Jump to a run, queue, or release control action",
                    shortcut="Cmd+K / Ctrl+K",
                    commands=[
                        CommandAction(id="search-runs", title="Search runs", section="Navigate", shortcut="/"),
                        CommandAction(id="open-queue", title="Open queue control", section="Operate"),
                        CommandAction(id="open-triage", title="Open triage center", section="Operate"),
                    ],
                ),
            ),
            navigation=[
                NavigationItem(name="Overview", route="/overview", section="Operate", icon="dashboard"),
                NavigationItem(name="Queue", route="/queue", section="Operate", icon="queue"),
                NavigationItem(name="Run Detail", route="/runs/detail", section="Operate", icon="activity"),
                NavigationItem(name="Triage", route="/triage", section="Operate", icon="alert"),
            ],
            surfaces=[
                ConsoleSurface(
                    name="Overview",
                    route="/overview",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                    ],
                    filters=[
                        FilterDefinition(
                            name="Team",
                            field="team",
                            control="select",
                            options=["all", "platform", "product"],
                        ),
                        FilterDefinition(
                            name="Time",
                            field="time_range",
                            control="segmented",
                            options=["24h", "7d", "30d"],
                            default_value="7d",
                        ),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["drill-down"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                ),
                ConsoleSurface(
                    name="Queue",
                    route="/queue",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                        GlobalAction(
                            action_id="bulk-approve",
                            label="Bulk Approve",
                            placement="topbar",
                            requires_selection=True,
                        ),
                    ],
                    filters=[
                        FilterDefinition(
                            name="Status",
                            field="status",
                            control="select",
                            options=["all", "queued", "approval"],
                        ),
                        FilterDefinition(name="Owner", field="owner", control="search"),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                    supports_bulk_actions=True,
                ),
                ConsoleSurface(
                    name="Run Detail",
                    route="/runs/detail",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                    ],
                    filters=[
                        FilterDefinition(name="Run", field="run_id", control="search"),
                        FilterDefinition(
                            name="Replay Mode",
                            field="replay_mode",
                            control="select",
                            options=["latest", "failure-only"],
                        ),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["drill-down"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                ),
                ConsoleSurface(
                    name="Triage",
                    route="/triage",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                        GlobalAction(
                            action_id="bulk-assign",
                            label="Bulk Assign",
                            placement="topbar",
                            requires_selection=True,
                        ),
                    ],
                    filters=[
                        FilterDefinition(
                            name="Severity",
                            field="severity",
                            control="select",
                            options=["all", "high", "critical"],
                        ),
                        FilterDefinition(
                            name="Workflow",
                            field="workflow",
                            control="select",
                            options=["all", "triage", "handoff"],
                        ),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                    supports_bulk_actions=True,
                ),
            ],
        ),
        contracts=[
            SurfaceInteractionContract(
                surface_name="Overview",
                required_action_ids=["drill-down", "export", "audit"],
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
                    denied_roles=["guest"],
                    audit_event="overview.access.denied",
                ),
                primary_persona="VP Eng",
                linked_wireframe_id="wf-overview",
                review_focus_areas=["metric hierarchy", "drill-down posture", "alert prioritization"],
                decision_prompts=[
                    "Is the executive KPI density still scannable within one screen?",
                    "Do risk and blocker cards point to the correct downstream investigation surface?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Queue",
                required_action_ids=["drill-down", "export", "audit"],
                requires_batch_actions=True,
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "cross-team-operator"],
                    denied_roles=["vp-eng", "guest"],
                    audit_event="queue.access.denied",
                ),
                primary_persona="Platform Admin",
                linked_wireframe_id="wf-queue",
                review_focus_areas=["batch approvals", "denied-role state", "audit rail"],
                decision_prompts=[
                    "Does the queue clearly separate selection, confirmation, and audit outcomes?",
                    "Is the denied-role treatment explicit enough for VP Eng and guest personas?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Run Detail",
                required_action_ids=["drill-down", "export", "audit"],
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
                    denied_roles=["guest"],
                    audit_event="run-detail.access.denied",
                ),
                primary_persona="Eng Lead",
                linked_wireframe_id="wf-run-detail",
                review_focus_areas=["replay context", "artifact evidence", "escalation path"],
                decision_prompts=[
                    "Can reviewers distinguish replay, compare, and escalated states without narration?",
                    "Is the audit trail visible at the moment an escalation decision is made?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Triage",
                required_action_ids=["drill-down", "export", "audit"],
                requires_batch_actions=True,
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "cross-team-operator"],
                    denied_roles=["vp-eng", "guest"],
                    audit_event="triage.access.denied",
                ),
                primary_persona="Cross-Team Operator",
                linked_wireframe_id="wf-triage",
                review_focus_areas=["handoff path", "bulk assignment", "ownership history"],
                decision_prompts=[
                    "Does the triage frame explain handoff consequences before ownership changes commit?",
                    "Is bulk assignment discoverable without overpowering the audit context?",
                ],
            ),
        ],
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


def render_information_architecture_report(
    architecture: InformationArchitecture,
    audit: InformationArchitectureAudit,
) -> str:
    lines = [
        "# Information Architecture Report",
        "",
        f"- Navigation Nodes: {audit.total_navigation_nodes}",
        f"- Routes: {audit.total_routes}",
        f"- Healthy: {audit.healthy}",
        "",
        "## Navigation Tree",
        "",
    ]

    if architecture.navigation_entries:
        for entry in architecture.navigation_entries:
            indent = "  " * entry.depth
            lines.append(f"- {indent}{entry.title} ({entry.path}) screen={entry.screen_id or 'none'}")
    else:
        lines.append("- None")

    lines.extend(["", "## Route Registry", ""])
    if architecture.routes:
        for route in architecture.routes:
            lines.append(
                f"- {route.path}: screen={route.screen_id} title={route.title} nav_node={route.nav_node_id or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    lines.append(f"- Duplicate routes: {', '.join(audit.duplicate_routes) if audit.duplicate_routes else 'none'}")
    if audit.missing_route_nodes:
        missing = ", ".join(f"{node_id}={path}" for node_id, path in sorted(audit.missing_route_nodes.items()))
    else:
        missing = "none"
    lines.append(f"- Missing route nodes: {missing}")
    if audit.secondary_nav_gaps:
        gaps = "; ".join(
            f"{section}={', '.join(paths)}" for section, paths in sorted(audit.secondary_nav_gaps.items())
        )
    else:
        gaps = "none"
    lines.append(f"- Secondary nav gaps: {gaps}")
    lines.append(f"- Orphan routes: {', '.join(audit.orphan_routes) if audit.orphan_routes else 'none'}")
    return "\n".join(lines) + "\n"


@dataclass(frozen=True)
class RolePermissionScenario:
    screen_id: str
    allowed_roles: List[str] = field(default_factory=list)
    denied_roles: List[str] = field(default_factory=list)
    audit_event: str = ""

    @property
    def missing_coverage(self) -> List[str]:
        missing: List[str] = []
        if not self.allowed_roles:
            missing.append("allowed-roles")
        if not self.denied_roles:
            missing.append("denied-roles")
        if not self.audit_event.strip():
            missing.append("audit-event")
        return missing

    def to_dict(self) -> Dict[str, object]:
        return {
            "screen_id": self.screen_id,
            "allowed_roles": list(self.allowed_roles),
            "denied_roles": list(self.denied_roles),
            "audit_event": self.audit_event,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "RolePermissionScenario":
        return cls(
            screen_id=str(data["screen_id"]),
            allowed_roles=[str(role) for role in data.get("allowed_roles", [])],
            denied_roles=[str(role) for role in data.get("denied_roles", [])],
            audit_event=str(data.get("audit_event", "")),
        )


@dataclass(frozen=True)
class DataAccuracyCheck:
    screen_id: str
    metric_id: str
    source_of_truth: str
    rendered_value: str
    tolerance: float = 0.0
    observed_delta: float = 0.0
    freshness_slo_seconds: int = 0
    observed_freshness_seconds: int = 0

    @property
    def within_tolerance(self) -> bool:
        return abs(self.observed_delta) <= self.tolerance

    @property
    def within_freshness_slo(self) -> bool:
        if self.freshness_slo_seconds <= 0:
            return True
        return self.observed_freshness_seconds <= self.freshness_slo_seconds

    @property
    def passes(self) -> bool:
        return self.within_tolerance and self.within_freshness_slo

    def to_dict(self) -> Dict[str, object]:
        return {
            "screen_id": self.screen_id,
            "metric_id": self.metric_id,
            "source_of_truth": self.source_of_truth,
            "rendered_value": self.rendered_value,
            "tolerance": self.tolerance,
            "observed_delta": self.observed_delta,
            "freshness_slo_seconds": self.freshness_slo_seconds,
            "observed_freshness_seconds": self.observed_freshness_seconds,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DataAccuracyCheck":
        return cls(
            screen_id=str(data["screen_id"]),
            metric_id=str(data["metric_id"]),
            source_of_truth=str(data.get("source_of_truth", "")),
            rendered_value=str(data.get("rendered_value", "")),
            tolerance=float(data.get("tolerance", 0.0)),
            observed_delta=float(data.get("observed_delta", 0.0)),
            freshness_slo_seconds=int(data.get("freshness_slo_seconds", 0)),
            observed_freshness_seconds=int(data.get("observed_freshness_seconds", 0)),
        )


@dataclass(frozen=True)
class PerformanceBudget:
    surface_id: str
    interaction: str
    target_p95_ms: int
    observed_p95_ms: int
    target_tti_ms: int = 0
    observed_tti_ms: int = 0

    @property
    def within_budget(self) -> bool:
        p95_ok = self.observed_p95_ms <= self.target_p95_ms
        tti_ok = self.target_tti_ms <= 0 or self.observed_tti_ms <= self.target_tti_ms
        return p95_ok and tti_ok

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_id": self.surface_id,
            "interaction": self.interaction,
            "target_p95_ms": self.target_p95_ms,
            "observed_p95_ms": self.observed_p95_ms,
            "target_tti_ms": self.target_tti_ms,
            "observed_tti_ms": self.observed_tti_ms,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "PerformanceBudget":
        return cls(
            surface_id=str(data["surface_id"]),
            interaction=str(data["interaction"]),
            target_p95_ms=int(data.get("target_p95_ms", 0)),
            observed_p95_ms=int(data.get("observed_p95_ms", 0)),
            target_tti_ms=int(data.get("target_tti_ms", 0)),
            observed_tti_ms=int(data.get("observed_tti_ms", 0)),
        )


@dataclass(frozen=True)
class UsabilityJourney:
    journey_id: str
    personas: List[str] = field(default_factory=list)
    critical_steps: List[str] = field(default_factory=list)
    expected_max_steps: int = 0
    observed_steps: int = 0
    keyboard_accessible: bool = False
    empty_state_guidance: bool = False
    recovery_support: bool = False

    @property
    def passes(self) -> bool:
        return (
            bool(self.personas)
            and bool(self.critical_steps)
            and self.expected_max_steps > 0
            and self.observed_steps <= self.expected_max_steps
            and self.keyboard_accessible
            and self.empty_state_guidance
            and self.recovery_support
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "journey_id": self.journey_id,
            "personas": list(self.personas),
            "critical_steps": list(self.critical_steps),
            "expected_max_steps": self.expected_max_steps,
            "observed_steps": self.observed_steps,
            "keyboard_accessible": self.keyboard_accessible,
            "empty_state_guidance": self.empty_state_guidance,
            "recovery_support": self.recovery_support,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UsabilityJourney":
        return cls(
            journey_id=str(data["journey_id"]),
            personas=[str(persona) for persona in data.get("personas", [])],
            critical_steps=[str(step) for step in data.get("critical_steps", [])],
            expected_max_steps=int(data.get("expected_max_steps", 0)),
            observed_steps=int(data.get("observed_steps", 0)),
            keyboard_accessible=bool(data.get("keyboard_accessible", False)),
            empty_state_guidance=bool(data.get("empty_state_guidance", False)),
            recovery_support=bool(data.get("recovery_support", False)),
        )


@dataclass(frozen=True)
class AuditRequirement:
    event_type: str
    required_fields: List[str] = field(default_factory=list)
    emitted_fields: List[str] = field(default_factory=list)
    retention_days: int = 0
    observed_retention_days: int = 0

    @property
    def missing_fields(self) -> List[str]:
        emitted = set(self.emitted_fields)
        return sorted(field for field in self.required_fields if field not in emitted)

    @property
    def retention_met(self) -> bool:
        if self.retention_days <= 0:
            return True
        return self.observed_retention_days >= self.retention_days

    @property
    def complete(self) -> bool:
        return not self.missing_fields and self.retention_met

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_type": self.event_type,
            "required_fields": list(self.required_fields),
            "emitted_fields": list(self.emitted_fields),
            "retention_days": self.retention_days,
            "observed_retention_days": self.observed_retention_days,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AuditRequirement":
        return cls(
            event_type=str(data["event_type"]),
            required_fields=[str(field_name) for field_name in data.get("required_fields", [])],
            emitted_fields=[str(field_name) for field_name in data.get("emitted_fields", [])],
            retention_days=int(data.get("retention_days", 0)),
            observed_retention_days=int(data.get("observed_retention_days", 0)),
        )


@dataclass
class UIAcceptanceSuite:
    name: str
    version: str
    role_permissions: List[RolePermissionScenario] = field(default_factory=list)
    data_accuracy_checks: List[DataAccuracyCheck] = field(default_factory=list)
    performance_budgets: List[PerformanceBudget] = field(default_factory=list)
    usability_journeys: List[UsabilityJourney] = field(default_factory=list)
    audit_requirements: List[AuditRequirement] = field(default_factory=list)
    documentation_complete: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "role_permissions": [scenario.to_dict() for scenario in self.role_permissions],
            "data_accuracy_checks": [check.to_dict() for check in self.data_accuracy_checks],
            "performance_budgets": [budget.to_dict() for budget in self.performance_budgets],
            "usability_journeys": [journey.to_dict() for journey in self.usability_journeys],
            "audit_requirements": [requirement.to_dict() for requirement in self.audit_requirements],
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIAcceptanceSuite":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            role_permissions=[
                RolePermissionScenario.from_dict(scenario) for scenario in data.get("role_permissions", [])
            ],
            data_accuracy_checks=[
                DataAccuracyCheck.from_dict(check) for check in data.get("data_accuracy_checks", [])
            ],
            performance_budgets=[
                PerformanceBudget.from_dict(budget) for budget in data.get("performance_budgets", [])
            ],
            usability_journeys=[
                UsabilityJourney.from_dict(journey) for journey in data.get("usability_journeys", [])
            ],
            audit_requirements=[
                AuditRequirement.from_dict(requirement) for requirement in data.get("audit_requirements", [])
            ],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class UIAcceptanceAudit:
    name: str
    version: str
    permission_gaps: List[str] = field(default_factory=list)
    failing_data_checks: List[str] = field(default_factory=list)
    failing_performance_budgets: List[str] = field(default_factory=list)
    failing_usability_journeys: List[str] = field(default_factory=list)
    incomplete_audit_trails: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def release_ready(self) -> bool:
        return (
            not self.permission_gaps
            and not self.failing_data_checks
            and not self.failing_performance_budgets
            and not self.failing_usability_journeys
            and not self.incomplete_audit_trails
            and self.documentation_complete
        )

    @property
    def readiness_score(self) -> float:
        checks = [
            not self.permission_gaps,
            not self.failing_data_checks,
            not self.failing_performance_budgets,
            not self.failing_usability_journeys,
            not self.incomplete_audit_trails,
            self.documentation_complete,
        ]
        passed = sum(1 for item in checks if item)
        return round((passed / len(checks)) * 100, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "permission_gaps": list(self.permission_gaps),
            "failing_data_checks": list(self.failing_data_checks),
            "failing_performance_budgets": list(self.failing_performance_budgets),
            "failing_usability_journeys": list(self.failing_usability_journeys),
            "incomplete_audit_trails": list(self.incomplete_audit_trails),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIAcceptanceAudit":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            permission_gaps=[str(item) for item in data.get("permission_gaps", [])],
            failing_data_checks=[str(item) for item in data.get("failing_data_checks", [])],
            failing_performance_budgets=[str(item) for item in data.get("failing_performance_budgets", [])],
            failing_usability_journeys=[str(item) for item in data.get("failing_usability_journeys", [])],
            incomplete_audit_trails=[str(item) for item in data.get("incomplete_audit_trails", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


class UIAcceptanceLibrary:
    def audit(self, suite: UIAcceptanceSuite) -> UIAcceptanceAudit:
        permission_gaps = [
            f"{scenario.screen_id}: missing={', '.join(scenario.missing_coverage)}"
            for scenario in suite.role_permissions
            if scenario.missing_coverage
        ]
        failing_data_checks = [
            f"{check.screen_id}.{check.metric_id}: delta={check.observed_delta} freshness={check.observed_freshness_seconds}s"
            for check in suite.data_accuracy_checks
            if not check.passes
        ]
        failing_performance_budgets = [
            f"{budget.surface_id}.{budget.interaction}: p95={budget.observed_p95_ms}ms"
            + (
                f" tti={budget.observed_tti_ms}ms"
                if budget.target_tti_ms > 0
                else ""
            )
            for budget in suite.performance_budgets
            if not budget.within_budget
        ]
        failing_usability_journeys = [
            f"{journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps}"
            for journey in suite.usability_journeys
            if not journey.passes
        ]
        incomplete_audit_trails = []
        for requirement in suite.audit_requirements:
            if requirement.complete:
                continue
            gaps = requirement.missing_fields
            parts: List[str] = []
            if gaps:
                parts.append(f"missing_fields={', '.join(gaps)}")
            if not requirement.retention_met:
                parts.append(
                    f"retention={requirement.observed_retention_days}/{requirement.retention_days}d"
                )
            incomplete_audit_trails.append(f"{requirement.event_type}: {' '.join(parts)}")

        return UIAcceptanceAudit(
            name=suite.name,
            version=suite.version,
            permission_gaps=permission_gaps,
            failing_data_checks=failing_data_checks,
            failing_performance_budgets=failing_performance_budgets,
            failing_usability_journeys=failing_usability_journeys,
            incomplete_audit_trails=incomplete_audit_trails,
            documentation_complete=suite.documentation_complete,
        )


def render_ui_acceptance_report(suite: UIAcceptanceSuite, audit: UIAcceptanceAudit) -> str:
    lines = [
        "# UI Acceptance Report",
        "",
        f"- Name: {suite.name}",
        f"- Version: {suite.version}",
        f"- Role/Permission Scenarios: {len(suite.role_permissions)}",
        f"- Data Accuracy Checks: {len(suite.data_accuracy_checks)}",
        f"- Performance Budgets: {len(suite.performance_budgets)}",
        f"- Usability Journeys: {len(suite.usability_journeys)}",
        f"- Audit Requirements: {len(suite.audit_requirements)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Coverage",
        "",
    ]

    if suite.role_permissions:
        for scenario in suite.role_permissions:
            denied = ", ".join(scenario.denied_roles) or "none"
            lines.append(
                f"- Role/Permission {scenario.screen_id}: allow={', '.join(scenario.allowed_roles) or 'none'} "
                f"deny={denied} audit_event={scenario.audit_event or 'none'}"
            )
    else:
        lines.append("- Role/Permission: none")

    if suite.data_accuracy_checks:
        for check in suite.data_accuracy_checks:
            lines.append(
                f"- Data Accuracy {check.screen_id}.{check.metric_id}: delta={check.observed_delta} "
                f"tolerance={check.tolerance} freshness={check.observed_freshness_seconds}/{check.freshness_slo_seconds}s"
            )
    else:
        lines.append("- Data Accuracy: none")

    if suite.performance_budgets:
        for budget in suite.performance_budgets:
            tti_text = (
                f" tti={budget.observed_tti_ms}/{budget.target_tti_ms}ms"
                if budget.target_tti_ms > 0
                else ""
            )
            lines.append(
                f"- Performance {budget.surface_id}.{budget.interaction}: "
                f"p95={budget.observed_p95_ms}/{budget.target_p95_ms}ms{tti_text}"
            )
    else:
        lines.append("- Performance: none")

    if suite.usability_journeys:
        for journey in suite.usability_journeys:
            lines.append(
                f"- Usability {journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps} "
                f"keyboard={journey.keyboard_accessible} empty_state={journey.empty_state_guidance} "
                f"recovery={journey.recovery_support}"
            )
    else:
        lines.append("- Usability: none")

    if suite.audit_requirements:
        for requirement in suite.audit_requirements:
            lines.append(
                f"- Audit {requirement.event_type}: fields={len(requirement.emitted_fields)}/{len(requirement.required_fields)} "
                f"retention={requirement.observed_retention_days}/{requirement.retention_days}d"
            )
    else:
        lines.append("- Audit: none")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        f"- Role/Permission gaps: {', '.join(audit.permission_gaps) if audit.permission_gaps else 'none'}"
    )
    lines.append(
        f"- Data accuracy failures: {', '.join(audit.failing_data_checks) if audit.failing_data_checks else 'none'}"
    )
    lines.append(
        "- Performance budget failures: "
        f"{', '.join(audit.failing_performance_budgets) if audit.failing_performance_budgets else 'none'}"
    )
    lines.append(
        "- Usability journey failures: "
        f"{', '.join(audit.failing_usability_journeys) if audit.failing_usability_journeys else 'none'}"
    )
    lines.append(
        f"- Audit completeness gaps: {', '.join(audit.incomplete_audit_trails) if audit.incomplete_audit_trails else 'none'}"
    )
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    return "\n".join(lines) + "\n"


PRIORITY_WEIGHTS = {"P0": 4, "P1": 3, "P2": 2, "P3": 1}
GOAL_STATUS_ORDER = {
    "done": 4,
    "on-track": 3,
    "at-risk": 2,
    "blocked": 1,
    "not-started": 0,
}
REQUIRED_RUN_CLOSEOUTS = ("validation-evidence", "git-push", "git-log-stat")
ALLOWED_SCOPE_STATUSES = {"frozen", "approved-exception", "proposed"}


@dataclass(frozen=True)
class FreezeException:
    issue_id: str
    reason: str
    approved_by: str = ""
    decision_note: str = ""

    @property
    def approved(self) -> bool:
        return bool(self.approved_by.strip())

    def to_dict(self) -> Dict[str, str]:
        return {
            "issue_id": self.issue_id,
            "reason": self.reason,
            "approved_by": self.approved_by,
            "decision_note": self.decision_note,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "FreezeException":
        return cls(
            issue_id=data["issue_id"],
            reason=data.get("reason", ""),
            approved_by=data.get("approved_by", ""),
            decision_note=data.get("decision_note", ""),
        )


@dataclass
class GovernanceBacklogItem:
    issue_id: str
    title: str
    phase: str
    owner: str = ""
    status: str = "planned"
    scope_status: str = "frozen"
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    required_closeout: List[str] = field(default_factory=lambda: list(REQUIRED_RUN_CLOSEOUTS))
    linked_epics: List[str] = field(default_factory=list)
    notes: str = ""

    @property
    def missing_closeout_requirements(self) -> List[str]:
        present = {item.strip().lower() for item in self.required_closeout if item.strip()}
        return [item for item in REQUIRED_RUN_CLOSEOUTS if item not in present]

    @property
    def governance_ready(self) -> bool:
        return (
            bool(self.owner.strip())
            and self.scope_status in ALLOWED_SCOPE_STATUSES
            and bool(self.acceptance_criteria)
            and bool(self.validation_plan)
            and not self.missing_closeout_requirements
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "phase": self.phase,
            "owner": self.owner,
            "status": self.status,
            "scope_status": self.scope_status,
            "acceptance_criteria": list(self.acceptance_criteria),
            "validation_plan": list(self.validation_plan),
            "required_closeout": list(self.required_closeout),
            "linked_epics": list(self.linked_epics),
            "notes": self.notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "GovernanceBacklogItem":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            phase=str(data.get("phase", "")),
            owner=str(data.get("owner", "")),
            status=str(data.get("status", "planned")),
            scope_status=str(data.get("scope_status", "frozen")),
            acceptance_criteria=[str(item) for item in data.get("acceptance_criteria", [])],
            validation_plan=[str(item) for item in data.get("validation_plan", [])],
            required_closeout=[str(item) for item in data.get("required_closeout", [])],
            linked_epics=[str(item) for item in data.get("linked_epics", [])],
            notes=str(data.get("notes", "")),
        )


@dataclass
class ScopeFreezeBoard:
    name: str
    version: str
    freeze_date: str
    freeze_owner: str
    backlog_items: List[GovernanceBacklogItem] = field(default_factory=list)
    exceptions: List[FreezeException] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "freeze_date": self.freeze_date,
            "freeze_owner": self.freeze_owner,
            "backlog_items": [item.to_dict() for item in self.backlog_items],
            "exceptions": [exception.to_dict() for exception in self.exceptions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ScopeFreezeBoard":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            freeze_date=str(data.get("freeze_date", "")),
            freeze_owner=str(data.get("freeze_owner", "")),
            backlog_items=[GovernanceBacklogItem.from_dict(item) for item in data.get("backlog_items", [])],
            exceptions=[FreezeException.from_dict(item) for item in data.get("exceptions", [])],
        )


@dataclass
class ScopeFreezeAudit:
    board_name: str
    version: str
    total_items: int
    duplicate_issue_ids: List[str] = field(default_factory=list)
    missing_owners: List[str] = field(default_factory=list)
    missing_acceptance: List[str] = field(default_factory=list)
    missing_validation: List[str] = field(default_factory=list)
    missing_closeout_requirements: Dict[str, List[str]] = field(default_factory=dict)
    unauthorized_scope_changes: List[str] = field(default_factory=list)
    invalid_scope_statuses: List[str] = field(default_factory=list)
    unapproved_exceptions: List[str] = field(default_factory=list)

    @property
    def release_ready(self) -> bool:
        return not (
            self.duplicate_issue_ids
            or self.missing_owners
            or self.missing_acceptance
            or self.missing_validation
            or self.missing_closeout_requirements
            or self.unauthorized_scope_changes
            or self.invalid_scope_statuses
            or self.unapproved_exceptions
        )

    @property
    def readiness_score(self) -> float:
        checks = [
            not self.duplicate_issue_ids,
            not self.missing_owners,
            not self.missing_acceptance,
            not self.missing_validation,
            not self.missing_closeout_requirements,
            not self.unauthorized_scope_changes,
            not self.invalid_scope_statuses,
            not self.unapproved_exceptions,
        ]
        passed = sum(1 for item in checks if item)
        return round((passed / len(checks)) * 100, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "board_name": self.board_name,
            "version": self.version,
            "total_items": self.total_items,
            "duplicate_issue_ids": list(self.duplicate_issue_ids),
            "missing_owners": list(self.missing_owners),
            "missing_acceptance": list(self.missing_acceptance),
            "missing_validation": list(self.missing_validation),
            "missing_closeout_requirements": {
                issue_id: list(requirements)
                for issue_id, requirements in self.missing_closeout_requirements.items()
            },
            "unauthorized_scope_changes": list(self.unauthorized_scope_changes),
            "invalid_scope_statuses": list(self.invalid_scope_statuses),
            "unapproved_exceptions": list(self.unapproved_exceptions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ScopeFreezeAudit":
        return cls(
            board_name=str(data["board_name"]),
            version=str(data["version"]),
            total_items=int(data.get("total_items", 0)),
            duplicate_issue_ids=[str(item) for item in data.get("duplicate_issue_ids", [])],
            missing_owners=[str(item) for item in data.get("missing_owners", [])],
            missing_acceptance=[str(item) for item in data.get("missing_acceptance", [])],
            missing_validation=[str(item) for item in data.get("missing_validation", [])],
            missing_closeout_requirements={
                str(issue_id): [str(requirement) for requirement in requirements]
                for issue_id, requirements in dict(data.get("missing_closeout_requirements", {})).items()
            },
            unauthorized_scope_changes=[str(item) for item in data.get("unauthorized_scope_changes", [])],
            invalid_scope_statuses=[str(item) for item in data.get("invalid_scope_statuses", [])],
            unapproved_exceptions=[str(item) for item in data.get("unapproved_exceptions", [])],
        )


class ScopeFreezeGovernance:
    def audit(self, board: ScopeFreezeBoard) -> ScopeFreezeAudit:
        counts: Dict[str, int] = {}
        exception_index = {exception.issue_id: exception for exception in board.exceptions}

        for item in board.backlog_items:
            counts[item.issue_id] = counts.get(item.issue_id, 0) + 1
        duplicate_issue_ids = sorted(issue_id for issue_id, count in counts.items() if count > 1)

        missing_owners = sorted(item.issue_id for item in board.backlog_items if not item.owner.strip())
        missing_acceptance = sorted(item.issue_id for item in board.backlog_items if not item.acceptance_criteria)
        missing_validation = sorted(item.issue_id for item in board.backlog_items if not item.validation_plan)
        missing_closeout_requirements = {
            item.issue_id: item.missing_closeout_requirements
            for item in board.backlog_items
            if item.missing_closeout_requirements
        }
        invalid_scope_statuses = sorted(
            item.issue_id for item in board.backlog_items if item.scope_status not in ALLOWED_SCOPE_STATUSES
        )

        unauthorized_scope_changes: List[str] = []
        for item in board.backlog_items:
            if item.scope_status != "proposed":
                continue
            exception = exception_index.get(item.issue_id)
            if exception is None or not exception.approved:
                unauthorized_scope_changes.append(item.issue_id)

        unapproved_exceptions = sorted(exception.issue_id for exception in board.exceptions if not exception.approved)

        return ScopeFreezeAudit(
            board_name=board.name,
            version=board.version,
            total_items=len(board.backlog_items),
            duplicate_issue_ids=duplicate_issue_ids,
            missing_owners=missing_owners,
            missing_acceptance=missing_acceptance,
            missing_validation=missing_validation,
            missing_closeout_requirements=missing_closeout_requirements,
            unauthorized_scope_changes=sorted(unauthorized_scope_changes),
            invalid_scope_statuses=invalid_scope_statuses,
            unapproved_exceptions=unapproved_exceptions,
        )


def render_scope_freeze_report(board: ScopeFreezeBoard, audit: ScopeFreezeAudit) -> str:
    lines = [
        "# Scope Freeze Governance Report",
        "",
        f"- Name: {board.name}",
        f"- Version: {board.version}",
        f"- Freeze Date: {board.freeze_date}",
        f"- Freeze Owner: {board.freeze_owner}",
        f"- Backlog Items: {len(board.backlog_items)}",
        f"- Exceptions: {len(board.exceptions)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Backlog",
        "",
    ]

    if board.backlog_items:
        for item in board.backlog_items:
            closeout = ", ".join(item.required_closeout) or "none"
            lines.append(
                f"- {item.issue_id}: phase={item.phase} owner={item.owner or 'none'} "
                f"status={item.status} scope={item.scope_status} closeout={closeout}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Freeze Exceptions", ""])
    if board.exceptions:
        for exception in board.exceptions:
            lines.append(
                f"- {exception.issue_id}: approved_by={exception.approved_by or 'pending'} reason={exception.reason or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    lines.append(
        f"- Duplicate issues: {', '.join(audit.duplicate_issue_ids) if audit.duplicate_issue_ids else 'none'}"
    )
    lines.append(f"- Missing owners: {', '.join(audit.missing_owners) if audit.missing_owners else 'none'}")
    lines.append(
        f"- Missing acceptance: {', '.join(audit.missing_acceptance) if audit.missing_acceptance else 'none'}"
    )
    lines.append(
        f"- Missing validation: {', '.join(audit.missing_validation) if audit.missing_validation else 'none'}"
    )
    if audit.missing_closeout_requirements:
        missing_closeout = "; ".join(
            f"{issue_id}={', '.join(requirements)}"
            for issue_id, requirements in sorted(audit.missing_closeout_requirements.items())
        )
    else:
        missing_closeout = "none"
    lines.append(f"- Missing closeout requirements: {missing_closeout}")
    lines.append(
        "- Unauthorized scope changes: "
        f"{', '.join(audit.unauthorized_scope_changes) if audit.unauthorized_scope_changes else 'none'}"
    )
    lines.append(
        f"- Invalid scope statuses: {', '.join(audit.invalid_scope_statuses) if audit.invalid_scope_statuses else 'none'}"
    )
    lines.append(
        f"- Unapproved exceptions: {', '.join(audit.unapproved_exceptions) if audit.unapproved_exceptions else 'none'}"
    )
    return "\n".join(lines) + "\n"


@dataclass(frozen=True)
class EvidenceLink:
    label: str
    target: str
    capability: str = ""
    note: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "label": self.label,
            "target": self.target,
            "capability": self.capability,
            "note": self.note,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EvidenceLink":
        return cls(
            label=str(data["label"]),
            target=str(data["target"]),
            capability=str(data.get("capability", "")),
            note=str(data.get("note", "")),
        )


@dataclass(frozen=True)
class CandidateEntry:
    candidate_id: str
    title: str
    theme: str
    priority: str
    owner: str
    outcome: str
    validation_command: str
    capabilities: List[str] = field(default_factory=list)
    evidence: List[str] = field(default_factory=list)
    evidence_links: List[EvidenceLink] = field(default_factory=list)
    dependencies: List[str] = field(default_factory=list)
    blockers: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> int:
        base = PRIORITY_WEIGHTS.get(self.priority.upper(), 0) * 25
        dependency_penalty = len(self.dependencies) * 10
        blocker_penalty = len(self.blockers) * 20
        evidence_bonus = min(len(self.evidence), 3) * 5
        return max(0, min(100, base + evidence_bonus - dependency_penalty - blocker_penalty))

    @property
    def ready(self) -> bool:
        return bool(self.capabilities) and bool(self.evidence) and not self.blockers

    def to_dict(self) -> Dict[str, object]:
        return {
            "candidate_id": self.candidate_id,
            "title": self.title,
            "theme": self.theme,
            "priority": self.priority,
            "owner": self.owner,
            "outcome": self.outcome,
            "validation_command": self.validation_command,
            "capabilities": list(self.capabilities),
            "evidence": list(self.evidence),
            "evidence_links": [link.to_dict() for link in self.evidence_links],
            "dependencies": list(self.dependencies),
            "blockers": list(self.blockers),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "CandidateEntry":
        return cls(
            candidate_id=str(data["candidate_id"]),
            title=str(data["title"]),
            theme=str(data["theme"]),
            priority=str(data["priority"]),
            owner=str(data["owner"]),
            outcome=str(data["outcome"]),
            validation_command=str(data["validation_command"]),
            capabilities=[str(item) for item in data.get("capabilities", [])],
            evidence=[str(item) for item in data.get("evidence", [])],
            evidence_links=[EvidenceLink.from_dict(item) for item in data.get("evidence_links", [])],
            dependencies=[str(item) for item in data.get("dependencies", [])],
            blockers=[str(item) for item in data.get("blockers", [])],
        )


@dataclass
class CandidateBacklog:
    epic_id: str
    title: str
    version: str
    candidates: List[CandidateEntry] = field(default_factory=list)

    @property
    def ranked_candidates(self) -> List[CandidateEntry]:
        return sorted(
            self.candidates,
            key=lambda candidate: (-candidate.readiness_score, candidate.candidate_id),
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "epic_id": self.epic_id,
            "title": self.title,
            "version": self.version,
            "candidates": [candidate.to_dict() for candidate in self.candidates],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "CandidateBacklog":
        return cls(
            epic_id=str(data["epic_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            candidates=[CandidateEntry.from_dict(item) for item in data.get("candidates", [])],
        )


@dataclass(frozen=True)
class EntryGate:
    gate_id: str
    name: str
    min_ready_candidates: int
    required_capabilities: List[str] = field(default_factory=list)
    required_evidence: List[str] = field(default_factory=list)
    required_baseline_version: str = ""
    max_blockers: int = 0

    def to_dict(self) -> Dict[str, object]:
        return {
            "gate_id": self.gate_id,
            "name": self.name,
            "min_ready_candidates": self.min_ready_candidates,
            "required_capabilities": list(self.required_capabilities),
            "required_evidence": list(self.required_evidence),
            "required_baseline_version": self.required_baseline_version,
            "max_blockers": self.max_blockers,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EntryGate":
        return cls(
            gate_id=str(data["gate_id"]),
            name=str(data["name"]),
            min_ready_candidates=int(data["min_ready_candidates"]),
            required_capabilities=[str(item) for item in data.get("required_capabilities", [])],
            required_evidence=[str(item) for item in data.get("required_evidence", [])],
            required_baseline_version=str(data.get("required_baseline_version", "")),
            max_blockers=int(data.get("max_blockers", 0)),
        )


@dataclass
class EntryGateDecision:
    gate_id: str
    passed: bool
    ready_candidate_ids: List[str] = field(default_factory=list)
    blocked_candidate_ids: List[str] = field(default_factory=list)
    missing_capabilities: List[str] = field(default_factory=list)
    missing_evidence: List[str] = field(default_factory=list)
    baseline_ready: bool = True
    baseline_findings: List[str] = field(default_factory=list)
    blocker_count: int = 0

    @property
    def summary(self) -> str:
        status = "PASS" if self.passed else "HOLD"
        return (
            f"{status}: ready={len(self.ready_candidate_ids)} "
            f"blocked={self.blocker_count} "
            f"missing_capabilities={len(self.missing_capabilities)} "
            f"missing_evidence={len(self.missing_evidence)} "
            f"baseline_findings={len(self.baseline_findings)}"
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "gate_id": self.gate_id,
            "passed": self.passed,
            "ready_candidate_ids": list(self.ready_candidate_ids),
            "blocked_candidate_ids": list(self.blocked_candidate_ids),
            "missing_capabilities": list(self.missing_capabilities),
            "missing_evidence": list(self.missing_evidence),
            "baseline_ready": self.baseline_ready,
            "baseline_findings": list(self.baseline_findings),
            "blocker_count": self.blocker_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EntryGateDecision":
        return cls(
            gate_id=str(data["gate_id"]),
            passed=bool(data["passed"]),
            ready_candidate_ids=[str(item) for item in data.get("ready_candidate_ids", [])],
            blocked_candidate_ids=[str(item) for item in data.get("blocked_candidate_ids", [])],
            missing_capabilities=[str(item) for item in data.get("missing_capabilities", [])],
            missing_evidence=[str(item) for item in data.get("missing_evidence", [])],
            baseline_ready=bool(data.get("baseline_ready", True)),
            baseline_findings=[str(item) for item in data.get("baseline_findings", [])],
            blocker_count=int(data.get("blocker_count", 0)),
        )


class CandidatePlanner:
    def evaluate_gate(
        self,
        backlog: CandidateBacklog,
        gate: EntryGate,
        baseline_audit: Optional[ScopeFreezeAudit] = None,
    ) -> EntryGateDecision:
        ready_candidates = [candidate for candidate in backlog.ranked_candidates if candidate.ready]
        blocked_candidates = [candidate for candidate in backlog.candidates if candidate.blockers]
        provided_capabilities = {capability for candidate in ready_candidates for capability in candidate.capabilities}
        provided_evidence = {item for candidate in ready_candidates for item in candidate.evidence}
        missing_capabilities = [
            capability
            for capability in gate.required_capabilities
            if capability not in provided_capabilities
        ]
        missing_evidence = [item for item in gate.required_evidence if item not in provided_evidence]
        baseline_findings = self._baseline_findings(gate, baseline_audit)
        baseline_ready = not baseline_findings
        passed = (
            len(ready_candidates) >= gate.min_ready_candidates
            and len(blocked_candidates) <= gate.max_blockers
            and not missing_capabilities
            and not missing_evidence
            and baseline_ready
        )
        return EntryGateDecision(
            gate_id=gate.gate_id,
            passed=passed,
            ready_candidate_ids=[candidate.candidate_id for candidate in ready_candidates],
            blocked_candidate_ids=[candidate.candidate_id for candidate in blocked_candidates],
            missing_capabilities=missing_capabilities,
            missing_evidence=missing_evidence,
            baseline_ready=baseline_ready,
            baseline_findings=baseline_findings,
            blocker_count=len(blocked_candidates),
        )

    def _baseline_findings(
        self,
        gate: EntryGate,
        baseline_audit: Optional[ScopeFreezeAudit],
    ) -> List[str]:
        if not gate.required_baseline_version:
            return []
        if baseline_audit is None:
            return [f"missing baseline audit for {gate.required_baseline_version}"]
        findings: List[str] = []
        if baseline_audit.version != gate.required_baseline_version:
            findings.append(
                f"baseline version mismatch: expected {gate.required_baseline_version}, got {baseline_audit.version}"
            )
        if not baseline_audit.release_ready:
            findings.append(
                f"baseline {baseline_audit.version} is not release ready ({baseline_audit.readiness_score:.1f})"
            )
        return findings


def render_candidate_backlog_report(
    backlog: CandidateBacklog,
    gate: EntryGate,
    decision: EntryGateDecision,
) -> str:
    lines = [
        "# V3 Candidate Backlog Report",
        "",
        f"- Epic: {backlog.epic_id} {backlog.title}",
        f"- Version: {backlog.version}",
        f"- Gate: {gate.name}",
        f"- Decision: {decision.summary}",
        "",
        "## Candidates",
    ]
    for candidate in backlog.ranked_candidates:
        lines.append(
            "- "
            f"{candidate.candidate_id}: {candidate.title} "
            f"priority={candidate.priority} owner={candidate.owner} "
            f"score={candidate.readiness_score} ready={candidate.ready}"
        )
        lines.append(
            "  "
            f"theme={candidate.theme} outcome={candidate.outcome} "
            f"capabilities={','.join(candidate.capabilities) or 'none'} "
            f"evidence={','.join(candidate.evidence) or 'none'} "
            f"blockers={','.join(candidate.blockers) or 'none'}"
        )
        lines.append(f"  validation={candidate.validation_command}")
        if candidate.dependencies:
            lines.append(f"  dependencies={','.join(candidate.dependencies)}")
        if candidate.evidence_links:
            lines.append("  evidence-links:")
            for link in candidate.evidence_links:
                qualifier = f" capability={link.capability}" if link.capability else ""
                note = f" note={link.note}" if link.note else ""
                lines.append(f"  - {link.label} -> {link.target}{qualifier}{note}")
    lines.extend(
        [
            "",
            "## Gate Findings",
            f"- Ready candidates: {', '.join(decision.ready_candidate_ids) or 'none'}",
            f"- Blocked candidates: {', '.join(decision.blocked_candidate_ids) or 'none'}",
            f"- Missing capabilities: {', '.join(decision.missing_capabilities) or 'none'}",
            f"- Missing evidence: {', '.join(decision.missing_evidence) or 'none'}",
            f"- Baseline ready: {decision.baseline_ready}",
            f"- Baseline findings: {', '.join(decision.baseline_findings) or 'none'}",
        ]
    )
    return "\n".join(lines)


def build_v3_candidate_backlog() -> CandidateBacklog:
    return CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Console release control center",
                theme="console-governance",
                priority="P0",
                owner="product-experience",
                outcome="Converge console shell governance, UI acceptance, and review-pack evidence into one release-control candidate.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_design_system.py "
                    "tests/test_console_ia.py tests/test_ui_review.py -q"
                ),
                capabilities=["release-gate", "console-shell", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="design-system-audit",
                        target="src/bigclaw/design_system.py",
                        capability="release-gate",
                        note="component inventory, accessibility, UI acceptance, and execution governance coverage",
                    ),
                    EvidenceLink(
                        label="console-ia-contract",
                        target="src/bigclaw/design_system.py",
                        capability="release-gate",
                        note="design system plus global navigation, top bar, filters, and state contracts",
                    ),
                    EvidenceLink(
                        label="ui-review-pack",
                        target="src/bigclaw/ui_review.py",
                        capability="release-gate",
                        note="review objectives, wireframes, interaction coverage, and open questions",
                    ),
                    EvidenceLink(
                        label="ui-acceptance-tests",
                        target="tests/test_design_system.py",
                        capability="release-gate",
                        note="role-permission, data accuracy, and performance audits",
                    ),
                    EvidenceLink(
                        label="console-shell-tests",
                        target="tests/test_console_ia.py",
                        capability="release-gate",
                        note="console shell and interaction draft release readiness",
                    ),
                    EvidenceLink(
                        label="review-pack-tests",
                        target="tests/test_ui_review.py",
                        capability="release-gate",
                        note="deterministic review packet validation",
                    ),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Operations command-center hardening",
                theme="ops-command-center",
                priority="P0",
                owner="engineering-operations",
                outcome="Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_operations.py "
                    "tests/test_evaluation.py -q && "
                    "(cd bigclaw-go && go test ./internal/product ./internal/worker ./internal/workflow ./internal/scheduler)"
                ),
                capabilities=["ops-control", "saved-views", "rollback-simulation"],
                evidence=["weekly-review", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="command-center-src",
                        target="src/bigclaw/operations.py",
                        capability="ops-control",
                        note="queue control center, dashboard builder, weekly review, and regression surfaces",
                    ),
                    EvidenceLink(
                        label="command-center-tests",
                        target="tests/test_control_center.py",
                        capability="ops-control",
                        note="queue control center validation",
                    ),
                    EvidenceLink(
                        label="operations-tests",
                        target="tests/test_operations.py",
                        capability="ops-control",
                        note="dashboard, weekly report, regression, and version-center coverage",
                    ),
                    EvidenceLink(
                        label="approval-contract",
                        target="src/bigclaw/execution_contract.py",
                        capability="ops-control",
                        note="approval permission and API role coverage contract",
                    ),
                    EvidenceLink(
                        label="approval-workflow",
                        target="src/bigclaw/workflow.py",
                        capability="ops-control",
                        note="approval workflow and closeout flow wiring",
                    ),
                    EvidenceLink(
                        label="workflow-tests",
                        target="bigclaw-go/internal/workflow/engine_test.go",
                        capability="ops-control",
                        note="acceptance gate and workpad journal validation",
                    ),
                    EvidenceLink(
                        label="execution-flow-tests",
                        target="bigclaw-go/internal/worker/runtime_test.go",
                        capability="ops-control",
                        note="execution handoff, closeout, and routed runtime evidence",
                    ),
                    EvidenceLink(
                        label="saved-views-src",
                        target="src/bigclaw/saved_views.py",
                        capability="saved-views",
                        note="saved views, digest subscriptions, and governed filters",
                    ),
                    EvidenceLink(
                        label="saved-views-tests",
                        target="bigclaw-go/internal/product/saved_views_test.go",
                        capability="saved-views",
                        note="Go-native saved-view audit coverage",
                    ),
                    EvidenceLink(
                        label="simulation-src",
                        target="src/bigclaw/operations.py",
                        capability="rollback-simulation",
                        note="simulation, replay, comparison evidence, and operator dashboard context",
                    ),
                    EvidenceLink(
                        label="simulation-tests",
                        target="tests/test_evaluation.py",
                        capability="rollback-simulation",
                        note="replay and benchmark validation",
                    ),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration-rollout",
                title="Agent orchestration rollout",
                theme="agent-orchestration",
                priority="P0",
                owner="orchestration-office",
                outcome="Carry entitlement-aware orchestration, handoff visibility, and commercialization proof into a candidate ready for release review.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_reports.py -q"
                ),
                capabilities=["commercialization", "handoff", "pilot-rollout"],
                evidence=["pilot-evidence", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="orchestration-plan-src",
                        target="src/bigclaw/orchestration.py",
                        capability="commercialization",
                        note="cross-team orchestration, entitlement-aware policy, and handoff decisions",
                    ),
                    EvidenceLink(
                        label="orchestration-report-src",
                        target="src/bigclaw/reports.py",
                        capability="commercialization",
                        note="orchestration canvas, portfolio rollups, and narrative exports",
                    ),
                    EvidenceLink(
                        label="orchestration-tests",
                        target="tests/test_orchestration.py",
                        capability="commercialization",
                        note="handoff and policy decision validation",
                    ),
                    EvidenceLink(
                        label="report-studio-tests",
                        target="tests/test_reports.py",
                        capability="commercialization",
                        note="report exports and downstream evidence sharing",
                    ),
                ],
            ),
        ],
    )


def build_v3_entry_gate() -> EntryGate:
    return EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
        max_blockers=0,
    )


@dataclass(frozen=True)
class WeeklyGoal:
    goal_id: str
    title: str
    owner: str
    status: str
    success_metric: str
    target_value: str
    current_value: str = ""
    dependencies: List[str] = field(default_factory=list)
    risks: List[str] = field(default_factory=list)

    @property
    def status_rank(self) -> int:
        return GOAL_STATUS_ORDER.get(self.status.strip().lower(), -1)

    @property
    def is_complete(self) -> bool:
        return self.status.strip().lower() == "done"

    @property
    def is_at_risk(self) -> bool:
        return self.status.strip().lower() in {"at-risk", "blocked"}

    def to_dict(self) -> Dict[str, object]:
        return {
            "goal_id": self.goal_id,
            "title": self.title,
            "owner": self.owner,
            "status": self.status,
            "success_metric": self.success_metric,
            "target_value": self.target_value,
            "current_value": self.current_value,
            "dependencies": list(self.dependencies),
            "risks": list(self.risks),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WeeklyGoal":
        return cls(
            goal_id=str(data["goal_id"]),
            title=str(data["title"]),
            owner=str(data["owner"]),
            status=str(data["status"]),
            success_metric=str(data["success_metric"]),
            target_value=str(data["target_value"]),
            current_value=str(data.get("current_value", "")),
            dependencies=[str(item) for item in data.get("dependencies", [])],
            risks=[str(item) for item in data.get("risks", [])],
        )


@dataclass(frozen=True)
class WeeklyExecutionPlan:
    week_number: int
    theme: str
    objective: str
    exit_criteria: List[str] = field(default_factory=list)
    deliverables: List[str] = field(default_factory=list)
    goals: List[WeeklyGoal] = field(default_factory=list)

    @property
    def completed_goals(self) -> int:
        return sum(goal.is_complete for goal in self.goals)

    @property
    def total_goals(self) -> int:
        return len(self.goals)

    @property
    def progress_percent(self) -> int:
        if not self.goals:
            return 0
        return int((self.completed_goals / len(self.goals)) * 100)

    @property
    def at_risk_goal_ids(self) -> List[str]:
        return [goal.goal_id for goal in self.goals if goal.is_at_risk]

    def to_dict(self) -> Dict[str, object]:
        return {
            "week_number": self.week_number,
            "theme": self.theme,
            "objective": self.objective,
            "exit_criteria": list(self.exit_criteria),
            "deliverables": list(self.deliverables),
            "goals": [goal.to_dict() for goal in self.goals],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WeeklyExecutionPlan":
        return cls(
            week_number=int(data["week_number"]),
            theme=str(data["theme"]),
            objective=str(data["objective"]),
            exit_criteria=[str(item) for item in data.get("exit_criteria", [])],
            deliverables=[str(item) for item in data.get("deliverables", [])],
            goals=[WeeklyGoal.from_dict(item) for item in data.get("goals", [])],
        )


@dataclass
class FourWeekExecutionPlan:
    plan_id: str
    title: str
    owner: str
    start_date: str
    weeks: List[WeeklyExecutionPlan] = field(default_factory=list)

    @property
    def total_goals(self) -> int:
        return sum(week.total_goals for week in self.weeks)

    @property
    def completed_goals(self) -> int:
        return sum(week.completed_goals for week in self.weeks)

    @property
    def overall_progress_percent(self) -> int:
        if self.total_goals == 0:
            return 0
        return int((self.completed_goals / self.total_goals) * 100)

    @property
    def at_risk_weeks(self) -> List[int]:
        return [week.week_number for week in self.weeks if week.at_risk_goal_ids]

    def goal_status_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for week in self.weeks:
            for goal in week.goals:
                counts[goal.status] = counts.get(goal.status, 0) + 1
        return counts

    def validate(self) -> None:
        week_numbers = [week.week_number for week in self.weeks]
        if week_numbers != [1, 2, 3, 4]:
            raise ValueError("Four-week execution plans must include weeks 1 through 4 in order")

    def to_dict(self) -> Dict[str, object]:
        return {
            "plan_id": self.plan_id,
            "title": self.title,
            "owner": self.owner,
            "start_date": self.start_date,
            "weeks": [week.to_dict() for week in self.weeks],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "FourWeekExecutionPlan":
        return cls(
            plan_id=str(data["plan_id"]),
            title=str(data["title"]),
            owner=str(data["owner"]),
            start_date=str(data["start_date"]),
            weeks=[WeeklyExecutionPlan.from_dict(item) for item in data.get("weeks", [])],
        )


def build_big_4701_execution_plan() -> FourWeekExecutionPlan:
    plan = FourWeekExecutionPlan(
        plan_id="BIG-4701",
        title="4周执行计划与周目标",
        owner="execution-office",
        start_date="2026-03-11",
        weeks=[
            WeeklyExecutionPlan(
                week_number=1,
                theme="Scope freeze and operating baseline",
                objective="Freeze scope, align owners, and establish validation and reporting cadence.",
                exit_criteria=[
                    "Scope freeze board published",
                    "Owners and validation commands assigned for all streams",
                ],
                deliverables=[
                    "Execution baseline report",
                    "Scope freeze audit snapshot",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w1-scope-freeze",
                        title="Lock the v4.0 scope and escalation path",
                        owner="program-office",
                        status="done",
                        success_metric="frozen backlog items",
                        target_value="5 epics aligned",
                        current_value="5 epics aligned",
                    ),
                    WeeklyGoal(
                        goal_id="w1-validation-matrix",
                        title="Assign validation commands and evidence owners",
                        owner="engineering-ops",
                        status="done",
                        success_metric="streams with validation owners",
                        target_value="5/5 streams",
                        current_value="5/5 streams",
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=2,
                theme="Build and integration",
                objective="Land the highest-risk implementation slices and wire cross-team dependencies.",
                exit_criteria=[
                    "P0 build items merged",
                    "Cross-team dependency review completed",
                ],
                deliverables=[
                    "Integrated build checkpoint",
                    "Dependency burn-down",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w2-p0-burndown",
                        title="Close the top P0 implementation gaps",
                        owner="engineering-platform",
                        status="on-track",
                        success_metric="P0 items merged",
                        target_value=">=3 merged",
                        current_value="2 merged",
                    ),
                    WeeklyGoal(
                        goal_id="w2-handoff-sync",
                        title="Resolve orchestration and console handoff dependencies",
                        owner="orchestration-office",
                        status="at-risk",
                        success_metric="open handoff blockers",
                        target_value="0 blockers",
                        current_value="1 blocker",
                        dependencies=["w2-p0-burndown"],
                        risks=["console entitlement contract is pending"],
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=3,
                theme="Stabilization and validation",
                objective="Drive regression triage, benchmark replay, and release-readiness evidence.",
                exit_criteria=[
                    "Regression backlog under control threshold",
                    "Benchmark comparison published",
                ],
                deliverables=[
                    "Stabilization report",
                    "Benchmark replay pack",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w3-regression-triage",
                        title="Reduce critical regressions before release gate",
                        owner="quality-ops",
                        status="not-started",
                        success_metric="critical regressions",
                        target_value="<=2 open",
                    ),
                    WeeklyGoal(
                        goal_id="w3-benchmark-pack",
                        title="Publish replay and weighted benchmark evidence",
                        owner="evaluation-lab",
                        status="not-started",
                        success_metric="benchmark evidence bundle",
                        target_value="1 bundle published",
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=4,
                theme="Launch decision and weekly operating rhythm",
                objective="Convert validation evidence into launch readiness and the post-launch weekly review cadence.",
                exit_criteria=[
                    "Launch decision signed off",
                    "Weekly operating review template adopted",
                ],
                deliverables=[
                    "Launch readiness packet",
                    "Weekly review operating template",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w4-launch-decision",
                        title="Complete launch readiness review",
                        owner="release-governance",
                        status="not-started",
                        success_metric="required sign-offs",
                        target_value="all sign-offs complete",
                    ),
                    WeeklyGoal(
                        goal_id="w4-weekly-rhythm",
                        title="Roll out the weekly KPI and issue review cadence",
                        owner="engineering-operations",
                        status="not-started",
                        success_metric="weekly review adoption",
                        target_value="1 recurring cadence active",
                    ),
                ],
            ),
        ],
    )
    plan.validate()
    return plan


def build_pilot_rollout_scorecard(
    *,
    adoption: float,
    convergence_improvement: float,
    review_efficiency: float,
    governance_incidents: int,
    evidence_completeness: float,
) -> Dict[str, object]:
    score = (
        adoption * 0.25
        + convergence_improvement * 0.25
        + review_efficiency * 0.2
        + evidence_completeness * 0.2
        + max(0.0, 100.0 - (governance_incidents * 20.0)) * 0.1
    )
    passed = score >= 75 and governance_incidents <= 2 and evidence_completeness >= 70
    return {
        "adoption": round(adoption, 1),
        "convergence_improvement": round(convergence_improvement, 1),
        "review_efficiency": round(review_efficiency, 1),
        "governance_incidents": int(governance_incidents),
        "evidence_completeness": round(evidence_completeness, 1),
        "rollout_score": round(score, 1),
        "recommendation": "go" if passed else "hold",
    }


def evaluate_candidate_gate(
    *,
    gate_decision: EntryGateDecision,
    rollout_scorecard: Dict[str, object],
) -> Dict[str, object]:
    readiness = bool(gate_decision.passed)
    rollout_ready = rollout_scorecard.get("recommendation") == "go"
    recommendation = "enable-by-default" if readiness and rollout_ready else "pilot-only"
    findings: List[str] = []
    if not readiness:
        findings.append(gate_decision.summary)
    if not rollout_ready:
        findings.append(
            "rollout score below threshold"
            f" ({rollout_scorecard.get('rollout_score', 'n/a')})"
        )
    return {
        "gate_passed": readiness,
        "rollout_recommendation": str(rollout_scorecard.get("recommendation", "hold")),
        "candidate_gate": recommendation,
        "findings": findings,
    }


def render_pilot_rollout_gate_report(result: Dict[str, object]) -> str:
    findings = result.get("findings") or []
    lines = [
        "# Pilot Rollout Candidate Gate",
        "",
        f"- Gate passed: {result.get('gate_passed')}",
        f"- Rollout recommendation: {result.get('rollout_recommendation')}",
        f"- Candidate gate: {result.get('candidate_gate')}",
    ]
    lines.append(f"- Findings: {', '.join(findings) if findings else 'none'}")
    return "\n".join(lines)


def render_four_week_execution_report(plan: FourWeekExecutionPlan) -> str:
    plan.validate()
    status_counts = plan.goal_status_counts()
    lines = [
        "# Four-Week Execution Plan",
        "",
        f"- Plan: {plan.plan_id} {plan.title}",
        f"- Owner: {plan.owner}",
        f"- Start date: {plan.start_date}",
        f"- Overall progress: {plan.completed_goals}/{plan.total_goals} goals complete ({plan.overall_progress_percent}%)",
        f"- At-risk weeks: {', '.join(str(week_number) for week_number in plan.at_risk_weeks) or 'none'}",
        (
            "- Goal status counts: "
            f"done={status_counts.get('done', 0)} "
            f"on-track={status_counts.get('on-track', 0)} "
            f"at-risk={status_counts.get('at-risk', 0)} "
            f"blocked={status_counts.get('blocked', 0)} "
            f"not-started={status_counts.get('not-started', 0)}"
        ),
        "",
        "## Weekly Plans",
    ]
    for week in plan.weeks:
        lines.extend(
            [
                (
                    f"- Week {week.week_number}: {week.theme} "
                    f"progress={week.completed_goals}/{week.total_goals} ({week.progress_percent}%)"
                ),
                f"  objective={week.objective}",
                f"  exit_criteria={', '.join(week.exit_criteria) or 'none'}",
                f"  deliverables={', '.join(week.deliverables) or 'none'}",
            ]
        )
        for goal in week.goals:
            lines.append(
                "  "
                f"- {goal.goal_id}: {goal.title} owner={goal.owner} status={goal.status} "
                f"metric={goal.success_metric} current={goal.current_value or 'n/a'} "
                f"target={goal.target_value}"
            )
            lines.append(
                "    "
                f"dependencies={','.join(goal.dependencies) or 'none'} "
                f"risks={','.join(goal.risks) or 'none'}"
            )
    return "\n".join(lines)
