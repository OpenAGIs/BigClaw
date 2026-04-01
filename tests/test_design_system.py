from bigclaw.design_system import (
    AuditRequirement,
    CommandAction,
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleInteractionAudit,
    ConsoleInteractionAuditor,
    ConsoleInteractionDraft,
    ConsoleSurface,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DataAccuracyCheck,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    FilterDefinition,
    GlobalAction,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationItem,
    NavigationNode,
    NavigationRoute,
    PerformanceBudget,
    RolePermissionScenario,
    SurfaceInteractionContract,
    SurfacePermissionRule,
    SurfaceState,
    UIAcceptanceAudit,
    UIAcceptanceLibrary,
    UIAcceptanceSuite,
    UsabilityJourney,
    build_big_4203_console_interaction_draft,
    render_console_interaction_report,
    render_console_ia_report,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
    render_ui_acceptance_report,
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
                    GlobalAction(action_id="bulk-approve", label="Bulk Approve", placement="topbar"),
                    GlobalAction(action_id="reassign", label="Reassign", placement="topbar"),
                ],
                filters=[FilterDefinition(name="Status", field="status", control="select", options=["all"])],
                states=[
                    SurfaceState(name="default"),
                    SurfaceState(name="loading", allowed_actions=["reassign"]),
                    SurfaceState(name="empty"),
                    SurfaceState(name="error", allowed_actions=["reassign"]),
                ],
            ),
            ConsoleSurface(
                name="Run Detail",
                route="/runs/detail",
                navigation_section="Operate",
                top_bar_actions=[GlobalAction(action_id="share", label="Share", placement="topbar")],
                filters=[FilterDefinition(name="Run", field="run_id", control="text")],
                states=[
                    SurfaceState(name="default"),
                    SurfaceState(name="loading"),
                    SurfaceState(name="empty"),
                    SurfaceState(name="error"),
                ],
            ),
            ConsoleSurface(
                name="Triage",
                route="/triage",
                navigation_section="Operate",
                top_bar_actions=[GlobalAction(action_id="triage", label="Triage", placement="topbar")],
                filters=[FilterDefinition(name="Severity", field="severity", control="select", options=["all"])],
                states=[
                    SurfaceState(name="default"),
                    SurfaceState(name="loading"),
                    SurfaceState(name="empty"),
                    SurfaceState(name="error"),
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
                        allowed_roles=["operator"],
                        denied_roles=["viewer"],
                        audit_event="ui.access.overview",
                    ),
                ),
                SurfaceInteractionContract(
                    surface_name="Queue",
                    required_action_ids=["bulk-approve", "reassign", "assign-owner"],
                    permission_rule=SurfacePermissionRule(
                        allowed_roles=["operator"],
                        denied_roles=[],
                        audit_event="",
                    ),
                    requires_batch_actions=True,
                ),
                SurfaceInteractionContract(
                    surface_name="Run Detail",
                    required_action_ids=["share", "timeline-sync"],
                    permission_rule=SurfacePermissionRule(),
                ),
                SurfaceInteractionContract(
                    surface_name="Ghost Surface",
                    required_action_ids=["ghost-action"],
                ),
            ],
        )

    audit = ConsoleInteractionAuditor().audit(draft)

    assert audit.release_ready is False
    assert audit.missing_surfaces == ["Ghost Surface"]
    assert audit.surfaces_missing_actions["Queue"] == ["assign-owner"]
    assert audit.surfaces_missing_actions["Run Detail"] == ["timeline-sync"]
    assert audit.surfaces_missing_batch_actions == ["Queue"]
    assert audit.permission_gaps["Queue"] == ["denied-roles", "audit-event"]
    assert audit.permission_gaps["Run Detail"] == ["allowed-roles", "denied-roles", "audit-event"]
    assert audit.uncovered_roles == []


def test_console_interaction_audit_round_trip_preserves_findings() -> None:
    audit = ConsoleInteractionAudit(
        name="BIG-4203 Four Critical Pages",
        version="v1",
        contract_count=4,
        missing_surfaces=["Ghost Surface"],
        surfaces_missing_actions={"Queue": ["assign-owner"]},
        surfaces_missing_batch_actions=["Queue"],
        permission_gaps={"Run Detail": ["allowed-roles", "denied-roles", "audit-event"]},
    )

    restored = ConsoleInteractionAudit.from_dict(audit.to_dict())

    assert restored == audit


def test_render_console_interaction_report_summarizes_release_gaps() -> None:
    draft = build_big_4203_console_interaction_draft()
    audit = ConsoleInteractionAuditor().audit(draft)

    report = render_console_interaction_report(draft, audit)

    assert "# Console Interaction Draft Report" in report
    assert "- Name: BIG-4203 Four Critical Pages" in report
    assert "- Version: v4.0-design-sprint" in report
    assert "- Release Ready: True" in report
    assert "- Missing surfaces: none" in report
    assert "- Queue: route=/queue required_actions=drill-down, export, audit available_actions=drill-down, export, audit, bulk-approve filters=2 states=default, loading, empty, error batch=required permissions=complete" in report
