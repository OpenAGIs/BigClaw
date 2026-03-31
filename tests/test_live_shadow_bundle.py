import json
from pathlib import Path




def test_checked_in_live_shadow_bundle_matches_expected_shape() -> None:
    manifest = json.loads(Path('bigclaw-go/docs/reports/live-shadow-index.json').read_text(encoding='utf-8'))
    latest = manifest['latest']
    rollup = manifest['drift_rollup']

    assert latest['run_id']
    assert latest['status'] == 'parity-ok'
    assert latest['severity'] == 'none'
    assert latest['summary']['drift_detected_count'] == 0
    assert latest['summary']['stale_inputs'] == 0
    assert latest['artifacts']['live_shadow_scorecard_path'].endswith('live-shadow-mirror-scorecard.json')
    assert rollup['status'] == 'parity-ok'
    assert rollup['summary']['highest_severity'] == 'none'
    assert rollup['summary']['latest_run_id'] == latest['run_id']
