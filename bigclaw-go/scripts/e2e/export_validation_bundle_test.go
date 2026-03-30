package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func e2eScriptDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Dir(filename)
}

func runPythonAssertion(t *testing.T, body string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required for export_validation_bundle.py compatibility tests")
	}
	scriptPath := filepath.Join(e2eScriptDir(t), "export_validation_bundle.py")
	code := `
import types
import pathlib
import tempfile

module_path = pathlib.Path(r'''` + scriptPath + `''')
source = module_path.read_text(encoding='utf-8')
module = types.ModuleType('export_validation_bundle')
module.__file__ = str(module_path)
exec(compile('from __future__ import annotations\n' + source, str(module_path), 'exec'), module.__dict__)
` + body

	cmd := exec.Command("python3", "-c", code)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python assertion failed: %v\n%s", err, string(output))
	}
}

func TestRenderIndexSurfacesContinuationWorkflowModeAndOutcome(t *testing.T) {
	runPythonAssertion(t, `
summary = {
    'run_id': '20260316T140138Z',
    'generated_at': '2026-03-16T14:48:42.581505+00:00',
    'status': 'succeeded',
    'bundle_path': 'docs/reports/live-validation-runs/20260316T140138Z',
    'summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/summary.json',
    'closeout_commands': ['cd bigclaw-go && ./scripts/e2e/run_all.sh'],
    'local': {
        'enabled': True,
        'status': 'succeeded',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json',
        'canonical_report_path': 'docs/reports/sqlite-smoke-report.json',
    },
    'kubernetes': {
        'enabled': False,
        'status': 'skipped',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json',
        'canonical_report_path': 'docs/reports/kubernetes-live-smoke-report.json',
    },
    'ray': {
        'enabled': False,
        'status': 'skipped',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json',
        'canonical_report_path': 'docs/reports/ray-live-smoke-report.json',
    },
    'broker': {
        'enabled': False,
        'status': 'skipped',
        'configuration_state': 'not_configured',
        'reason': 'not_configured',
        'bundle_summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json',
        'canonical_summary_path': 'docs/reports/broker-validation-summary.json',
        'bundle_bootstrap_summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json',
        'canonical_bootstrap_summary_path': 'docs/reports/broker-bootstrap-review-summary.json',
        'validation_pack_path': 'docs/reports/broker-failover-fault-injection-validation-pack.md',
        'backend': None,
        'bootstrap_ready': False,
        'runtime_posture': 'contract_only',
        'live_adapter_implemented': False,
        'proof_boundary': 'broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof',
        'config_completeness': {
            'driver': False,
            'urls': False,
            'topic': False,
            'consumer_group': False,
        },
        'validation_errors': ['broker event log config missing driver, urls, topic'],
    },
}
continuation_gate = {
    'path': 'docs/reports/validation-bundle-continuation-policy-gate.json',
    'status': 'policy-hold',
    'recommendation': 'hold',
    'enforcement': {'mode': 'hold', 'outcome': 'hold', 'exit_code': 2},
    'summary': {'latest_run_id': '20260316T140138Z', 'failing_check_count': 2, 'workflow_exit_code': 2},
    'reviewer_path': {'digest_path': 'docs/reports/validation-bundle-continuation-digest.md'},
    'next_actions': ['rerun cd bigclaw-go && ./scripts/e2e/run_all.sh to refresh the latest validation bundle'],
}

index_text = module.render_index(summary, [], continuation_gate, [], [])
assert '- Workflow mode:' in index_text and 'hold' in index_text
assert '- Workflow outcome:' in index_text and 'hold' in index_text
assert '- Workflow exit code on current evidence:' in index_text and '2' in index_text
assert '### broker' in index_text
assert '- Configuration state:' in index_text and 'not_configured' in index_text
assert '- Runtime posture:' in index_text and 'contract_only' in index_text
assert '- Live adapter implemented:' in index_text and 'False' in index_text
assert '- Validation error:' in index_text and 'broker event log config missing driver, urls, topic' in index_text
assert '- Reason:' in index_text and 'not_configured' in index_text
`)
}

func TestBuildBrokerSectionWritesNotConfiguredSummaryWhenDisabled(t *testing.T) {
	runPythonAssertion(t, `
with tempfile.TemporaryDirectory() as tmpdir:
    tmp_root = pathlib.Path(tmpdir)
    bundle_dir = tmp_root / 'docs' / 'reports' / 'live-validation-runs' / 'run-1'
    bundle_dir.mkdir(parents=True, exist_ok=True)
    bootstrap_summary_path = tmp_root / 'docs' / 'reports' / 'broker-bootstrap-review-summary-source.json'
    bootstrap_summary_path.parent.mkdir(parents=True, exist_ok=True)
    bootstrap_summary_path.write_text(
        '{"ready": false, "runtime_posture": "contract_only", "live_adapter_implemented": false, '
        '"proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof", '
        '"config_completeness": {"driver": false, "urls": false, "topic": false, "consumer_group": false}, '
        '"validation_errors": ["broker event log config missing driver, urls, topic"]}',
        encoding='utf-8',
    )

    section = module.build_broker_section(
        enabled=False,
        backend='',
        root=tmp_root,
        bundle_dir=bundle_dir,
        bootstrap_summary_path=bootstrap_summary_path,
        report_path=None,
    )

    assert section['status'] == 'skipped'
    assert section['configuration_state'] == 'not_configured'
    assert section['reason'] == 'not_configured'
    assert section['bundle_summary_path'] == 'docs/reports/live-validation-runs/run-1/broker-validation-summary.json'
    assert section['runtime_posture'] == 'contract_only'
    assert not section['live_adapter_implemented']
    assert not section['config_completeness']['driver']
    assert (bundle_dir / 'broker-validation-summary.json').exists()
    assert (tmp_root / 'docs/reports/broker-validation-summary.json').exists()
    assert (bundle_dir / 'broker-bootstrap-review-summary.json').exists()
    assert (tmp_root / 'docs/reports/broker-bootstrap-review-summary.json').exists()
`)
}

func TestBuildComponentSectionEmitsK8sMatrixAndFailureRootCause(t *testing.T) {
	runPythonAssertion(t, `
with tempfile.TemporaryDirectory() as tmpdir:
    tmp_root = pathlib.Path(tmpdir)
    bundle_dir = tmp_root / 'docs' / 'reports' / 'live-validation-runs' / 'run-k8s'
    bundle_dir.mkdir(parents=True, exist_ok=True)

    report_path = tmp_root / 'tmp' / 'kubernetes-smoke-report.json'
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(
        '''
{
  "task": {
    "id": "kubernetes-smoke-failed",
    "required_executor": "kubernetes"
  },
  "status": {
    "state": "dead_letter",
    "latest_event": {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  },
  "events": [
    {
      "id": "evt-routed",
      "type": "scheduler.routed",
      "timestamp": "2026-03-23T11:00:00Z",
      "payload": {
        "reason": "browser workloads default to kubernetes executor"
      }
    },
    {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  ]
}
        '''.strip(),
        encoding='utf-8',
    )
    stdout_path = tmp_root / 'tmp' / 'kubernetes.stdout.log'
    stderr_path = tmp_root / 'tmp' / 'kubernetes.stderr.log'
    stdout_path.write_text('starting kubernetes smoke\n', encoding='utf-8')
    stderr_path.write_text('lease lost during replay\n', encoding='utf-8')

    section = module.build_component_section(
        name='kubernetes',
        enabled=True,
        root=tmp_root,
        bundle_dir=bundle_dir,
        report_path=report_path,
        stdout_path=stdout_path,
        stderr_path=stderr_path,
    )

    assert section['status'] == 'dead_letter'
    assert section['validation_matrix']['lane'] == 'k8s'
    assert section['validation_matrix']['executor'] == 'kubernetes'
    assert section['routing_reason'] == 'browser workloads default to kubernetes executor'
    assert section['failure_root_cause']['status'] == 'captured'
    assert section['failure_root_cause']['event_type'] == 'task.dead_letter'
    assert section['failure_root_cause']['message'] == 'lease lost during replay'
    assert section['failure_root_cause']['location'] == 'docs/reports/live-validation-runs/run-k8s/kubernetes.stderr.log'
`)
}

func TestRenderIndexSurfacesValidationMatrixAndFailureRootCause(t *testing.T) {
	runPythonAssertion(t, `
summary = {
    'run_id': '20260323T030000Z',
    'generated_at': '2026-03-23T03:10:00+00:00',
    'status': 'failed',
    'bundle_path': 'docs/reports/live-validation-runs/20260323T030000Z',
    'summary_path': 'docs/reports/live-validation-runs/20260323T030000Z/summary.json',
    'closeout_commands': ['cd bigclaw-go && ./scripts/e2e/run_all.sh'],
    'local': {
        'enabled': True,
        'status': 'succeeded',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
        'canonical_report_path': 'docs/reports/sqlite-smoke-report.json',
        'validation_matrix': {
            'lane': 'local',
            'executor': 'local',
            'enabled': True,
            'status': 'succeeded',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
        },
        'failure_root_cause': {
            'status': 'not_triggered',
            'event_type': 'task.completed',
            'message': '',
            'location': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
        },
    },
    'kubernetes': {
        'enabled': True,
        'status': 'dead_letter',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
        'canonical_report_path': 'docs/reports/kubernetes-live-smoke-report.json',
        'validation_matrix': {
            'lane': 'k8s',
            'executor': 'kubernetes',
            'enabled': True,
            'status': 'dead_letter',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
            'root_cause_event_type': 'task.dead_letter',
            'root_cause_location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
            'root_cause_message': 'lease lost during replay',
        },
        'failure_root_cause': {
            'status': 'captured',
            'event_type': 'task.dead_letter',
            'message': 'lease lost during replay',
            'location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
        },
    },
    'ray': {
        'enabled': True,
        'status': 'succeeded',
        'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
        'canonical_report_path': 'docs/reports/ray-live-smoke-report.json',
        'validation_matrix': {
            'lane': 'ray',
            'executor': 'ray',
            'enabled': True,
            'status': 'succeeded',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
        },
        'failure_root_cause': {
            'status': 'not_triggered',
            'event_type': 'task.completed',
            'message': '',
            'location': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
        },
    },
    'validation_matrix': [
        {
            'lane': 'local',
            'executor': 'local',
            'enabled': True,
            'status': 'succeeded',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
        },
        {
            'lane': 'k8s',
            'executor': 'kubernetes',
            'enabled': True,
            'status': 'dead_letter',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
            'root_cause_event_type': 'task.dead_letter',
            'root_cause_location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
            'root_cause_message': 'lease lost during replay',
        },
        {
            'lane': 'ray',
            'executor': 'ray',
            'enabled': True,
            'status': 'succeeded',
            'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
        },
    ],
}

index_text = module.render_index(summary, [], None, [], [])
assert '## Validation matrix' in index_text
assert '- Lane' in index_text and 'k8s' in index_text and 'kubernetes' in index_text and 'dead_letter' in index_text and 'kubernetes-live-smoke-report.json' in index_text
assert '- Lane' in index_text and 'root cause' in index_text and 'task.dead_letter' in index_text and 'kubernetes.stderr.log' in index_text and 'lease lost during replay' in index_text
assert '- Failure root cause:' in index_text and 'captured' in index_text and 'task.dead_letter' in index_text and 'kubernetes.stderr.log' in index_text
`)
}
