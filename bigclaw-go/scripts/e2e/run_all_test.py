#!/usr/bin/env python3
import json
import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SOURCE_RUN_ALL = REPO_ROOT / 'scripts' / 'e2e' / 'run_all.sh'


class RunAllTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmpdir.cleanup)
        self.root = Path(self.tmpdir.name)
        scripts_dir = self.root / 'scripts' / 'e2e'
        scripts_dir.mkdir(parents=True, exist_ok=True)
        run_all_path = scripts_dir / 'run_all.sh'
        run_all_path.write_text(SOURCE_RUN_ALL.read_text(encoding='utf-8'), encoding='utf-8')
        run_all_path.chmod(run_all_path.stat().st_mode | stat.S_IXUSR)

    def write_file(self, relpath: str, content: str, *, executable: bool = False) -> None:
        path = self.root / relpath
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(textwrap.dedent(content), encoding='utf-8')
        if executable:
            path.chmod(path.stat().st_mode | stat.S_IXUSR)

    def install_stubs(self) -> None:
        self.write_file(
            'scripts/e2e/run_task_smoke',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            report_path = pathlib.Path(args[args.index('--report-path') + 1])
            report_path.parent.mkdir(parents=True, exist_ok=True)
            report_path.write_text(json.dumps({'status': 'succeeded', 'all_ok': True}), encoding='utf-8')
            """,
            executable=True,
        )
        self.write_file(
            'scripts/e2e/broker_bootstrap_summary.go',
            """\
            package main

            import (
                "flag"
                "os"
            )

            func main() {
                output := flag.String("output", "", "output")
                flag.Parse()
                if err := os.WriteFile(*output, []byte("{\\"ready\\":false,\\"runtime_posture\\":\\"contract_only\\",\\"live_adapter_implemented\\":false}\\n"), 0o644); err != nil {
                    panic(err)
                }
            }
            """,
        )
        self.write_file(
            'scripts/e2e/export_validation_bundle',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            root = pathlib.Path(args[args.index('--go-root') + 1])
            bundle_dir = root / args[args.index('--bundle-dir') + 1]
            bundle_dir.mkdir(parents=True, exist_ok=True)
            calls_path = root / 'calls.jsonl'
            gate_path = root / 'docs/reports/validation-bundle-continuation-policy-gate.json'
            payload = {
                'gate_exists': gate_path.exists(),
                'run_broker': args[args.index('--run-broker') + 1],
                'broker_backend': args[args.index('--broker-backend') + 1],
                'broker_report_path': args[args.index('--broker-report-path') + 1],
                'broker_bootstrap_summary_path': args[args.index('--broker-bootstrap-summary-path') + 1],
            }
            with calls_path.open('a', encoding='utf-8') as handle:
                handle.write(json.dumps(payload) + '\\n')
            """,
            executable=True,
        )
        self.write_file(
            'scripts/e2e/validation_bundle_continuation_scorecard',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            output = pathlib.Path(args[args.index('--output') + 1])
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
            """,
            executable=True,
        )
        self.write_file(
            'scripts/e2e/validation_bundle_continuation_policy_gate',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            mode = args[args.index('--enforcement-mode') + 1]
            output = pathlib.Path(args[args.index('--output') + 1])
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(json.dumps({'status': 'policy-go', 'recommendation': 'go', 'enforcement': {'mode': mode, 'outcome': 'pass', 'exit_code': 0}}), encoding='utf-8')
            """,
            executable=True,
        )

    def test_run_all_rerenders_bundle_after_gate_refresh(self) -> None:
        self.install_stubs()
        env = os.environ.copy()
        env.update(
            {
                'BIGCLAW_E2E_RUN_KUBERNETES': '0',
                'BIGCLAW_E2E_RUN_RAY': '0',
                'BIGCLAW_E2E_RUN_LOCAL': '1',
                'BIGCLAW_E2E_RUN_BROKER': '1',
                'BIGCLAW_E2E_BROKER_BACKEND': 'stub',
                'BIGCLAW_E2E_BROKER_REPORT_PATH': 'docs/reports/broker-failover-stub-report.json',
            }
        )

        result = subprocess.run(
            [str(self.root / 'scripts' / 'e2e' / 'run_all.sh')],
            cwd=self.root,
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        calls = [
            json.loads(line)
            for line in (self.root / 'calls.jsonl').read_text(encoding='utf-8').splitlines()
            if line.strip()
        ]
        self.assertEqual(len(calls), 2)
        self.assertFalse(calls[0]['gate_exists'])
        self.assertTrue(calls[1]['gate_exists'])
        self.assertEqual(calls[0]['run_broker'], '1')
        self.assertEqual(calls[0]['broker_backend'], 'stub')
        self.assertEqual(calls[0]['broker_report_path'], 'docs/reports/broker-failover-stub-report.json')
        self.assertEqual(calls[0]['broker_bootstrap_summary_path'], 'docs/reports/broker-bootstrap-review-summary.json')

    def test_run_all_defaults_to_hold_mode(self) -> None:
        self.install_stubs()
        self.write_file(
            'scripts/e2e/validation_bundle_continuation_policy_gate',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            mode = args[args.index('--enforcement-mode') + 1]
            output = pathlib.Path(args[args.index('--output') + 1])
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
            """,
            executable=True,
        )

        env = os.environ.copy()
        env.update(
            {
                'BIGCLAW_E2E_RUN_KUBERNETES': '0',
                'BIGCLAW_E2E_RUN_RAY': '0',
                'BIGCLAW_E2E_RUN_LOCAL': '1',
            }
        )

        result = subprocess.run(
            [str(self.root / 'scripts' / 'e2e' / 'run_all.sh')],
            cwd=self.root,
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        gate = json.loads(
            (self.root / 'docs' / 'reports' / 'validation-bundle-continuation-policy-gate.json').read_text(
                encoding='utf-8'
            )
        )
        self.assertEqual(gate['enforcement']['mode'], 'hold')

    def test_legacy_enforce_alias_still_maps_to_fail_mode(self) -> None:
        self.install_stubs()
        self.write_file(
            'scripts/e2e/validation_bundle_continuation_policy_gate',
            """\
            #!/usr/bin/env python3
            import json
            import pathlib
            import sys

            args = sys.argv[1:]
            mode = args[args.index('--enforcement-mode') + 1]
            output = pathlib.Path(args[args.index('--output') + 1])
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
            """,
            executable=True,
        )

        env = os.environ.copy()
        env.update(
            {
                'BIGCLAW_E2E_RUN_KUBERNETES': '0',
                'BIGCLAW_E2E_RUN_RAY': '0',
                'BIGCLAW_E2E_RUN_LOCAL': '1',
                'BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE': '1',
            }
        )

        result = subprocess.run(
            [str(self.root / 'scripts' / 'e2e' / 'run_all.sh')],
            cwd=self.root,
            env=env,
            capture_output=True,
            text=True,
            check=False,
        )

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        gate = json.loads(
            (self.root / 'docs' / 'reports' / 'validation-bundle-continuation-policy-gate.json').read_text(
                encoding='utf-8'
            )
        )
        self.assertEqual(gate['enforcement']['mode'], 'fail')


if __name__ == '__main__':
    unittest.main()
