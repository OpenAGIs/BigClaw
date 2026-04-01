import importlib


def test_purged_top_level_modules_remain_importable_via_package_shims() -> None:
    audit_events = importlib.import_module("bigclaw.audit_events")
    collaboration = importlib.import_module("bigclaw.collaboration")
    governance = importlib.import_module("bigclaw.governance")

    assert audit_events.FLOW_HANDOFF_EVENT == "execution.flow_handoff"
    assert collaboration.merge_collaboration_threads is not None
    assert governance.ScopeFreezeGovernance is not None
