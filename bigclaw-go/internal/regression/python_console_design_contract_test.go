package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonConsoleIAContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "console_ia_contract.py")
	script := `import json
import sys

from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.console_ia import (
    ConsoleIA,
    ConsoleIAAuditor,
    ConsoleInteractionAuditor,
    ConsoleInteractionDraft,
    ConsoleSurface,
    FilterDefinition,
    GlobalAction,
    NavigationItem,
    SurfaceInteractionContract,
    SurfacePermissionRule,
    SurfaceState,
    build_big_4203_console_interaction_draft,
    render_console_interaction_report,
    render_console_ia_report,
)
from bigclaw.design_system import CommandAction, ConsoleCommandEntry, ConsoleTopBar

architecture = ConsoleIA(
    name="BigClaw Console IA",
    version="v3",
    top_bar=ConsoleTopBar(
        name="BigClaw Global Header",
        search_placeholder="Search runs, issues, commands",
        environment_options=["Production", "Staging"],
        time_range_options=["24h", "7d"],
        alert_channels=["approvals"],
        documentation_complete=True,
        accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
        command_entry=ConsoleCommandEntry(
            trigger_label="Command Menu",
            placeholder="Type a command",
            shortcut="Cmd+K / Ctrl+K",
            commands=[CommandAction(id="search-runs", title="Search runs", section="Navigate")],
        ),
    ),
    navigation=[
        NavigationItem(name="Overview", route="/overview", section="Operate", icon="dashboard", badge_count=2)
    ],
    surfaces=[
        ConsoleSurface(
            name="Overview",
            route="/overview",
            navigation_section="Operate",
            top_bar_actions=[GlobalAction(action_id="refresh", label="Refresh", placement="topbar")],
            filters=[
                FilterDefinition(
                    name="Team",
                    field="team",
                    control="select",
                    options=["all", "platform"],
                    default_value="all",
                )
            ],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["refresh"]),
                SurfaceState(name="empty", allowed_actions=["refresh"]),
                SurfaceState(name="error", allowed_actions=["refresh"]),
            ],
        )
    ],
)

restored = ConsoleIA.from_dict(architecture.to_dict())

gap_architecture = ConsoleIA(
    name="BigClaw Console IA",
    version="v3",
    top_bar=ConsoleTopBar(
        name="Incomplete Header",
        search_placeholder="",
        environment_options=["Production"],
        time_range_options=["24h"],
        documentation_complete=False,
        accessibility_requirements=["focus-visible"],
        command_entry=ConsoleCommandEntry(trigger_label="", placeholder="", shortcut="Cmd+K"),
    ),
    navigation=[
        NavigationItem(name="Overview", route="/overview", section="Operate"),
        NavigationItem(name="Ghost", route="/ghost", section="Operate"),
    ],
    surfaces=[
        ConsoleSurface(
            name="Overview",
            route="/overview",
            navigation_section="Operate",
            top_bar_actions=[GlobalAction(action_id="refresh", label="Refresh", placement="topbar")],
            filters=[FilterDefinition(name="Team", field="team", control="select", options=["all"])],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["refresh"]),
                SurfaceState(name="empty", allowed_actions=["refresh"]),
                SurfaceState(name="error", allowed_actions=["refresh"]),
            ],
        ),
        ConsoleSurface(
            name="Queue",
            route="/queue",
            navigation_section="Operate",
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading"),
                SurfaceState(name="empty", allowed_actions=["retry"]),
            ],
        ),
    ],
)
gap_audit = ConsoleIAAuditor().audit(gap_architecture)

report_architecture = ConsoleIA(
    name="BigClaw Console IA",
    version="v3",
    top_bar=ConsoleTopBar(
        name="BigClaw Global Header",
        search_placeholder="Search runs, issues, commands",
        environment_options=["Production", "Staging"],
        time_range_options=["24h", "7d", "30d"],
        alert_channels=["approvals", "sla"],
        documentation_complete=True,
        accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
        command_entry=ConsoleCommandEntry(
            trigger_label="Command Menu",
            placeholder="Type a command or jump to a run",
            shortcut="Cmd+K / Ctrl+K",
            commands=[
                CommandAction(id="search-runs", title="Search runs", section="Navigate", shortcut="/"),
                CommandAction(id="open-alerts", title="Open alerts", section="Monitor"),
            ],
        ),
    ),
    navigation=[NavigationItem(name="Overview", route="/overview", section="Operate")],
    surfaces=[
        ConsoleSurface(
            name="Overview",
            route="/overview",
            navigation_section="Operate",
            top_bar_actions=[GlobalAction(action_id="refresh", label="Refresh", placement="topbar")],
            filters=[FilterDefinition(name="Team", field="team", control="select", options=["all"])],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["refresh"]),
                SurfaceState(name="empty", allowed_actions=["refresh"]),
                SurfaceState(name="error", allowed_actions=["refresh"]),
            ],
        )
    ],
)
report_audit = ConsoleIAAuditor().audit(report_architecture)
ia_report = render_console_ia_report(report_architecture, report_audit)

interaction_architecture = ConsoleIA(
    name="BigClaw Console IA",
    version="v3",
    top_bar=ConsoleTopBar(
        name="BigClaw Global Header",
        search_placeholder="Search runs, issues, commands",
        environment_options=["Production", "Staging"],
        time_range_options=["24h", "7d"],
        alert_channels=["approvals"],
        documentation_complete=True,
        accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
        command_entry=ConsoleCommandEntry(
            trigger_label="Command Menu",
            placeholder="Type a command",
            shortcut="Cmd+K / Ctrl+K",
            commands=[CommandAction(id="search-runs", title="Search runs", section="Navigate")],
        ),
    ),
    navigation=[
        NavigationItem(name="Overview", route="/overview", section="Operate"),
        NavigationItem(name="Queue", route="/queue", section="Operate"),
        NavigationItem(name="Run Detail", route="/runs/detail", section="Operate"),
        NavigationItem(name="Triage", route="/triage", section="Operate"),
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
            filters=[FilterDefinition(name="Team", field="team", control="select", options=["all"])],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["export"]),
                SurfaceState(name="empty", allowed_actions=["export"]),
                SurfaceState(name="error", allowed_actions=["audit"]),
            ],
        ),
        ConsoleSurface(
            name="Queue",
            route="/queue",
            navigation_section="Operate",
            top_bar_actions=[
                GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
            ],
            filters=[FilterDefinition(name="Status", field="status", control="select", options=["all"])],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["audit"]),
                SurfaceState(name="empty", allowed_actions=["audit"]),
            ],
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
            filters=[FilterDefinition(name="Run", field="run_id", control="search")],
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
                GlobalAction(action_id="bulk-assign", label="Bulk Assign", placement="topbar", requires_selection=True),
            ],
            states=[
                SurfaceState(name="default"),
                SurfaceState(name="loading", allowed_actions=["export"]),
                SurfaceState(name="empty", allowed_actions=["audit"]),
                SurfaceState(name="error", allowed_actions=["audit"]),
            ],
        ),
    ],
)
draft = ConsoleInteractionDraft(
    name="BIG-4203 Four Critical Pages",
    version="v1",
    architecture=interaction_architecture,
    contracts=[
        SurfaceInteractionContract(
            surface_name="Overview",
            required_action_ids=["drill-down", "export", "audit"],
            permission_rule=SurfacePermissionRule(
                allowed_roles=["admin", "operator"],
                denied_roles=["viewer"],
                audit_event="overview.access.denied",
            ),
        ),
        SurfaceInteractionContract(
            surface_name="Queue",
            required_action_ids=["drill-down", "export", "audit"],
            requires_batch_actions=True,
            permission_rule=SurfacePermissionRule(
                allowed_roles=["admin", "operator"],
                denied_roles=["viewer"],
            ),
        ),
        SurfaceInteractionContract(
            surface_name="Run Detail",
            required_action_ids=["drill-down", "export", "audit"],
            permission_rule=SurfacePermissionRule(
                allowed_roles=["admin", "operator", "viewer"],
                denied_roles=[],
                audit_event="run-detail.access.denied",
            ),
        ),
        SurfaceInteractionContract(
            surface_name="Triage",
            required_action_ids=["drill-down", "export", "audit"],
            requires_filters=True,
            requires_batch_actions=True,
            permission_rule=SurfacePermissionRule(
                allowed_roles=["admin", "operator"],
                denied_roles=["viewer"],
                audit_event="triage.access.denied",
            ),
        ),
    ],
)
interaction_audit = ConsoleInteractionAuditor().audit(draft)

release_draft = build_big_4203_console_interaction_draft()
release_audit = ConsoleInteractionAuditor().audit(release_draft)
release_report = render_console_interaction_report(release_draft, release_audit)

broken_draft = build_big_4203_console_interaction_draft()
broken_draft.required_roles.append("finance-reviewer")
broken_draft.contracts[0].primary_persona = ""
broken_draft.contracts[0].linked_wireframe_id = ""
broken_draft.contracts[0].review_focus_areas = []
broken_draft.contracts[0].decision_prompts = []
broken_audit = ConsoleInteractionAuditor().audit(broken_draft)

print(json.dumps({
    "round_trip_ok": restored == architecture,
    "gap_audit": {
        "surfaces_missing_filters": gap_audit.surfaces_missing_filters,
        "surfaces_missing_actions": gap_audit.surfaces_missing_actions,
        "missing_capabilities": gap_audit.top_bar_audit.missing_capabilities,
        "release_ready": gap_audit.top_bar_audit.release_ready,
        "surfaces_missing_states": gap_audit.surfaces_missing_states,
        "states_missing_actions": gap_audit.states_missing_actions,
        "unresolved_state_actions": gap_audit.unresolved_state_actions,
        "orphan_navigation_routes": gap_audit.orphan_navigation_routes,
        "unnavigable_surfaces": gap_audit.unnavigable_surfaces,
        "readiness_score": gap_audit.readiness_score,
    },
    "ia_report_checks": {
        "title": "# Console Information Architecture Report" in ia_report,
        "top_bar": "- Name: BigClaw Global Header" in ia_report,
        "release_ready": "- Release Ready: True" in ia_report,
        "surface_line": "- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none" in ia_report,
        "gap_line": "- Undefined state actions: none" in ia_report,
    },
    "interaction_audit": {
        "surfaces_missing_filters": interaction_audit.surfaces_missing_filters,
        "surfaces_missing_actions": interaction_audit.surfaces_missing_actions,
        "surfaces_missing_batch_actions": interaction_audit.surfaces_missing_batch_actions,
        "surfaces_missing_states": interaction_audit.surfaces_missing_states,
        "permission_gaps": interaction_audit.permission_gaps,
        "readiness_score": interaction_audit.readiness_score,
        "release_ready": interaction_audit.release_ready,
    },
    "release_draft": {
        "required_roles": release_draft.required_roles,
        "release_ready": release_audit.release_ready,
        "uncovered_roles": release_audit.uncovered_roles,
        "has_roles_line": "- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator" in release_report,
        "has_persona_line": "persona=VP Eng wireframe=wf-overview" in release_report,
        "has_review_focus_line": "review_focus=metric hierarchy,drill-down posture,alert prioritization" in release_report,
    },
    "broken_draft": {
        "uncovered_roles": broken_audit.uncovered_roles,
        "surfaces_missing_primary_personas": broken_audit.surfaces_missing_primary_personas,
        "surfaces_missing_wireframe_links": broken_audit.surfaces_missing_wireframe_links,
        "surfaces_missing_review_focus": broken_audit.surfaces_missing_review_focus,
        "surfaces_missing_decision_prompts": broken_audit.surfaces_missing_decision_prompts,
        "release_ready": broken_audit.release_ready,
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write console ia contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run console ia contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		RoundTripOK bool `json:"round_trip_ok"`
		GapAudit    struct {
			SurfacesMissingFilters []string                       `json:"surfaces_missing_filters"`
			SurfacesMissingActions []string                       `json:"surfaces_missing_actions"`
			MissingCapabilities    []string                       `json:"missing_capabilities"`
			ReleaseReady           bool                           `json:"release_ready"`
			SurfacesMissingStates  map[string][]string            `json:"surfaces_missing_states"`
			StatesMissingActions   map[string][]string            `json:"states_missing_actions"`
			UnresolvedStateActions map[string]map[string][]string `json:"unresolved_state_actions"`
			OrphanNavigationRoutes []string                       `json:"orphan_navigation_routes"`
			UnnavigableSurfaces    []string                       `json:"unnavigable_surfaces"`
			ReadinessScore         float64                        `json:"readiness_score"`
		} `json:"gap_audit"`
		IAReportChecks struct {
			Title        bool `json:"title"`
			TopBar       bool `json:"top_bar"`
			ReleaseReady bool `json:"release_ready"`
			SurfaceLine  bool `json:"surface_line"`
			GapLine      bool `json:"gap_line"`
		} `json:"ia_report_checks"`
		InteractionAudit struct {
			SurfacesMissingFilters      []string            `json:"surfaces_missing_filters"`
			SurfacesMissingActions      map[string][]string `json:"surfaces_missing_actions"`
			SurfacesMissingBatchActions []string            `json:"surfaces_missing_batch_actions"`
			SurfacesMissingStates       map[string][]string `json:"surfaces_missing_states"`
			PermissionGaps              map[string][]string `json:"permission_gaps"`
			ReadinessScore              float64             `json:"readiness_score"`
			ReleaseReady                bool                `json:"release_ready"`
		} `json:"interaction_audit"`
		ReleaseDraft struct {
			RequiredRoles      []string `json:"required_roles"`
			ReleaseReady       bool     `json:"release_ready"`
			UncoveredRoles     []string `json:"uncovered_roles"`
			HasRolesLine       bool     `json:"has_roles_line"`
			HasPersonaLine     bool     `json:"has_persona_line"`
			HasReviewFocusLine bool     `json:"has_review_focus_line"`
		} `json:"release_draft"`
		BrokenDraft struct {
			UncoveredRoles                 []string `json:"uncovered_roles"`
			SurfacesMissingPrimaryPersonas []string `json:"surfaces_missing_primary_personas"`
			SurfacesMissingWireframeLinks  []string `json:"surfaces_missing_wireframe_links"`
			SurfacesMissingReviewFocus     []string `json:"surfaces_missing_review_focus"`
			SurfacesMissingDecisionPrompts []string `json:"surfaces_missing_decision_prompts"`
			ReleaseReady                   bool     `json:"release_ready"`
		} `json:"broken_draft"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode console ia contract output: %v\n%s", err, string(output))
	}

	if !decoded.RoundTripOK {
		t.Fatal("expected console ia round trip to remain stable")
	}
	if len(decoded.GapAudit.SurfacesMissingFilters) != 1 || decoded.GapAudit.SurfacesMissingFilters[0] != "Queue" ||
		len(decoded.GapAudit.SurfacesMissingActions) != 1 || decoded.GapAudit.SurfacesMissingActions[0] != "Queue" ||
		len(decoded.GapAudit.MissingCapabilities) != 5 || decoded.GapAudit.ReleaseReady ||
		len(decoded.GapAudit.SurfacesMissingStates["Queue"]) != 1 || decoded.GapAudit.SurfacesMissingStates["Queue"][0] != "error" ||
		len(decoded.GapAudit.StatesMissingActions["Queue"]) != 1 || decoded.GapAudit.StatesMissingActions["Queue"][0] != "loading" ||
		len(decoded.GapAudit.UnresolvedStateActions["Queue"]["empty"]) != 1 || decoded.GapAudit.UnresolvedStateActions["Queue"]["empty"][0] != "retry" ||
		len(decoded.GapAudit.OrphanNavigationRoutes) != 1 || decoded.GapAudit.OrphanNavigationRoutes[0] != "/ghost" ||
		len(decoded.GapAudit.UnnavigableSurfaces) != 1 || decoded.GapAudit.UnnavigableSurfaces[0] != "Queue" ||
		decoded.GapAudit.ReadinessScore != 0 {
		t.Fatalf("unexpected console ia gap audit payload: %+v", decoded.GapAudit)
	}
	if !decoded.IAReportChecks.Title || !decoded.IAReportChecks.TopBar || !decoded.IAReportChecks.ReleaseReady || !decoded.IAReportChecks.SurfaceLine || !decoded.IAReportChecks.GapLine {
		t.Fatalf("expected console ia report checks to pass, got %+v", decoded.IAReportChecks)
	}
	if len(decoded.InteractionAudit.SurfacesMissingFilters) != 1 || decoded.InteractionAudit.SurfacesMissingFilters[0] != "Triage" ||
		len(decoded.InteractionAudit.SurfacesMissingActions["Queue"]) != 1 || decoded.InteractionAudit.SurfacesMissingActions["Queue"][0] != "export" ||
		len(decoded.InteractionAudit.SurfacesMissingBatchActions) != 1 || decoded.InteractionAudit.SurfacesMissingBatchActions[0] != "Queue" ||
		len(decoded.InteractionAudit.SurfacesMissingStates["Queue"]) != 1 || decoded.InteractionAudit.SurfacesMissingStates["Queue"][0] != "error" ||
		len(decoded.InteractionAudit.PermissionGaps["Queue"]) != 1 || decoded.InteractionAudit.PermissionGaps["Queue"][0] != "audit-event" ||
		len(decoded.InteractionAudit.PermissionGaps["Run Detail"]) != 1 || decoded.InteractionAudit.PermissionGaps["Run Detail"][0] != "denied-roles" ||
		decoded.InteractionAudit.ReadinessScore != 0 || decoded.InteractionAudit.ReleaseReady {
		t.Fatalf("unexpected console interaction audit payload: %+v", decoded.InteractionAudit)
	}
	if len(decoded.ReleaseDraft.RequiredRoles) != 4 || !decoded.ReleaseDraft.ReleaseReady || len(decoded.ReleaseDraft.UncoveredRoles) != 0 || !decoded.ReleaseDraft.HasRolesLine || !decoded.ReleaseDraft.HasPersonaLine || !decoded.ReleaseDraft.HasReviewFocusLine {
		t.Fatalf("unexpected release draft payload: %+v", decoded.ReleaseDraft)
	}
	if len(decoded.BrokenDraft.UncoveredRoles) != 1 || decoded.BrokenDraft.UncoveredRoles[0] != "finance-reviewer" ||
		len(decoded.BrokenDraft.SurfacesMissingPrimaryPersonas) != 1 || decoded.BrokenDraft.SurfacesMissingPrimaryPersonas[0] != "Overview" ||
		len(decoded.BrokenDraft.SurfacesMissingWireframeLinks) != 1 || decoded.BrokenDraft.SurfacesMissingWireframeLinks[0] != "Overview" ||
		len(decoded.BrokenDraft.SurfacesMissingReviewFocus) != 1 || decoded.BrokenDraft.SurfacesMissingReviewFocus[0] != "Overview" ||
		len(decoded.BrokenDraft.SurfacesMissingDecisionPrompts) != 1 || decoded.BrokenDraft.SurfacesMissingDecisionPrompts[0] != "Overview" ||
		decoded.BrokenDraft.ReleaseReady {
		t.Fatalf("unexpected broken draft payload: %+v", decoded.BrokenDraft)
	}
}

func TestLane8PythonDesignSystemContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "design_system_contract.py")
	script := `import json
import sys

from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.design_system import (
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    DesignSystem,
    DesignToken,
    InformationArchitecture,
    NavigationNode,
    NavigationRoute,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
)

system = DesignSystem(
    name="BigClaw Console UI",
    version="v2",
    tokens=[
        DesignToken(name="color.action.primary", category="color", value="#4455ff"),
        DesignToken(name="spacing.control.md", category="spacing", value="12px"),
        DesignToken(name="radius.md", category="radius", value="8px"),
    ],
    components=[
        ComponentSpec(
            name="Button",
            readiness="stable",
            documentation_complete=True,
            accessibility_requirements=["focus-visible", "keyboard-activation"],
            variants=[
                ComponentVariant(
                    name="primary",
                    tokens=["color.action.primary", "spacing.control.md"],
                    states=["default", "hover", "disabled"],
                )
            ],
        ),
        ComponentSpec(
            name="CommandBar",
            readiness="beta",
            documentation_complete=False,
            variants=[
                ComponentVariant(
                    name="global",
                    tokens=["spacing.control.md"],
                    states=["default", "hover"],
                )
            ],
        ),
    ],
)
restored = DesignSystem.from_dict(system.to_dict())
audit = ComponentLibrary().audit(system)
report = render_design_system_report(
    DesignSystem(
        name="BigClaw Console UI",
        version="v2",
        tokens=[
            DesignToken(name="color.action.primary", category="color", value="#4455ff"),
            DesignToken(name="spacing.control.md", category="spacing", value="12px"),
        ],
        components=[
            ComponentSpec(
                name="Button",
                readiness="stable",
                documentation_complete=True,
                accessibility_requirements=["focus-visible"],
                variants=[
                    ComponentVariant(
                        name="primary",
                        tokens=["color.action.primary", "spacing.control.md"],
                        states=["default", "hover", "disabled"],
                    )
                ],
            )
        ],
    ),
    ComponentLibrary().audit(
        DesignSystem(
            name="BigClaw Console UI",
            version="v2",
            tokens=[
                DesignToken(name="color.action.primary", category="color", value="#4455ff"),
                DesignToken(name="spacing.control.md", category="spacing", value="12px"),
            ],
            components=[
                ComponentSpec(
                    name="Button",
                    readiness="stable",
                    documentation_complete=True,
                    accessibility_requirements=["focus-visible"],
                    variants=[
                        ComponentVariant(
                            name="primary",
                            tokens=["color.action.primary", "spacing.control.md"],
                            states=["default", "hover", "disabled"],
                        )
                    ],
                )
            ],
        )
    ),
)

top_bar = ConsoleTopBar(
    name="BigClaw Global Header",
    search_placeholder="Search runs, issues, commands",
    environment_options=["Production", "Staging"],
    time_range_options=["24h", "7d", "30d"],
    alert_channels=["approvals", "sla"],
    documentation_complete=True,
    accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
    command_entry=ConsoleCommandEntry(
        trigger_label="Command Menu",
        placeholder="Type a command or jump to a run",
        shortcut="Cmd+K / Ctrl+K",
        commands=[
            CommandAction(id="search-runs", title="Search runs", section="Navigate", shortcut="/"),
            CommandAction(id="open-alerts", title="Open alerts", section="Monitor"),
        ],
    ),
)
top_bar_audit = ConsoleChromeLibrary().audit_top_bar(top_bar)
top_bar_report = render_console_top_bar_report(top_bar, top_bar_audit)

bad_top_bar = ConsoleTopBar(
    name="Incomplete Header",
    search_placeholder="",
    environment_options=["Production"],
    time_range_options=["24h"],
    command_entry=ConsoleCommandEntry(trigger_label="", placeholder="", shortcut="Cmd+K"),
    documentation_complete=False,
    accessibility_requirements=["focus-visible"],
)
bad_top_bar_audit = ConsoleChromeLibrary().audit_top_bar(bad_top_bar)

architecture = InformationArchitecture(
    global_nav=[
        NavigationNode(
            node_id="workbench",
            title="Workbench",
            segment="workbench",
            screen_id="workbench-home",
            children=[
                NavigationNode(
                    node_id="workbench-runs",
                    title="Runs",
                    segment="runs",
                    screen_id="run-index",
                ),
                NavigationNode(
                    node_id="workbench-replays",
                    title="Replays",
                    segment="replays",
                    screen_id="replay-index",
                ),
            ],
        )
    ],
    routes=[
        NavigationRoute(
            path="/workbench/runs",
            screen_id="run-index",
            title="Runs",
            nav_node_id="workbench-runs",
        ),
        NavigationRoute(
            path="/workbench/runs",
            screen_id="run-index-v2",
            title="Runs V2",
            nav_node_id="workbench-runs",
        ),
        NavigationRoute(
            path="/settings",
            screen_id="settings-home",
            title="Settings",
            nav_node_id="settings",
        ),
    ],
)
ia_restored = InformationArchitecture.from_dict(architecture.to_dict())
ia_audit = architecture.audit()
ia_report = render_information_architecture_report(architecture, ia_audit)

print(json.dumps({
    "round_trip_ok": restored == system,
    "audit": {
        "release_ready_components": audit.release_ready_components,
        "components_missing_docs": audit.components_missing_docs,
        "components_missing_accessibility": audit.components_missing_accessibility,
        "components_missing_states": audit.components_missing_states,
        "token_orphans": audit.token_orphans,
        "readiness_score": audit.readiness_score,
    },
    "report_checks": {
        "title": "# Design System Report" in report,
        "release_ready_components": "- Release Ready Components: 1" in report,
        "button_line": "- Button: readiness=stable docs=True a11y=True states=default, hover, disabled missing_states=none undefined_tokens=none" in report,
        "orphans": "- Orphan tokens: none" in report,
    },
    "top_bar": {
        "release_ready": top_bar_audit.release_ready,
        "command_count": top_bar_audit.command_count,
        "command_shortcut_supported": top_bar_audit.command_shortcut_supported,
        "report_has_shortcut": "- Command Shortcut: Cmd+K / Ctrl+K" in top_bar_report,
        "report_has_search": "- search-runs: Search runs [Navigate] shortcut=/" in top_bar_report,
    },
    "bad_top_bar": {
        "missing_capabilities": bad_top_bar_audit.missing_capabilities,
        "release_ready": bad_top_bar_audit.release_ready,
    },
    "information_architecture": {
        "round_trip_ok": ia_restored == architecture,
        "duplicate_routes": ia_audit.duplicate_routes,
        "missing_route_nodes": ia_audit.missing_route_nodes,
        "secondary_nav_gaps": ia_audit.secondary_nav_gaps,
        "orphan_routes": ia_audit.orphan_routes,
        "report_checks": {
            "title": "# Information Architecture Report" in ia_report,
            "healthy": "- Healthy: False" in ia_report,
            "duplicate_routes": "- Duplicate routes: /workbench/runs" in ia_report,
            "orphan_routes": "- Orphan routes: /settings" in ia_report,
        },
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write design system contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run design system contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		RoundTripOK bool `json:"round_trip_ok"`
		Audit       struct {
			ReleaseReadyComponents         []string `json:"release_ready_components"`
			ComponentsMissingDocs          []string `json:"components_missing_docs"`
			ComponentsMissingAccessibility []string `json:"components_missing_accessibility"`
			ComponentsMissingStates        []string `json:"components_missing_states"`
			TokenOrphans                   []string `json:"token_orphans"`
			ReadinessScore                 float64  `json:"readiness_score"`
		} `json:"audit"`
		ReportChecks struct {
			Title                  bool `json:"title"`
			ReleaseReadyComponents bool `json:"release_ready_components"`
			ButtonLine             bool `json:"button_line"`
			Orphans                bool `json:"orphans"`
		} `json:"report_checks"`
		TopBar struct {
			ReleaseReady             bool `json:"release_ready"`
			CommandCount             int  `json:"command_count"`
			CommandShortcutSupported bool `json:"command_shortcut_supported"`
			ReportHasShortcut        bool `json:"report_has_shortcut"`
			ReportHasSearch          bool `json:"report_has_search"`
		} `json:"top_bar"`
		BadTopBar struct {
			MissingCapabilities []string `json:"missing_capabilities"`
			ReleaseReady        bool     `json:"release_ready"`
		} `json:"bad_top_bar"`
		InformationArchitecture struct {
			RoundTripOK       bool                `json:"round_trip_ok"`
			DuplicateRoutes   []string            `json:"duplicate_routes"`
			MissingRouteNodes map[string]string   `json:"missing_route_nodes"`
			SecondaryNavGaps  map[string][]string `json:"secondary_nav_gaps"`
			OrphanRoutes      []string            `json:"orphan_routes"`
			ReportChecks      struct {
				Title           bool `json:"title"`
				Healthy         bool `json:"healthy"`
				DuplicateRoutes bool `json:"duplicate_routes"`
				OrphanRoutes    bool `json:"orphan_routes"`
			} `json:"report_checks"`
		} `json:"information_architecture"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode design system contract output: %v\n%s", err, string(output))
	}

	if !decoded.RoundTripOK {
		t.Fatal("expected design system round trip to remain stable")
	}
	if len(decoded.Audit.ReleaseReadyComponents) != 1 || decoded.Audit.ReleaseReadyComponents[0] != "Button" ||
		len(decoded.Audit.ComponentsMissingDocs) != 1 || decoded.Audit.ComponentsMissingDocs[0] != "CommandBar" ||
		len(decoded.Audit.ComponentsMissingAccessibility) != 1 || decoded.Audit.ComponentsMissingAccessibility[0] != "CommandBar" ||
		len(decoded.Audit.ComponentsMissingStates) != 1 || decoded.Audit.ComponentsMissingStates[0] != "CommandBar" ||
		len(decoded.Audit.TokenOrphans) != 1 || decoded.Audit.TokenOrphans[0] != "radius.md" ||
		decoded.Audit.ReadinessScore != 35 {
		t.Fatalf("unexpected design system audit payload: %+v", decoded.Audit)
	}
	if !decoded.ReportChecks.Title || !decoded.ReportChecks.ReleaseReadyComponents || !decoded.ReportChecks.ButtonLine || !decoded.ReportChecks.Orphans {
		t.Fatalf("expected design system report checks to pass, got %+v", decoded.ReportChecks)
	}
	if !decoded.TopBar.ReleaseReady || decoded.TopBar.CommandCount != 2 || !decoded.TopBar.CommandShortcutSupported || !decoded.TopBar.ReportHasShortcut || !decoded.TopBar.ReportHasSearch {
		t.Fatalf("unexpected top bar payload: %+v", decoded.TopBar)
	}
	if len(decoded.BadTopBar.MissingCapabilities) != 5 || decoded.BadTopBar.ReleaseReady {
		t.Fatalf("unexpected bad top bar payload: %+v", decoded.BadTopBar)
	}
	if !decoded.InformationArchitecture.RoundTripOK ||
		len(decoded.InformationArchitecture.DuplicateRoutes) != 1 || decoded.InformationArchitecture.DuplicateRoutes[0] != "/workbench/runs" ||
		decoded.InformationArchitecture.MissingRouteNodes["workbench"] != "/workbench" ||
		len(decoded.InformationArchitecture.SecondaryNavGaps["Workbench"]) != 1 || decoded.InformationArchitecture.SecondaryNavGaps["Workbench"][0] != "/workbench" ||
		len(decoded.InformationArchitecture.OrphanRoutes) != 1 || decoded.InformationArchitecture.OrphanRoutes[0] != "/settings" ||
		!decoded.InformationArchitecture.ReportChecks.Title || !decoded.InformationArchitecture.ReportChecks.Healthy || !decoded.InformationArchitecture.ReportChecks.DuplicateRoutes || !decoded.InformationArchitecture.ReportChecks.OrphanRoutes {
		t.Fatalf("unexpected information architecture payload: %+v", decoded.InformationArchitecture)
	}
}
