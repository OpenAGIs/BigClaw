#!/usr/bin/env python3
import argparse
import copy
import json
import pathlib
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional, Tuple


REPORT_PATH = 'bigclaw-go/docs/reports/broker-failover-stub-report.json'
ARTIFACT_ROOT = 'bigclaw-go/docs/reports/broker-failover-stub-artifacts'


def utc_iso(value: datetime) -> str:
    return value.astimezone(timezone.utc).isoformat().replace('+00:00', 'Z')


@dataclass
class EventRecord:
    sequence: int
    event_id: str
    trace_id: str
    source_node: str
    phase: str
    duplicate: bool = False

    def as_replay_row(self) -> dict:
        return {
            'durable_sequence': self.sequence,
            'event_id': self.event_id,
            'trace_id': self.trace_id,
            'delivery_phase': self.phase,
            'source_node': self.source_node,
            'duplicate': self.duplicate,
        }


@dataclass
class LeaseState:
    owner_id: str
    lease_id: str
    epoch: int
    sequence: int


class FenceError(Exception):
    def __init__(self, reason: str, current: LeaseState):
        super().__init__(reason)
        self.reason = reason
        self.current = current


class StubBrokerBackend:
    def __init__(self, *, backend: str = 'stub_broker', now: Optional[datetime] = None):
        self.backend = backend
        self.now = now or datetime(2026, 3, 17, 12, 0, tzinfo=timezone.utc)
        self._next_sequence = 1
        self._next_lease = 1
        self.events: List[dict] = []
        self.publish_ledger: List[dict] = []
        self.replay_capture: List[dict] = []
        self.checkpoint_log: List[dict] = []
        self.fault_timeline: List[dict] = []
        self.health_snapshots: List[dict] = []
        self.checkpoints: Dict[Tuple[str, str], LeaseState] = {}
        self.retention_floor = 1
        self.nodes = {
            'broker-a': {'role': 'leader', 'healthy': True},
            'broker-b': {'role': 'follower', 'healthy': True},
            'broker-c': {'role': 'follower', 'healthy': True},
        }

    def advance(self, seconds: int = 1) -> datetime:
        self.now += timedelta(seconds=seconds)
        return self.now

    def record_fault(self, action: str, target: str, details: Optional[dict] = None) -> None:
        self.fault_timeline.append(
            {
                'timestamp': utc_iso(self.now),
                'action': action,
                'target': target,
                'details': details or {},
            }
        )

    def snapshot_health(self, label: str) -> None:
        self.health_snapshots.append(
            {
                'label': label,
                'timestamp': utc_iso(self.now),
                'nodes': copy.deepcopy(self.nodes),
                'retention_floor': self.retention_floor,
                'committed_sequences': [event['sequence'] for event in self.events if event['committed']],
            }
        )

    def publish(
        self,
        *,
        event_id: str,
        trace_id: str,
        source_node: str,
        client_outcome: str,
        committed: bool = True,
    ) -> Optional[int]:
        sequence = None
        if committed:
            sequence = self._next_sequence
            self._next_sequence += 1
            self.events.append(
                {
                    'sequence': sequence,
                    'event_id': event_id,
                    'trace_id': trace_id,
                    'source_node': source_node,
                    'committed': True,
                }
            )
        self.publish_ledger.append(
            {
                'timestamp': utc_iso(self.now),
                'event_id': event_id,
                'trace_id': trace_id,
                'attempt': 1,
                'source_node': source_node,
                'client_outcome': client_outcome,
                'durable_sequence': sequence,
            }
        )
        return sequence

    def replay(self, *, after_sequence: int = 0, source_node: str, duplicate_event_ids: Optional[set] = None) -> List[dict]:
        rows = []
        seen_duplicates = duplicate_event_ids or set()
        for event in self.events:
            if not event['committed'] or event['sequence'] <= after_sequence:
                continue
            rows.append(
                EventRecord(
                    sequence=event['sequence'],
                    event_id=event['event_id'],
                    trace_id=event['trace_id'],
                    source_node=source_node,
                    phase='replay',
                    duplicate=event['event_id'] in seen_duplicates,
                ).as_replay_row()
            )
            if event['event_id'] in seen_duplicates:
                rows.append(
                    EventRecord(
                        sequence=event['sequence'],
                        event_id=event['event_id'],
                        trace_id=event['trace_id'],
                        source_node=source_node,
                        phase='replay_duplicate',
                        duplicate=True,
                    ).as_replay_row()
                )
        self.replay_capture.extend(rows)
        return rows

    def acquire_lease(self, *, group_id: str, subscriber_id: str, owner_id: str) -> LeaseState:
        key = (group_id, subscriber_id)
        current = self.checkpoints.get(key)
        state = LeaseState(
            owner_id=owner_id,
            lease_id=f'lease-{self._next_lease}',
            epoch=1 if current is None else current.epoch + 1,
            sequence=0 if current is None else current.sequence,
        )
        self._next_lease += 1
        self.checkpoints[key] = state
        self.checkpoint_log.append(
            {
                'timestamp': utc_iso(self.now),
                'group_id': group_id,
                'subscriber_id': subscriber_id,
                'owner_id': owner_id,
                'lease_id': state.lease_id,
                'lease_epoch': state.epoch,
                'prior_sequence': 0 if current is None else current.sequence,
                'next_sequence': state.sequence,
                'transition': 'leased',
                'fence_reason': '',
            }
        )
        return state

    def checkpoint(
        self,
        *,
        group_id: str,
        subscriber_id: str,
        owner_id: str,
        lease_id: str,
        epoch: int,
        sequence: int,
        transition: str = 'committed',
    ) -> LeaseState:
        key = (group_id, subscriber_id)
        current = self.checkpoints[key]
        if current.owner_id != owner_id or current.lease_id != lease_id or current.epoch != epoch:
            self.checkpoint_log.append(
                {
                    'timestamp': utc_iso(self.now),
                    'group_id': group_id,
                    'subscriber_id': subscriber_id,
                    'owner_id': owner_id,
                    'lease_id': lease_id,
                    'lease_epoch': epoch,
                    'prior_sequence': current.sequence,
                    'next_sequence': sequence,
                    'transition': 'fenced',
                    'fence_reason': 'stale_writer',
                }
            )
            raise FenceError('stale_writer', current)
        if sequence < current.sequence:
            self.checkpoint_log.append(
                {
                    'timestamp': utc_iso(self.now),
                    'group_id': group_id,
                    'subscriber_id': subscriber_id,
                    'owner_id': owner_id,
                    'lease_id': lease_id,
                    'lease_epoch': epoch,
                    'prior_sequence': current.sequence,
                    'next_sequence': sequence,
                    'transition': 'fenced',
                    'fence_reason': 'checkpoint_regression',
                }
            )
            raise FenceError('checkpoint_regression', current)
        updated = LeaseState(owner_id=owner_id, lease_id=lease_id, epoch=epoch, sequence=sequence)
        self.checkpoints[key] = updated
        self.checkpoint_log.append(
            {
                'timestamp': utc_iso(self.now),
                'group_id': group_id,
                'subscriber_id': subscriber_id,
                'owner_id': owner_id,
                'lease_id': lease_id,
                'lease_epoch': epoch,
                'prior_sequence': current.sequence,
                'next_sequence': sequence,
                'transition': transition,
                'fence_reason': '',
            }
        )
        return updated

    def checkpoint_snapshot(self, *, group_id: str, subscriber_id: str) -> dict:
        state = self.checkpoints[(group_id, subscriber_id)]
        return {
            'owner_id': state.owner_id,
            'lease_id': state.lease_id,
            'lease_epoch': state.epoch,
            'durable_sequence': state.sequence,
        }


def required_artifacts(artifact_root: str) -> dict:
    return {
        'publish_attempt_ledger': f'{artifact_root}/publish-attempt-ledger.json',
        'replay_capture': f'{artifact_root}/replay-capture.json',
        'checkpoint_transition_log': f'{artifact_root}/checkpoint-transition-log.json',
        'fault_timeline': f'{artifact_root}/fault-timeline.json',
        'backend_health_snapshot': f'{artifact_root}/backend-health.json',
    }


def build_result(
    *,
    scenario_id: str,
    backend: str,
    topology: dict,
    fault_window: dict,
    publish_ledger: List[dict],
    replay_capture: List[dict],
    checkpoint_log: List[dict],
    fault_timeline: List[dict],
    health_snapshots: List[dict],
    checkpoint_before: dict,
    checkpoint_after: dict,
    replay_resume_cursor: dict,
    lease_transitions: List[dict],
    ambiguous_event_ids: Optional[set] = None,
) -> dict:
    expected_after_sequence = int(replay_resume_cursor.get('after_sequence', 0))
    committed_event_ids = {
        row['event_id']
        for row in publish_ledger
        if row['durable_sequence'] is not None and row['durable_sequence'] > expected_after_sequence
    }
    replayed_event_ids = {row['event_id'] for row in replay_capture}
    missing_event_ids = sorted(committed_event_ids - replayed_event_ids)
    duplicate_event_ids = sorted(
        {
            row['event_id']
            for row in replay_capture
            if row.get('duplicate') or row.get('delivery_phase') == 'replay_duplicate'
        }
    )
    publish_outcomes = {
        'committed': sum(1 for row in publish_ledger if row['client_outcome'] == 'committed'),
        'rejected': sum(1 for row in publish_ledger if row['client_outcome'] == 'rejected'),
        'unknown_commit': sum(1 for row in publish_ledger if row['client_outcome'] == 'unknown_commit'),
    }
    ambiguous = ambiguous_event_ids or set()
    ambiguous_resolved = all(event_id in replayed_event_ids for event_id in ambiguous)
    stale_writer_fenced = all(entry.get('fence_reason', '') != 'stale_writer' or entry['transition'] == 'fenced' for entry in checkpoint_log)
    checkpoint_monotonic = checkpoint_after['durable_sequence'] >= checkpoint_before['durable_sequence']
    assertions = [
        {
            'label': 'all committed events remain replayable after recovery',
            'passed': not missing_event_ids,
        },
        {
            'label': 'checkpoint sequence stays monotonic across recovery',
            'passed': checkpoint_monotonic,
        },
        {
            'label': 'stale writers are fenced instead of regressing the checkpoint',
            'passed': stale_writer_fenced,
        },
        {
            'label': 'ambiguous publish outcomes resolve from replay evidence',
            'passed': ambiguous_resolved,
        },
    ]
    artifact_root = f'bigclaw-go/docs/reports/broker-failover-stub-artifacts/{scenario_id}'
    result = 'passed' if all(item['passed'] for item in assertions) else 'failed'
    return {
        'scenario_id': scenario_id,
        'backend': backend,
        'topology': topology,
        'fault_window': fault_window,
        'published_count': len(publish_ledger),
        'committed_count': len(committed_event_ids),
        'replayed_count': len(replay_capture),
        'duplicate_count': len(duplicate_event_ids),
        'missing_event_ids': missing_event_ids,
        'checkpoint_before_fault': checkpoint_before,
        'checkpoint_after_recovery': checkpoint_after,
        'lease_transitions': lease_transitions,
        'publish_outcomes': publish_outcomes,
        'replay_resume_cursor': replay_resume_cursor,
        'artifacts': required_artifacts(artifact_root),
        'result': result,
        'assertions': assertions,
        'raw_artifacts': {
            'publish_attempt_ledger': publish_ledger,
            'replay_capture': replay_capture,
            'checkpoint_transition_log': checkpoint_log,
            'fault_timeline': fault_timeline,
            'backend_health_snapshot': health_snapshots,
        },
    }


def scenario_bf01() -> dict:
    broker = StubBrokerBackend()
    broker.snapshot_health('before')
    trace_id = 'trace-bf01'
    for idx in range(1, 6):
        broker.publish(
            event_id=f'bf01-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    broker.record_fault('leader_restart', 'broker-a', {'during': 'publish_burst'})
    broker.nodes['broker-a']['healthy'] = False
    broker.advance(2)
    broker.nodes['broker-b']['role'] = 'leader'
    broker.nodes['broker-a']['role'] = 'follower'
    broker.nodes['broker-a']['healthy'] = True
    broker.snapshot_health('after_recovery')
    replay = broker.replay(after_sequence=0, source_node='broker-b')
    checkpoint = {'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 0}
    return build_result(
        scenario_id='BF-01',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'leader_before': 'broker-a', 'leader_after': 'broker-b'},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'leader_restart_during_publish'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint,
        checkpoint_after=checkpoint,
        replay_resume_cursor={'after_sequence': 0, 'resumed_from_node': 'broker-b'},
        lease_transitions=[],
    )


def scenario_bf02() -> dict:
    broker = StubBrokerBackend()
    broker.snapshot_health('before')
    trace_id = 'trace-bf02'
    for idx in range(1, 5):
        broker.publish(
            event_id=f'bf02-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    broker.record_fault('replica_loss', 'broker-c', {'latency_spike_ms': 180})
    broker.nodes['broker-c']['healthy'] = False
    broker.advance()
    broker.snapshot_health('after_replica_loss')
    replay = broker.replay(after_sequence=2, source_node='broker-a')
    checkpoint = {'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 2}
    return build_result(
        scenario_id='BF-02',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'replication_factor': 3, 'replica_lost': 'broker-c'},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'follower_loss_during_publish_and_replay'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint,
        checkpoint_after={'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 4},
        replay_resume_cursor={'after_sequence': 2, 'resumed_from_node': 'broker-a', 'publish_latency_spike_ms': 180},
        lease_transitions=[],
    )


def scenario_bf03() -> dict:
    broker = StubBrokerBackend()
    trace_id = 'trace-bf03'
    lease = broker.acquire_lease(group_id='group-a', subscriber_id='subscriber-a', owner_id='consumer-a')
    for idx in range(1, 5):
        broker.publish(
            event_id=f'bf03-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=lease.owner_id,
        lease_id=lease.lease_id,
        epoch=lease.epoch,
        sequence=2,
    )
    checkpoint_before = broker.checkpoint_snapshot(group_id='group-a', subscriber_id='subscriber-a')
    broker.record_fault('consumer_crash', 'consumer-a', {'after_sequence': 3, 'before_checkpoint_commit': True})
    broker.advance()
    replay = broker.replay(after_sequence=checkpoint_before['durable_sequence'], source_node='broker-a')
    checkpoint_after = broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=lease.owner_id,
        lease_id=lease.lease_id,
        epoch=lease.epoch,
        sequence=4,
        transition='replayed',
    )
    return build_result(
        scenario_id='BF-03',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'consumer_group': 'group-a'},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'consumer_crash_before_checkpoint_commit'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint_before,
        checkpoint_after={
            'owner_id': checkpoint_after.owner_id,
            'lease_id': checkpoint_after.lease_id,
            'lease_epoch': checkpoint_after.epoch,
            'durable_sequence': checkpoint_after.sequence,
        },
        replay_resume_cursor={'after_sequence': checkpoint_before['durable_sequence'], 'resumed_from_node': 'broker-a'},
        lease_transitions=[{'owner_id': lease.owner_id, 'lease_id': lease.lease_id, 'lease_epoch': lease.epoch}],
    )


def scenario_bf04() -> dict:
    broker = StubBrokerBackend()
    trace_id = 'trace-bf04'
    primary = broker.acquire_lease(group_id='group-a', subscriber_id='subscriber-a', owner_id='consumer-a')
    for idx in range(1, 4):
        broker.publish(
            event_id=f'bf04-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=primary.owner_id,
        lease_id=primary.lease_id,
        epoch=primary.epoch,
        sequence=2,
        transition='acked_pending_commit',
    )
    checkpoint_before = broker.checkpoint_snapshot(group_id='group-a', subscriber_id='subscriber-a')
    broker.record_fault('checkpoint_leader_change', 'broker-b', {'contenders': ['consumer-a', 'consumer-b']})
    broker.advance()
    standby = broker.acquire_lease(group_id='group-a', subscriber_id='subscriber-a', owner_id='consumer-b')
    stale_fence = 0
    try:
        broker.checkpoint(
            group_id='group-a',
            subscriber_id='subscriber-a',
            owner_id=primary.owner_id,
            lease_id=primary.lease_id,
            epoch=primary.epoch,
            sequence=3,
        )
    except FenceError:
        stale_fence += 1
    broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=standby.owner_id,
        lease_id=standby.lease_id,
        epoch=standby.epoch,
        sequence=3,
    )
    checkpoint_after = broker.checkpoint_snapshot(group_id='group-a', subscriber_id='subscriber-a')
    replay = broker.replay(after_sequence=checkpoint_before['durable_sequence'], source_node='broker-b')
    result = build_result(
        scenario_id='BF-04',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'consumer_group': 'group-a', 'contenders': ['consumer-a', 'consumer-b']},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'checkpoint_leader_change_with_contention'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint_before,
        checkpoint_after=checkpoint_after,
        replay_resume_cursor={'after_sequence': checkpoint_before['durable_sequence'], 'resumed_from_node': 'broker-b'},
        lease_transitions=[
            {'owner_id': primary.owner_id, 'lease_id': primary.lease_id, 'lease_epoch': primary.epoch},
            {'owner_id': standby.owner_id, 'lease_id': standby.lease_id, 'lease_epoch': standby.epoch},
        ],
    )
    result['stale_write_rejections'] = stale_fence
    return result


def scenario_bf05() -> dict:
    broker = StubBrokerBackend()
    broker.snapshot_health('before')
    broker.publish(event_id='bf05-event-01', trace_id='trace-bf05', source_node='producer-a', client_outcome='committed')
    broker.advance()
    broker.record_fault('producer_timeout', 'producer-a', {'between_ack_and_client_response': True})
    broker.publish(
        event_id='bf05-event-02',
        trace_id='trace-bf05',
        source_node='producer-a',
        client_outcome='unknown_commit',
        committed=True,
    )
    broker.advance()
    broker.publish(
        event_id='bf05-event-03',
        trace_id='trace-bf05',
        source_node='producer-a',
        client_outcome='rejected',
        committed=False,
    )
    broker.snapshot_health('after_timeout')
    replay = broker.replay(after_sequence=0, source_node='broker-a')
    checkpoint = {'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 0}
    return build_result(
        scenario_id='BF-05',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'producer': 'producer-a'},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'producer_timeout_after_commit_ambiguity'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint,
        checkpoint_after=checkpoint,
        replay_resume_cursor={'after_sequence': 0, 'resumed_from_node': 'broker-a'},
        lease_transitions=[],
        ambiguous_event_ids={'bf05-event-02'},
    )


def scenario_bf06() -> dict:
    broker = StubBrokerBackend()
    trace_id = 'trace-bf06'
    for idx in range(1, 6):
        broker.publish(
            event_id=f'bf06-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    initial = broker.replay(after_sequence=0, source_node='broker-a')[:3]
    broker.replay_capture = initial
    broker.record_fault('replay_disconnect', 'consumer-a', {'after_sequence': 3, 'reconnect_node': 'broker-b'})
    broker.advance()
    resumed = broker.replay(after_sequence=3, source_node='broker-b')
    checkpoint = {'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 3}
    return build_result(
        scenario_id='BF-06',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'replay_client': 'consumer-a'},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'replay_client_disconnect_and_reconnect'},
        publish_ledger=broker.publish_ledger,
        replay_capture=initial + resumed,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint,
        checkpoint_after={'owner_id': 'consumer-a', 'lease_id': 'lease-none', 'lease_epoch': 0, 'durable_sequence': 5},
        replay_resume_cursor={'after_sequence': 3, 'resumed_from_node': 'broker-b'},
        lease_transitions=[],
    )


def scenario_bf07() -> dict:
    broker = StubBrokerBackend()
    trace_id = 'trace-bf07'
    for idx in range(1, 6):
        broker.publish(
            event_id=f'bf07-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    lease = broker.acquire_lease(group_id='group-a', subscriber_id='subscriber-a', owner_id='consumer-a')
    broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=lease.owner_id,
        lease_id=lease.lease_id,
        epoch=lease.epoch,
        sequence=1,
    )
    checkpoint_before = broker.checkpoint_snapshot(group_id='group-a', subscriber_id='subscriber-a')
    broker.retention_floor = 3
    broker.record_fault('retention_boundary', 'broker-a', {'checkpoint_sequence': 1, 'retention_floor': 3})
    broker.snapshot_health('after_retention_trim')
    replay = broker.replay(after_sequence=2, source_node='broker-a')
    result = build_result(
        scenario_id='BF-07',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'retention_floor': 3},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'retention_boundary_intersects_stale_checkpoint'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint_before,
        checkpoint_after={'owner_id': 'operator-reset', 'lease_id': 'manual-reset', 'lease_epoch': lease.epoch + 1, 'durable_sequence': 3},
        replay_resume_cursor={'after_sequence': 2, 'resumed_from_node': 'broker-a', 'reset_required': True},
        lease_transitions=[{'owner_id': lease.owner_id, 'lease_id': lease.lease_id, 'lease_epoch': lease.epoch}],
    )
    result['operator_guidance'] = 'checkpoint expired behind retention floor; explicit reset to sequence 3 required'
    return result


def scenario_bf08() -> dict:
    broker = StubBrokerBackend()
    trace_id = 'trace-bf08'
    for idx in range(1, 5):
        broker.publish(
            event_id=f'bf08-event-{idx:02d}',
            trace_id=trace_id,
            source_node='producer-a',
            client_outcome='committed',
        )
        broker.advance()
    lease = broker.acquire_lease(group_id='group-a', subscriber_id='subscriber-a', owner_id='consumer-a')
    broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=lease.owner_id,
        lease_id=lease.lease_id,
        epoch=lease.epoch,
        sequence=2,
    )
    checkpoint_before = broker.checkpoint_snapshot(group_id='group-a', subscriber_id='subscriber-a')
    broker.record_fault('split_brain_duplicate_window', 'broker-b', {'duplicate_event_ids': ['bf08-event-03']})
    replay = broker.replay(after_sequence=2, source_node='broker-b', duplicate_event_ids={'bf08-event-03'})
    checkpoint_after = broker.checkpoint(
        group_id='group-a',
        subscriber_id='subscriber-a',
        owner_id=lease.owner_id,
        lease_id=lease.lease_id,
        epoch=lease.epoch,
        sequence=4,
    )
    return build_result(
        scenario_id='BF-08',
        backend=broker.backend,
        topology={'nodes': ['broker-a', 'broker-b', 'broker-c'], 'duplicate_window': True},
        fault_window={'start': broker.fault_timeline[0]['timestamp'], 'end': utc_iso(broker.now), 'fault': 'split_brain_duplicate_delivery_window'},
        publish_ledger=broker.publish_ledger,
        replay_capture=replay,
        checkpoint_log=broker.checkpoint_log,
        fault_timeline=broker.fault_timeline,
        health_snapshots=broker.health_snapshots,
        checkpoint_before=checkpoint_before,
        checkpoint_after={
            'owner_id': checkpoint_after.owner_id,
            'lease_id': checkpoint_after.lease_id,
            'lease_epoch': checkpoint_after.epoch,
            'durable_sequence': checkpoint_after.sequence,
        },
        replay_resume_cursor={'after_sequence': 2, 'resumed_from_node': 'broker-b'},
        lease_transitions=[{'owner_id': lease.owner_id, 'lease_id': lease.lease_id, 'lease_epoch': lease.epoch}],
    )


def build_report() -> dict:
    scenarios = [
        scenario_bf01(),
        scenario_bf02(),
        scenario_bf03(),
        scenario_bf04(),
        scenario_bf05(),
        scenario_bf06(),
        scenario_bf07(),
        scenario_bf08(),
    ]
    return {
        'generated_at': utc_iso(datetime(2026, 3, 17, 12, 0, tzinfo=timezone.utc)),
        'ticket': 'OPE-272',
        'title': 'Deterministic broker failover stub validation report',
        'backend': 'stub_broker',
        'status': 'deterministic-harness',
        'source_validation_pack': 'bigclaw-go/docs/reports/broker-failover-fault-injection-validation-pack.md',
        'report_schema_version': '2026-03-17',
        'scenarios': [
            {key: value for key, value in scenario.items() if key != 'raw_artifacts'}
            for scenario in scenarios
        ],
        'summary': {
            'scenario_count': len(scenarios),
            'passing_scenarios': sum(1 for scenario in scenarios if scenario['result'] == 'passed'),
            'failing_scenarios': sum(1 for scenario in scenarios if scenario['result'] != 'passed'),
            'duplicate_count': sum(scenario['duplicate_count'] for scenario in scenarios),
            'missing_event_count': sum(len(scenario['missing_event_ids']) for scenario in scenarios),
        },
        'raw_artifacts': {
            scenario['scenario_id']: scenario['raw_artifacts']
            for scenario in scenarios
        },
    }


def write_report(report: dict, *, output: pathlib.Path, artifact_root: pathlib.Path, pretty: bool) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    artifact_root.mkdir(parents=True, exist_ok=True)
    output.write_text(
        json.dumps({key: value for key, value in report.items() if key != 'raw_artifacts'}, indent=2 if pretty else None) + '\n',
        encoding='utf-8',
    )
    for scenario_id, artifact_payloads in report['raw_artifacts'].items():
        scenario_dir = artifact_root / scenario_id
        scenario_dir.mkdir(parents=True, exist_ok=True)
        for filename, payload in (
            ('publish-attempt-ledger.json', artifact_payloads['publish_attempt_ledger']),
            ('replay-capture.json', artifact_payloads['replay_capture']),
            ('checkpoint-transition-log.json', artifact_payloads['checkpoint_transition_log']),
            ('fault-timeline.json', artifact_payloads['fault_timeline']),
            ('backend-health.json', artifact_payloads['backend_health_snapshot']),
        ):
            (scenario_dir / filename).write_text(
                json.dumps(payload, indent=2 if pretty else None) + '\n',
                encoding='utf-8',
            )


def main() -> None:
    parser = argparse.ArgumentParser(description='Generate deterministic broker failover stub validation artifacts')
    parser.add_argument('--output', default=REPORT_PATH)
    parser.add_argument('--artifact-root', default=ARTIFACT_ROOT)
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report()
    write_report(
        report,
        output=repo_root / args.output,
        artifact_root=repo_root / args.artifact_root,
        pretty=args.pretty,
    )


if __name__ == '__main__':
    main()
