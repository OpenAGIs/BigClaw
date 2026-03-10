from bigclaw.design_system import (
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    DesignSystem,
    DesignToken,
    render_design_system_report,
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
    assert audit.token_orphans == ["radius.md"]
    assert audit.readiness_score == 40.0


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
    assert "- Button: readiness=stable docs=True a11y=True states=default, hover, disabled" in report
    assert "- Orphan tokens: none" in report
