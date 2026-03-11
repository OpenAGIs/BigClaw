from bigclaw.design_system import (
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationNode,
    NavigationRoute,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
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
