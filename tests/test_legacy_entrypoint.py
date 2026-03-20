from bigclaw.__main__ import LEGACY_ENTRYPOINT_NOTICE, build_parser
from bigclaw.service import LEGACY_SERVER_NOTICE, legacy_server_banner


def test_legacy_python_entrypoint_is_marked_for_migration_only():
    parser = build_parser()
    assert parser.description == "BigClaw legacy Python developer utilities"
    assert parser.epilog == LEGACY_ENTRYPOINT_NOTICE
    assert "Go mainline" in LEGACY_ENTRYPOINT_NOTICE
    assert "migration or legacy-path validation" in LEGACY_ENTRYPOINT_NOTICE


def test_legacy_python_server_banner_points_to_go_mainline():
    banner = legacy_server_banner("127.0.0.1", 8008, ".")
    assert banner.startswith(LEGACY_SERVER_NOTICE)
    assert "go run ./cmd/bigclawd" in banner
    assert "migration or legacy-path validation" in banner
