import bigclaw
import bigclaw.reports
import bigclaw.runtime


def test_runtime_compatibility_surface_resolves_via_legacy_shim() -> None:
    assert bigclaw.runtime is bigclaw.legacy_shim
    assert bigclaw.runtime.__file__.endswith("src/bigclaw/legacy_shim.py")


def test_reports_compatibility_surface_resolves_via_legacy_source_asset() -> None:
    assert bigclaw.reports.__file__.endswith("src/bigclaw/_legacy/reports.legacy")
    assert hasattr(bigclaw.reports, "render_repo_sync_audit_report")
