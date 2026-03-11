from bigclaw.saved_views import (
    AlertDigestSubscription,
    SavedView,
    SavedViewCatalog,
    SavedViewCatalogAudit,
    SavedViewFilter,
    SavedViewLibrary,
    render_saved_view_report,
)


def test_saved_view_catalog_round_trip_preserves_manifest_shape() -> None:
    catalog = SavedViewCatalog(
        name="BigClaw Views",
        version="v3",
        views=[
            SavedView(
                view_id="view-ops-needs-approval",
                name="Needs Approval",
                route="/operations/overview",
                owner="ops",
                visibility="team",
                filters=[
                    SavedViewFilter(field="status", operator="=", value="needs-approval"),
                    SavedViewFilter(field="priority", operator="in", value="p0,p1"),
                ],
                sort_by="-updated_at",
                pinned=True,
                is_default=True,
            )
        ],
        subscriptions=[
            AlertDigestSubscription(
                subscription_id="digest-ops-daily",
                saved_view_id="view-ops-needs-approval",
                channel="email",
                cadence="daily",
                recipients=["ops@bigclaw.dev"],
            )
        ],
    )

    restored = SavedViewCatalog.from_dict(catalog.to_dict())

    assert restored == catalog


def test_saved_view_catalog_audit_surfaces_configuration_gaps() -> None:
    catalog = SavedViewCatalog(
        name="BigClaw Views",
        version="v3",
        views=[
            SavedView(
                view_id="view-a",
                name="Needs Approval",
                route="/operations/overview",
                owner="ops",
                visibility="team",
                filters=[SavedViewFilter(field="status", operator="=", value="needs-approval")],
                is_default=True,
            ),
            SavedView(
                view_id="view-b",
                name="Needs Approval",
                route="/operations/overview",
                owner="ops",
                visibility="company",
                is_default=True,
            ),
        ],
        subscriptions=[
            AlertDigestSubscription(
                subscription_id="digest-missing-view",
                saved_view_id="view-z",
                channel="pagerduty",
                cadence="monthly",
            )
        ],
    )

    audit = SavedViewLibrary().audit(catalog)

    assert audit.duplicate_view_names == {"/operations/overview:ops": ["Needs Approval"]}
    assert audit.invalid_visibility_views == ["Needs Approval"]
    assert audit.views_missing_filters == ["Needs Approval"]
    assert audit.duplicate_default_views == {"/operations/overview:ops": ["Needs Approval", "Needs Approval"]}
    assert audit.orphan_subscriptions == ["digest-missing-view"]
    assert audit.subscriptions_missing_recipients == ["digest-missing-view"]
    assert audit.subscriptions_with_invalid_channel == ["digest-missing-view"]
    assert audit.subscriptions_with_invalid_cadence == ["digest-missing-view"]
    assert audit.readiness_score == 0.0


def test_saved_view_catalog_audit_round_trip_preserves_findings() -> None:
    audit = SavedViewCatalogAudit(
        catalog_name="BigClaw Views",
        version="v3",
        view_count=2,
        subscription_count=1,
        duplicate_view_names={"/operations/overview:ops": ["Needs Approval"]},
        invalid_visibility_views=["Needs Approval"],
        views_missing_filters=["Needs Approval"],
        duplicate_default_views={"/operations/overview:ops": ["Needs Approval", "Needs Approval"]},
        orphan_subscriptions=["digest-missing-view"],
        subscriptions_missing_recipients=["digest-missing-view"],
        subscriptions_with_invalid_channel=["digest-missing-view"],
        subscriptions_with_invalid_cadence=["digest-missing-view"],
    )

    restored = SavedViewCatalogAudit.from_dict(audit.to_dict())

    assert restored == audit


def test_render_saved_view_report_summarizes_views_and_digest_coverage() -> None:
    catalog = SavedViewCatalog(
        name="BigClaw Views",
        version="v3",
        views=[
            SavedView(
                view_id="view-ops-needs-approval",
                name="Needs Approval",
                route="/operations/overview",
                owner="ops",
                visibility="team",
                filters=[
                    SavedViewFilter(field="status", operator="=", value="needs-approval"),
                ],
                sort_by="-updated_at",
                pinned=True,
                is_default=True,
            )
        ],
        subscriptions=[
            AlertDigestSubscription(
                subscription_id="digest-ops-daily",
                saved_view_id="view-ops-needs-approval",
                channel="email",
                cadence="daily",
                recipients=["ops@bigclaw.dev"],
            )
        ],
    )

    audit = SavedViewLibrary().audit(catalog)
    report = render_saved_view_report(catalog, audit)

    assert "# Saved Views & Alert Digests Report" in report
    assert "- Saved Views: 1" in report
    assert (
        "- Needs Approval: route=/operations/overview owner=ops visibility=team "
        "filters=status=needs-approval sort=-updated_at pinned=True default=True"
    ) in report
    assert (
        "- digest-ops-daily: view=view-ops-needs-approval channel=email cadence=daily "
        "recipients=ops@bigclaw.dev include_empty=False muted=False"
    ) in report
    assert "- Duplicate view names: none" in report
    assert "- Orphan subscriptions: none" in report
