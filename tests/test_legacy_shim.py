from pathlib import Path

from bigclaw.legacy_shim import (
    LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag,
    build_bigclawctl_exec_args,
    build_workspace_bootstrap_args,
    build_workspace_validate_args,
    repo_root_from_script,
    translate_workspace_validate_args,
)


def test_append_missing_flag_and_exec_arg_builders_are_deterministic() -> None:
    repo_root = Path("/tmp/bigclaw")

    args = append_missing_flag(["--foo", "bar"], "--repo-url", "git@example.com/repo.git")
    argv = build_bigclawctl_exec_args(repo_root, ["workspace", "bootstrap"], ["--issue", "BIG-1"])

    assert "--repo-url" in args
    assert argv == ["bash", "/tmp/bigclaw/scripts/ops/bigclawctl", "workspace", "bootstrap", "--issue", "BIG-1"]
    assert "compatibility shim" in LEGACY_PYTHON_WRAPPER_NOTICE


def test_workspace_translation_helpers_preserve_expected_flags() -> None:
    forwarded = ["--report-file", "out.json", "--no-cleanup", "--issues", "BIG-1", "BIG-2", "--verbose"]
    translated = translate_workspace_validate_args(forwarded)
    repo_root = Path("/tmp/bigclaw")
    bootstrap = build_workspace_bootstrap_args(repo_root, ["--issue", "BIG-1"])
    validate = build_workspace_validate_args(repo_root, forwarded)

    assert translated == ["--report", "out.json", "--cleanup=false", "--issues", "BIG-1,BIG-2", "--verbose"]
    assert "--repo-url" in bootstrap and "--cache-key" in bootstrap
    assert validate[:3] == ["bash", "/tmp/bigclaw/scripts/ops/bigclawctl", "workspace"]
    assert repo_root_from_script("/tmp/bigclaw/src/bigclaw/legacy_shim.py").name == repo_root.name
