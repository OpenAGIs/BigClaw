import os
import subprocess
import sys
from pathlib import Path

from bigclaw.legacy_shim import (
    LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag,
    build_github_sync_args,
    build_refill_args,
    build_workspace_bootstrap_args,
    build_workspace_runtime_bootstrap_args,
    build_workspace_validate_args,
    repo_root_from_script,
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


def test_workspace_runtime_wrapper_targets_go_shim():
    assert build_workspace_runtime_bootstrap_args(REPO_ROOT, ['bootstrap', '--json']) == [
        'bash', '/repo/scripts/ops/bigclawctl', 'workspace', 'bootstrap', '--json'
    ]


def test_repo_root_from_script_climbs_to_repository_root():
    assert repo_root_from_script('/repo/scripts/ops/bigclaw_refill_queue.py') == REPO_ROOT


def test_dev_smoke_shim_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/dev_smoke.py'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'smoke_ok local' in result.stdout


def test_refill_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/ops/bigclaw_refill_queue.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl refill [flags]' in result.stdout


def test_create_issues_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/create_issues.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl create-issues [flags]' in result.stdout


def test_github_sync_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/ops/bigclaw_github_sync.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl github-sync <install|status|sync> [flags]' in result.stdout


def test_workspace_bootstrap_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/ops/bigclaw_workspace_bootstrap.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]' in result.stdout


def test_symphony_workspace_bootstrap_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/ops/symphony_workspace_bootstrap.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]' in result.stdout


def test_symphony_workspace_validate_shim_help_runs_without_pythonpath():
    repo_root = Path(__file__).resolve().parents[1]
    env = dict(os.environ)
    env.pop('PYTHONPATH', None)
    result = subprocess.run(
        [sys.executable, 'scripts/ops/symphony_workspace_validate.py', '--help'],
        cwd=repo_root,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    assert 'usage: bigclawctl workspace validate [flags]' in result.stdout
