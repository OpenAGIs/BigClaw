import importlib


def test_purged_top_level_modules_remain_importable_via_package_shims() -> None:
    audit_events = importlib.import_module("bigclaw.audit_events")
    collaboration = importlib.import_module("bigclaw.collaboration")
    console_ia = importlib.import_module("bigclaw.console_ia")
    design_system = importlib.import_module("bigclaw.design_system")
    governance = importlib.import_module("bigclaw.governance")
    evaluation = importlib.import_module("bigclaw.evaluation")

    assert audit_events.FLOW_HANDOFF_EVENT == "execution.flow_handoff"
    assert collaboration.merge_collaboration_threads is not None
    assert console_ia.ConsoleIAAuditor is not None
    assert design_system.DesignSystemAudit is not None
    assert governance.ScopeFreezeGovernance is not None
    assert evaluation.BenchmarkRunner is not None
