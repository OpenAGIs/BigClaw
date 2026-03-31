from pathlib import Path

from bigclaw.console_ia import (
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ConsoleInteractionAudit,
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
from bigclaw.design_system import (
    AuditRequirement,
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DataAccuracyCheck,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationNode,
    NavigationRoute,
    PerformanceBudget,
    RolePermissionScenario,
    UIAcceptanceAudit,
    UIAcceptanceLibrary,
    UIAcceptanceSuite,
    UsabilityJourney,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
    render_ui_acceptance_report,
)
from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewBlocker,
    ReviewBlockerEvent,
    ReviewDecision,
    ReviewObjective,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_audit_density_board,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_checklist_traceability_board,
    render_ui_review_decision_followup_tracker,
    render_ui_review_decision_log,
    render_ui_review_escalation_dashboard,
    render_ui_review_escalation_handoff_ledger,
    render_ui_review_exception_log,
    render_ui_review_exception_matrix,
    render_ui_review_freeze_approval_trail,
    render_ui_review_freeze_exception_board,
    render_ui_review_freeze_renewal_tracker,
    render_ui_review_handoff_ack_ledger,
    render_ui_review_interaction_coverage_board,
    render_ui_review_objective_coverage_board,
    render_ui_review_open_question_tracker,
    render_ui_review_owner_escalation_digest,
    render_ui_review_owner_review_queue,
    render_ui_review_owner_workload_board,
    render_ui_review_pack_html,
    render_ui_review_pack_report,
    render_ui_review_persona_readiness_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_review_summary_board,
    render_ui_review_role_coverage_board,
    render_ui_review_role_matrix,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_log,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_wireframe_readiness_board,
    write_ui_review_pack_bundle,
)


def test_component_release_ready_requires_docs_accessibility_and_states():
    component = ComponentSpec(
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
    )

    assert component.release_ready is True
    assert component.token_names == ["color.action.primary", "spacing.control.md"]
    assert component.missing_required_states == []


def test_design_system_round_trip_preserves_manifest_shape():
    system = DesignSystem(
        name="BigClaw Console UI",
        version="v2",
        tokens=[
            DesignToken(
                name="color.action.primary",
                category="color",
                value="#4455ff",
                semantic_role="action-primary",
            )
        ],
        components=[
            ComponentSpec(
                name="Button",
                readiness="stable",
                slots=["icon", "label"],
                documentation_complete=True,
                accessibility_requirements=["focus-visible"],
                variants=[
                    ComponentVariant(
                        name="primary",
                        tokens=["color.action.primary"],
                        states=["default", "hover", "disabled"],
                        usage_notes="Use for primary CTA.",
                    )
                ],
            )
        ],
    )

    restored = DesignSystem.from_dict(system.to_dict())

    assert restored == system


def test_design_system_audit_surfaces_release_gaps_and_orphan_tokens():
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

    audit = ComponentLibrary().audit(system)

    assert audit.release_ready_components == ["Button"]
    assert audit.components_missing_docs == ["CommandBar"]
    assert audit.components_missing_accessibility == ["CommandBar"]
    assert audit.components_missing_states == ["CommandBar"]
    assert audit.undefined_token_refs == {}
    assert audit.token_orphans == ["radius.md"]
    assert audit.readiness_score == 35.0


def test_design_system_audit_flags_undefined_token_references():
    system = DesignSystem(
        name="BigClaw Console UI",
        version="v2",
        tokens=[DesignToken(name="spacing.control.md", category="spacing", value="12px")],
        components=[
            ComponentSpec(
                name="SideNav",
                readiness="stable",
                documentation_complete=True,
                accessibility_requirements=["focus-visible"],
                variants=[
                    ComponentVariant(
                        name="default",
                        tokens=["spacing.control.md", "color.surface.nav"],
                        states=["default", "hover", "disabled"],
                    )
                ],
            )
        ],
    )

    audit = ComponentLibrary().audit(system)

    assert audit.release_ready_components == []
    assert audit.undefined_token_refs == {"SideNav": ["color.surface.nav"]}



def test_design_system_audit_round_trip_preserves_governance_findings():
    audit = DesignSystemAudit(
        system_name="BigClaw Console UI",
        version="v2",
        token_counts={"color": 3, "spacing": 2},
        component_count=2,
        release_ready_components=["Button"],
        components_missing_docs=["CommandBar"],
        components_missing_accessibility=["CommandBar"],
        components_missing_states=["CommandBar"],
        undefined_token_refs={"SideNav": ["color.surface.nav"]},
        token_orphans=["radius.md"],
    )

    restored = DesignSystemAudit.from_dict(audit.to_dict())

    assert restored == audit



def test_render_design_system_report_summarizes_inventory_and_gaps():
    system = DesignSystem(
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
    audit = ComponentLibrary().audit(system)

    report = render_design_system_report(system, audit)

    assert "# Design System Report" in report
    assert "- Release Ready Components: 1" in report
    assert "- color: 1" in report
    assert "- Button: readiness=stable docs=True a11y=True states=default, hover, disabled missing_states=none undefined_tokens=none" in report
    assert "- Missing interaction states: none" in report
    assert "- Undefined token refs: none" in report
    assert "- Orphan tokens: none" in report

def test_console_top_bar_round_trip_preserves_command_entry_manifest():
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
            recent_queries_enabled=True,
            commands=[
                CommandAction(id="search-runs", title="Search runs", section="Navigate", shortcut="/"),
                CommandAction(id="open-alerts", title="Open alerts", section="Monitor"),
            ],
        ),
    )

    restored = ConsoleTopBar.from_dict(top_bar.to_dict())

    assert restored == top_bar


def test_console_top_bar_audit_checks_ticket_capabilities_and_shortcuts():
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
                CommandAction(id="search-runs", title="Search runs", section="Navigate"),
                CommandAction(id="switch-env", title="Switch environment", section="Context"),
            ],
        ),
    )

    audit = ConsoleChromeLibrary().audit_top_bar(top_bar)

    assert audit == ConsoleTopBarAudit(
        name="BigClaw Global Header",
        missing_capabilities=[],
        documentation_complete=True,
        accessibility_complete=True,
        command_shortcut_supported=True,
        command_count=2,
    )
    assert audit.release_ready is True


def test_console_top_bar_audit_flags_missing_global_entry_capabilities():
    top_bar = ConsoleTopBar(
        name="Incomplete Header",
        search_placeholder="",
        environment_options=["Production"],
        time_range_options=["24h"],
        command_entry=ConsoleCommandEntry(
            trigger_label="",
            placeholder="",
            shortcut="Cmd+K",
        ),
        documentation_complete=False,
        accessibility_requirements=["focus-visible"],
    )

    audit = ConsoleChromeLibrary().audit_top_bar(top_bar)

    assert audit.missing_capabilities == [
        "global-search",
        "time-range-switch",
        "environment-switch",
        "alert-entry",
        "command-shell",
    ]
    assert audit.documentation_complete is False
    assert audit.accessibility_complete is False
    assert audit.command_shortcut_supported is False
    assert audit.release_ready is False


def test_render_console_top_bar_report_summarizes_global_header_and_shell():
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
    audit = ConsoleChromeLibrary().audit_top_bar(top_bar)

    report = render_console_top_bar_report(top_bar, audit)

    assert "# Console Top Bar Report" in report
    assert "- Command Shortcut: Cmd+K / Ctrl+K" in report
    assert "- Release Ready: True" in report
    assert "- search-runs: Search runs [Navigate] shortcut=/" in report
    assert "- Missing capabilities: none" in report
    assert "- Cmd/Ctrl+K supported: True" in report


def test_information_architecture_round_trip_and_route_resolution():
    architecture = InformationArchitecture(
        global_nav=[
            NavigationNode(
                node_id="ops",
                title="Operations",
                segment="operations",
                screen_id="operations-overview",
                children=[
                    NavigationNode(
                        node_id="ops-queue",
                        title="Queue Control",
                        segment="queue",
                        screen_id="queue-control",
                    ),
                    NavigationNode(
                        node_id="ops-triage",
                        title="Triage Center",
                        segment="triage",
                        screen_id="triage-center",
                    ),
                ],
            )
        ],
        routes=[
            NavigationRoute(
                path="/operations",
                screen_id="operations-overview",
                title="Operations",
                nav_node_id="ops",
            ),
            NavigationRoute(
                path="/operations/queue",
                screen_id="queue-control",
                title="Queue Control",
                nav_node_id="ops-queue",
            ),
            NavigationRoute(
                path="/operations/triage",
                screen_id="triage-center",
                title="Triage Center",
                nav_node_id="ops-triage",
            ),
        ],
    )

    restored = InformationArchitecture.from_dict(architecture.to_dict())

    assert restored == architecture
    assert [entry.path for entry in architecture.navigation_entries] == [
        "/operations",
        "/operations/queue",
        "/operations/triage",
    ]
    assert architecture.resolve_route("operations/queue") == NavigationRoute(
        path="/operations/queue",
        screen_id="queue-control",
        title="Queue Control",
        nav_node_id="ops-queue",
    )


def test_information_architecture_audit_flags_duplicates_secondary_gaps_and_orphans():
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

    audit = architecture.audit()

    assert audit.healthy is False
    assert audit.duplicate_routes == ["/workbench/runs"]
    assert audit.missing_route_nodes == {
        "workbench": "/workbench",
        "workbench-replays": "/workbench/replays",
    }
    assert audit.secondary_nav_gaps == {"Workbench": ["/workbench"]}
    assert audit.orphan_routes == ["/settings"]


def test_information_architecture_audit_round_trip_and_report():
    audit = InformationArchitectureAudit(
        total_navigation_nodes=3,
        total_routes=2,
        duplicate_routes=["/workbench/runs"],
        missing_route_nodes={"workbench": "/workbench"},
        secondary_nav_gaps={"Workbench": ["/workbench"]},
        orphan_routes=["/settings"],
    )

    restored = InformationArchitectureAudit.from_dict(audit.to_dict())

    assert restored == audit

    architecture = InformationArchitecture(
        global_nav=[
            NavigationNode(node_id="workbench", title="Workbench", segment="workbench", screen_id="workbench-home")
        ],
        routes=[
            NavigationRoute(
                path="/settings",
                screen_id="settings-home",
                title="Settings",
                nav_node_id="settings",
            )
        ],
    )

    report = render_information_architecture_report(architecture, audit)

    assert "# Information Architecture Report" in report
    assert "- Healthy: False" in report
    assert "- Workbench (/workbench) screen=workbench-home" in report
    assert "- /settings: screen=settings-home title=Settings nav_node=settings" in report
    assert "- Duplicate routes: /workbench/runs" in report
    assert "- Missing route nodes: workbench=/workbench" in report
    assert "- Secondary nav gaps: Workbench=/workbench" in report
    assert "- Orphan routes: /settings" in report


def test_ui_acceptance_suite_round_trip_preserves_acceptance_manifest():
    suite = UIAcceptanceSuite(
        name="BIG-1701 v3.0 UI Acceptance",
        version="v3.0",
        role_permissions=[
            RolePermissionScenario(
                screen_id="run-detail",
                allowed_roles=["admin", "operator"],
                denied_roles=["viewer"],
                audit_event="ui.access.denied",
            )
        ],
        data_accuracy_checks=[
            DataAccuracyCheck(
                screen_id="sla-dashboard",
                metric_id="breach-count",
                source_of_truth="warehouse.sla_daily",
                rendered_value="12",
                tolerance=0.0,
                observed_delta=0.0,
                freshness_slo_seconds=300,
                observed_freshness_seconds=120,
            )
        ],
        performance_budgets=[
            PerformanceBudget(
                surface_id="triage-center",
                interaction="initial-load",
                target_p95_ms=1200,
                observed_p95_ms=980,
                target_tti_ms=1800,
                observed_tti_ms=1400,
            )
        ],
        usability_journeys=[
            UsabilityJourney(
                journey_id="approve-high-risk-run",
                personas=["operator"],
                critical_steps=["open queue", "inspect run", "approve"],
                expected_max_steps=4,
                observed_steps=3,
                keyboard_accessible=True,
                empty_state_guidance=True,
                recovery_support=True,
            )
        ],
        audit_requirements=[
            AuditRequirement(
                event_type="run.approval.changed",
                required_fields=["run_id", "actor_role", "decision"],
                emitted_fields=["run_id", "actor_role", "decision"],
                retention_days=90,
                observed_retention_days=120,
            )
        ],
        documentation_complete=True,
    )

    restored = UIAcceptanceSuite.from_dict(suite.to_dict())

    assert restored == suite


def test_ui_acceptance_audit_detects_permission_accuracy_perf_usability_and_audit_gaps():
    suite = UIAcceptanceSuite(
        name="BIG-1701 v3.0 UI Acceptance",
        version="v3.0",
        role_permissions=[
            RolePermissionScenario(
                screen_id="operations-overview",
                allowed_roles=["admin"],
                denied_roles=[],
                audit_event="",
            )
        ],
        data_accuracy_checks=[
            DataAccuracyCheck(
                screen_id="sla-dashboard",
                metric_id="breach-count",
                source_of_truth="warehouse.sla_daily",
                rendered_value="12",
                tolerance=0.0,
                observed_delta=2.0,
                freshness_slo_seconds=300,
                observed_freshness_seconds=901,
            )
        ],
        performance_budgets=[
            PerformanceBudget(
                surface_id="triage-center",
                interaction="initial-load",
                target_p95_ms=1200,
                observed_p95_ms=1480,
                target_tti_ms=1800,
                observed_tti_ms=2400,
            )
        ],
        usability_journeys=[
            UsabilityJourney(
                journey_id="reassign-alert",
                personas=["operator"],
                critical_steps=["open alert", "assign owner", "save"],
                expected_max_steps=3,
                observed_steps=5,
                keyboard_accessible=False,
                empty_state_guidance=True,
                recovery_support=False,
            )
        ],
        audit_requirements=[
            AuditRequirement(
                event_type="permission.override.used",
                required_fields=["actor_role", "screen_id", "reason_code"],
                emitted_fields=["actor_role", "screen_id"],
                retention_days=180,
                observed_retention_days=30,
            )
        ],
        documentation_complete=False,
    )

    audit = UIAcceptanceLibrary().audit(suite)

    assert audit == UIAcceptanceAudit(
        name="BIG-1701 v3.0 UI Acceptance",
        version="v3.0",
        permission_gaps=["operations-overview: missing=denied-roles, audit-event"],
        failing_data_checks=["sla-dashboard.breach-count: delta=2.0 freshness=901s"],
        failing_performance_budgets=["triage-center.initial-load: p95=1480ms tti=2400ms"],
        failing_usability_journeys=["reassign-alert: steps=5/3"],
        incomplete_audit_trails=["permission.override.used: missing_fields=reason_code retention=30/180d"],
        documentation_complete=False,
    )
    assert audit.readiness_score == 0.0
    assert audit.release_ready is False


def test_render_ui_acceptance_report_summarizes_release_readiness():
    suite = UIAcceptanceSuite(
        name="BIG-1701 v3.0 UI Acceptance",
        version="v3.0",
        role_permissions=[
            RolePermissionScenario(
                screen_id="run-detail",
                allowed_roles=["admin", "operator"],
                denied_roles=["viewer"],
                audit_event="ui.access.denied",
            )
        ],
        data_accuracy_checks=[
            DataAccuracyCheck(
                screen_id="sla-dashboard",
                metric_id="breach-count",
                source_of_truth="warehouse.sla_daily",
                rendered_value="12",
                tolerance=0.0,
                observed_delta=0.0,
                freshness_slo_seconds=300,
                observed_freshness_seconds=120,
            )
        ],
        performance_budgets=[
            PerformanceBudget(
                surface_id="triage-center",
                interaction="initial-load",
                target_p95_ms=1200,
                observed_p95_ms=980,
                target_tti_ms=1800,
                observed_tti_ms=1400,
            )
        ],
        usability_journeys=[
            UsabilityJourney(
                journey_id="approve-high-risk-run",
                personas=["operator"],
                critical_steps=["open queue", "inspect run", "approve"],
                expected_max_steps=4,
                observed_steps=3,
                keyboard_accessible=True,
                empty_state_guidance=True,
                recovery_support=True,
            )
        ],
        audit_requirements=[
            AuditRequirement(
                event_type="run.approval.changed",
                required_fields=["run_id", "actor_role", "decision"],
                emitted_fields=["run_id", "actor_role", "decision"],
                retention_days=90,
                observed_retention_days=120,
            )
        ],
        documentation_complete=True,
    )

    audit = UIAcceptanceLibrary().audit(suite)
    report = render_ui_acceptance_report(suite, audit)

    assert "# UI Acceptance Report" in report
    assert "- Readiness Score: 100.0" in report
    assert "- Release Ready: True" in report
    assert "- Role/Permission run-detail: allow=admin, operator deny=viewer audit_event=ui.access.denied" in report
    assert "- Data Accuracy sla-dashboard.breach-count: delta=0.0 tolerance=0.0 freshness=120/300s" in report
    assert "- Performance triage-center.initial-load: p95=980/1200ms tti=1400/1800ms" in report
    assert "- Usability approve-high-risk-run: steps=3/4 keyboard=True empty_state=True recovery=True" in report
    assert "- Audit completeness gaps: none" in report


def test_console_ia_round_trip_preserves_manifest_shape() -> None:
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

    assert restored == architecture


def test_console_ia_audit_surfaces_global_interaction_gaps() -> None:
    architecture = ConsoleIA(
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

    audit = ConsoleIAAuditor().audit(architecture)

    assert audit.surfaces_missing_filters == ["Queue"]
    assert audit.surfaces_missing_actions == ["Queue"]
    assert audit.top_bar_audit.missing_capabilities == [
        "global-search",
        "time-range-switch",
        "environment-switch",
        "alert-entry",
        "command-shell",
    ]
    assert audit.top_bar_audit.release_ready is False
    assert audit.surfaces_missing_states == {"Queue": ["error"]}
    assert audit.states_missing_actions == {"Queue": ["loading"]}
    assert audit.unresolved_state_actions == {"Queue": {"empty": ["retry"]}}
    assert audit.orphan_navigation_routes == ["/ghost"]
    assert audit.unnavigable_surfaces == ["Queue"]
    assert audit.readiness_score == 0.0


def test_console_ia_audit_round_trip_preserves_findings() -> None:
    audit = ConsoleIAAudit(
        system_name="BigClaw Console IA",
        version="v3",
        surface_count=2,
        navigation_count=1,
        top_bar_audit=ConsoleIAAuditor().audit(
            ConsoleIA(
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
            )
        ).top_bar_audit,
        surfaces_missing_filters=["Queue"],
        surfaces_missing_actions=["Queue"],
        surfaces_missing_states={"Queue": ["error"]},
        states_missing_actions={"Queue": ["loading"]},
        unresolved_state_actions={"Queue": {"empty": ["retry"]}},
        orphan_navigation_routes=["/ghost"],
        unnavigable_surfaces=["Queue"],
    )

    restored = ConsoleIAAudit.from_dict(audit.to_dict())

    assert restored == audit


def test_render_console_ia_report_summarizes_surface_coverage() -> None:
    architecture = ConsoleIA(
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

    audit = ConsoleIAAuditor().audit(architecture)
    report = render_console_ia_report(architecture, audit)

    assert "# Console Information Architecture Report" in report
    assert "- Name: BigClaw Global Header" in report
    assert "- Release Ready: True" in report
    assert "- Navigation Items: 1" in report
    assert "- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none" in report
    assert "- Surfaces missing filters: none" in report
    assert "- Undefined state actions: none" in report


def test_console_interaction_draft_round_trip_preserves_four_page_manifest() -> None:
    draft = ConsoleInteractionDraft(
        name="BIG-4203 Four Critical Pages",
        version="v1",
        architecture=ConsoleIA(
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
                ConsoleSurface(name="Overview", route="/overview", navigation_section="Operate"),
                ConsoleSurface(name="Queue", route="/queue", navigation_section="Operate"),
                ConsoleSurface(name="Run Detail", route="/runs/detail", navigation_section="Operate"),
                ConsoleSurface(name="Triage", route="/triage", navigation_section="Operate"),
            ],
        ),
        contracts=[
            SurfaceInteractionContract(surface_name="Overview"),
            SurfaceInteractionContract(surface_name="Queue", requires_batch_actions=True),
            SurfaceInteractionContract(surface_name="Run Detail"),
            SurfaceInteractionContract(surface_name="Triage"),
        ],
    )

    restored = ConsoleInteractionDraft.from_dict(draft.to_dict())

    assert restored == draft


def test_console_interaction_audit_surfaces_missing_actions_permissions_and_batch_ops() -> None:
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
                    GlobalAction(
                        action_id="bulk-assign",
                        label="Bulk Assign",
                        placement="topbar",
                        requires_selection=True,
                    ),
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
        architecture=architecture,
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

    audit = ConsoleInteractionAuditor().audit(draft)

    assert audit == ConsoleInteractionAudit(
        name="BIG-4203 Four Critical Pages",
        version="v1",
        contract_count=4,
        missing_surfaces=[],
        surfaces_missing_filters=["Triage"],
        surfaces_missing_actions={"Queue": ["export"]},
        surfaces_missing_batch_actions=["Queue"],
        surfaces_missing_states={"Queue": ["error"]},
        permission_gaps={
            "Queue": ["audit-event"],
            "Run Detail": ["denied-roles"],
        },
    )
    assert audit.readiness_score == 0.0
    assert audit.release_ready is False


def test_render_console_interaction_report_summarizes_critical_page_contracts() -> None:
    draft = ConsoleInteractionDraft(
        name="BIG-4203 Four Critical Pages",
        version="v1",
        architecture=ConsoleIA(
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
                    filters=[FilterDefinition(name="Status", field="status", control="select", options=["all"])],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
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
                        GlobalAction(
                            action_id="bulk-assign",
                            label="Bulk Assign",
                            placement="topbar",
                            requires_selection=True,
                        ),
                    ],
                    filters=[FilterDefinition(name="Severity", field="severity", control="select", options=["all"])],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                ),
            ],
        ),
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
                    audit_event="queue.access.denied",
                ),
            ),
            SurfaceInteractionContract(
                surface_name="Run Detail",
                required_action_ids=["drill-down", "export", "audit"],
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["admin", "operator", "viewer"],
                    denied_roles=["guest"],
                    audit_event="run-detail.access.denied",
                ),
            ),
            SurfaceInteractionContract(
                surface_name="Triage",
                required_action_ids=["drill-down", "export", "audit"],
                requires_batch_actions=True,
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["admin", "operator"],
                    denied_roles=["viewer"],
                    audit_event="triage.access.denied",
                ),
            ),
        ],
    )

    audit = ConsoleInteractionAuditor().audit(draft)
    report = render_console_interaction_report(draft, audit)

    assert "# Console Interaction Draft Report" in report
    assert "- Critical Pages: 4" in report
    assert "- Required Roles: none" in report
    assert "- Readiness Score: 100.0" in report
    assert "- Release Ready: True" in report
    assert "- Overview: route=/overview required_actions=drill-down, export, audit available_actions=drill-down, export, audit filters=1 states=default, loading, empty, error batch=optional permissions=complete" in report
    assert "- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=1 states=default, loading, empty, error batch=required permissions=complete" in report
    assert "- Permission gaps: none" in report


def test_build_big_4203_console_interaction_draft_is_release_ready() -> None:
    draft = build_big_4203_console_interaction_draft()

    audit = ConsoleInteractionAuditor().audit(draft)
    report = render_console_interaction_report(draft, audit)

    assert draft.required_roles == [
        "eng-lead",
        "platform-admin",
        "vp-eng",
        "cross-team-operator",
    ]
    assert draft.requires_frame_contracts is True
    assert audit.release_ready is True
    assert audit.uncovered_roles == []
    assert "- Required Roles: eng-lead, platform-admin, vp-eng, cross-team-operator" in report
    assert "persona=VP Eng wireframe=wf-overview" in report
    assert "review_focus=metric hierarchy,drill-down posture,alert prioritization" in report
    assert "- Uncovered roles: none" in report
    assert "- Pages missing personas: none" in report
    assert "- Pages missing wireframe links: none" in report


def test_console_interaction_audit_flags_uncovered_required_roles() -> None:
    draft = build_big_4203_console_interaction_draft()
    draft.required_roles.append("finance-reviewer")

    audit = ConsoleInteractionAuditor().audit(draft)

    assert audit.uncovered_roles == ["finance-reviewer"]
    assert audit.release_ready is False


def test_console_interaction_audit_flags_missing_frame_contract_details() -> None:
    draft = build_big_4203_console_interaction_draft()
    draft.contracts[0].primary_persona = ""
    draft.contracts[0].linked_wireframe_id = ""
    draft.contracts[0].review_focus_areas = []
    draft.contracts[0].decision_prompts = []

    audit = ConsoleInteractionAuditor().audit(draft)

    assert audit.surfaces_missing_primary_personas == ["Overview"]
    assert audit.surfaces_missing_wireframe_links == ["Overview"]
    assert audit.surfaces_missing_review_focus == ["Overview"]
    assert audit.surfaces_missing_decision_prompts == ["Overview"]
    assert audit.release_ready is False

def build_review_pack() -> UIReviewPack:
    return UIReviewPack(
        issue_id="BIG-4204",
        title="UI评审包输出",
        version="v4.0-review-pack",
        objectives=[
            ReviewObjective(
                objective_id="obj-alignment",
                title="Align reviewers on the release-control story",
                persona="product-experience",
                outcome="Reviewers see the scope, stakes, and success criteria before page-level critique.",
                success_signal="The kickoff frame is sufficient to decide whether the slice is review-ready.",
                priority="P0",
                dependencies=["BIG-1103", "BIG-1701"],
            )
        ],
        wireframes=[
            WireframeSurface(
                surface_id="wf-overview",
                name="Review overview board",
                device="desktop",
                entry_point="Epic 11 review hub",
                primary_blocks=["header", "objective strip", "wireframe rail", "decision log"],
                review_notes=["Highlight unresolved dependencies before approval."],
            )
        ],
        interactions=[
            InteractionFlow(
                flow_id="flow-frame-switch",
                name="Switch between wireframes and interaction notes",
                trigger="Reviewer selects a surface from the wireframe rail",
                system_response="The board swaps the focus frame and preserves reviewer comments.",
                states=["default", "focus", "with-comments"],
                exceptions=["Warn when leaving a frame with unsaved notes."],
            )
        ],
        open_questions=[
            OpenQuestion(
                question_id="oq-mobile-depth",
                theme="scope",
                question="Should the first review pack cover mobile wireframes or desktop only?",
                owner="product-experience",
                impact="Changes review breadth and the number of required surfaces.",
            )
        ],
    )


def test_ui_review_pack_round_trip_preserves_manifest_shape() -> None:
    pack = build_review_pack()

    restored = UIReviewPack.from_dict(pack.to_dict())

    assert restored == pack


def test_ui_review_pack_audit_flags_missing_sections_and_coverage_gaps() -> None:
    pack = UIReviewPack(
        issue_id="BIG-4204",
        title="UI评审包输出",
        version="v4.0-review-pack",
        objectives=[
            ReviewObjective(
                objective_id="obj-incomplete",
                title="Incomplete objective",
                persona="product-experience",
                outcome="Create a frame for review.",
                success_signal="",
            )
        ],
        wireframes=[
            WireframeSurface(
                surface_id="wf-empty",
                name="Empty frame",
                device="desktop",
                entry_point="Review hub",
            )
        ],
        interactions=[
            InteractionFlow(
                flow_id="flow-empty",
                name="Unspecified interaction",
                trigger="Reviewer opens the page",
                system_response="The system loads the frame.",
            )
        ],
    )

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.missing_sections == ["open_questions"]
    assert audit.objectives_missing_signals == ["obj-incomplete"]
    assert audit.wireframes_missing_blocks == ["wf-empty"]
    assert audit.interactions_missing_states == ["flow-empty"]


def test_ui_review_pack_audit_allows_open_questions_while_marking_pack_ready() -> None:
    pack = build_review_pack()

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is True
    assert audit.unresolved_question_ids == ["oq-mobile-depth"]
    assert audit.missing_sections == []


def test_render_ui_review_pack_report_summarizes_review_shape_and_findings() -> None:
    pack = build_review_pack()
    audit = UIReviewPackAuditor().audit(pack)

    report = render_ui_review_pack_report(pack, audit)

    assert "# UI Review Pack" in report
    assert "- Issue: BIG-4204 UI评审包输出" in report
    assert "- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0 blockers=0 timeline_events=0" in report
    assert (
        "- obj-alignment: Align reviewers on the release-control story "
        "persona=product-experience priority=P0"
    ) in report
    assert "- Unresolved questions: oq-mobile-depth" in report


def test_build_big_4204_review_pack_is_ready_for_design_sprint_review() -> None:
    pack = build_big_4204_review_pack()

    audit = UIReviewPackAuditor().audit(pack)
    report = render_ui_review_pack_report(pack, audit)

    assert audit.ready is True
    assert len(pack.objectives) == 4
    assert len(pack.wireframes) == 4
    assert len(pack.interactions) == 4
    assert len(pack.open_questions) == 3
    assert len(pack.reviewer_checklist) == 8
    assert len(pack.decision_log) == 4
    assert len(pack.role_matrix) == 8
    assert len(pack.signoff_log) == 4
    assert len(pack.blocker_log) == 1
    assert len(pack.blocker_timeline) == 2
    assert pack.requires_reviewer_checklist is True
    assert pack.requires_decision_log is True
    assert pack.requires_role_matrix is True
    assert pack.requires_signoff_log is True
    assert pack.requires_blocker_log is True
    assert pack.requires_blocker_timeline is True
    assert "obj-queue-governance" in report
    assert "## Review Summary Board" in report
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in report
    assert "## Objective Coverage Board" in report
    assert "- covered: 2" in report
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in report
    assert "## Persona Readiness Board" in report
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in report
    assert "wf-triage: Triage and handoff board" in report
    assert "## Wireframe Readiness Board" in report
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in report
    assert "flow-run-replay: Run replay with evidence audit" in report
    assert "## Interaction Coverage Board" in report
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in report
    assert "## Open Question Tracker" in report
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in report
    assert "chk-queue-batch-approval: surface=wf-queue owner=Platform Admin status=ready" in report
    assert "dec-queue-vp-summary: surface=wf-queue owner=VP Eng status=proposed" in report
    assert "role-queue-platform-admin: surface=wf-queue role=Platform Admin status=ready" in report
    assert "## Checklist Traceability Board" in report
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in report
    assert "## Decision Follow-up Tracker" in report
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in report
    assert "## Role Coverage Board" in report
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in report
    assert "## Signoff Dependency Board" in report
    assert "- blocked: 1" in report
    assert "- clear: 3" in report
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in report
    assert "assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z sla=at-risk due_at=2026-03-15T18:00:00Z cadence=daily" in report
    assert "sig-run-detail-eng-lead: surface=wf-run-detail role=Eng Lead assignment=role-run-detail-eng-lead status=pending" in report
    assert "blk-run-detail-copy-final: surface=wf-run-detail signoff=sig-run-detail-eng-lead owner=product-experience status=open severity=medium" in report
    assert "evt-run-detail-copy-escalated: blocker=blk-run-detail-copy-final actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in report
    assert "## Review Exceptions" in report
    assert "exc-blk-run-detail-copy-final: type=blocker source=blk-run-detail-copy-final surface=wf-run-detail owner=product-experience status=open severity=medium" in report
    assert "## Sign-off SLA Dashboard" in report
    assert "- at-risk: 1" in report
    assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in report
    assert "## Sign-off Reminder Queue" in report
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in report
    assert "## Reminder Cadence Board" in report
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in report
    assert "## Sign-off Breach Board" in report
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in report
    assert "## Escalation Dashboard" in report
    assert "- engineering-director: blockers=0 signoffs=1 total=1" in report
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in report
    assert "## Escalation Handoff Ledger" in report
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in report
    assert "## Handoff Ack Ledger" in report
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in report
    assert "## Owner Escalation Digest" in report
    assert "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2" in report
    assert "## Owner Workload Board" in report
    assert "- Owners: 7" in report
    assert "- Items: 8" in report
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 reminders=0 renewals=0 total=2" in report
    assert "load-queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open lane=queue" in report
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in report
    assert "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal" in report
    assert "## Review Freeze Exception Board" in report
    assert "## Freeze Approval Trail" in report
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in report
    assert "## Freeze Renewal Tracker" in report
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in report
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in report
    assert "## Review Exception Matrix" in report
    assert "- product-experience: blockers=1 signoffs=0 total=1" in report
    assert "## Audit Density Board" in report
    assert "- Surfaces: 4" in report
    assert "- Load bands: 3" in report
    assert "- active: 2" in report
    assert "- dense: 1" in report
    assert "- light: 1" in report
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in report
    assert "checklist=2 decisions=1 assignments=2 signoffs=1 blockers=1 timeline=2 blocks=4 notes=2" in report
    assert "## Owner Review Queue" in report
    assert "queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in report
    assert "## Blocker Timeline Summary" in report
    assert "- escalated: 1" in report
    assert "- Wireframes missing checklist coverage: none" in report
    assert "- Checklist items missing role links: none" in report
    assert "- Wireframes missing decision coverage: none" in report
    assert "- Unresolved decisions missing follow-ups: none" in report
    assert "- Wireframes missing role assignments: none" in report
    assert "- Wireframes missing signoff coverage: none" in report
    assert "- Blockers missing signoff links: none" in report
    assert "- Freeze exceptions missing owners: none" in report
    assert "- Freeze exceptions missing windows: none" in report
    assert "- Freeze exceptions missing approvers: none" in report
    assert "- Freeze exceptions missing approval dates: none" in report
    assert "- Freeze exceptions missing renewal owners: none" in report
    assert "- Freeze exceptions missing renewal dates: none" in report
    assert "- Blockers missing timeline events: none" in report
    assert "- Closed blockers missing resolution events: none" in report
    assert "- Orphan blocker timeline blocker ids: none" in report
    assert "- Handoff events missing targets: none" in report
    assert "- Handoff events missing artifacts: none" in report
    assert "- Handoff events missing ack owners: none" in report
    assert "- Handoff events missing ack dates: none" in report
    assert "- Unresolved required signoffs without blockers: none" in report
    assert "- Unresolved decision ids: dec-queue-vp-summary" in report
    assert "- Decisions missing role links: none" in report
    assert "- Signoffs missing requested dates: none" in report
    assert "- Signoffs missing due dates: none" in report
    assert "- Signoffs missing escalation owners: none" in report
    assert "- Signoffs missing reminder owners: none" in report
    assert "- Signoffs missing next reminders: none" in report
    assert "- Signoffs missing reminder cadence: none" in report
    assert "- Signoffs with breached SLA: none" in report
    assert "- Unresolved required signoff ids: sig-run-detail-eng-lead" in report
    assert "- Unresolved questions: oq-role-density, oq-alert-priority, oq-handoff-evidence" in report


def test_ui_review_pack_audit_flags_missing_checklist_coverage_and_evidence() -> None:
    pack = build_big_4204_review_pack()
    pack.reviewer_checklist = [
        ReviewerChecklistItem(
            item_id="chk-overview-kpi-scan",
            surface_id="wf-overview",
            prompt="Verify the KPI strip still supports one-screen executive scanning before drill-down.",
            owner="VP Eng",
            status="ready",
            evidence_links=[],
        )
    ]

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.wireframes_missing_checklists == ["wf-queue", "wf-run-detail", "wf-triage"]
    assert audit.checklist_items_missing_evidence == ["chk-overview-kpi-scan"]
    assert audit.orphan_checklist_surfaces == []


def test_ui_review_pack_audit_flags_missing_decision_coverage() -> None:
    pack = build_big_4204_review_pack()
    pack.decision_log = [
        ReviewDecision(
            decision_id="dec-overview-alert-stack",
            surface_id="wf-overview",
            owner="product-experience",
            summary="Keep approval and regression alerts in one stacked priority rail.",
            rationale="Reviewers need one comparison lane before jumping into queue or triage surfaces.",
            status="accepted",
        )
    ]

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.wireframes_missing_decisions == ["wf-queue", "wf-run-detail", "wf-triage"]
    assert audit.orphan_decision_surfaces == []
    assert audit.unresolved_decision_ids == []


def test_ui_review_pack_audit_flags_missing_role_matrix_coverage() -> None:
    pack = build_big_4204_review_pack()
    pack.role_matrix = [
        ReviewRoleAssignment(
            assignment_id="role-overview-vp-eng",
            surface_id="wf-overview",
            role="VP Eng",
            responsibilities=["approve overview scan path"],
            checklist_item_ids=["chk-overview-kpi-scan"],
            decision_ids=["dec-overview-alert-stack"],
            status="ready",
        )
    ]

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.wireframes_missing_role_assignments == ["wf-queue", "wf-run-detail", "wf-triage"]
    assert audit.orphan_role_assignment_surfaces == []
    assert audit.role_assignments_missing_responsibilities == []
    assert audit.role_assignments_missing_checklist_links == []
    assert audit.role_assignments_missing_decision_links == []
    assert audit.checklist_items_missing_role_links == [
        "chk-overview-alert-hierarchy",
        "chk-queue-batch-approval",
        "chk-queue-role-density",
        "chk-run-audit-density",
        "chk-run-replay-context",
        "chk-triage-bulk-assign",
        "chk-triage-handoff-clarity",
    ]
    assert audit.decisions_missing_role_links == [
        "dec-queue-vp-summary",
        "dec-run-detail-audit-rail",
        "dec-triage-handoff-density",
    ]


def test_ui_review_pack_audit_flags_missing_signoff_coverage_and_assignment_links() -> None:
    pack = build_big_4204_review_pack()
    pack.signoff_log = [
        ReviewSignoff(
            signoff_id="sig-overview-vp-eng",
            assignment_id="role-overview-missing",
            surface_id="wf-overview",
            role="VP Eng",
            status="approved",
            evidence_links=["chk-overview-kpi-scan"],
        )
    ]

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.wireframes_missing_signoffs == ["wf-queue", "wf-run-detail", "wf-triage"]
    assert audit.orphan_signoff_surfaces == []
    assert audit.signoffs_missing_assignments == ["sig-overview-vp-eng"]
    assert audit.signoffs_missing_evidence == []
    assert audit.unresolved_required_signoff_ids == []


def test_ui_review_pack_audit_flags_missing_signoff_sla_metadata() -> None:
    pack = build_big_4204_review_pack()
    pack.signoff_log[2] = ReviewSignoff(
        signoff_id="sig-run-detail-eng-lead",
        assignment_id="role-run-detail-eng-lead",
        surface_id="wf-run-detail",
        role="Eng Lead",
        status="pending",
        evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
        notes="Waiting for final replay-state copy review.",
        requested_at="",
        due_at="",
        escalation_owner="",
        sla_status="breached",
        reminder_owner="",
        reminder_channel="slack",
        last_reminder_at="2026-03-14T09:45:00Z",
        next_reminder_at="",
    )

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.signoffs_missing_requested_dates == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_due_dates == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_escalation_owners == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_reminder_owners == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_next_reminders == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_reminder_cadence == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_with_breached_sla == ["sig-run-detail-eng-lead"]


def test_ui_review_pack_audit_flags_unresolved_decision_without_follow_up() -> None:
    pack = build_big_4204_review_pack()
    pack.decision_log[1] = ReviewDecision(
        decision_id="dec-queue-vp-summary",
        surface_id="wf-queue",
        owner="VP Eng",
        summary="Route VP Eng to a summary-first queue state instead of operator controls.",
        rationale="The VP Eng persona needs queue visibility without accidental action affordances.",
        status="proposed",
        follow_up="",
    )

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.unresolved_decision_ids == ["dec-queue-vp-summary"]
    assert audit.unresolved_decisions_missing_follow_ups == ["dec-queue-vp-summary"]


def test_ui_review_pack_audit_flags_missing_freeze_and_handoff_metadata() -> None:
    pack = build_big_4204_review_pack()
    pack.blocker_log[0] = ReviewBlocker(
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
        freeze_owner="",
        freeze_until="",
        freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
        freeze_approved_by="",
        freeze_approved_at="",
    )
    pack.blocker_timeline[1] = ReviewBlockerEvent(
        event_id="evt-run-detail-copy-escalated",
        blocker_id="blk-run-detail-copy-final",
        actor="design-program-manager",
        status="escalated",
        summary="Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
        timestamp="2026-03-14T09:30:00Z",
        next_action="Refresh the run-detail frame annotations after the wording review completes.",
        handoff_from="product-experience",
        handoff_to="",
        channel="design-critique",
        artifact_ref="",
    )

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.freeze_exceptions_missing_owners == ["blk-run-detail-copy-final"]
    assert audit.freeze_exceptions_missing_until == ["blk-run-detail-copy-final"]
    assert audit.freeze_exceptions_missing_approvers == ["blk-run-detail-copy-final"]
    assert audit.freeze_exceptions_missing_approval_dates == ["blk-run-detail-copy-final"]
    assert audit.freeze_exceptions_missing_renewal_owners == ["blk-run-detail-copy-final"]
    assert audit.freeze_exceptions_missing_renewal_dates == ["blk-run-detail-copy-final"]
    assert audit.handoff_events_missing_targets == ["evt-run-detail-copy-escalated"]
    assert audit.handoff_events_missing_artifacts == ["evt-run-detail-copy-escalated"]
    assert audit.handoff_events_missing_ack_owners == ["evt-run-detail-copy-escalated"]
    assert audit.handoff_events_missing_ack_dates == ["evt-run-detail-copy-escalated"]


def test_ui_review_pack_audit_flags_unresolved_signoff_without_blocker() -> None:
    pack = build_big_4204_review_pack()
    pack.blocker_log = []

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.unresolved_required_signoff_ids == ["sig-run-detail-eng-lead"]
    assert audit.unresolved_required_signoffs_without_blockers == ["sig-run-detail-eng-lead"]
    assert audit.blockers_missing_signoff_links == []
    assert audit.blockers_missing_escalation_owners == []
    assert audit.blockers_missing_next_actions == []



def test_ui_review_pack_audit_flags_waived_signoff_without_metadata() -> None:
    pack = build_big_4204_review_pack()
    pack.signoff_log[2] = ReviewSignoff(
        signoff_id="sig-run-detail-eng-lead",
        assignment_id="role-run-detail-eng-lead",
        surface_id="wf-run-detail",
        role="Eng Lead",
        status="waived",
        evidence_links=[],
        notes="Design review accepted a temporary waiver pending copy cleanup.",
    )
    pack.blocker_log = []
    pack.blocker_timeline = []

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.waived_signoffs_missing_metadata == ["sig-run-detail-eng-lead"]
    assert audit.signoffs_missing_evidence == []
    assert audit.unresolved_required_signoff_ids == []



def test_ui_review_pack_audit_flags_missing_or_invalid_blocker_timeline() -> None:
    pack = build_big_4204_review_pack()
    pack.blocker_timeline = []

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.blockers_missing_timeline_events == ["blk-run-detail-copy-final"]
    assert audit.closed_blockers_missing_resolution_events == []
    assert audit.orphan_blocker_timeline_blocker_ids == []



def test_ui_review_pack_audit_flags_closed_blocker_without_resolution_event_and_orphans() -> None:
    pack = build_big_4204_review_pack()
    pack.blocker_log[0] = ReviewBlocker(
        blocker_id="blk-run-detail-copy-final",
        surface_id="wf-run-detail",
        signoff_id="sig-run-detail-eng-lead",
        owner="product-experience",
        summary="Replay-state copy review is closed pending audit trail confirmation.",
        status="closed",
        severity="medium",
        escalation_owner="design-program-manager",
        next_action="Archive the blocker after the final review sync.",
    )
    pack.blocker_timeline = [
        ReviewBlockerEvent(
            event_id="evt-run-detail-copy-opened",
            blocker_id="blk-run-detail-copy-final",
            actor="product-experience",
            status="opened",
            summary="Tracked the replay-state wording gap during review prep.",
            timestamp="2026-03-13T10:00:00Z",
            next_action="Review wording changes with Eng Lead.",
        ),
        ReviewBlockerEvent(
            event_id="evt-orphan-blocker",
            blocker_id="blk-missing",
            actor="design-program-manager",
            status="escalated",
            summary="Unexpected timeline entry remained after blocker merge cleanup.",
            timestamp="2026-03-14T11:00:00Z",
            next_action="Remove orphaned timeline entry from the bundle.",
        ),
    ]

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.blockers_missing_timeline_events == []
    assert audit.closed_blockers_missing_resolution_events == ["blk-run-detail-copy-final"]
    assert audit.orphan_blocker_timeline_blocker_ids == ["blk-missing"]


def test_render_ui_review_signoff_sla_and_escalation_dashboards() -> None:
    pack = build_big_4204_review_pack()

    signoff_sla = render_ui_review_signoff_sla_dashboard(pack)
    signoff_reminder = render_ui_review_signoff_reminder_queue(pack)
    signoff_breach = render_ui_review_signoff_breach_board(pack)
    escalation_dashboard = render_ui_review_escalation_dashboard(pack)
    handoff_ledger = render_ui_review_escalation_handoff_ledger(pack)
    owner_digest = render_ui_review_owner_escalation_digest(pack)

    assert "# UI Review Sign-off SLA Dashboard" in signoff_sla
    assert "- Sign-offs: 4" in signoff_sla
    assert "- Escalation owners: 4" in signoff_sla
    assert "- at-risk: 1" in signoff_sla
    assert "- met: 3" in signoff_sla
    assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in signoff_sla
    assert "# UI Review Sign-off Reminder Queue" in signoff_reminder
    assert "- Reminders: 1" in signoff_reminder
    assert "- design-program-manager: reminders=1" in signoff_reminder
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in signoff_reminder
    assert "# UI Review Sign-off Breach Board" in signoff_breach
    assert "- Breach items: 1" in signoff_breach
    assert "- engineering-director: 1" in signoff_breach
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in signoff_breach
    assert "# UI Review Escalation Dashboard" in escalation_dashboard
    assert "- Items: 2" in escalation_dashboard
    assert "- design-program-manager: blockers=1 signoffs=0 total=1" in escalation_dashboard
    assert "- engineering-director: blockers=0 signoffs=1 total=1" in escalation_dashboard
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in escalation_dashboard
    assert "# UI Review Escalation Handoff Ledger" in handoff_ledger
    assert "- Handoffs: 1" in handoff_ledger
    assert "- design-critique: 1" in handoff_ledger
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in handoff_ledger
    assert "from=product-experience to=Eng Lead channel=design-critique artifact=wf-run-detail#copy-v5" in handoff_ledger
    assert "# UI Review Owner Escalation Digest" in owner_digest
    assert "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2" in owner_digest
    assert "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in owner_digest


def test_render_ui_review_exception_matrix_includes_signoff_and_blocker_counts() -> None:
    pack = build_big_4204_review_pack()
    pack.signoff_log[2] = ReviewSignoff(
        signoff_id="sig-run-detail-eng-lead",
        assignment_id="role-run-detail-eng-lead",
        surface_id="wf-run-detail",
        role="Eng Lead",
        status="waived",
        evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
        notes="Temporary waiver approved pending copy lock.",
        waiver_owner="Eng Lead",
        waiver_reason="Copy review is deferred to the next wording pass.",
    )

    exception_matrix = render_ui_review_exception_matrix(pack)

    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- Exceptions: 2" in exception_matrix
    assert "- Owners: 2" in exception_matrix
    assert "- Surfaces: 1" in exception_matrix
    assert "- Eng Lead: blockers=0 signoffs=1 total=1" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "- open: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "- waived: blockers=0 signoffs=1 total=1" in exception_matrix
    assert "- wf-run-detail: blockers=1 signoffs=1 total=2" in exception_matrix



def test_render_ui_review_freeze_exception_board() -> None:
    pack = build_big_4204_review_pack()

    freeze_board = render_ui_review_freeze_exception_board(pack)

    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "- Exceptions: 1" in freeze_board
    assert "- release-director: blockers=1 signoffs=0 total=1" in freeze_board
    assert "- wf-run-detail: blockers=1 signoffs=0 total=1" in freeze_board
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in freeze_board


def test_render_ui_review_freeze_approval_trail() -> None:
    pack = build_big_4204_review_pack()

    freeze_trail = render_ui_review_freeze_approval_trail(pack)

    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "- Approvals: 1" in freeze_trail
    assert "- release-director: 1" in freeze_trail
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in freeze_trail




def test_render_ui_review_summary_persona_and_interaction_boards() -> None:
    pack = build_big_4204_review_pack()

    review_summary = render_ui_review_review_summary_board(pack)
    persona_readiness = render_ui_review_persona_readiness_board(pack)
    interaction_coverage = render_ui_review_interaction_coverage_board(pack)

    assert "# UI Review Review Summary Board" in review_summary
    assert "- Categories: 6" in review_summary
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in review_summary
    assert "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2" in review_summary
    assert "summary-interactions: category=interactions total=4 covered=4 watch=0 missing=0" in review_summary
    assert "summary-actions: category=actions total=8 queue=6 reminder=1 renewal=1" in review_summary
    assert "# UI Review Persona Readiness Board" in persona_readiness
    assert "- Personas: 4" in persona_readiness
    assert "- Objectives: 4" in persona_readiness
    assert "- blocked: 1" in persona_readiness
    assert "- at-risk: 1" in persona_readiness
    assert "- ready: 2" in persona_readiness
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in persona_readiness
    assert "objective_ids=obj-run-detail-investigation surfaces=wf-run-detail queue_ids=queue-sig-run-detail-eng-lead blocker_ids=blk-run-detail-copy-final" in persona_readiness
    assert "# UI Review Interaction Coverage Board" in interaction_coverage
    assert "- Interactions: 4" in interaction_coverage
    assert "- Surfaces: 4" in interaction_coverage
    assert "- covered: 4" in interaction_coverage
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in interaction_coverage
    assert "checklist=chk-triage-handoff-clarity,chk-triage-bulk-assign open_checklist=none trigger=Cross-Team Operator bulk-assigns a finding set or opens the handoff panel" in interaction_coverage


def test_render_ui_review_objective_wireframe_and_question_boards() -> None:
    pack = build_big_4204_review_pack()

    objective_coverage = render_ui_review_objective_coverage_board(pack)
    wireframe_readiness = render_ui_review_wireframe_readiness_board(pack)
    question_tracker = render_ui_review_open_question_tracker(pack)

    assert "# UI Review Objective Coverage Board" in objective_coverage
    assert "- Objectives: 4" in objective_coverage
    assert "- Personas: 4" in objective_coverage
    assert "- blocked: 1" in objective_coverage
    assert "- covered: 2" in objective_coverage
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in objective_coverage
    assert "dependency_ids=BIG-4203,OPE-72,OPE-73 assignments=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final" in objective_coverage
    assert "# UI Review Wireframe Readiness Board" in wireframe_readiness
    assert "- Wireframes: 4" in wireframe_readiness
    assert "- Devices: 1" in wireframe_readiness
    assert "- at-risk: 2" in wireframe_readiness
    assert "- blocked: 1" in wireframe_readiness
    assert "- ready: 1" in wireframe_readiness
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in wireframe_readiness
    assert "checklist_open=1 decisions_open=0 assignments_open=1 signoffs_open=1 blockers_open=1 signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final blocks=4 notes=2" in wireframe_readiness
    assert "# UI Review Open Question Tracker" in question_tracker
    assert "- Questions: 3" in question_tracker
    assert "- Owners: 3" in question_tracker
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in question_tracker
    assert "checklist=chk-queue-role-density flows=none impact=Changes denial-path copy, button placement, and review criteria for queue and triage pages." in question_tracker


def test_render_ui_review_traceability_and_role_coverage_boards() -> None:
    pack = build_big_4204_review_pack()

    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)

    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "- Checklist items: 8" in checklist_traceability
    assert "- Owners: 7" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "- Decisions: 4" in decision_followup
    assert "- Owners: 4" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "linked_assignments=role-queue-platform-admin,role-queue-product-experience linked_checklists=chk-queue-batch-approval,chk-queue-role-density follow_up=Resolve after the next design critique with policy owners." in decision_followup
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "- Assignments: 8" in role_coverage
    assert "- Surfaces: 4" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "signoff=sig-run-detail-eng-lead signoff_status=pending" in role_coverage


def test_render_ui_review_dependency_workload_and_density_boards() -> None:
    pack = build_big_4204_review_pack()

    signoff_dependency = render_ui_review_signoff_dependency_board(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    audit_density = render_ui_review_audit_density_board(pack)

    assert "# UI Review Signoff Dependency Board" in signoff_dependency
    assert "- Sign-offs: 4" in signoff_dependency
    assert "- blocked: 1" in signoff_dependency
    assert "- clear: 3" in signoff_dependency
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in signoff_dependency
    assert "assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z sla=at-risk due_at=2026-03-15T18:00:00Z cadence=daily" in signoff_dependency
    assert "# UI Review Owner Workload Board" in owner_workload
    assert "- Owners: 7" in owner_workload
    assert "- Items: 8" in owner_workload
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 reminders=0 renewals=0 total=2" in owner_workload
    assert "load-queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open lane=queue" in owner_workload
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in owner_workload
    assert "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal" in owner_workload
    assert "# UI Review Audit Density Board" in audit_density
    assert "- Surfaces: 4" in audit_density
    assert "- Load bands: 3" in audit_density
    assert "- active: 2" in audit_density
    assert "- dense: 1" in audit_density
    assert "- light: 1" in audit_density
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in audit_density
    assert "checklist=2 decisions=1 assignments=2 signoffs=1 blockers=1 timeline=2 blocks=4 notes=2" in audit_density


def test_render_ui_review_owner_review_queue_groups_actionable_items() -> None:
    pack = build_big_4204_review_pack()

    owner_queue = render_ui_review_owner_review_queue(pack)

    assert "# UI Review Owner Review Queue" in owner_queue
    assert "- Owners: 5" in owner_queue
    assert "- Queue items: 6" in owner_queue
    assert "- engineering-operations: blockers=0 checklist=1 decisions=0 signoffs=0 total=1" in owner_queue
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 total=2" in owner_queue
    assert "- queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open" in owner_queue
    assert "- queue-dec-queue-vp-summary: owner=VP Eng type=decision source=dec-queue-vp-summary surface=wf-queue status=proposed" in owner_queue
    assert "- queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in owner_queue
    assert "- queue-blk-run-detail-copy-final: owner=product-experience type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open" in owner_queue



def test_render_ui_review_exception_log_and_timeline_summary() -> None:
    pack = build_big_4204_review_pack()

    signoff_sla = render_ui_review_signoff_sla_dashboard(pack)
    signoff_reminder = render_ui_review_signoff_reminder_queue(pack)
    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    reminder_cadence = render_ui_review_reminder_cadence_board(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)
    signoff_breach = render_ui_review_signoff_breach_board(pack)
    escalation_dashboard = render_ui_review_escalation_dashboard(pack)
    handoff_ledger = render_ui_review_escalation_handoff_ledger(pack)
    handoff_ack = render_ui_review_handoff_ack_ledger(pack)
    owner_digest = render_ui_review_owner_escalation_digest(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    freeze_board = render_ui_review_freeze_exception_board(pack)
    freeze_trail = render_ui_review_freeze_approval_trail(pack)
    freeze_renewal = render_ui_review_freeze_renewal_tracker(pack)
    exception_log = render_ui_review_exception_log(pack)
    exception_matrix = render_ui_review_exception_matrix(pack)
    audit_density = render_ui_review_audit_density_board(pack)
    owner_review_queue = render_ui_review_owner_review_queue(pack)
    timeline_summary = render_ui_review_blocker_timeline_summary(pack)

    assert "# UI Review Sign-off SLA Dashboard" in signoff_sla
    assert "- at-risk: 1" in signoff_sla
    assert "# UI Review Sign-off Reminder Queue" in signoff_reminder
    assert "- Reminders: 1" in signoff_reminder
    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "# UI Review Reminder Cadence Board" in reminder_cadence
    assert "- Cadences: 1" in reminder_cadence
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in reminder_cadence
    assert "# UI Review Sign-off Breach Board" in signoff_breach
    assert "- Breach items: 1" in signoff_breach
    assert "# UI Review Escalation Dashboard" in escalation_dashboard
    assert "- engineering-director: blockers=0 signoffs=1 total=1" in escalation_dashboard
    assert "# UI Review Escalation Handoff Ledger" in handoff_ledger
    assert "- design-critique: 1" in handoff_ledger
    assert "# UI Review Handoff Ack Ledger" in handoff_ack
    assert "- Ack owners: 1" in handoff_ack
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in handoff_ack
    assert "# UI Review Owner Escalation Digest" in owner_digest
    assert "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2" in owner_digest
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "- release-director: blockers=1 signoffs=0 total=1" in freeze_board
    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "- Approvals: 1" in freeze_trail
    assert "# UI Review Freeze Renewal Tracker" in freeze_renewal
    assert "- Renewal owners: 1" in freeze_renewal
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in freeze_renewal
    assert "# UI Review Exception Log" in exception_log
    assert "- Exceptions: 1" in exception_log
    assert "exc-blk-run-detail-copy-final" in exception_log
    assert "evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z" in exception_log
    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "# UI Review Audit Density Board" in audit_density
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in audit_density
    assert "# UI Review Owner Review Queue" in owner_review_queue
    assert "- Queue items: 6" in owner_review_queue
    assert "# UI Review Blocker Timeline Summary" in timeline_summary
    assert "- Events: 2" in timeline_summary
    assert "- opened: 1" in timeline_summary
    assert "- escalated: 1" in timeline_summary
    assert "- design-program-manager: 1" in timeline_summary
    assert "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in timeline_summary


def test_render_ui_review_html_and_bundle_export(tmp_path) -> None:
    pack = build_big_4204_review_pack()
    audit = UIReviewPackAuditor().audit(pack)

    html = render_ui_review_pack_html(pack, audit)
    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_log = render_ui_review_decision_log(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    review_summary = render_ui_review_review_summary_board(pack)
    objective_coverage = render_ui_review_objective_coverage_board(pack)
    persona_readiness = render_ui_review_persona_readiness_board(pack)
    wireframe_readiness = render_ui_review_wireframe_readiness_board(pack)
    interaction_coverage = render_ui_review_interaction_coverage_board(pack)
    question_tracker = render_ui_review_open_question_tracker(pack)
    role_matrix = render_ui_review_role_matrix(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)
    signoff_dependency = render_ui_review_signoff_dependency_board(pack)
    signoff_log = render_ui_review_signoff_log(pack)
    blocker_log = render_ui_review_blocker_log(pack)
    blocker_timeline = render_ui_review_blocker_timeline(pack)
    signoff_sla = render_ui_review_signoff_sla_dashboard(pack)
    signoff_reminder = render_ui_review_signoff_reminder_queue(pack)
    reminder_cadence = render_ui_review_reminder_cadence_board(pack)
    signoff_breach = render_ui_review_signoff_breach_board(pack)
    escalation_dashboard = render_ui_review_escalation_dashboard(pack)
    handoff_ledger = render_ui_review_escalation_handoff_ledger(pack)
    handoff_ack = render_ui_review_handoff_ack_ledger(pack)
    owner_digest = render_ui_review_owner_escalation_digest(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    freeze_board = render_ui_review_freeze_exception_board(pack)
    freeze_trail = render_ui_review_freeze_approval_trail(pack)
    freeze_renewal = render_ui_review_freeze_renewal_tracker(pack)
    exception_log = render_ui_review_exception_log(pack)
    exception_matrix = render_ui_review_exception_matrix(pack)
    audit_density = render_ui_review_audit_density_board(pack)
    owner_review_queue = render_ui_review_owner_review_queue(pack)
    timeline_summary = render_ui_review_blocker_timeline_summary(pack)
    artifacts = write_ui_review_pack_bundle(str(tmp_path), pack)

    assert "<h2>Decision Log</h2>" in html
    assert "<h2>Checklist Traceability Board</h2>" in html
    assert "<h2>Decision Follow-up Tracker</h2>" in html
    assert "<h2>Review Summary Board</h2>" in html
    assert "<h2>Objective Coverage Board</h2>" in html
    assert "<h2>Persona Readiness Board</h2>" in html
    assert "<h2>Wireframe Readiness Board</h2>" in html
    assert "<h2>Interaction Coverage Board</h2>" in html
    assert "<h2>Open Question Tracker</h2>" in html
    assert "<h2>Role Matrix</h2>" in html
    assert "<h2>Role Coverage Board</h2>" in html
    assert "<h2>Signoff Dependency Board</h2>" in html
    assert "<h2>Sign-off Log</h2>" in html
    assert "<h2>Sign-off SLA Dashboard</h2>" in html
    assert "<h2>Sign-off Reminder Queue</h2>" in html
    assert "<h2>Reminder Cadence Board</h2>" in html
    assert "<h2>Sign-off Breach Board</h2>" in html
    assert "<h2>Escalation Dashboard</h2>" in html
    assert "<h2>Escalation Handoff Ledger</h2>" in html
    assert "<h2>Handoff Ack Ledger</h2>" in html
    assert "<h2>Owner Escalation Digest</h2>" in html
    assert "<h2>Owner Workload Board</h2>" in html
    assert "<h2>Blocker Log</h2>" in html
    assert "<h2>Blocker Timeline</h2>" in html
    assert "<h2>Review Freeze Exception Board</h2>" in html
    assert "<h2>Freeze Approval Trail</h2>" in html
    assert "<h2>Freeze Renewal Tracker</h2>" in html
    assert "<h2>Review Exceptions</h2>" in html
    assert "<h2>Review Exception Matrix</h2>" in html
    assert "<h2>Audit Density Board</h2>" in html
    assert "<h2>Owner Review Queue</h2>" in html
    assert "<h2>Blocker Timeline Summary</h2>" in html
    assert "dec-queue-vp-summary" in html
    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Log" in decision_log
    assert "dec-run-detail-audit-rail" in decision_log
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "# UI Review Review Summary Board" in review_summary
    assert "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2" in review_summary
    assert "# UI Review Objective Coverage Board" in objective_coverage
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in objective_coverage
    assert "# UI Review Persona Readiness Board" in persona_readiness
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in persona_readiness
    assert "# UI Review Wireframe Readiness Board" in wireframe_readiness
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in wireframe_readiness
    assert "# UI Review Interaction Coverage Board" in interaction_coverage
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in interaction_coverage
    assert "# UI Review Open Question Tracker" in question_tracker
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in question_tracker
    assert "# UI Review Role Matrix" in role_matrix
    assert "role-triage-platform-admin" in role_matrix
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "# UI Review Signoff Dependency Board" in signoff_dependency
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in signoff_dependency
    assert "# UI Review Sign-off Log" in signoff_log
    assert "sig-run-detail-eng-lead" in signoff_log
    assert "# UI Review Sign-off SLA Dashboard" in signoff_sla
    assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in signoff_sla
    assert "# UI Review Sign-off Reminder Queue" in signoff_reminder
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in signoff_reminder
    assert "# UI Review Reminder Cadence Board" in reminder_cadence
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in reminder_cadence
    assert "# UI Review Sign-off Breach Board" in signoff_breach
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in signoff_breach
    assert "# UI Review Escalation Dashboard" in escalation_dashboard
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in escalation_dashboard
    assert "# UI Review Escalation Handoff Ledger" in handoff_ledger
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in handoff_ledger
    assert "# UI Review Handoff Ack Ledger" in handoff_ack
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in handoff_ack
    assert "# UI Review Owner Escalation Digest" in owner_digest
    assert "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in owner_digest
    assert "# UI Review Owner Workload Board" in owner_workload
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in owner_workload
    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in freeze_board
    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in freeze_trail
    assert "# UI Review Freeze Renewal Tracker" in freeze_renewal
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in freeze_renewal
    assert "# UI Review Blocker Log" in blocker_log
    assert "blk-run-detail-copy-final" in blocker_log
    assert "# UI Review Blocker Timeline" in blocker_timeline
    assert "evt-run-detail-copy-escalated" in blocker_timeline
    assert "# UI Review Exception Log" in exception_log
    assert "exc-blk-run-detail-copy-final" in exception_log
    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "# UI Review Owner Review Queue" in owner_review_queue
    assert "- Queue items: 6" in owner_review_queue
    assert "# UI Review Blocker Timeline Summary" in timeline_summary
    assert "- escalated: 1" in timeline_summary
    assert Path(artifacts.markdown_path).exists()
    assert Path(artifacts.html_path).exists()
    assert Path(artifacts.decision_log_path).exists()
    assert Path(artifacts.review_summary_board_path).exists()
    assert Path(artifacts.objective_coverage_board_path).exists()
    assert Path(artifacts.persona_readiness_board_path).exists()
    assert Path(artifacts.wireframe_readiness_board_path).exists()
    assert Path(artifacts.interaction_coverage_board_path).exists()
    assert Path(artifacts.open_question_tracker_path).exists()
    assert Path(artifacts.checklist_traceability_board_path).exists()
    assert Path(artifacts.decision_followup_tracker_path).exists()
    assert Path(artifacts.role_matrix_path).exists()
    assert Path(artifacts.role_coverage_board_path).exists()
    assert Path(artifacts.signoff_dependency_board_path).exists()
    assert Path(artifacts.signoff_log_path).exists()
    assert Path(artifacts.signoff_sla_dashboard_path).exists()
    assert Path(artifacts.signoff_reminder_queue_path).exists()
    assert Path(artifacts.reminder_cadence_board_path).exists()
    assert Path(artifacts.signoff_breach_board_path).exists()
    assert Path(artifacts.escalation_dashboard_path).exists()
    assert Path(artifacts.escalation_handoff_ledger_path).exists()
    assert Path(artifacts.handoff_ack_ledger_path).exists()
    assert Path(artifacts.owner_escalation_digest_path).exists()
    assert Path(artifacts.owner_workload_board_path).exists()
    assert Path(artifacts.blocker_log_path).exists()
    assert Path(artifacts.blocker_timeline_path).exists()
    assert Path(artifacts.freeze_exception_board_path).exists()
    assert Path(artifacts.freeze_approval_trail_path).exists()
    assert Path(artifacts.freeze_renewal_tracker_path).exists()
    assert Path(artifacts.exception_log_path).exists()
    assert Path(artifacts.exception_matrix_path).exists()
    assert Path(artifacts.audit_density_board_path).exists()
    assert Path(artifacts.owner_review_queue_path).exists()
    assert Path(artifacts.blocker_timeline_summary_path).exists()
    assert "Decision Log" in Path(artifacts.html_path).read_text()
    assert "Checklist Traceability Board" in Path(artifacts.html_path).read_text()
    assert "Decision Follow-up Tracker" in Path(artifacts.html_path).read_text()
    assert "Review Summary Board" in Path(artifacts.html_path).read_text()
    assert "Objective Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Persona Readiness Board" in Path(artifacts.html_path).read_text()
    assert "Wireframe Readiness Board" in Path(artifacts.html_path).read_text()
    assert "Interaction Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Open Question Tracker" in Path(artifacts.html_path).read_text()
    assert "Role Matrix" in Path(artifacts.html_path).read_text()
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in Path(artifacts.review_summary_board_path).read_text()
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in Path(artifacts.persona_readiness_board_path).read_text()
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in Path(artifacts.interaction_coverage_board_path).read_text()
    assert "Role Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Signoff Dependency Board" in Path(artifacts.html_path).read_text()
    assert "Sign-off Log" in Path(artifacts.html_path).read_text()
    assert "Sign-off SLA Dashboard" in Path(artifacts.html_path).read_text()
    assert "Sign-off Reminder Queue" in Path(artifacts.html_path).read_text()
    assert "Reminder Cadence Board" in Path(artifacts.html_path).read_text()
    assert "Sign-off Breach Board" in Path(artifacts.html_path).read_text()
    assert "Escalation Dashboard" in Path(artifacts.html_path).read_text()
    assert "Escalation Handoff Ledger" in Path(artifacts.html_path).read_text()
    assert "Handoff Ack Ledger" in Path(artifacts.html_path).read_text()
    assert "Owner Escalation Digest" in Path(artifacts.html_path).read_text()
    assert "Owner Workload Board" in Path(artifacts.html_path).read_text()
    assert "Blocker Log" in Path(artifacts.html_path).read_text()
    assert "Blocker Timeline" in Path(artifacts.html_path).read_text()
    assert "Review Freeze Exception Board" in Path(artifacts.html_path).read_text()
    assert "Freeze Approval Trail" in Path(artifacts.html_path).read_text()
    assert "Freeze Renewal Tracker" in Path(artifacts.html_path).read_text()
    assert "Review Exceptions" in Path(artifacts.html_path).read_text()
    assert "Review Exception Matrix" in Path(artifacts.html_path).read_text()
    assert "Audit Density Board" in Path(artifacts.html_path).read_text()
    assert "Owner Review Queue" in Path(artifacts.html_path).read_text()
    assert "Blocker Timeline Summary" in Path(artifacts.html_path).read_text()
    assert "dec-triage-handoff-density" in Path(artifacts.decision_log_path).read_text()
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in Path(artifacts.objective_coverage_board_path).read_text()
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in Path(artifacts.wireframe_readiness_board_path).read_text()
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in Path(artifacts.open_question_tracker_path).read_text()
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in Path(artifacts.checklist_traceability_board_path).read_text()
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in Path(artifacts.decision_followup_tracker_path).read_text()
    assert "role-run-detail-eng-lead" in Path(artifacts.role_matrix_path).read_text()
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in Path(artifacts.role_coverage_board_path).read_text()
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in Path(artifacts.signoff_dependency_board_path).read_text()
    assert "sig-queue-platform-admin" in Path(artifacts.signoff_log_path).read_text()
    assert "- at-risk: 1" in Path(artifacts.signoff_sla_dashboard_path).read_text()
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in Path(artifacts.signoff_reminder_queue_path).read_text()
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in Path(artifacts.reminder_cadence_board_path).read_text()
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in Path(artifacts.signoff_breach_board_path).read_text()
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in Path(artifacts.escalation_dashboard_path).read_text()
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in Path(artifacts.escalation_handoff_ledger_path).read_text()
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in Path(artifacts.handoff_ack_ledger_path).read_text()
    assert "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in Path(artifacts.owner_escalation_digest_path).read_text()
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in Path(artifacts.owner_workload_board_path).read_text()
    assert "blk-run-detail-copy-final" in Path(artifacts.blocker_log_path).read_text()
    assert "evt-run-detail-copy-opened" in Path(artifacts.blocker_timeline_path).read_text()
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in Path(artifacts.freeze_exception_board_path).read_text()
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in Path(artifacts.freeze_approval_trail_path).read_text()
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in Path(artifacts.freeze_renewal_tracker_path).read_text()
    assert "exc-blk-run-detail-copy-final" in Path(artifacts.exception_log_path).read_text()
    assert "- product-experience: blockers=1 signoffs=0 total=1" in Path(artifacts.exception_matrix_path).read_text()
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in Path(artifacts.audit_density_board_path).read_text()
    assert "- Queue items: 6" in Path(artifacts.owner_review_queue_path).read_text()
    assert "- escalated: 1" in Path(artifacts.blocker_timeline_summary_path).read_text()
