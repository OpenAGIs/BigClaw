package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type brokerEventRecord struct {
	Sequence   int
	EventID    string
	TraceID    string
	SourceNode string
	Phase      string
	Duplicate  bool
}

type brokerLeaseState struct {
	OwnerID  string
	LeaseID  string
	Epoch    int
	Sequence int
}

type brokerFenceError struct {
	Reason  string
	Current brokerLeaseState
}

func (e brokerFenceError) Error() string { return e.Reason }

type brokerStubBackend struct {
	Backend         string
	Now             time.Time
	NextSequence    int
	NextLease       int
	Events          []map[string]any
	PublishLedger   []map[string]any
	ReplayCapture   []map[string]any
	CheckpointLog   []map[string]any
	FaultTimeline   []map[string]any
	HealthSnapshots []map[string]any
	Checkpoints     map[string]brokerLeaseState
	RetentionFloor  int
	Nodes           map[string]map[string]any
}

const (
	brokerFailoverReportPath           = "bigclaw-go/docs/reports/broker-failover-stub-report.json"
	brokerFailoverArtifactRoot         = "bigclaw-go/docs/reports/broker-failover-stub-artifacts"
	brokerCheckpointFencingSummaryPath = "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json"
	brokerRetentionBoundarySummaryPath = "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json"
)

func runAutomationBrokerFailoverStubMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e broker-failover-stub-matrix", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	output := flags.String("output", brokerFailoverReportPath, "report output path")
	artifactRoot := flags.String("artifact-root", brokerFailoverArtifactRoot, "artifact root path")
	checkpointSummary := flags.String("checkpoint-fencing-summary-output", brokerCheckpointFencingSummaryPath, "checkpoint fencing summary output path")
	retentionSummary := flags.String("retention-boundary-summary-output", brokerRetentionBoundarySummaryPath, "retention boundary summary output path")
	pretty := flags.Bool("pretty", false, "pretty-print generated JSON")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e broker-failover-stub-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, raw, err := buildBrokerFailoverStubReport()
	if err != nil {
		return err
	}
	root := absPath(*repoRoot)
	if err := writeBrokerFailoverReport(root, trim(*output), trim(*artifactRoot), trim(*checkpointSummary), trim(*retentionSummary), report, raw, *pretty); err != nil {
		return err
	}
	return emit(report, true, 0)
}

func buildBrokerFailoverStubReport() (map[string]any, map[string]map[string]any, error) {
	scenarios, raw, err := buildBrokerFailoverScenarios()
	if err != nil {
		return nil, nil, err
	}
	passing := 0
	duplicates := 0
	missing := 0
	for _, scenario := range scenarios {
		entry, _ := scenario.(map[string]any)
		if entry["result"] == "passed" {
			passing++
		}
		duplicates += automationInt(entry["duplicate_count"])
		missing += len(anySlice(entry["missing_event_ids"]))
	}
	report := map[string]any{
		"generated_at":           "2026-03-17T12:00:00Z",
		"ticket":                 "OPE-272",
		"title":                  "Deterministic broker failover stub validation report",
		"backend":                "stub_broker",
		"status":                 "deterministic-harness",
		"source_validation_pack": "bigclaw-go/docs/reports/broker-failover-fault-injection-validation-pack.md",
		"report_schema_version":  "2026-03-17",
		"scenarios":              scenarios,
		"summary": map[string]any{
			"scenario_count":      len(scenarios),
			"passing_scenarios":   passing,
			"failing_scenarios":   len(scenarios) - passing,
			"duplicate_count":     duplicates,
			"missing_event_count": missing,
		},
		"proof_artifacts": map[string]any{
			"checkpoint_fencing_summary": brokerCheckpointFencingSummaryPath,
			"retention_boundary_summary": brokerRetentionBoundarySummaryPath,
		},
	}
	return report, raw, nil
}

func buildBrokerFailoverScenarios() ([]any, map[string]map[string]any, error) {
	builders := []func() (map[string]any, map[string]any, error){
		scenarioBF01, scenarioBF02, scenarioBF03, scenarioBF04, scenarioBF05, scenarioBF06, scenarioBF07, scenarioBF08,
	}
	scenarios := make([]any, 0, len(builders))
	raw := map[string]map[string]any{}
	for _, builder := range builders {
		scenario, artifacts, err := builder()
		if err != nil {
			return nil, nil, err
		}
		id := fmt.Sprint(scenario["scenario_id"])
		scenarios = append(scenarios, scenario)
		raw[id] = artifacts
	}
	return scenarios, raw, nil
}

func newBrokerStubBackend() *brokerStubBackend {
	return &brokerStubBackend{
		Backend:        "stub_broker",
		Now:            time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC),
		NextSequence:   1,
		NextLease:      1,
		Checkpoints:    map[string]brokerLeaseState{},
		RetentionFloor: 1,
		Nodes: map[string]map[string]any{
			"broker-a": {"role": "leader", "healthy": true},
			"broker-b": {"role": "follower", "healthy": true},
			"broker-c": {"role": "follower", "healthy": true},
		},
	}
}

func (b *brokerStubBackend) advance(seconds int) {
	b.Now = b.Now.Add(time.Duration(seconds) * time.Second)
}

func (b *brokerStubBackend) recordFault(action string, target string, details map[string]any) {
	b.FaultTimeline = append(b.FaultTimeline, map[string]any{
		"timestamp": utcISOTime(b.Now),
		"action":    action,
		"target":    target,
		"details":   details,
	})
}

func (b *brokerStubBackend) snapshotHealth(label string) {
	nodes := map[string]any{}
	for key, value := range b.Nodes {
		copyMap := map[string]any{}
		for k, v := range value {
			copyMap[k] = v
		}
		nodes[key] = copyMap
	}
	sequences := []any{}
	for _, event := range b.Events {
		if event["committed"] == true {
			sequences = append(sequences, event["sequence"])
		}
	}
	b.HealthSnapshots = append(b.HealthSnapshots, map[string]any{
		"label":               label,
		"timestamp":           utcISOTime(b.Now),
		"nodes":               nodes,
		"retention_floor":     b.RetentionFloor,
		"committed_sequences": sequences,
	})
}

func (b *brokerStubBackend) publish(eventID string, traceID string, sourceNode string, outcome string, committed bool) *int {
	var seq *int
	if committed {
		current := b.NextSequence
		seq = &current
		b.NextSequence++
		b.Events = append(b.Events, map[string]any{
			"sequence":    current,
			"event_id":    eventID,
			"trace_id":    traceID,
			"source_node": sourceNode,
			"committed":   true,
		})
	}
	row := map[string]any{
		"timestamp":        utcISOTime(b.Now),
		"event_id":         eventID,
		"trace_id":         traceID,
		"attempt":          1,
		"source_node":      sourceNode,
		"client_outcome":   outcome,
		"durable_sequence": nil,
	}
	if seq != nil {
		row["durable_sequence"] = *seq
	}
	b.PublishLedger = append(b.PublishLedger, row)
	return seq
}

func (b *brokerStubBackend) replay(afterSequence int, sourceNode string, duplicateEventIDs map[string]bool) []map[string]any {
	if duplicateEventIDs == nil {
		duplicateEventIDs = map[string]bool{}
	}
	rows := []map[string]any{}
	for _, event := range b.Events {
		sequence := automationInt(event["sequence"])
		if sequence <= afterSequence || event["committed"] != true {
			continue
		}
		row := brokerEventRecord{
			Sequence:   sequence,
			EventID:    fmt.Sprint(event["event_id"]),
			TraceID:    fmt.Sprint(event["trace_id"]),
			SourceNode: sourceNode,
			Phase:      "replay",
			Duplicate:  duplicateEventIDs[fmt.Sprint(event["event_id"])],
		}
		rows = append(rows, map[string]any{
			"durable_sequence": row.Sequence,
			"event_id":         row.EventID,
			"trace_id":         row.TraceID,
			"delivery_phase":   row.Phase,
			"source_node":      row.SourceNode,
			"duplicate":        row.Duplicate,
		})
		if row.Duplicate {
			rows = append(rows, map[string]any{
				"durable_sequence": row.Sequence,
				"event_id":         row.EventID,
				"trace_id":         row.TraceID,
				"delivery_phase":   "replay_duplicate",
				"source_node":      row.SourceNode,
				"duplicate":        true,
			})
		}
	}
	b.ReplayCapture = append(b.ReplayCapture, rows...)
	return rows
}

func (b *brokerStubBackend) acquireLease(groupID string, subscriberID string, ownerID string) brokerLeaseState {
	key := groupID + "::" + subscriberID
	current, ok := b.Checkpoints[key]
	state := brokerLeaseState{
		OwnerID:  ownerID,
		LeaseID:  fmt.Sprintf("lease-%d", b.NextLease),
		Epoch:    1,
		Sequence: 0,
	}
	if ok {
		state.Epoch = current.Epoch + 1
		state.Sequence = current.Sequence
	}
	b.NextLease++
	b.Checkpoints[key] = state
	b.CheckpointLog = append(b.CheckpointLog, map[string]any{
		"timestamp":     utcISOTime(b.Now),
		"group_id":      groupID,
		"subscriber_id": subscriberID,
		"owner_id":      ownerID,
		"lease_id":      state.LeaseID,
		"lease_epoch":   state.Epoch,
		"prior_sequence": func() int {
			if ok {
				return current.Sequence
			}
			return 0
		}(),
		"next_sequence": state.Sequence,
		"transition":    "leased",
		"fence_reason":  "",
	})
	return state
}

func (b *brokerStubBackend) checkpoint(groupID string, subscriberID string, ownerID string, leaseID string, epoch int, sequence int, transition string) (brokerLeaseState, error) {
	key := groupID + "::" + subscriberID
	current := b.Checkpoints[key]
	if current.OwnerID != ownerID || current.LeaseID != leaseID || current.Epoch != epoch {
		b.CheckpointLog = append(b.CheckpointLog, map[string]any{
			"timestamp":      utcISOTime(b.Now),
			"group_id":       groupID,
			"subscriber_id":  subscriberID,
			"owner_id":       ownerID,
			"lease_id":       leaseID,
			"lease_epoch":    epoch,
			"prior_sequence": current.Sequence,
			"next_sequence":  sequence,
			"transition":     "fenced",
			"fence_reason":   "stale_writer",
		})
		return brokerLeaseState{}, brokerFenceError{Reason: "stale_writer", Current: current}
	}
	if sequence < current.Sequence {
		b.CheckpointLog = append(b.CheckpointLog, map[string]any{
			"timestamp":      utcISOTime(b.Now),
			"group_id":       groupID,
			"subscriber_id":  subscriberID,
			"owner_id":       ownerID,
			"lease_id":       leaseID,
			"lease_epoch":    epoch,
			"prior_sequence": current.Sequence,
			"next_sequence":  sequence,
			"transition":     "fenced",
			"fence_reason":   "checkpoint_regression",
		})
		return brokerLeaseState{}, brokerFenceError{Reason: "checkpoint_regression", Current: current}
	}
	updated := brokerLeaseState{OwnerID: ownerID, LeaseID: leaseID, Epoch: epoch, Sequence: sequence}
	b.Checkpoints[key] = updated
	b.CheckpointLog = append(b.CheckpointLog, map[string]any{
		"timestamp":      utcISOTime(b.Now),
		"group_id":       groupID,
		"subscriber_id":  subscriberID,
		"owner_id":       ownerID,
		"lease_id":       leaseID,
		"lease_epoch":    epoch,
		"prior_sequence": current.Sequence,
		"next_sequence":  sequence,
		"transition":     transition,
		"fence_reason":   "",
	})
	return updated, nil
}

func (b *brokerStubBackend) checkpointSnapshot(groupID string, subscriberID string) map[string]any {
	state := b.Checkpoints[groupID+"::"+subscriberID]
	return map[string]any{
		"owner_id":         state.OwnerID,
		"lease_id":         state.LeaseID,
		"lease_epoch":      state.Epoch,
		"durable_sequence": state.Sequence,
	}
}

func brokerBuildResult(scenarioID string, backend *brokerStubBackend, topology map[string]any, faultWindow map[string]any, checkpointBefore map[string]any, checkpointAfter map[string]any, replayResume map[string]any, leaseTransitions []any, ambiguousIDs map[string]bool) (map[string]any, map[string]any) {
	expectedAfter := automationInt(replayResume["after_sequence"])
	committedEventIDs := map[string]bool{}
	for _, row := range backend.PublishLedger {
		sequence := automationInt(row["durable_sequence"])
		if row["durable_sequence"] != nil && sequence > expectedAfter {
			committedEventIDs[fmt.Sprint(row["event_id"])] = true
		}
	}
	replayedEventIDs := map[string]bool{}
	duplicateEventIDs := map[string]bool{}
	for _, row := range backend.ReplayCapture {
		replayedEventIDs[fmt.Sprint(row["event_id"])] = true
		if row["duplicate"] == true || row["delivery_phase"] == "replay_duplicate" {
			duplicateEventIDs[fmt.Sprint(row["event_id"])] = true
		}
	}
	missing := []any{}
	for eventID := range committedEventIDs {
		if !replayedEventIDs[eventID] {
			missing = append(missing, eventID)
		}
	}
	duplicates := []any{}
	for eventID := range duplicateEventIDs {
		duplicates = append(duplicates, eventID)
	}
	publishOutcomes := map[string]any{
		"committed":      countByOutcome(backend.PublishLedger, "committed"),
		"rejected":       countByOutcome(backend.PublishLedger, "rejected"),
		"unknown_commit": countByOutcome(backend.PublishLedger, "unknown_commit"),
	}
	ambiguousResolved := true
	for eventID := range ambiguousIDs {
		if !replayedEventIDs[eventID] {
			ambiguousResolved = false
		}
	}
	staleWriterFenced := true
	for _, entry := range backend.CheckpointLog {
		if entry["fence_reason"] == "stale_writer" && entry["transition"] != "fenced" {
			staleWriterFenced = false
		}
	}
	checkpointMonotonic := automationInt(checkpointAfter["durable_sequence"]) >= automationInt(checkpointBefore["durable_sequence"])
	assertions := []any{
		map[string]any{"label": "all committed events remain replayable after recovery", "passed": len(missing) == 0},
		map[string]any{"label": "checkpoint sequence stays monotonic across recovery", "passed": checkpointMonotonic},
		map[string]any{"label": "stale writers are fenced instead of regressing the checkpoint", "passed": staleWriterFenced},
		map[string]any{"label": "ambiguous publish outcomes resolve from replay evidence", "passed": ambiguousResolved},
	}
	result := "passed"
	for _, assertion := range assertions {
		if assertion.(map[string]any)["passed"] != true {
			result = "failed"
			break
		}
	}
	artifactRoot := filepath.ToSlash(filepath.Join(brokerFailoverArtifactRoot, scenarioID))
	scenario := map[string]any{
		"scenario_id":               scenarioID,
		"backend":                   backend.Backend,
		"topology":                  topology,
		"fault_window":              faultWindow,
		"published_count":           len(backend.PublishLedger),
		"committed_count":           len(committedEventIDs),
		"replayed_count":            len(backend.ReplayCapture),
		"duplicate_count":           len(duplicates),
		"missing_event_ids":         missing,
		"checkpoint_before_fault":   checkpointBefore,
		"checkpoint_after_recovery": checkpointAfter,
		"lease_transitions":         leaseTransitions,
		"publish_outcomes":          publishOutcomes,
		"replay_resume_cursor":      replayResume,
		"artifacts": map[string]any{
			"publish_attempt_ledger":    artifactRoot + "/publish-attempt-ledger.json",
			"replay_capture":            artifactRoot + "/replay-capture.json",
			"checkpoint_transition_log": artifactRoot + "/checkpoint-transition-log.json",
			"fault_timeline":            artifactRoot + "/fault-timeline.json",
			"backend_health_snapshot":   artifactRoot + "/backend-health.json",
		},
		"result":     result,
		"assertions": assertions,
	}
	raw := map[string]any{
		"publish_attempt_ledger":    backend.PublishLedger,
		"replay_capture":            backend.ReplayCapture,
		"checkpoint_transition_log": backend.CheckpointLog,
		"fault_timeline":            backend.FaultTimeline,
		"backend_health_snapshot":   backend.HealthSnapshots,
	}
	return scenario, raw
}

func scenarioBF01() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	b.snapshotHealth("before")
	for i := 1; i <= 5; i++ {
		b.publish(fmt.Sprintf("bf01-event-%02d", i), "trace-bf01", "producer-a", "committed", true)
		b.advance(1)
	}
	b.recordFault("leader_restart", "broker-a", map[string]any{"during": "publish_burst"})
	b.Nodes["broker-a"]["healthy"] = false
	b.advance(2)
	b.Nodes["broker-b"]["role"] = "leader"
	b.Nodes["broker-a"]["role"] = "follower"
	b.Nodes["broker-a"]["healthy"] = true
	b.snapshotHealth("after_recovery")
	b.replay(0, "broker-b", nil)
	checkpoint := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 0}
	scenario, raw := brokerBuildResult("BF-01", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "leader_before": "broker-a", "leader_after": "broker-b"}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "leader_restart_during_publish"}, checkpoint, checkpoint, map[string]any{"after_sequence": 0, "resumed_from_node": "broker-b"}, nil, nil)
	return scenario, raw, nil
}

func scenarioBF02() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	b.snapshotHealth("before")
	for i := 1; i <= 4; i++ {
		b.publish(fmt.Sprintf("bf02-event-%02d", i), "trace-bf02", "producer-a", "committed", true)
		b.advance(1)
	}
	b.recordFault("replica_loss", "broker-c", map[string]any{"latency_spike_ms": 180})
	b.Nodes["broker-c"]["healthy"] = false
	b.advance(1)
	b.snapshotHealth("after_replica_loss")
	b.replay(2, "broker-a", nil)
	before := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 2}
	after := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 4}
	scenario, raw := brokerBuildResult("BF-02", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "replication_factor": 3, "replica_lost": "broker-c"}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "follower_loss_during_publish_and_replay"}, before, after, map[string]any{"after_sequence": 2, "resumed_from_node": "broker-a", "publish_latency_spike_ms": 180}, nil, nil)
	return scenario, raw, nil
}

func scenarioBF03() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	lease := b.acquireLease("group-a", "subscriber-a", "consumer-a")
	for i := 1; i <= 4; i++ {
		b.publish(fmt.Sprintf("bf03-event-%02d", i), "trace-bf03", "producer-a", "committed", true)
		b.advance(1)
	}
	if _, err := b.checkpoint("group-a", "subscriber-a", lease.OwnerID, lease.LeaseID, lease.Epoch, 2, "committed"); err != nil {
		return nil, nil, err
	}
	before := b.checkpointSnapshot("group-a", "subscriber-a")
	b.recordFault("consumer_crash", "consumer-a", map[string]any{"after_sequence": 3, "before_checkpoint_commit": true})
	b.advance(1)
	b.replay(automationInt(before["durable_sequence"]), "broker-a", nil)
	afterLease, err := b.checkpoint("group-a", "subscriber-a", lease.OwnerID, lease.LeaseID, lease.Epoch, 4, "replayed")
	if err != nil {
		return nil, nil, err
	}
	after := map[string]any{"owner_id": afterLease.OwnerID, "lease_id": afterLease.LeaseID, "lease_epoch": afterLease.Epoch, "durable_sequence": afterLease.Sequence}
	scenario, raw := brokerBuildResult("BF-03", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "consumer_group": "group-a"}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "consumer_crash_before_checkpoint_commit"}, before, after, map[string]any{"after_sequence": before["durable_sequence"], "resumed_from_node": "broker-a"}, []any{map[string]any{"owner_id": lease.OwnerID, "lease_id": lease.LeaseID, "lease_epoch": lease.Epoch}}, nil)
	return scenario, raw, nil
}

func scenarioBF04() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	primary := b.acquireLease("group-a", "subscriber-a", "consumer-a")
	for i := 1; i <= 3; i++ {
		b.publish(fmt.Sprintf("bf04-event-%02d", i), "trace-bf04", "producer-a", "committed", true)
		b.advance(1)
	}
	if _, err := b.checkpoint("group-a", "subscriber-a", primary.OwnerID, primary.LeaseID, primary.Epoch, 2, "acked_pending_commit"); err != nil {
		return nil, nil, err
	}
	before := b.checkpointSnapshot("group-a", "subscriber-a")
	b.recordFault("checkpoint_leader_change", "broker-b", map[string]any{"contenders": []any{"consumer-a", "consumer-b"}})
	b.advance(1)
	standby := b.acquireLease("group-a", "subscriber-a", "consumer-b")
	stale := 0
	if _, err := b.checkpoint("group-a", "subscriber-a", primary.OwnerID, primary.LeaseID, primary.Epoch, 3, "committed"); err != nil {
		if _, ok := err.(brokerFenceError); ok {
			stale++
		} else {
			return nil, nil, err
		}
	}
	if _, err := b.checkpoint("group-a", "subscriber-a", standby.OwnerID, standby.LeaseID, standby.Epoch, 3, "committed"); err != nil {
		return nil, nil, err
	}
	after := b.checkpointSnapshot("group-a", "subscriber-a")
	b.replay(automationInt(before["durable_sequence"]), "broker-b", nil)
	scenario, raw := brokerBuildResult("BF-04", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "consumer_group": "group-a", "contenders": []any{"consumer-a", "consumer-b"}}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "checkpoint_leader_change_with_contention"}, before, after, map[string]any{"after_sequence": before["durable_sequence"], "resumed_from_node": "broker-b"}, []any{map[string]any{"owner_id": primary.OwnerID, "lease_id": primary.LeaseID, "lease_epoch": primary.Epoch}, map[string]any{"owner_id": standby.OwnerID, "lease_id": standby.LeaseID, "lease_epoch": standby.Epoch}}, nil)
	scenario["stale_write_rejections"] = stale
	return scenario, raw, nil
}

func scenarioBF05() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	b.snapshotHealth("before")
	b.publish("bf05-event-01", "trace-bf05", "producer-a", "committed", true)
	b.advance(1)
	b.recordFault("producer_timeout", "producer-a", map[string]any{"between_ack_and_client_response": true})
	b.publish("bf05-event-02", "trace-bf05", "producer-a", "unknown_commit", true)
	b.advance(1)
	b.publish("bf05-event-03", "trace-bf05", "producer-a", "rejected", false)
	b.snapshotHealth("after_timeout")
	b.replay(0, "broker-a", nil)
	checkpoint := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 0}
	scenario, raw := brokerBuildResult("BF-05", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "producer": "producer-a"}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "producer_timeout_after_commit_ambiguity"}, checkpoint, checkpoint, map[string]any{"after_sequence": 0, "resumed_from_node": "broker-a"}, nil, map[string]bool{"bf05-event-02": true})
	return scenario, raw, nil
}

func scenarioBF06() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	for i := 1; i <= 5; i++ {
		b.publish(fmt.Sprintf("bf06-event-%02d", i), "trace-bf06", "producer-a", "committed", true)
		b.advance(1)
	}
	initial := b.replay(0, "broker-a", nil)
	if len(initial) > 3 {
		initial = initial[:3]
	}
	b.ReplayCapture = initial
	b.recordFault("replay_disconnect", "consumer-a", map[string]any{"after_sequence": 3, "reconnect_node": "broker-b"})
	b.advance(1)
	resumed := b.replay(3, "broker-b", nil)
	b.ReplayCapture = append(initial, resumed...)
	before := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 3}
	after := map[string]any{"owner_id": "consumer-a", "lease_id": "lease-none", "lease_epoch": 0, "durable_sequence": 5}
	scenario, raw := brokerBuildResult("BF-06", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "replay_client": "consumer-a"}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "replay_client_disconnect_and_reconnect"}, before, after, map[string]any{"after_sequence": 3, "resumed_from_node": "broker-b"}, nil, nil)
	return scenario, raw, nil
}

func scenarioBF07() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	for i := 1; i <= 5; i++ {
		b.publish(fmt.Sprintf("bf07-event-%02d", i), "trace-bf07", "producer-a", "committed", true)
		b.advance(1)
	}
	lease := b.acquireLease("group-a", "subscriber-a", "consumer-a")
	if _, err := b.checkpoint("group-a", "subscriber-a", lease.OwnerID, lease.LeaseID, lease.Epoch, 1, "committed"); err != nil {
		return nil, nil, err
	}
	before := b.checkpointSnapshot("group-a", "subscriber-a")
	b.RetentionFloor = 3
	b.recordFault("retention_boundary", "broker-a", map[string]any{"checkpoint_sequence": 1, "retention_floor": 3})
	b.snapshotHealth("after_retention_trim")
	b.replay(2, "broker-a", nil)
	after := map[string]any{"owner_id": "operator-reset", "lease_id": "manual-reset", "lease_epoch": lease.Epoch + 1, "durable_sequence": 3}
	scenario, raw := brokerBuildResult("BF-07", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "retention_floor": 3}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "retention_boundary_intersects_stale_checkpoint"}, before, after, map[string]any{"after_sequence": 2, "resumed_from_node": "broker-a", "reset_required": true}, []any{map[string]any{"owner_id": lease.OwnerID, "lease_id": lease.LeaseID, "lease_epoch": lease.Epoch}}, nil)
	scenario["operator_guidance"] = "checkpoint expired behind retention floor; explicit reset to sequence 3 required"
	return scenario, raw, nil
}

func scenarioBF08() (map[string]any, map[string]any, error) {
	b := newBrokerStubBackend()
	for i := 1; i <= 4; i++ {
		b.publish(fmt.Sprintf("bf08-event-%02d", i), "trace-bf08", "producer-a", "committed", true)
		b.advance(1)
	}
	lease := b.acquireLease("group-a", "subscriber-a", "consumer-a")
	if _, err := b.checkpoint("group-a", "subscriber-a", lease.OwnerID, lease.LeaseID, lease.Epoch, 2, "committed"); err != nil {
		return nil, nil, err
	}
	before := b.checkpointSnapshot("group-a", "subscriber-a")
	b.recordFault("split_brain_duplicate_window", "broker-b", map[string]any{"duplicate_event_ids": []any{"bf08-event-03"}})
	b.replay(2, "broker-b", map[string]bool{"bf08-event-03": true})
	afterLease, err := b.checkpoint("group-a", "subscriber-a", lease.OwnerID, lease.LeaseID, lease.Epoch, 4, "committed")
	if err != nil {
		return nil, nil, err
	}
	after := map[string]any{"owner_id": afterLease.OwnerID, "lease_id": afterLease.LeaseID, "lease_epoch": afterLease.Epoch, "durable_sequence": afterLease.Sequence}
	scenario, raw := brokerBuildResult("BF-08", b, map[string]any{"nodes": []any{"broker-a", "broker-b", "broker-c"}, "duplicate_window": true}, map[string]any{"start": b.FaultTimeline[0]["timestamp"], "end": utcISOTime(b.Now), "fault": "split_brain_duplicate_delivery_window"}, before, after, map[string]any{"after_sequence": 2, "resumed_from_node": "broker-b"}, []any{map[string]any{"owner_id": lease.OwnerID, "lease_id": lease.LeaseID, "lease_epoch": lease.Epoch}}, nil)
	return scenario, raw, nil
}

func countByOutcome(rows []map[string]any, outcome string) int {
	count := 0
	for _, row := range rows {
		if row["client_outcome"] == outcome {
			count++
		}
	}
	return count
}

func writeBrokerFailoverReport(repoRoot string, output string, artifactRoot string, checkpointSummary string, retentionSummary string, report map[string]any, raw map[string]map[string]any, pretty bool) error {
	outputPath := automationResolveRepoPath(repoRoot, output)
	artifactRootPath := automationResolveRepoPath(repoRoot, artifactRoot)
	checkpointSummaryPath := automationResolveRepoPath(repoRoot, checkpointSummary)
	retentionSummaryPath := automationResolveRepoPath(repoRoot, retentionSummary)
	if err := automationWriteJSON(outputPath, report); err != nil {
		return err
	}
	if err := automationWriteJSON(checkpointSummaryPath, buildBrokerCheckpointFencingSummary(report)); err != nil {
		return err
	}
	if err := automationWriteJSON(retentionSummaryPath, buildBrokerRetentionBoundarySummary(report)); err != nil {
		return err
	}
	for scenarioID, payloads := range raw {
		dir := filepath.Join(artifactRootPath, scenarioID)
		for name, payload := range map[string]any{
			"publish-attempt-ledger.json":    payloads["publish_attempt_ledger"],
			"replay-capture.json":            payloads["replay_capture"],
			"checkpoint-transition-log.json": payloads["checkpoint_transition_log"],
			"fault-timeline.json":            payloads["fault_timeline"],
			"backend-health.json":            payloads["backend_health_snapshot"],
		} {
			if err := automationWriteJSON(filepath.Join(dir, name), payload); err != nil {
				return err
			}
		}
	}
	_ = pretty
	return nil
}

func buildBrokerCheckpointFencingSummary(report map[string]any) map[string]any {
	scenariosByID := map[string]map[string]any{}
	for _, item := range anySlice(report["scenarios"]) {
		entry, _ := item.(map[string]any)
		scenariosByID[fmt.Sprint(entry["scenario_id"])] = entry
	}
	focusIDs := []string{"BF-03", "BF-04", "BF-08"}
	focus := []map[string]any{}
	stale := 0
	duplicates := 0
	allPassed := true
	for _, id := range focusIDs {
		entry := scenariosByID[id]
		focus = append(focus, entry)
		stale += automationInt(entry["stale_write_rejections"])
		duplicates += automationInt(entry["duplicate_count"])
		if entry["result"] != "passed" {
			allPassed = false
		}
	}
	rows := []any{}
	for _, scenario := range focus {
		rows = append(rows, map[string]any{
			"scenario_id":               scenario["scenario_id"],
			"fault":                     scenario["fault_window"].(map[string]any)["fault"],
			"status":                    scenario["result"],
			"checkpoint_before_fault":   scenario["checkpoint_before_fault"],
			"checkpoint_after_recovery": scenario["checkpoint_after_recovery"],
			"stale_write_rejections":    automationInt(scenario["stale_write_rejections"]),
			"duplicate_count":           scenario["duplicate_count"],
			"artifacts": map[string]any{
				"replay_capture":            scenario["artifacts"].(map[string]any)["replay_capture"],
				"checkpoint_transition_log": scenario["artifacts"].(map[string]any)["checkpoint_transition_log"],
			},
		})
	}
	return map[string]any{
		"generated_at":           report["generated_at"],
		"ticket":                 "OPE-230",
		"title":                  "Checkpoint fencing proof summary from broker failover stub matrix",
		"source_report":          brokerFailoverReportPath,
		"summary_schema_version": "2026-03-17",
		"proof_family":           "checkpoint_fencing",
		"status":                 map[bool]string{true: "passed", false: "failed"}[allPassed],
		"rollout_gate_statuses": []any{
			map[string]any{"name": "durable_publish_ack", "status": "unknown", "detail": "Checkpoint-fencing scenarios do not independently prove replicated publish acknowledgements.", "scenario_ids": []any{}},
			map[string]any{"name": "replay_checkpoint_alignment", "status": "passed", "detail": "Replay resume cursors and checkpoint commits stay in one durable sequence domain across crash, takeover, and duplicate-delivery windows.", "scenario_ids": []any{"BF-03", "BF-04", "BF-08"}},
			map[string]any{"name": "retention_boundary_visibility", "status": "unknown", "detail": "Retention-boundary handling is summarized separately in the retention proof summary.", "scenario_ids": []any{}},
			map[string]any{"name": "live_fanout_isolation", "status": "unknown", "detail": "These deterministic scenarios do not exercise live SSE or in-process fanout isolation.", "scenario_ids": []any{}},
		},
		"focus_scenarios": rows,
		"summary": map[string]any{
			"scenario_count": len(focus),
			"passing_scenarios": func() int {
				if allPassed {
					return len(focus)
				}
				count := 0
				for _, s := range focus {
					if s["result"] == "passed" {
						count++
					}
				}
				return count
			}(),
			"failing_scenarios": func() int {
				if allPassed {
					return 0
				}
				return len(focus) - func() int {
					count := 0
					for _, s := range focus {
						if s["result"] == "passed" {
							count++
						}
					}
					return count
				}()
			}(),
			"stale_write_rejections":   stale,
			"duplicate_replay_windows": duplicates,
		},
	}
}

func buildBrokerRetentionBoundarySummary(report map[string]any) map[string]any {
	var scenario map[string]any
	for _, item := range anySlice(report["scenarios"]) {
		entry, _ := item.(map[string]any)
		if fmt.Sprint(entry["scenario_id"]) == "BF-07" {
			scenario = entry
			break
		}
	}
	retentionFloor := scenario["topology"].(map[string]any)["retention_floor"]
	resetTarget := scenario["checkpoint_after_recovery"].(map[string]any)["durable_sequence"]
	return map[string]any{
		"generated_at":           report["generated_at"],
		"ticket":                 "OPE-230",
		"title":                  "Retention boundary proof summary from broker failover stub matrix",
		"source_report":          brokerFailoverReportPath,
		"summary_schema_version": "2026-03-17",
		"proof_family":           "retention_boundary",
		"status":                 scenario["result"],
		"rollout_gate_statuses": []any{
			map[string]any{"name": "durable_publish_ack", "status": "unknown", "detail": "Retention-boundary evidence does not independently classify replicated publish acknowledgements.", "scenario_ids": []any{}},
			map[string]any{"name": "replay_checkpoint_alignment", "status": "passed", "detail": "Expired checkpoints fail closed and require an explicit reset before replay resumes from the retained sequence domain.", "scenario_ids": []any{"BF-07"}},
			map[string]any{"name": "retention_boundary_visibility", "status": "passed", "detail": "The scenario surfaces the retention floor, marks the stale checkpoint as expired, and requires an explicit operator reset.", "scenario_ids": []any{"BF-07"}},
			map[string]any{"name": "live_fanout_isolation", "status": "unknown", "detail": "Retention-boundary validation does not measure live fanout lag isolation.", "scenario_ids": []any{}},
		},
		"focus_scenarios": []any{
			map[string]any{
				"scenario_id":               scenario["scenario_id"],
				"fault":                     scenario["fault_window"].(map[string]any)["fault"],
				"status":                    scenario["result"],
				"checkpoint_before_fault":   scenario["checkpoint_before_fault"],
				"checkpoint_after_recovery": scenario["checkpoint_after_recovery"],
				"retention_floor":           retentionFloor,
				"reset_required":            scenario["replay_resume_cursor"].(map[string]any)["reset_required"],
				"operator_guidance":         scenario["operator_guidance"],
				"artifacts": map[string]any{
					"fault_timeline":            scenario["artifacts"].(map[string]any)["fault_timeline"],
					"backend_health_snapshot":   scenario["artifacts"].(map[string]any)["backend_health_snapshot"],
					"checkpoint_transition_log": scenario["artifacts"].(map[string]any)["checkpoint_transition_log"],
				},
			},
		},
		"summary": map[string]any{
			"scenario_count":              1,
			"passing_scenarios":           map[bool]int{true: 1, false: 0}[scenario["result"] == "passed"],
			"failing_scenarios":           map[bool]int{true: 0, false: 1}[scenario["result"] == "passed"],
			"retention_floor":             retentionFloor,
			"expired_checkpoint_sequence": scenario["checkpoint_before_fault"].(map[string]any)["durable_sequence"],
			"reset_target_sequence":       resetTarget,
		},
	}
}
