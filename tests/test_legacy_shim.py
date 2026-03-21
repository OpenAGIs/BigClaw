from pathlib import Path

from bigclaw.legacy_shim import (
    LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag,
    build_github_sync_args,
    build_refill_args,
    build_workspace_bootstrap_args,
    build_workspace_validate_args,
    translate_workspace_validate_args,
)


REPO_ROOT = Path('/repo')


def test_append_missing_flag_preserves_existing_values():
    assert append_missing_flag(['--repo-url', 'ssh://example/repo.git'], '--repo-url', 'git@github.com:OpenAGIs/BigClaw.git') == ['--repo-url', 'ssh://example/repo.git']
    assert append_missing_flag(['--cache-key=openagis-bigclaw'], '--cache-key', 'other') == ['--cache-key=openagis-bigclaw']


def test_workspace_bootstrap_wrapper_injects_go_defaults():
    argv = build_workspace_bootstrap_args(REPO_ROOT, ['--workspace', '/tmp/ws'])
    assert argv[:4] == ['bash', '/repo/scripts/ops/bigclawctl', 'workspace', 'bootstrap']
    assert '--repo-url' in argv
    assert 'git@github.com:OpenAGIs/BigClaw.git' in argv
    assert '--cache-key' in argv
    assert 'openagis-bigclaw' in argv


def test_workspace_validate_wrapper_translates_legacy_flags():
    translated = translate_workspace_validate_args([
        '--repo-url', 'git@github.com:OpenAGIs/BigClaw.git',
        '--workspace-root', '/tmp/ws',
        '--issues', 'BIG-1', 'BIG-2',
        '--report-file', '/tmp/report.md',
        '--no-cleanup',
        '--json',
    ])
    assert translated == [
        '--repo-url', 'git@github.com:OpenAGIs/BigClaw.git',
        '--workspace-root', '/tmp/ws',
        '--issues', 'BIG-1,BIG-2',
        '--report', '/tmp/report.md',
        '--cleanup=false',
        '--json',
    ]
    argv = build_workspace_validate_args(REPO_ROOT, ['--issues', 'BIG-1', 'BIG-2'])
    assert argv[:4] == ['bash', '/repo/scripts/ops/bigclawctl', 'workspace', 'validate']
    assert argv[4:] == ['--issues', 'BIG-1,BIG-2']


def test_github_sync_and_refill_wrappers_target_go_shim():
    assert build_github_sync_args(REPO_ROOT, ['status', '--json']) == ['bash', '/repo/scripts/ops/bigclawctl', 'github-sync', 'status', '--json']
    assert build_refill_args(REPO_ROOT, ['--apply']) == ['bash', '/repo/scripts/ops/bigclawctl', 'refill', '--apply']
    assert 'compatibility shim during migration' in LEGACY_PYTHON_WRAPPER_NOTICE
