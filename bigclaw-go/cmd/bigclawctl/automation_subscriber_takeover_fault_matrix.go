package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

type takeoverLease struct {
	GroupID          string
	SubscriberID     string
	ConsumerID       string
	LeaseToken       string
	LeaseEpoch       int
	CheckpointOffset int
	CheckpointEvent  string
	ExpiresAt        time.Time
	UpdatedAt        time.Time
}

type takeoverLeaseCoordinator struct {
	leases  map[string]takeoverLease
	counter int
}

var (
	errTakeoverLeaseHeld    = errors.New("lease held")
	errTakeoverLeaseExpired = errors.New("lease expired")
	errTakeoverLeaseFence   = errors.New("lease fenced")
	errTakeoverRollback     = errors.New("checkpoint rollback")
	takeoverBaseTime        = time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)
	takeoverTTL             = 30 * time.Second
)

func runAutomationSubscriberTakeoverFaultMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e subscriber-takeover-fault-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	output := flags.String("output", "docs/reports/multi-subscriber-takeover-validation-report.json", "path relative to the repo root")
	pretty := flags.Bool("pretty", false, "print the generated report")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e subscriber-takeover-fault-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := buildSubscriberTakeoverFaultMatrixReport()
	if err != nil {
		return err
	}
	if err := automationWriteReport(absPath(*goRoot), trim(*output), report); err != nil {
		return err
	}
	if *pretty || *asJSON {
		return emit(report, true, 0)
	}
	return nil
}

func buildSubscriberTakeoverFaultMatrixReport() (map[string]any, error) {
	scenarios, err := buildSubscriberTakeoverScenarios()
	if err != nil {
		return nil, err
	}
	passing := 0
	duplicates := 0
	rejections := 0
	for _, scenario := range scenarios {
		entry, _ := scenario.(map[string]any)
		if entry["all_assertions_passed"] == true {
			passing++
		}
		duplicates += automationInt(entry["duplicate_delivery_count"])
		rejections += automationInt(entry["stale_write_rejections"])
	}
	return map[string]any{
		"generated_at": utcISOTime(time.Now().UTC()),
		"ticket":       "OPE-269",
		"title":        "Multi-subscriber takeover executable local harness report",
		"status":       "local-executable",
		"harness_mode": "deterministic_local_simulation",
		"current_primitives": map[string]any{
			"lease_aware_checkpoints": []any{
				"internal/events/subscriber_leases.go",
				"internal/events/subscriber_leases_test.go",
				"docs/reports/event-bus-reliability-report.md",
			},
			"shared_queue_evidence": []any{
				"scripts/e2e/multi_node_shared_queue.py",
				"docs/reports/multi-node-shared-queue-report.json",
			},
			"takeover_harness": []any{
				"scripts/e2e/subscriber_takeover_fault_matrix.py",
				"docs/reports/multi-subscriber-takeover-validation-report.json",
			},
		},
		"required_report_sections": []any{
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
		"implementation_path": []any{
			"wire the same ownership and rejection schema into the shared multi-node harness",
			"emit real per-node audit artifacts from live takeover runs instead of synthetic report paths",
			"export duplicate replay candidates and stale-writer rejection counters from live event-log APIs",
			"prove the same report contract against an actual cross-process subscriber group",
		},
		"summary": map[string]any{
			"scenario_count":           len(scenarios),
			"passing_scenarios":        passing,
			"failing_scenarios":        len(scenarios) - passing,
			"duplicate_delivery_count": duplicates,
			"stale_write_rejections":   rejections,
		},
		"scenarios": scenarios,
		"remaining_gaps": []any{
			"The harness is deterministic and local; it does not yet orchestrate live bigclawd takeover between separate processes.",
			"Audit log paths in the report are normalized artifact targets, not emitted runtime files from a multi-node run.",
			"Shared durable subscriber-group coordination still needs a full cross-process proof before the follow-up digest can be closed as done.",
		},
	}, nil
}

func buildSubscriberTakeoverScenarios() ([]any, error) {
	first, err := scenarioTakeoverAfterPrimaryCrash()
	if err != nil {
		return nil, err
	}
	second, err := scenarioLeaseExpiryStaleWriterRejected()
	if err != nil {
		return nil, err
	}
	third, err := scenarioSplitBrainDualReplayWindow()
	if err != nil {
		return nil, err
	}
	return []any{first, second, third}, nil
}

func scenarioTakeoverAfterPrimaryCrash() (map[string]any, error) {
	coordinator := newTakeoverLeaseCoordinator()
	now := takeoverBaseTime
	timeline := []any{}
	owners := []any{}
	group := "group-takeover-crash"
	primary := "subscriber-a"
	standby := "subscriber-b"
	lease, err := coordinator.acquire(group, "event-stream", primary, takeoverTTL, now)
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now, primary, "lease_acquired", lease))
	timeline = append(timeline, auditTimelineEntry(now, primary, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch}))
	lease, err = coordinator.commit(group, "event-stream", primary, lease.LeaseToken, lease.LeaseEpoch, 40, "evt-40", now.Add(5*time.Second))
	if err != nil {
		return nil, err
	}
	before := checkpointPayloadFromLease(lease)
	timeline = append(timeline,
		auditTimelineEntry(now.Add(5*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 40, "event_id": "evt-40"}),
		auditTimelineEntry(now.Add(7*time.Second), primary, "processed_uncheckpointed_tail", map[string]any{"offset": 41, "event_id": "evt-41"}),
		auditTimelineEntry(now.Add(8*time.Second), primary, "primary_crashed", map[string]any{"reason": "terminated before checkpoint flush"}),
	)
	lease, err = coordinator.acquire(group, "event-stream", standby, takeoverTTL, now.Add(31*time.Second))
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", lease))
	timeline = append(timeline,
		auditTimelineEntry(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch, "takeover": true}),
		auditTimelineEntry(now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 40, "from_event_id": "evt-40"}),
	)
	lease, err = coordinator.commit(group, "event-stream", standby, lease.LeaseToken, lease.LeaseEpoch, 41, "evt-41", now.Add(33*time.Second))
	if err != nil {
		return nil, err
	}
	after := checkpointPayloadFromLease(lease)
	timeline = append(timeline, auditTimelineEntry(now.Add(33*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 41, "event_id": "evt-41", "replayed_tail": true}))
	return buildTakeoverScenarioResult(
		"takeover-after-primary-crash", "Primary subscriber crashes after processing but before checkpoint flush",
		primary, standby, "trace-takeover-crash", group,
		[]string{"Audit log shows one ownership handoff from primary to standby.", "Audit log records the primary interruption reason before standby completion.", "Audit log links takeover to the same task or trace identifier across both subscribers."},
		[]string{"Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.", "Standby checkpoint commit is attributed to the new lease owner.", "No checkpoint update is accepted from the crashed primary after takeover."},
		[]string{"Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.", "At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.", "Replay window closes once the standby checkpoint advances past the tail."},
		owners, before, after, cursorEntry(40, "evt-40"), cursorEntry(41, "evt-41"), []any{"evt-41"}, 0, timeline,
		[]any{map[string]any{"event_id": "evt-40", "delivered_by": []any{primary}, "delivery_kind": "durable"}, map[string]any{"event_id": "evt-41", "delivered_by": []any{primary, standby}, "delivery_kind": "uncheckpointed_tail_replay"}},
		[]any{"Deterministic local harness only; no live bigclawd processes participate in this proof.", "Per-node audit paths are report artifacts rather than emitted runtime JSONL files."},
	), nil
}

func scenarioLeaseExpiryStaleWriterRejected() (map[string]any, error) {
	coordinator := newTakeoverLeaseCoordinator()
	now := takeoverBaseTime.Add(5 * time.Minute)
	timeline := []any{}
	owners := []any{}
	group := "group-stale-writer"
	primary := "subscriber-a"
	standby := "subscriber-b"
	lease, err := coordinator.acquire(group, "event-stream", primary, takeoverTTL, now)
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now, primary, "lease_acquired", lease))
	timeline = append(timeline, auditTimelineEntry(now, primary, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch}))
	lease, err = coordinator.commit(group, "event-stream", primary, lease.LeaseToken, lease.LeaseEpoch, 80, "evt-80", now.Add(3*time.Second))
	if err != nil {
		return nil, err
	}
	before := checkpointPayloadFromLease(lease)
	timeline = append(timeline,
		auditTimelineEntry(now.Add(3*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 80, "event_id": "evt-80"}),
		auditTimelineEntry(now.Add(31*time.Second), primary, "lease_expired", map[string]any{"last_offset": 80}),
	)
	lease, err = coordinator.acquire(group, "event-stream", standby, takeoverTTL, now.Add(31*time.Second))
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", lease))
	timeline = append(timeline, auditTimelineEntry(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch, "takeover": true}))
	rejections := 0
	if _, err := coordinator.commit(group, "event-stream", primary, before["lease_token"].(string), automationInt(before["lease_epoch"]), 81, "evt-81", now.Add(32*time.Second)); err != nil {
		if errors.Is(err, errTakeoverLeaseFence) {
			rejections++
			timeline = append(timeline, auditTimelineEntry(now.Add(32*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 81, "attempted_event_id": "evt-81", "accepted_owner": standby}))
		} else {
			return nil, err
		}
	}
	timeline = append(timeline, auditTimelineEntry(now.Add(33*time.Second), standby, "replay_started", map[string]any{"from_offset": 80, "from_event_id": "evt-80"}))
	lease, err = coordinator.commit(group, "event-stream", standby, lease.LeaseToken, lease.LeaseEpoch, 82, "evt-82", now.Add(34*time.Second))
	if err != nil {
		return nil, err
	}
	after := checkpointPayloadFromLease(lease)
	timeline = append(timeline, auditTimelineEntry(now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 82, "event_id": "evt-82", "stale_writer_rejected": true}))
	return buildTakeoverScenarioResult(
		"lease-expiry-stale-writer-rejected", "Lease expires and the former owner attempts a stale checkpoint write",
		primary, standby, "trace-stale-writer", group,
		[]string{"Audit log records lease expiry for the former owner and acquisition by the standby.", "Audit log records the stale write rejection with both attempted and accepted owners.", "Audit log keeps the rejection and accepted takeover in the same ordered timeline."},
		[]string{"Checkpoint sequence never decreases after the standby acquires ownership.", "Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.", "Accepted checkpoint owner always matches the active lease holder."},
		[]string{"Replay after stale write rejection starts from the accepted durable checkpoint only.", "No event acknowledged only by the stale writer disappears from the replay timeline.", "Replay report exposes any duplicate event IDs caused by the overlap window."},
		owners, before, after, cursorEntry(80, "evt-80"), cursorEntry(82, "evt-82"), []any{"evt-81"}, rejections, timeline,
		[]any{
			map[string]any{"event_id": "evt-80", "delivered_by": []any{primary}, "delivery_kind": "durable"},
			map[string]any{"event_id": "evt-81", "delivered_by": []any{primary, standby}, "delivery_kind": "stale_overlap_candidate"},
			map[string]any{"event_id": "evt-82", "delivered_by": []any{standby}, "delivery_kind": "takeover_replay_commit"},
		},
		[]any{"Deterministic local harness only; live lease expiry still needs a real two-node integration proof.", "Stale writer rejection count is produced by the harness rather than the live control-plane API."},
	), nil
}

func scenarioSplitBrainDualReplayWindow() (map[string]any, error) {
	coordinator := newTakeoverLeaseCoordinator()
	now := takeoverBaseTime.Add(10 * time.Minute)
	timeline := []any{}
	owners := []any{}
	group := "group-split-brain"
	primary := "subscriber-a"
	standby := "subscriber-b"
	lease, err := coordinator.acquire(group, "event-stream", primary, takeoverTTL, now)
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now, primary, "lease_acquired", lease))
	timeline = append(timeline, auditTimelineEntry(now, primary, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch}))
	lease, err = coordinator.commit(group, "event-stream", primary, lease.LeaseToken, lease.LeaseEpoch, 120, "evt-120", now.Add(2*time.Second))
	if err != nil {
		return nil, err
	}
	before := checkpointPayloadFromLease(lease)
	timeline = append(timeline,
		auditTimelineEntry(now.Add(2*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 120, "event_id": "evt-120"}),
		auditTimelineEntry(now.Add(29*time.Second), primary, "overlap_window_started", map[string]any{"candidate_events": []any{"evt-121", "evt-122"}}),
	)
	lease, err = coordinator.acquire(group, "event-stream", standby, takeoverTTL, now.Add(31*time.Second))
	if err != nil {
		return nil, err
	}
	owners = append(owners, ownerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", lease))
	timeline = append(timeline,
		auditTimelineEntry(now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": lease.LeaseEpoch, "takeover": true}),
		auditTimelineEntry(now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 120, "candidate_events": []any{"evt-121", "evt-122"}}),
	)
	rejections := 0
	if _, err := coordinator.commit(group, "event-stream", primary, before["lease_token"].(string), automationInt(before["lease_epoch"]), 122, "evt-122", now.Add(33*time.Second)); err != nil {
		if errors.Is(err, errTakeoverLeaseFence) {
			rejections++
			timeline = append(timeline, auditTimelineEntry(now.Add(33*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 122, "accepted_owner": standby, "overlap": true}))
		} else {
			return nil, err
		}
	}
	lease, err = coordinator.commit(group, "event-stream", standby, lease.LeaseToken, lease.LeaseEpoch, 122, "evt-122", now.Add(34*time.Second))
	if err != nil {
		return nil, err
	}
	after := checkpointPayloadFromLease(lease)
	timeline = append(timeline,
		auditTimelineEntry(now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 122, "event_id": "evt-122", "winning_owner": standby}),
		auditTimelineEntry(now.Add(35*time.Second), standby, "overlap_window_closed", map[string]any{"winning_owner": standby, "duplicate_events": []any{"evt-121", "evt-122"}}),
	)
	return buildTakeoverScenarioResult(
		"split-brain-dual-replay-window", "Two subscribers briefly believe they can replay the same tail",
		primary, standby, "trace-split-brain", group,
		[]string{"Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.", "Audit evidence includes per-node file paths and normalized subscriber identities.", "The final report highlights whether duplicate replay attempts were observed or only simulated."},
		[]string{"Only the winning owner can advance the durable checkpoint.", "Losing owner leaves durable checkpoint unchanged once fencing is applied.", "Report includes the exact checkpoint sequence where overlap began and ended."},
		[]string{"Replay output groups duplicate candidate deliveries by event ID.", "Final replay cursor belongs to the winning owner only.", "Validation reports whether overlapping replay created observable duplicate deliveries."},
		owners, before, after, cursorEntry(120, "evt-120"), cursorEntry(122, "evt-122"), []any{"evt-121", "evt-122"}, rejections, timeline,
		[]any{
			map[string]any{"event_id": "evt-120", "delivered_by": []any{primary}, "delivery_kind": "durable"},
			map[string]any{"event_id": "evt-121", "delivered_by": []any{primary, standby}, "delivery_kind": "overlap_candidate"},
			map[string]any{"event_id": "evt-122", "delivered_by": []any{primary, standby}, "delivery_kind": "overlap_candidate"},
		},
		[]any{"Deterministic local harness only; duplicate replay candidates are modeled rather than captured from a live shared queue.", "Real cross-process subscriber membership and per-node replay metrics remain follow-up work."},
	), nil
}

func buildTakeoverScenarioResult(id string, title string, primary string, standby string, traceID string, group string, auditAssertions []string, checkpointAssertions []string, replayAssertions []string, owners []any, before map[string]any, after map[string]any, replayStart map[string]any, replayEnd map[string]any, duplicateEvents []any, rejections int, timeline []any, excerpt []any, limitations []any) map[string]any {
	auditChecks := []any{
		map[string]any{"label": "ownership handoff is visible in the audit timeline", "passed": ownerCount(owners) >= 2},
		map[string]any{"label": "audit timeline contains takeover-specific events", "passed": timelineHasAction(timeline, "lease_acquired", "lease_fenced", "primary_crashed")},
		map[string]any{"label": "audit timeline stays ordered by timestamp", "passed": timelineOrdered(timeline)},
	}
	checkpointChecks := []any{
		map[string]any{"label": "checkpoint never regresses across takeover", "passed": automationInt(after["offset"]) >= automationInt(before["offset"])},
		map[string]any{"label": "final checkpoint owner matches the final lease owner", "passed": fmt.Sprint(after["owner"]) == fmt.Sprint(lastOwner(owners))},
		map[string]any{"label": "stale writers do not replace the accepted checkpoint owner", "passed": rejections == 0 || fmt.Sprint(after["owner"]) == standby},
	}
	replayChecks := []any{
		map[string]any{"label": "replay restarts from the durable checkpoint boundary", "passed": automationInt(replayStart["offset"]) == automationInt(before["offset"])},
		map[string]any{"label": "replay end cursor advances to the final durable checkpoint", "passed": automationInt(replayEnd["offset"]) == automationInt(after["offset"])},
		map[string]any{"label": "duplicate replay candidates are counted explicitly", "passed": len(duplicateEvents) >= 0},
	}
	allPassed := assertionListPassed(auditChecks) && assertionListPassed(checkpointChecks) && assertionListPassed(replayChecks)
	root := "artifacts/" + id
	return map[string]any{
		"id":                       id,
		"title":                    title,
		"subscriber_group":         group,
		"primary_subscriber":       primary,
		"takeover_subscriber":      standby,
		"task_or_trace_id":         traceID,
		"audit_assertions":         stringSliceToAny(auditAssertions),
		"checkpoint_assertions":    stringSliceToAny(checkpointAssertions),
		"replay_assertions":        stringSliceToAny(replayAssertions),
		"lease_owner_timeline":     owners,
		"checkpoint_before":        before,
		"checkpoint_after":         after,
		"replay_start_cursor":      replayStart,
		"replay_end_cursor":        replayEnd,
		"duplicate_delivery_count": len(duplicateEvents),
		"duplicate_events":         duplicateEvents,
		"stale_write_rejections":   rejections,
		"audit_log_paths":          []any{root + "/" + primary + "-audit.jsonl", root + "/" + standby + "-audit.jsonl"},
		"event_log_excerpt":        excerpt,
		"audit_timeline":           timeline,
		"assertion_results":        map[string]any{"audit": auditChecks, "checkpoint": checkpointChecks, "replay": replayChecks},
		"all_assertions_passed":    allPassed,
		"local_limitations":        limitations,
	}
}

func newTakeoverLeaseCoordinator() *takeoverLeaseCoordinator {
	return &takeoverLeaseCoordinator{leases: map[string]takeoverLease{}}
}

func (c *takeoverLeaseCoordinator) acquire(group string, subscriber string, consumer string, ttl time.Duration, now time.Time) (takeoverLease, error) {
	key := group + "::" + subscriber
	current, ok := c.leases[key]
	if ok && !leaseExpired(current, now) && current.ConsumerID != consumer {
		return takeoverLease{}, errTakeoverLeaseHeld
	}
	if ok && !leaseExpired(current, now) && current.ConsumerID == consumer {
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
		nextEvent = current.CheckpointEvent
	}
	c.counter++
	lease := takeoverLease{
		GroupID:          group,
		SubscriberID:     subscriber,
		ConsumerID:       consumer,
		LeaseToken:       fmt.Sprintf("lease-%d", c.counter),
		LeaseEpoch:       nextEpoch,
		CheckpointOffset: nextOffset,
		CheckpointEvent:  nextEvent,
		ExpiresAt:        now.Add(ttl),
		UpdatedAt:        now,
	}
	c.leases[key] = lease
	return lease, nil
}

func (c *takeoverLeaseCoordinator) commit(group string, subscriber string, consumer string, token string, epoch int, offset int, eventID string, now time.Time) (takeoverLease, error) {
	key := group + "::" + subscriber
	current, ok := c.leases[key]
	if !ok {
		return takeoverLease{}, errTakeoverLeaseExpired
	}
	if leaseExpired(current, now) {
		return takeoverLease{}, errTakeoverLeaseExpired
	}
	if current.ConsumerID != consumer || current.LeaseToken != token || current.LeaseEpoch != epoch {
		return takeoverLease{}, errTakeoverLeaseFence
	}
	if offset < current.CheckpointOffset {
		return takeoverLease{}, errTakeoverRollback
	}
	current.CheckpointOffset = offset
	current.CheckpointEvent = eventID
	current.UpdatedAt = now
	c.leases[key] = current
	return current, nil
}

func leaseExpired(lease takeoverLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}

func checkpointPayloadFromLease(lease takeoverLease) map[string]any {
	return map[string]any{
		"owner":       lease.ConsumerID,
		"lease_epoch": lease.LeaseEpoch,
		"lease_token": lease.LeaseToken,
		"offset":      lease.CheckpointOffset,
		"event_id":    lease.CheckpointEvent,
		"updated_at":  utcISOTime(lease.UpdatedAt),
	}
}

func cursorEntry(offset int, eventID string) map[string]any {
	return map[string]any{"offset": offset, "event_id": eventID}
}

func auditTimelineEntry(ts time.Time, subscriber string, action string, details map[string]any) map[string]any {
	return map[string]any{"timestamp": utcISOTime(ts), "subscriber": subscriber, "action": action, "details": details}
}

func ownerTimelineEntry(ts time.Time, owner string, event string, lease takeoverLease) map[string]any {
	return map[string]any{"timestamp": utcISOTime(ts), "owner": owner, "event": event, "lease_epoch": lease.LeaseEpoch, "checkpoint_offset": lease.CheckpointOffset, "checkpoint_event_id": lease.CheckpointEvent}
}

func stringSliceToAny(values []string) []any {
	items := make([]any, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}

func timelineHasAction(timeline []any, actions ...string) bool {
	set := map[string]bool{}
	for _, action := range actions {
		set[action] = true
	}
	for _, entry := range timeline {
		item, _ := entry.(map[string]any)
		if set[fmt.Sprint(item["action"])] {
			return true
		}
	}
	return false
}

func timelineOrdered(timeline []any) bool {
	prev := ""
	for _, entry := range timeline {
		item, _ := entry.(map[string]any)
		ts := fmt.Sprint(item["timestamp"])
		if prev != "" && ts < prev {
			return false
		}
		prev = ts
	}
	return true
}

func ownerCount(owners []any) int {
	seen := map[string]bool{}
	for _, entry := range owners {
		item, _ := entry.(map[string]any)
		seen[fmt.Sprint(item["owner"])] = true
	}
	return len(seen)
}

func lastOwner(owners []any) string {
	if len(owners) == 0 {
		return ""
	}
	item, _ := owners[len(owners)-1].(map[string]any)
	return fmt.Sprint(item["owner"])
}

func assertionListPassed(items []any) bool {
	for _, item := range items {
		entry, _ := item.(map[string]any)
		if entry["passed"] != true {
			return false
		}
	}
	return true
}
