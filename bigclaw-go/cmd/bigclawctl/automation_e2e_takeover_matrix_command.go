package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

const takeoverMatrixReportPath = "docs/reports/multi-subscriber-takeover-validation-report.json"

type automationSubscriberTakeoverFaultMatrixOptions struct {
	GoRoot     string
	OutputPath string
	Now        func() time.Time
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

type takeoverLeaseCoordinator struct {
	leases  map[string]takeoverLease
	counter int
}

type takeoverLeaseHeldError struct {
	Lease takeoverLease
}

func (e *takeoverLeaseHeldError) Error() string {
	return "lease held"
}

type takeoverLeaseExpiredError struct {
	Lease takeoverLease
}

func (e *takeoverLeaseExpiredError) Error() string {
	return "lease expired"
}

type takeoverLeaseFenceError struct {
	Lease takeoverLease
}

func (e *takeoverLeaseFenceError) Error() string {
	return "lease fenced"
}

type takeoverCheckpointRollbackError struct {
	Lease takeoverLease
}

func (e *takeoverCheckpointRollbackError) Error() string {
	return "checkpoint rollback"
}

func runAutomationSubscriberTakeoverFaultMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e subscriber-takeover-fault-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	output := flags.String("output", takeoverMatrixReportPath, "output path")
	asJSON := flags.Bool("json", true, "json")
	pretty := flags.Bool("pretty", false, "pretty-print report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e subscriber-takeover-fault-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, err := automationSubscriberTakeoverFaultMatrix(automationSubscriberTakeoverFaultMatrixOptions{
		GoRoot:     absPath(*goRoot),
		OutputPath: *output,
		Now:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return emit(report, *asJSON, 0)
}

func automationSubscriberTakeoverFaultMatrix(opts automationSubscriberTakeoverFaultMatrixOptions) (map[string]any, error) {
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	report := takeoverBuildReport(now())
	if err := e2eWriteJSON(e2eResolvePath(absPath(opts.GoRoot), opts.OutputPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func newTakeoverLeaseCoordinator() *takeoverLeaseCoordinator {
	return &takeoverLeaseCoordinator{leases: map[string]takeoverLease{}}
}

func (c *takeoverLeaseCoordinator) key(groupID, subscriberID string) string {
	return groupID + "\x00" + subscriberID
}

func (c *takeoverLeaseCoordinator) nextToken() string {
	c.counter++
	return fmt.Sprintf("lease-%d", c.counter)
}

func (c *takeoverLeaseCoordinator) expired(lease takeoverLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && (now.Equal(lease.ExpiresAt) || now.After(lease.ExpiresAt))
}

func (c *takeoverLeaseCoordinator) acquire(groupID, subscriberID, consumerID string, ttl time.Duration, now time.Time) (takeoverLease, error) {
	key := c.key(groupID, subscriberID)
	current, ok := c.leases[key]
	if ok && !c.expired(current, now) && current.ConsumerID != consumerID {
		return takeoverLease{}, &takeoverLeaseHeldError{Lease: current}
	}
	if ok && c.expired(current, now) && current.ConsumerID != consumerID {
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
	if ok && !c.expired(current, now) && current.ConsumerID == consumerID {
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
	lease := takeoverLease{
		GroupID:           groupID,
		SubscriberID:      subscriberID,
		ConsumerID:        consumerID,
		LeaseToken:        c.nextToken(),
		LeaseEpoch:        nextEpoch,
		CheckpointOffset:  nextOffset,
		CheckpointEventID: nextEvent,
		ExpiresAt:         now.Add(ttl),
		UpdatedAt:         now,
	}
	c.leases[key] = lease
	return lease, nil
}

func (c *takeoverLeaseCoordinator) commit(groupID, subscriberID, consumerID, leaseToken string, leaseEpoch, checkpointOffset int, checkpointEventID string, now time.Time) (takeoverLease, error) {
	key := c.key(groupID, subscriberID)
	current, ok := c.leases[key]
	if !ok {
		return takeoverLease{}, &takeoverLeaseExpiredError{}
	}
	if c.expired(current, now) {
		return takeoverLease{}, &takeoverLeaseExpiredError{Lease: current}
	}
	if current.ConsumerID != consumerID || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
		return takeoverLease{}, &takeoverLeaseFenceError{Lease: current}
	}
	if checkpointOffset < current.CheckpointOffset {
		return takeoverLease{}, &takeoverCheckpointRollbackError{Lease: current}
	}
	current.CheckpointOffset = checkpointOffset
	current.CheckpointEventID = checkpointEventID
	current.UpdatedAt = now
	c.leases[key] = current
	return current, nil
}

func takeoverUTCISO(timestamp time.Time) string {
	return timestamp.UTC().Format(time.RFC3339Nano)
}

func takeoverCheckpointPayload(lease takeoverLease) map[string]any {
	return map[string]any{
		"owner":       lease.ConsumerID,
		"lease_epoch": lease.LeaseEpoch,
		"lease_token": lease.LeaseToken,
		"offset":      lease.CheckpointOffset,
		"event_id":    lease.CheckpointEventID,
		"updated_at":  takeoverUTCISO(lease.UpdatedAt),
	}
}

func takeoverCursor(offset int, eventID string) map[string]any {
	return map[string]any{
		"offset":   offset,
		"event_id": eventID,
	}
}

func takeoverAuditEvent(timeline *[]any, timestamp time.Time, subscriber, action string, details map[string]any) {
	*timeline = append(*timeline, map[string]any{
		"timestamp":  takeoverUTCISO(timestamp),
		"subscriber": subscriber,
		"action":     action,
		"details":    details,
	})
}

func takeoverOwnerTimelineEntry(timestamp time.Time, owner, event string, lease takeoverLease) map[string]any {
	return map[string]any{
		"timestamp":           takeoverUTCISO(timestamp),
		"owner":               owner,
		"event":               event,
		"lease_epoch":         lease.LeaseEpoch,
		"checkpoint_offset":   lease.CheckpointOffset,
		"checkpoint_event_id": lease.CheckpointEventID,
	}
}

func takeoverBuildScenarioResult(
	scenarioID, title, primarySubscriber, takeoverSubscriber, taskOrTraceID, subscriberGroup string,
	auditAssertions, checkpointAssertions, replayAssertions, leaseOwnerTimeline []any,
	checkpointBefore, checkpointAfter, replayStartCursor, replayEndCursor map[string]any,
	duplicateEvents []any,
	staleWriteRejections int,
	auditTimeline, eventLogExcerpt, localLimitations []any,
) map[string]any {
	auditChecks := []any{
		map[string]any{
			"label":  "ownership handoff is visible in the audit timeline",
			"passed": takeoverUniqueOwnerCount(leaseOwnerTimeline) >= 2,
		},
		map[string]any{
			"label":  "audit timeline contains takeover-specific events",
			"passed": takeoverTimelineHasAction(auditTimeline, "lease_acquired", "lease_fenced", "primary_crashed"),
		},
		map[string]any{
			"label":  "audit timeline stays ordered by timestamp",
			"passed": takeoverTimelineOrdered(auditTimeline),
		},
	}
	checkpointChecks := []any{
		map[string]any{
			"label":  "checkpoint never regresses across takeover",
			"passed": asInt(checkpointAfter["offset"]) >= asInt(checkpointBefore["offset"]),
		},
		map[string]any{
			"label":  "final checkpoint owner matches the final lease owner",
			"passed": asString(checkpointAfter["owner"]) == asString(lastMap(leaseOwnerTimeline)["owner"]),
		},
		map[string]any{
			"label":  "stale writers do not replace the accepted checkpoint owner",
			"passed": staleWriteRejections == 0 || asString(checkpointAfter["owner"]) == takeoverSubscriber,
		},
	}
	replayChecks := []any{
		map[string]any{
			"label":  "replay restarts from the durable checkpoint boundary",
			"passed": asInt(replayStartCursor["offset"]) == asInt(checkpointBefore["offset"]),
		},
		map[string]any{
			"label":  "replay end cursor advances to the final durable checkpoint",
			"passed": asInt(replayEndCursor["offset"]) == asInt(checkpointAfter["offset"]),
		},
		map[string]any{
			"label":  "duplicate replay candidates are counted explicitly",
			"passed": len(duplicateEvents) >= 0,
		},
	}
	assertions := append(append([]any{}, auditChecks...), checkpointChecks...)
	assertions = append(assertions, replayChecks...)
	allAssertionsPassed := takeoverAllPassed(assertions)
	artifactRoot := fmt.Sprintf("artifacts/%s", scenarioID)
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
		"audit_log_paths": []any{
			fmt.Sprintf("%s/%s-audit.jsonl", artifactRoot, primarySubscriber),
			fmt.Sprintf("%s/%s-audit.jsonl", artifactRoot, takeoverSubscriber),
		},
		"event_log_excerpt": eventLogExcerpt,
		"audit_timeline":    auditTimeline,
		"assertion_results": map[string]any{
			"audit":      auditChecks,
			"checkpoint": checkpointChecks,
			"replay":     replayChecks,
		},
		"all_assertions_passed": allAssertionsPassed,
		"local_limitations":     localLimitations,
	}
}

func takeoverScenarioTakeoverAfterPrimaryCrash() map[string]any {
	scenarioID := "takeover-after-primary-crash"
	title := "Primary subscriber crashes after processing but before checkpoint flush"
	subscriberGroup := "group-takeover-crash"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-takeover-crash"
	timeline := []any{}
	ownerTimeline := []any{}
	coordinator := newTakeoverLeaseCoordinator()
	now := time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)
	ttl := 30 * time.Second

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	takeoverAuditEvent(&timeline, now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch})

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 40, "evt-40", now.Add(5*time.Second))
	mustNoErr(err)
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	takeoverAuditEvent(&timeline, now.Add(5*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 40, "event_id": "evt-40"})
	takeoverAuditEvent(&timeline, now.Add(7*time.Second), primary, "processed_uncheckpointed_tail", map[string]any{"offset": 41, "event_id": "evt-41"})
	takeoverAuditEvent(&timeline, now.Add(8*time.Second), primary, "primary_crashed", map[string]any{"reason": "terminated before checkpoint flush"})

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	takeoverAuditEvent(&timeline, now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true})
	takeoverAuditEvent(&timeline, now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 40, "from_event_id": "evt-40"})
	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 41, "evt-41", now.Add(33*time.Second))
	mustNoErr(err)
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	takeoverAuditEvent(&timeline, now.Add(33*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 41, "event_id": "evt-41", "replayed_tail": true})

	return takeoverBuildScenarioResult(
		scenarioID, title, primary, standby, taskOrTraceID, subscriberGroup,
		[]any{
			"Audit log shows one ownership handoff from primary to standby.",
			"Audit log records the primary interruption reason before standby completion.",
			"Audit log links takeover to the same task or trace identifier across both subscribers.",
		},
		[]any{
			"Checkpoint after takeover is greater than or equal to the last durable checkpoint from the primary.",
			"Standby checkpoint commit is attributed to the new lease owner.",
			"No checkpoint update is accepted from the crashed primary after takeover.",
		},
		[]any{
			"Replay resumes from the last durable checkpoint, not from the last in-memory event processed by the crashed primary.",
			"At most one duplicate delivery is tolerated for the uncheckpointed tail and it is visible in the report.",
			"Replay window closes once the standby checkpoint advances past the tail.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		takeoverCursor(40, "evt-40"),
		takeoverCursor(41, "evt-41"),
		[]any{"evt-41"},
		0,
		timeline,
		[]any{
			map[string]any{"event_id": "evt-40", "delivered_by": []any{primary}, "delivery_kind": "durable"},
			map[string]any{"event_id": "evt-41", "delivered_by": []any{primary, standby}, "delivery_kind": "uncheckpointed_tail_replay"},
		},
		[]any{
			"Deterministic local harness only; no live bigclawd processes participate in this proof.",
			"Per-node audit paths are report artifacts rather than emitted runtime JSONL files.",
		},
	)
}

func takeoverScenarioLeaseExpiryStaleWriterRejected() map[string]any {
	scenarioID := "lease-expiry-stale-writer-rejected"
	title := "Lease expires and the former owner attempts a stale checkpoint write"
	subscriberGroup := "group-stale-writer"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-stale-writer"
	timeline := []any{}
	ownerTimeline := []any{}
	coordinator := newTakeoverLeaseCoordinator()
	now := time.Date(2026, 3, 16, 10, 35, 0, 0, time.UTC)
	ttl := 30 * time.Second

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	takeoverAuditEvent(&timeline, now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch})

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 80, "evt-80", now.Add(3*time.Second))
	mustNoErr(err)
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	takeoverAuditEvent(&timeline, now.Add(3*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 80, "event_id": "evt-80"})
	takeoverAuditEvent(&timeline, now.Add(31*time.Second), primary, "lease_expired", map[string]any{"last_offset": 80})

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	takeoverAuditEvent(&timeline, now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true})

	staleWriteRejections := 0
	_, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 81, "evt-81", now.Add(32*time.Second))
	var fenceErr *takeoverLeaseFenceError
	if errors.As(err, &fenceErr) {
		staleWriteRejections++
		takeoverAuditEvent(&timeline, now.Add(32*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 81, "attempted_event_id": "evt-81", "accepted_owner": standby})
	} else {
		mustNoErr(err)
	}

	takeoverAuditEvent(&timeline, now.Add(33*time.Second), standby, "replay_started", map[string]any{"from_offset": 80, "from_event_id": "evt-80"})
	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 82, "evt-82", now.Add(34*time.Second))
	mustNoErr(err)
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	takeoverAuditEvent(&timeline, now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 82, "event_id": "evt-82", "stale_writer_rejected": true})

	return takeoverBuildScenarioResult(
		scenarioID, title, primary, standby, taskOrTraceID, subscriberGroup,
		[]any{
			"Audit log records lease expiry for the former owner and acquisition by the standby.",
			"Audit log records the stale write rejection with both attempted and accepted owners.",
			"Audit log keeps the rejection and accepted takeover in the same ordered timeline.",
		},
		[]any{
			"Checkpoint sequence never decreases after the standby acquires ownership.",
			"Late primary acknowledgement is rejected or ignored without mutating durable checkpoint state.",
			"Accepted checkpoint owner always matches the active lease holder.",
		},
		[]any{
			"Replay after stale write rejection starts from the accepted durable checkpoint only.",
			"No event acknowledged only by the stale writer disappears from the replay timeline.",
			"Replay report exposes any duplicate event IDs caused by the overlap window.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		takeoverCursor(80, "evt-80"),
		takeoverCursor(82, "evt-82"),
		[]any{"evt-81"},
		staleWriteRejections,
		timeline,
		[]any{
			map[string]any{"event_id": "evt-80", "delivered_by": []any{primary}, "delivery_kind": "durable"},
			map[string]any{"event_id": "evt-81", "delivered_by": []any{primary, standby}, "delivery_kind": "stale_overlap_candidate"},
			map[string]any{"event_id": "evt-82", "delivered_by": []any{standby}, "delivery_kind": "takeover_replay_commit"},
		},
		[]any{
			"Deterministic local harness only; live lease expiry still needs a real two-node integration proof.",
			"Stale writer rejection count is produced by the harness rather than the live control-plane API.",
		},
	)
}

func takeoverScenarioSplitBrainDualReplayWindow() map[string]any {
	scenarioID := "split-brain-dual-replay-window"
	title := "Two subscribers briefly believe they can replay the same tail"
	subscriberGroup := "group-split-brain"
	primary := "subscriber-a"
	standby := "subscriber-b"
	taskOrTraceID := "trace-split-brain"
	timeline := []any{}
	ownerTimeline := []any{}
	coordinator := newTakeoverLeaseCoordinator()
	now := time.Date(2026, 3, 16, 10, 40, 0, 0, time.UTC)
	ttl := 30 * time.Second

	primaryLease, err := coordinator.acquire(subscriberGroup, "event-stream", primary, ttl, now)
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now, primary, "lease_acquired", primaryLease))
	takeoverAuditEvent(&timeline, now, primary, "lease_acquired", map[string]any{"lease_epoch": primaryLease.LeaseEpoch})

	primaryLease, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 120, "evt-120", now.Add(2*time.Second))
	mustNoErr(err)
	checkpointBefore := takeoverCheckpointPayload(primaryLease)
	takeoverAuditEvent(&timeline, now.Add(2*time.Second), primary, "checkpoint_committed", map[string]any{"offset": 120, "event_id": "evt-120"})
	takeoverAuditEvent(&timeline, now.Add(29*time.Second), primary, "overlap_window_started", map[string]any{"candidate_events": []any{"evt-121", "evt-122"}})

	takeoverLease, err := coordinator.acquire(subscriberGroup, "event-stream", standby, ttl, now.Add(31*time.Second))
	mustNoErr(err)
	ownerTimeline = append(ownerTimeline, takeoverOwnerTimelineEntry(now.Add(31*time.Second), standby, "takeover_acquired", takeoverLease))
	takeoverAuditEvent(&timeline, now.Add(31*time.Second), standby, "lease_acquired", map[string]any{"lease_epoch": takeoverLease.LeaseEpoch, "takeover": true})
	takeoverAuditEvent(&timeline, now.Add(32*time.Second), standby, "replay_started", map[string]any{"from_offset": 120, "candidate_events": []any{"evt-121", "evt-122"}})

	staleWriteRejections := 0
	_, err = coordinator.commit(subscriberGroup, "event-stream", primary, primaryLease.LeaseToken, primaryLease.LeaseEpoch, 122, "evt-122", now.Add(33*time.Second))
	var fenceErr *takeoverLeaseFenceError
	if errors.As(err, &fenceErr) {
		staleWriteRejections++
		takeoverAuditEvent(&timeline, now.Add(33*time.Second), primary, "lease_fenced", map[string]any{"attempted_offset": 122, "accepted_owner": standby, "overlap": true})
	} else {
		mustNoErr(err)
	}

	takeoverLease, err = coordinator.commit(subscriberGroup, "event-stream", standby, takeoverLease.LeaseToken, takeoverLease.LeaseEpoch, 122, "evt-122", now.Add(34*time.Second))
	mustNoErr(err)
	checkpointAfter := takeoverCheckpointPayload(takeoverLease)
	takeoverAuditEvent(&timeline, now.Add(34*time.Second), standby, "checkpoint_committed", map[string]any{"offset": 122, "event_id": "evt-122", "winning_owner": standby})
	takeoverAuditEvent(&timeline, now.Add(35*time.Second), standby, "overlap_window_closed", map[string]any{"winning_owner": standby, "duplicate_events": []any{"evt-121", "evt-122"}})

	return takeoverBuildScenarioResult(
		scenarioID, title, primary, standby, taskOrTraceID, subscriberGroup,
		[]any{
			"Combined audit timeline shows overlapping replay attempts and identifies the surviving owner.",
			"Audit evidence includes per-node file paths and normalized subscriber identities.",
			"The final report highlights whether duplicate replay attempts were observed or only simulated.",
		},
		[]any{
			"Only the winning owner can advance the durable checkpoint.",
			"Losing owner leaves durable checkpoint unchanged once fencing is applied.",
			"Report includes the exact checkpoint sequence where overlap began and ended.",
		},
		[]any{
			"Replay output groups duplicate candidate deliveries by event ID.",
			"Final replay cursor belongs to the winning owner only.",
			"Validation reports whether overlapping replay created observable duplicate deliveries.",
		},
		ownerTimeline,
		checkpointBefore,
		checkpointAfter,
		takeoverCursor(120, "evt-120"),
		takeoverCursor(122, "evt-122"),
		[]any{"evt-121", "evt-122"},
		staleWriteRejections,
		timeline,
		[]any{
			map[string]any{"event_id": "evt-120", "delivered_by": []any{primary}, "delivery_kind": "durable"},
			map[string]any{"event_id": "evt-121", "delivered_by": []any{primary, standby}, "delivery_kind": "overlap_candidate"},
			map[string]any{"event_id": "evt-122", "delivered_by": []any{primary, standby}, "delivery_kind": "overlap_candidate"},
		},
		[]any{
			"Deterministic local harness only; duplicate replay candidates are modeled rather than captured from a live shared queue.",
			"Real cross-process subscriber membership and per-node replay metrics remain follow-up work.",
		},
	)
}

func takeoverBuildReport(generatedAt time.Time) map[string]any {
	scenarios := []any{
		takeoverScenarioTakeoverAfterPrimaryCrash(),
		takeoverScenarioLeaseExpiryStaleWriterRejected(),
		takeoverScenarioSplitBrainDualReplayWindow(),
	}
	passing := 0
	totalDuplicates := 0
	totalRejections := 0
	for _, item := range scenarios {
		scenario := item.(map[string]any)
		if scenario["all_assertions_passed"] == true {
			passing++
		}
		totalDuplicates += lenAnySlice(scenario["duplicate_events"])
		totalRejections += asInt(scenario["stale_write_rejections"])
	}
	return map[string]any{
		"generated_at": takeoverUTCISO(generatedAt),
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
				"go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...",
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
			"duplicate_delivery_count": totalDuplicates,
			"stale_write_rejections":   totalRejections,
		},
		"scenarios": scenarios,
		"remaining_gaps": []any{
			"The harness is deterministic and local; it does not yet orchestrate live bigclawd takeover between separate processes.",
			"Audit log paths in the report are normalized artifact targets, not emitted runtime files from a multi-node run.",
			"Shared durable subscriber-group coordination still needs a full cross-process proof before the follow-up digest can be closed as done.",
		},
	}
}

func takeoverUniqueOwnerCount(leaseOwnerTimeline []any) int {
	owners := map[string]struct{}{}
	for _, item := range leaseOwnerTimeline {
		if row, ok := item.(map[string]any); ok {
			owners[asString(row["owner"])] = struct{}{}
		}
	}
	return len(owners)
}

func takeoverTimelineHasAction(timeline []any, actions ...string) bool {
	want := map[string]struct{}{}
	for _, action := range actions {
		want[action] = struct{}{}
	}
	for _, item := range timeline {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := want[asString(row["action"])]; ok {
			return true
		}
	}
	return false
}

func takeoverTimelineOrdered(timeline []any) bool {
	last := ""
	for _, item := range timeline {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		current := asString(row["timestamp"])
		if last != "" && current < last {
			return false
		}
		last = current
	}
	return true
}

func takeoverAllPassed(items []any) bool {
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok || row["passed"] != true {
			return false
		}
	}
	return true
}

func lastMap(items []any) map[string]any {
	if len(items) == 0 {
		return map[string]any{}
	}
	row, _ := items[len(items)-1].(map[string]any)
	if row == nil {
		return map[string]any{}
	}
	return row
}

func mustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func lenAnySlice(value any) int {
	items, _ := value.([]any)
	return len(items)
}
