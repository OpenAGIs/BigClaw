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
from bigclaw.design_system import CommandAction, ConsoleCommandEntry, ConsoleTopBar


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
