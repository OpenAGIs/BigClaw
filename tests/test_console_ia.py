from bigclaw.console_ia import (
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ConsoleSurface,
    FilterDefinition,
    GlobalAction,
    NavigationItem,
    SurfaceState,
    render_console_ia_report,
)


def test_console_ia_round_trip_preserves_manifest_shape() -> None:
    architecture = ConsoleIA(
        name="BigClaw Console IA",
        version="v3",
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
    assert "- Navigation Items: 1" in report
    assert "- Overview: route=/overview filters=Team actions=Refresh states=default, loading, empty, error missing_states=none states_without_actions=none unresolved_state_actions=none" in report
    assert "- Surfaces missing filters: none" in report
    assert "- Undefined state actions: none" in report
