from bigclaw.__main__ import LEGACY_ENTRYPOINT_NOTICE, build_parser


def test_legacy_python_entrypoint_is_marked_for_migration_only():
    parser = build_parser()
    assert parser.description == "BigClaw legacy Python developer utilities"
    assert parser.epilog == LEGACY_ENTRYPOINT_NOTICE
    assert "Go mainline" in LEGACY_ENTRYPOINT_NOTICE
    assert "migration or legacy-path validation" in LEGACY_ENTRYPOINT_NOTICE
