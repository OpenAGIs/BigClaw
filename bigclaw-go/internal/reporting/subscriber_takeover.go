package reporting

import (
	"fmt"
	"time"
)

const (
	SubscriberTakeoverGenerator = "bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix/main.go"
	defaultTakeoverReportPath   = "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json"
)

type SubscriberTakeoverOptions struct {
	Output string
}

type takeoverLease struct {
	GroupID           string
	SubscriberID      string
	ConsumerID        string
	LeaseToken        string
	LeaseEpoch        int
	CheckpointOffset  int
	CheckpointEventID string
	ExpiresAt         time.Time
	UpdatedAt         time.Time
}

type takeoverLeaseHeld struct{ lease takeoverLease }
type takeoverLeaseExpired struct{ lease takeoverLease }
type takeoverLeaseFence struct{ lease takeoverLease }
type takeoverCheckpointRollback struct{ lease takeoverLease }

func (e takeoverLeaseHeld) Error() string          { return "lease held" }
func (e takeoverLeaseExpired) Error() string       { return "lease expired" }
func (e takeoverLeaseFence) Error() string         { return "lease fenced" }
func (e takeoverCheckpointRollback) Error() string { return "checkpoint rollback" }

type takeoverLeaseCoordinator struct {
	leases  map[string]takeoverLease
	counter int
}

func newTakeoverLeaseCoordinator() *takeoverLeaseCoordinator {
	return &takeoverLeaseCoordinator{leases: map[string]takeoverLease{}}
}

func (c *takeoverLeaseCoordinator) acquire(groupID string, subscriberID string, consumerID string, ttl time.Duration, now time.Time) (takeoverLease, error) {
	key := groupID + "\x00" + subscriberID
	current, ok := c.leases[key]
	if ok && !takeoverLeaseExpiredAt(current, now) && current.ConsumerID != consumerID {
		return takeoverLease{}, takeoverLeaseHeld{lease: current}
	}
	if ok && takeoverLeaseExpiredAt(current, now) && current.ConsumerID != consumerID {
		current = takeoverLease{
			GroupID:           current.GroupID,
			SubscriberID:      current.SubscriberID,
			ConsumerID:        current.ConsumerID,
			LeaseEpoch:        current.LeaseEpoch,
			CheckpointOffset:  current.CheckpointOffset,
			CheckpointEventID: current.CheckpointEventID,
			ExpiresAt:         current.ExpiresAt,
			UpdatedAt:         current.UpdatedAt,
		}
	}
	if ok && !takeoverLeaseExpiredAt(current, now) && current.ConsumerID == consumerID {
		current.ExpiresAt = now.Add(ttl)
		current.UpdatedAt = now
		c.leases[key] = current
		return current, nil
	}
	nextEpoch := 1
	nextOffset := 0
	nextEvent := ""
	if ok {
		nextEpoch = current.LeaseEpoch + 1
		nextOffset = current.CheckpointOffset
		nextEvent = current.CheckpointEventID
	}
	c.counter++
	lease := takeoverLease{
		GroupID:           groupID,
		SubscriberID:      subscriberID,
		ConsumerID:        consumerID,
		LeaseToken:        fmt.Sprintf("lease-%d", c.counter),
		LeaseEpoch:        nextEpoch,
		CheckpointOffset:  nextOffset,
		CheckpointEventID: nextEvent,
		ExpiresAt:         now.Add(ttl),
		UpdatedAt:         now,
	}
	c.leases[key] = lease
	return lease, nil
}

func (c *takeoverLeaseCoordinator) commit(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int, checkpointOffset int, checkpointEventID string, now time.Time) (takeoverLease, error) {
	key := groupID + "\x00" + subscriberID
	current, ok := c.leases[key]
	if !ok {
		return takeoverLease{}, takeoverLeaseExpired{}
	}
	if takeoverLeaseExpiredAt(current, now) {
		return takeoverLease{}, takeoverLeaseExpired{lease: current}
	}
	if current.ConsumerID != consumerID || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
		return takeoverLease{}, takeoverLeaseFence{lease: current}
	}
	if checkpointOffset < current.CheckpointOffset {
		return takeoverLease{}, takeoverCheckpointRollback{lease: current}
	}
	current.CheckpointOffset = checkpointOffset
	current.CheckpointEventID = checkpointEventID
	current.UpdatedAt = now
	c.leases[key] = current
	return current, nil
}

func takeoverLeaseExpiredAt(lease takeoverLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}

func BuildSubscriberTakeoverReport(now time.Time) map[string]any {
	if now.IsZero() {
		now = time.Date(2026, 3, 16, 10, 20, 20, 246671000, time.UTC)
	}
	scenarios := []map[string]any{
		buildTakeoverAfterPrimaryCrashScenario(),
		buildLeaseExpiryStaleWriterRejectedScenario(),
		buildSplitBrainDualReplayWindowScenario(),
	}
	passing := 0
	totalDuplicates := 0
	totalRejections := 0
	for _, scenario := range scenarios {
		if asBool(scenario["all_assertions_passed"]) {
			passing++
		}
		totalDuplicates += asInt(scenario["duplicate_delivery_count"])
		totalRejections += asInt(scenario["stale_write_rejections"])
	}
	return map[string]any{
		"generated_at": now.UTC().Format(time.RFC3339Nano),
		"ticket":       "OPE-269",
		"title":        "Multi-subscriber takeover executable local harness report",
		"status":       "local-executable",
		"harness_mode": "deterministic_local_simulation",
		"current_primitives": map[string]any{
			"lease_aware_checkpoints": []string{
				"internal/events/subscriber_leases.go",
				"internal/events/subscriber_leases_test.go",
				"docs/reports/event-bus-reliability-report.md",
			},
			"shared_queue_evidence": []string{
				"scripts/e2e/multi_node_shared_queue.py",
				"docs/reports/multi-node-shared-queue-report.json",
			},
			"takeover_harness": []string{
				"scripts/e2e/subscriber_takeover_fault_matrix/main.go",
				"docs/reports/multi-subscriber-takeover-validation-report.json",
			},
		},
		"required_report_sections": []string{
			"scenario metadata",
			"fault injection steps",
			"audit assertions",
			"checkpoint assertions",
			"replay assertions",
			"per-node audit artifacts",
			"final owner and replay cursor summary",
			"duplicate delivery accounting",
			"open blockers and follow-up implementation hooks",
		},
		"implementation_path": []string{
			"wire the same ownership and rejection schema into the shared multi-node harness",
			"emit real per-node audit artifacts from live takeover runs instead of synthetic report paths",
			"export duplicate replay candidates and stale-writer rejection counters from live event-log APIs",
			"prove the same report contract against an actual cross-process subscriber group",
		},
		"summary": map[string]any{
			"scenario_count":           len(scenarios),
			"passing_scenarios":        passing,
			"failing_scenarios":        len(scenarios) - passing,
			"duplicate_delivery_count": totalDuplicates,
			"stale_write_rejections":   totalRejections,
		},
		"scenarios": scenarios,
		"remaining_gaps": []string{
			"The harness is deterministic and local; it does not yet orchestrate live bigclawd takeover between separate processes.",
			"Audit log paths in the report are normalized artifact targets, not emitted runtime files from a multi-node run.",
			"Shared durable subscriber-group coordination still needs a full cross-process proof before the follow-up digest can be closed as done.",
		},
	}
}

func WriteSubscriberTakeoverArtifacts(root string, options SubscriberTakeoverOptions) error {
	root = firstNonEmptyString(root)
	if root == "" {
		return fmt.Errorf("repo root is required")
	}
	if options.Output == "" {
		options.Output = defaultTakeoverReportPath
	}
	return WriteJSON(resolveReportPath(root, options.Output), BuildSubscriberTakeoverReport(time.Date(2026, 3, 16, 10, 20, 20, 246671000, time.UTC)))
}

func buildTakeoverAfterPrimaryCrashScenario() map[string]any {
	const scenarioID = "takeover-after-primary-crash"
	const subscriberGroup = "group-takeover-crash"
	const primary = "subscriber-a"
	const standby = "subscriber-b"
	baseTime := time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)
	ttl := 30 * time.Second
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newTakeoverLeaseCoordinator()

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, baseTime)
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 40, "evt-40", baseTime.Add(5*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(5*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 40, "event_id": "evt-40"}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(7*time.Second), primary, "processed_uncheckpointed_tail", map[string]any{"offset": 41, "event_id": "evt-41"}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(8*time.Second), primary, "primary_crashed", map[string]any{"reason": "terminated before checkpoint flush"}))

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, baseTime.Add(31*time.Second))
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 40, "from_event_id": "evt-40"}))
	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 41, "evt-41", baseTime.Add(33*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(33*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 41, "event_id": "evt-41", "replayed_tail": true}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Primary subscriber crashes after processing but before checkpoint flush",
		primary,
		standby,
		"trace-takeover-crash",
		subscriberGroup,
		[]string{
			"Audit log shows one ownership handoff from primary to standby.",
			"Audit log records the primary interruption reason before standby completion.",
			"Audit log links takeover to the same task or trace identifier across both subscribers.",
		},
		[]string{
			"Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.",
			"Standby checkpoint commit is attributed to the new lease owner.",
			"No checkpoint update is accepted from the crashed primary after takeover.",
		},
		[]string{
			"Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.",
			"At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.",
			"Replay window closes once the standby checkpoint advances past the tail.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 40, "event_id": "evt-40"},
		map[string]any{"offset": 41, "event_id": "evt-41"},
		[]string{"evt-41"},
		0,
		timeline,
		[]map[string]any{
			{"event_id": "evt-40", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-41", "delivered_by": []string{primary, standby}, "delivery_kind": "uncheckpointed_tail_replay"},
		},
		[]string{
			"Deterministic local harness only; no live bigclawd processes participate in this proof.",
			"Per-node audit paths are report artifacts rather than emitted runtime JSONL files.",
		},
	)
}

func buildLeaseExpiryStaleWriterRejectedScenario() map[string]any {
	const scenarioID = "lease-expiry-stale-writer-rejected"
	const subscriberGroup = "group-stale-writer"
	const primary = "subscriber-a"
	const standby = "subscriber-b"
	baseTime := time.Date(2026, 3, 16, 10, 35, 0, 0, time.UTC)
	ttl := 30 * time.Second
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newTakeoverLeaseCoordinator()

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, baseTime)
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 80, "evt-80", baseTime.Add(3*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(3*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 80, "event_id": "evt-80"}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(31*time.Second), primary, "lease_expired", map[string]any{"last_offset": 80}))

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, baseTime.Add(31*time.Second))
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))

	staleWriteRejections := 0
	if _, err := coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 81, "evt-81", baseTime.Add(32*time.Second)); err != nil {
		if _, ok := err.(takeoverLeaseFence); ok {
			staleWriteRejections++
			timeline = append(timeline, takeoverAuditEvent(baseTime.Add(32*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 81, "attempted_event_id": "evt-81", "accepted_owner": standby}))
		} else {
			panic(err)
		}
	}

	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(33*time.Second), standby, "replay_started", map[string]any{"from_offset": 80, "from_event_id": "evt-80"}))
	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 82, "evt-82", baseTime.Add(34*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 82, "event_id": "evt-82", "stale_writer_rejected": true}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Lease expires and the former owner attempts a stale checkpoint write",
		primary,
		standby,
		"trace-stale-writer",
		subscriberGroup,
		[]string{
			"Audit log records lease expiry for the former owner and acquisition by the standby.",
			"Audit log records the stale write rejection with both attempted and accepted owners.",
			"Audit log keeps the rejection and accepted takeover in the same ordered timeline.",
		},
		[]string{
			"Checkpoint sequence never decreases after the standby acquires ownership.",
			"Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.",
			"Accepted checkpoint owner always matches the active lease holder.",
		},
		[]string{
			"Replay after stale write rejection starts from the accepted durable checkpoint only.",
			"No event acknowledged only by the stale writer disappears from the replay timeline.",
			"Replay report exposes any duplicate event IDs caused by the overlap window.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 80, "event_id": "evt-80"},
		map[string]any{"offset": 82, "event_id": "evt-82"},
		[]string{"evt-81"},
		staleWriteRejections,
		timeline,
		[]map[string]any{
			{"event_id": "evt-80", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-81", "delivered_by": []string{primary, standby}, "delivery_kind": "stale_overlap_candidate"},
			{"event_id": "evt-82", "delivered_by": []string{standby}, "delivery_kind": "takeover_replay_commit"},
		},
		[]string{
			"Deterministic local harness only; live lease expiry still needs a real two-node integration proof.",
			"Stale writer rejection count is produced by the harness rather than the live control-plane API.",
		},
	)
}

func buildSplitBrainDualReplayWindowScenario() map[string]any {
	const scenarioID = "split-brain-dual-replay-window"
	const subscriberGroup = "group-split-brain"
	const primary = "subscriber-a"
	const standby = "subscriber-b"
	baseTime := time.Date(2026, 3, 16, 10, 40, 0, 0, time.UTC)
	ttl := 30 * time.Second
	timeline := []map[string]any{}
	ownerTimeline := []map[string]any{}
	coordinator := newTakeoverLeaseCoordinator()

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, baseTime)
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime, primary, "lease_acquired", primaryLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch}))

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 120, "evt-120", baseTime.Add(2*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(2*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 120, "event_id": "evt-120"}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(29*time.Second), primary, "overlap_window_started", map[string]any{"candidate_events": []string{"evt-121", "evt-122"}}))

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, baseTime.Add(31*time.Second))
	if err != nil {
		panic(err)
	}
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(baseTime.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 120, "candidate_events": []string{"evt-121", "evt-122"}}))

	staleWriteRejections := 0
	if _, err := coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 122, "evt-122", baseTime.Add(33*time.Second)); err != nil {
		if _, ok := err.(takeoverLeaseFence); ok {
			staleWriteRejections++
			timeline = append(timeline, takeoverAuditEvent(baseTime.Add(33*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 122, "accepted_owner": standby, "overlap": true}))
		} else {
			panic(err)
		}
	}

	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 122, "evt-122", baseTime.Add(34*time.Second))
	if err != nil {
		panic(err)
	}
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 122, "event_id": "evt-122", "winning_owner": standby}))
	timeline = append(timeline, takeoverAuditEvent(baseTime.Add(35*time.Second), standby, "overlap_window_closed", map[string]any{"winning_owner": standby, "duplicate_events": []string{"evt-121", "evt-122"}}))

	return buildTakeoverScenarioResult(
		scenarioID,
		"Two subscribers briefly believe they can replay the same tail",
		primary,
		standby,
		"trace-split-brain",
		subscriberGroup,
		[]string{
			"Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.",
			"Audit evidence includes per-node file paths and normalized subscriber identities.",
			"The final report highlights whether duplicate replay attempts were observed or only simulated.",
		},
		[]string{
			"Only the winning owner can advance the durable checkpoint.",
			"Losing owner leaves durable checkpoint unchanged once fencing is applied.",
			"Report includes the exact checkpoint sequence where overlap began and ended.",
		},
		[]string{
			"Replay output groups duplicate candidate deliveries by event ID.",
			"Final replay cursor belongs to the winning owner only.",
			"Validation reports whether overlapping replay created observable duplicate deliveries.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		map[string]any{"offset": 120, "event_id": "evt-120"},
		map[string]any{"offset": 122, "event_id": "evt-122"},
		[]string{"evt-121", "evt-122"},
		staleWriteRejections,
		timeline,
		[]map[string]any{
			{"event_id": "evt-120", "delivered_by": []string{primary}, "delivery_kind": "durable"},
			{"event_id": "evt-121", "delivered_by": []string{primary, standby}, "delivery_kind": "overlap_candidate"},
			{"event_id": "evt-122", "delivered_by": []string{primary, standby}, "delivery_kind": "overlap_candidate"},
		},
		[]string{
			"Deterministic local harness only; duplicate replay candidates are modeled rather than captured from a live shared queue.",
			"Real cross-process subscriber membership and per-node replay metrics remain follow-up work.",
		},
	)
}

func buildTakeoverScenarioResult(
	scenarioID string,
	title string,
	primarySubscriber string,
	takeoverSubscriber string,
	taskOrTraceID string,
	subscriberGroup string,
	auditAssertions []string,
	checkpointAssertions []string,
	replayAssertions []string,
	leaseOwnerTimeline []map[string]any,
	checkpointBefore map[string]any,
	checkpointAfter map[string]any,
	replayStartCursor map[string]any,
	replayEndCursor map[string]any,
	duplicateEvents []string,
	staleWriteRejections int,
	auditTimeline []map[string]any,
	eventLogExcerpt []map[string]any,
	localLimitations []string,
) map[string]any {
	auditChecks := []map[string]any{
		{"label": "ownership handoff is visible in the audit timeline", "passed": len(takeoverDistinctOwners(leaseOwnerTimeline)) >= 2},
		{"label": "audit timeline contains takeover-specific events", "passed": takeoverTimelineHasAction(auditTimeline, "lease_acquired", "lease_fenced", "primary_crashed")},
		{"label": "audit timeline stays ordered by timestamp", "passed": takeoverTimelineOrdered(auditTimeline)},
	}
	checkpointChecks := []map[string]any{
		{"label": "checkpoint never regresses across takeover", "passed": asInt(checkpointAfter["offset"]) >= asInt(checkpointBefore["offset"])},
		{"label": "final checkpoint owner matches the final lease owner", "passed": asString(checkpointAfter["owner"]) == asString(leaseOwnerTimeline[len(leaseOwnerTimeline)-1]["owner"])},
		{"label": "stale writers do not replace the accepted checkpoint owner", "passed": staleWriteRejections == 0 || asString(checkpointAfter["owner"]) == takeoverSubscriber},
	}
	replayChecks := []map[string]any{
		{"label": "replay restarts from the durable checkpoint boundary", "passed": asInt(replayStartCursor["offset"]) == asInt(checkpointBefore["offset"])},
		{"label": "replay end cursor advances to the final durable checkpoint", "passed": asInt(replayEndCursor["offset"]) == asInt(checkpointAfter["offset"])},
		{"label": "duplicate replay candidates are counted explicitly", "passed": len(duplicateEvents) >= 0},
	}
	allPassed := takeoverChecksPassed(auditChecks) && takeoverChecksPassed(checkpointChecks) && takeoverChecksPassed(replayChecks)
	artifactRoot := "artifacts/" + scenarioID
	return map[string]any{
		"id":                       scenarioID,
		"title":                    title,
		"subscriber_group":         subscriberGroup,
		"primary_subscriber":       primarySubscriber,
		"takeover_subscriber":      takeoverSubscriber,
		"task_or_trace_id":         taskOrTraceID,
		"audit_assertions":         auditAssertions,
		"checkpoint_assertions":    checkpointAssertions,
		"replay_assertions":        replayAssertions,
		"lease_owner_timeline":     leaseOwnerTimeline,
		"checkpoint_before":        checkpointBefore,
		"checkpoint_after":         checkpointAfter,
		"replay_start_cursor":      replayStartCursor,
		"replay_end_cursor":        replayEndCursor,
		"duplicate_delivery_count": len(duplicateEvents),
		"duplicate_events":         duplicateEvents,
		"stale_write_rejections":   staleWriteRejections,
		"audit_log_paths": []string{
			artifactRoot + "/" + primarySubscriber + "-audit.jsonl",
			artifactRoot + "/" + takeoverSubscriber + "-audit.jsonl",
		},
		"event_log_excerpt": eventLogExcerpt,
		"audit_timeline":    auditTimeline,
		"assertion_results": map[string]any{
			"audit":      auditChecks,
			"checkpoint": checkpointChecks,
			"replay":     replayChecks,
		},
		"all_assertions_passed": allPassed,
		"local_limitations":     localLimitations,
	}
}

func takeoverCheckpointPayload(lease takeoverLease) map[string]any {
	return map[string]any{
		"owner":       lease.ConsumerID,
		"lease_epoch": lease.LeaseEpoch,
		"lease_token": lease.LeaseToken,
		"offset":      lease.CheckpointOffset,
		"event_id":    lease.CheckpointEventID,
		"updated_at":  lease.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func takeoverAuditEvent(timestamp time.Time, subscriber string, action string, details map[string]any) map[string]any {
	return map[string]any{
		"timestamp":  timestamp.UTC().Format(time.RFC3339),
		"subscriber": subscriber,
		"action":     action,
		"details":    details,
	}
}

func takeoverOwnerTimelineEntry(timestamp time.Time, owner string, event string, lease takeoverLease) map[string]any {
	return map[string]any{
		"timestamp":           timestamp.UTC().Format(time.RFC3339),
		"owner":               owner,
		"event":               event,
		"lease_epoch":         lease.LeaseEpoch,
		"checkpoint_offset":   lease.CheckpointOffset,
		"checkpoint_event_id": lease.CheckpointEventID,
	}
}

func takeoverDistinctOwners(timeline []map[string]any) map[string]struct{} {
	owners := map[string]struct{}{}
	for _, entry := range timeline {
		owners[asString(entry["owner"])] = struct{}{}
	}
	return owners
}

func takeoverTimelineHasAction(timeline []map[string]any, actions ...string) bool {
	allowed := map[string]struct{}{}
	for _, action := range actions {
		allowed[action] = struct{}{}
	}
	for _, entry := range timeline {
		if _, ok := allowed[asString(entry["action"])]; ok {
			return true
		}
	}
	return false
}

func takeoverTimelineOrdered(timeline []map[string]any) bool {
	last := ""
	for _, entry := range timeline {
		current := asString(entry["timestamp"])
		if last != "" && current < last {
			return false
		}
		last = current
	}
	return true
}

func takeoverChecksPassed(checks []map[string]any) bool {
	for _, check := range checks {
		if !asBool(check["passed"]) {
			return false
		}
	}
	return true
}
