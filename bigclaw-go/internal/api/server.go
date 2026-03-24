package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/flow"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type WorkerStatusProvider interface {
	Snapshot() worker.Status
}

type WorkerPoolStatusProvider interface {
	WorkerStatusProvider
	Snapshots() []worker.Status
}

type Server struct {
	Recorder         *observability.Recorder
	Queue            queue.Queue
	Executors        []domain.ExecutorKind
	Bus              *events.Bus
	EventPlan        events.DurabilityPlan
	EventLog         events.EventLog
	SubscriberLeases events.SubscriberLeaseStore
	Now              func() time.Time
	Worker           WorkerStatusProvider
	Control          *control.Controller
	FlowStore        *flow.Store
	SchedulerPolicy  *scheduler.PolicyStore
	SchedulerRuntime *scheduler.Scheduler
}

type checkpointDiagnostics struct {
	SubscriberID       string                       `json:"subscriber_id"`
	Status             string                       `json:"status"`
	Reason             string                       `json:"reason,omitempty"`
	SuggestedAction    string                       `json:"suggested_action,omitempty"`
	Checkpoint         *events.SubscriberCheckpoint `json:"checkpoint,omitempty"`
	RetentionWatermark *events.RetentionWatermark   `json:"retention_watermark,omitempty"`
}

type checkpointResetFacetCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type checkpointResetSnapshot struct {
	RecentCount  int                           `json:"recent_count"`
	BySubscriber []checkpointResetFacetCount   `json:"by_subscriber"`
	ByReason     []checkpointResetFacetCount   `json:"by_reason"`
	Recent       []events.CheckpointResetAudit `json:"recent"`
}

type checkpointExpiredError struct {
	Diagnostics checkpointDiagnostics
}

func (e checkpointExpiredError) Error() string {
	return fmt.Sprintf("checkpoint for subscriber %s expired: %s", e.Diagnostics.SubscriberID, e.Diagnostics.SuggestedAction)
}

func (s *Server) Handler() http.Handler {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Control == nil {
		s.Control = control.New()
	}
	if s.FlowStore == nil {
		s.FlowStore = flow.NewStore()
	}
	if s.SchedulerPolicy == nil {
		s.SchedulerPolicy = scheduler.NewDefaultPolicyStore()
	}
	if s.SchedulerRuntime == nil {
		s.SchedulerRuntime = scheduler.NewWithStores(s.SchedulerPolicy, nil)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/metrics", s.handleMetrics)
	if store := s.logServiceStore(); store != nil {
		mux.Handle("/internal/events/log/", http.StripPrefix("/internal/events/log", events.NewEventLogServiceHandler(store)))
	}
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		taskID := r.URL.Query().Get("task_id")
		traceID := r.URL.Query().Get("trace_id")
		subscriberID := strings.TrimSpace(r.URL.Query().Get("subscriber_id"))
		afterID, err := s.resolveAfterID(subscriberID, replayCursorFromRequest(r))
		if err != nil {
			var expired checkpointExpiredError
			if errors.As(err, &expired) {
				writeJSON(w, http.StatusConflict, map[string]any{"code": "checkpoint_expired", "error": "checkpoint expired", "checkpoint_diagnostics": expired.Diagnostics})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		eventTypes := parseEventTypes(r.URL.Query()["event_type"])
		if s.EventLog == nil && s.Bus != nil {
			busLimit := limit
			if len(eventTypes) > 0 {
				busLimit = 0
			}
			history, cursor := s.Bus.ReplayWindow(busLimit, afterID, taskID, traceID)
			history = limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit)
			writeReplayCursorHeaders(w, cursor)
			payload := map[string]any{
				"events":        history,
				"cursor":        cursor,
				"subscriber_id": subscriberID,
				"after_id":      afterID,
				"next_after_id": nextAfterID(history, afterID),
				"event_types":   eventTypes,
				"backend":       "memory",
				"durable":       false,
			}
			if watermark := s.retentionWatermark(); watermark != nil {
				payload["retention_watermark"] = watermark
			}
			writeJSON(w, http.StatusOK, payload)
			return
		}
		history, backend, err := s.queryEvents(taskID, traceID, afterID, eventTypes, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if backend == "memory" {
			history = events.WithDeliveryBatch(history, domain.EventDeliveryModeReplay)
		}
		payload := map[string]any{
			"events":        history,
			"subscriber_id": subscriberID,
			"after_id":      afterID,
			"next_after_id": nextAfterID(history, afterID),
			"event_types":   eventTypes,
			"backend":       backend,
			"durable":       s.eventLogDurable(),
		}
		if watermark := s.retentionWatermark(); watermark != nil {
			payload["retention_watermark"] = watermark
		}
		writeJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/subscriber-groups/leases", s.handleSubscriberGroupLease)
	mux.HandleFunc("/subscriber-groups/checkpoints", s.handleSubscriberGroupCheckpoint)
	mux.HandleFunc("/subscriber-groups/", s.handleSubscriberGroupLeaseStatus)
	mux.HandleFunc("/stream/events/checkpoints/", s.handleStreamEventCheckpoint)
	mux.HandleFunc("/stream/events", s.handleStreamEvents)
	mux.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		writeJSON(w, http.StatusOK, map[string]any{"audit": s.Recorder.EventsByTask("", limit)})
	})
	mux.HandleFunc("/replay/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		taskID := strings.TrimPrefix(r.URL.Path, "/replay/")
		if taskID == "" {
			http.Error(w, "missing task id", http.StatusBadRequest)
			return
		}
		recorded := s.Recorder.EventsByTask(taskID, 1000)
		writeJSON(w, http.StatusOK, map[string]any{
			"task_id":           taskID,
			"timeline":          events.WithDeliveryBatch(recorded, domain.EventDeliveryModeReplay),
			"consumer_contract": replayConsumerContract(),
		})
	})
	mux.HandleFunc("/deadletters", s.handleDeadLetters)
	mux.HandleFunc("/deadletters/", s.handleDeadLetterAction)
	mux.HandleFunc("/debug/traces", s.handleDebugTraces)
	mux.HandleFunc("/debug/traces/", s.handleDebugTrace)
	mux.HandleFunc(coordinationLeaderEndpoint, s.handleCoordinationLeader)
	mux.HandleFunc("/debug/status", func(w http.ResponseWriter, r *http.Request) {
		rolloutScorecard := s.EventPlan.RolloutScorecard
		team := strings.TrimSpace(r.URL.Query().Get("team"))
		project := strings.TrimSpace(r.URL.Query().Get("project"))
		clawHostTasks := filterClawHostPolicyTasks(s.clawHostPolicyTasks(r.Context()), team, project)
		payload := map[string]any{
			"queue_size":                      s.Queue.Size(context.Background()),
			"audit_events":                    len(s.Recorder.Logs()),
			"executors":                       s.executorNames(),
			"event_durability":                s.EventPlan,
			"event_durability_rollout":        rolloutScorecard,
			"event_log":                       s.eventLogCapabilities(r.Context()),
			"admission_policy_summary":        admissionPolicySummaryPayload(),
			"coordination_capability_surface": coordinationCapabilitySurfacePayload(),
			"coordination_leader_election":    s.coordinationLeaderElectionPayload(),
			"leader_election_capability":      leaderElectionCapabilitySurfacePayload(),
			"delivery_ack_readiness":          deliveryAckReadinessPayload(),
			"publish_ack_outcomes":            publishAckOutcomeSurfacePayload(),
			"sequence_bridge_surface":         sequenceBridgeSurfacePayload(),
			"retention_expiry_surface":        retentionExpirySurfacePayload(),
			"live_shadow_mirror_scorecard":    liveShadowMirrorPayload(),
			"rollback_trigger_surface":        rollbackTriggerSurfacePayload(),
			"broker_stub_fanout_isolation":    brokerStubFanoutIsolationPayload(),
			"provider_live_handoff_isolation": providerLiveHandoffIsolationPayload(),
			"filters": map[string]any{
				"team":    team,
				"project": project,
			},
			"clawhost_policy_surface":        clawHostPolicySurfacePayload(clawHostTasks, team, project),
			"clawhost_workflow_surface":      clawHostWorkflowSurfacePayload(clawHostTasks),
			"clawhost_rollout_surface":       clawHostRolloutSurfacePayload(clawHostTasks, team, project),
			"clawhost_readiness_surface":     clawHostReadinessSurfacePayload(clawHostTasks),
			"clawhost_recovery_surface":      clawHostRecoverySurfacePayload(clawHostTasks),
			"broker_bootstrap_surface":       brokerBootstrapSurfacePayload(),
			"broker_review_bundle":           brokerReviewBundleSurfacePayload(),
			"validation_bundle_continuation": validationBundleContinuationGatePayload(),
		}
		if s.Worker != nil {
			payload["worker"] = s.Worker.Snapshot()
		}
		if pool := s.workerPoolSummary(); pool != nil {
			payload["worker_pool"] = pool
			if s.Now != nil {
				payload["worker_pool_health"] = workerPoolHealth(s.Now(), pool)
			} else {
				payload["worker_pool_health"] = workerPoolHealth(time.Now(), pool)
			}
		}
		if s.Control != nil {
			payload["control"] = s.Control.Snapshot()
		}
		if watermark := s.retentionWatermark(); watermark != nil {
			payload["retention_watermark"] = watermark
		}
		if s.EventLog != nil {
			payload["event_log"] = map[string]any{
				"backend":      s.EventLog.Backend(),
				"capabilities": s.EventLog.Capabilities(),
			}
		}
		if checkpointResets := s.checkpointResetAuditSnapshot(5); checkpointResets != nil {
			payload["checkpoint_resets"] = checkpointResets
		}
		writeJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/v2/dashboard/engineering", s.handleV2EngineeringDashboard)
	mux.HandleFunc("/v2/dashboard/operations", s.handleV2OperationsDashboard)
	mux.HandleFunc("/v2/triage/center", s.handleV2TriageCenter)
	mux.HandleFunc("/v2/regression/center", s.handleV2RegressionCenter)
	mux.HandleFunc("/v2/control-center", s.handleV2ControlCenter)
	mux.HandleFunc("/v2/control-center/audit", s.handleV2ControlCenterAudit)
	mux.HandleFunc("/v2/control-center/actions", s.handleV2ControlCenterAction)
	mux.HandleFunc("/v2/control-center/policy", s.handleV2ControlCenterPolicy)
	mux.HandleFunc("/v2/control-center/policy/export", s.handleV2ControlCenterPolicyExport)
	mux.HandleFunc("/v2/control-center/policy/reload", s.handleV2ControlCenterPolicyReload)
	mux.Handle("/internal/scheduler/fairness/", http.StripPrefix("/internal/scheduler/fairness", s.schedulerRuntime().FairnessServiceHandler()))
	mux.HandleFunc("/v2/reports/weekly", s.handleV2WeeklyReport)
	mux.HandleFunc("/v2/reports/weekly/export", s.handleV2WeeklyReportExport)
	mux.HandleFunc("/v2/reports/distributed", s.handleV2DistributedReport)
	mux.HandleFunc("/v2/reports/distributed/export", s.handleV2DistributedReportExport)
	mux.HandleFunc("/v2/flows/templates", s.handleV2FlowTemplates)
	mux.HandleFunc("/v2/flows/templates/", s.handleV2FlowTemplateAction)
	mux.HandleFunc("/v2/flows/overview", s.handleV2FlowOverview)
	mux.HandleFunc("/v2/prd/intake", s.handleV2PRDIntake)
	mux.HandleFunc("/v2/intake/connectors/", s.handleV2IntakeConnectorIssues)
	mux.HandleFunc("/v2/intake/issues/map", s.handleV2IntakeIssueMap)
	mux.HandleFunc("/v2/launch/checklist", s.handleV2LaunchChecklist)
	mux.HandleFunc("/v2/support/handoff", s.handleV2SupportHandoff)
	mux.HandleFunc("/v2/workflows/definitions/render", s.handleV2WorkflowDefinitionRender)
	mux.HandleFunc("/v2/navigation", s.handleV2Navigation)
	mux.HandleFunc("/v2/home", s.handleV2Home)
	mux.HandleFunc("/v2/design-system", s.handleV2DesignSystem)
	mux.HandleFunc("/v2/saved-views", s.handleV2SavedViews)
	mux.HandleFunc("/v2/saved-views/export", s.handleV2SavedViewsExport)
	mux.HandleFunc("/v2/dashboard-run-contract", s.handleV2DashboardRunContract)
	mux.HandleFunc("/v2/dashboard-run-contract/export", s.handleV2DashboardRunContractExport)
	mux.HandleFunc("/v2/clawhost/fleet", s.handleV2ClawHostFleet)
	mux.HandleFunc("/v2/clawhost/fleet/export", s.handleV2ClawHostFleetExport)
	mux.HandleFunc("/v2/clawhost/rollout-planner", s.handleV2ClawHostRolloutPlanner)
	mux.HandleFunc("/v2/clawhost/rollout-planner/export", s.handleV2ClawHostRolloutPlannerExport)
	mux.HandleFunc("/v2/clawhost/workflows", s.handleV2ClawHostWorkflows)
	mux.HandleFunc("/v2/clawhost/workflows/export", s.handleV2ClawHostWorkflowsExport)
	mux.HandleFunc("/v2/clawhost/recovery-scorecard", s.handleV2ClawHostRecoveryScorecard)
	mux.HandleFunc("/v2/clawhost/recovery-scorecard/export", s.handleV2ClawHostRecoveryScorecardExport)
	mux.HandleFunc("/v2/billing/usage", s.handleV2BillingUsage)
	mux.HandleFunc("/v2/billing/entitlements", s.handleV2BillingEntitlements)
	mux.HandleFunc("/v2/runs", s.handleV2Runs)
	mux.HandleFunc("/v2/runs/", s.handleV2RunDetail)
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateTask(w, r)
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"message": "use POST /tasks or GET /tasks/{id}"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/", s.handleTaskStatus)
	return mux
}

func replayConsumerContract() map[string]any {
	return map[string]any{
		"event_id_field":         "id",
		"idempotency_key_field":  "delivery.idempotency_key",
		"replay_indicator_field": "delivery.replay",
		"delivery_mode_field":    "delivery.mode",
		"replay_mode":            string(domain.EventDeliveryModeReplay),
		"live_mode":              string(domain.EventDeliveryModeLive),
	}
}

type leaseRequestPayload struct {
	GroupID      string `json:"group_id"`
	SubscriberID string `json:"subscriber_id"`
	ConsumerID   string `json:"consumer_id"`
	TTLSeconds   int64  `json:"ttl_seconds"`
}

type checkpointRequestPayload struct {
	GroupID          string `json:"group_id"`
	SubscriberID     string `json:"subscriber_id"`
	ConsumerID       string `json:"consumer_id"`
	LeaseToken       string `json:"lease_token"`
	LeaseEpoch       int64  `json:"lease_epoch"`
	CheckpointOffset uint64 `json:"checkpoint_offset"`
	CheckpointEvent  string `json:"checkpoint_event_id"`
}

func subscriberLeaseSequenceBridgePayload(lease events.SubscriberLease) map[string]any {
	return map[string]any{
		"portable_sequence": uint64(lease.CheckpointOffset),
		"provider_offset":   uint64(lease.CheckpointOffset),
		"ownership_epoch":   lease.LeaseEpoch,
		"offset_domain":     "lease_checkpoint_offset",
		"mapping_status":    "lease_checkpoint_offset_mirrors_portable_sequence",
	}
}

func checkpointSequenceBridgePayload(checkpoint events.SubscriberCheckpoint) map[string]any {
	return map[string]any{
		"portable_sequence": checkpoint.EventSequence,
		"provider_offset":   checkpoint.EventSequence,
		"offset_domain":     "event_sequence",
		"mapping_status":    "checkpoint_event_sequence_is_portable_sequence",
	}
}

func (s *Server) publishSubscriberLeaseEvent(eventType domain.EventType, now time.Time, lease events.SubscriberLease, payload map[string]any) {
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["group_id"] = lease.GroupID
	payload["subscriber_id"] = lease.SubscriberID
	payload["consumer_id"] = lease.ConsumerID
	payload["lease_token"] = lease.LeaseToken
	payload["lease_epoch"] = lease.LeaseEpoch
	payload["checkpoint_offset"] = lease.CheckpointOffset
	payload["checkpoint_event_id"] = lease.CheckpointEvent
	payload["sequence_bridge"] = subscriberLeaseSequenceBridgePayload(lease)
	if !lease.ExpiresAt.IsZero() {
		payload["expires_at"] = lease.ExpiresAt.UTC()
	}
	if !lease.UpdatedAt.IsZero() {
		payload["updated_at"] = lease.UpdatedAt.UTC()
	}
	s.publish(domain.Event{
		ID:        fmt.Sprintf("subscriber-%s-%s-%d", strings.ReplaceAll(string(eventType), ".", "-"), subscriberLeaseEventKeyPart(lease.GroupID, lease.SubscriberID), now.UnixNano()),
		Type:      eventType,
		Timestamp: now,
		Payload:   payload,
	})
}

func subscriberLeaseEventKeyPart(groupID string, subscriberID string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-", ".", "-")
	group := replacer.Replace(strings.TrimSpace(groupID))
	subscriber := replacer.Replace(strings.TrimSpace(subscriberID))
	if group == "" {
		group = "group"
	}
	if subscriber == "" {
		subscriber = "subscriber"
	}
	return group + "-" + subscriber
}

func (s *Server) handleSubscriberGroupLease(w http.ResponseWriter, r *http.Request) {
	if s.SubscriberLeases == nil {
		http.Error(w, "subscriber lease coordinator unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var payload leaseRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode lease request: %v", err), http.StatusBadRequest)
			return
		}
		now := s.Now()
		previous, hadPrevious := s.SubscriberLeases.Get(payload.GroupID, payload.SubscriberID)
		lease, err := s.SubscriberLeases.Acquire(events.LeaseRequest{
			GroupID:      payload.GroupID,
			SubscriberID: payload.SubscriberID,
			ConsumerID:   payload.ConsumerID,
			TTL:          time.Duration(payload.TTLSeconds) * time.Second,
			Now:          now,
		})
		if err != nil {
			status := http.StatusBadRequest
			if err == events.ErrLeaseHeld {
				status = http.StatusConflict
				s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseRejected, now, lease, map[string]any{
					"attempted_consumer_id": payload.ConsumerID,
					"requested_ttl_seconds": payload.TTLSeconds,
					"reason":                err.Error(),
				})
			}
			writeJSON(w, status, map[string]any{"error": err.Error(), "lease": lease, "sequence_bridge": subscriberLeaseSequenceBridgePayload(lease)})
			return
		}
		expiredTakeover := hadPrevious && previous.ConsumerID != "" && previous.ConsumerID != lease.ConsumerID && !previous.ExpiresAt.IsZero() && !now.Before(previous.ExpiresAt)
		if expiredTakeover {
			s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseExpired, now, previous, map[string]any{
				"expired_consumer_id":   previous.ConsumerID,
				"takeover_consumer_id":  lease.ConsumerID,
				"expired_checkpoint_at": previous.CheckpointOffset,
			})
			s.publishSubscriberLeaseEvent(domain.EventSubscriberTakeoverSucceeded, now, lease, map[string]any{
				"previous_consumer_id":  previous.ConsumerID,
				"takeover":              true,
				"requested_ttl_seconds": payload.TTLSeconds,
			})
		} else {
			s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseAcquired, now, lease, map[string]any{
				"renewal":               hadPrevious && previous.ConsumerID == lease.ConsumerID,
				"requested_ttl_seconds": payload.TTLSeconds,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"lease": lease, "sequence_bridge": subscriberLeaseSequenceBridgePayload(lease)})
	case http.MethodDelete:
		var payload checkpointRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode lease release: %v", err), http.StatusBadRequest)
			return
		}
		err := s.SubscriberLeases.Release(payload.GroupID, payload.SubscriberID, payload.ConsumerID, payload.LeaseToken, payload.LeaseEpoch)
		if err != nil {
			status := http.StatusConflict
			if err == events.ErrLeaseExpired {
				status = http.StatusGone
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"released": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSubscriberGroupCheckpoint(w http.ResponseWriter, r *http.Request) {
	if s.SubscriberLeases == nil {
		http.Error(w, "subscriber lease coordinator unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload checkpointRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("decode checkpoint commit: %v", err), http.StatusBadRequest)
		return
	}
	now := s.Now()
	lease, err := s.SubscriberLeases.Commit(events.CheckpointCommit{
		GroupID:          payload.GroupID,
		SubscriberID:     payload.SubscriberID,
		ConsumerID:       payload.ConsumerID,
		LeaseToken:       payload.LeaseToken,
		LeaseEpoch:       payload.LeaseEpoch,
		CheckpointOffset: payload.CheckpointOffset,
		CheckpointEvent:  payload.CheckpointEvent,
		Now:              now,
	})
	if err != nil {
		status := http.StatusConflict
		switch err {
		case events.ErrCheckpointRollback:
			status = http.StatusPreconditionFailed
		case events.ErrLeaseExpired:
			status = http.StatusGone
		}
		s.publishSubscriberLeaseEvent(domain.EventSubscriberCheckpointRejected, now, lease, map[string]any{
			"attempted_consumer_id":         payload.ConsumerID,
			"attempted_lease_token":         payload.LeaseToken,
			"attempted_lease_epoch":         payload.LeaseEpoch,
			"attempted_checkpoint_offset":   payload.CheckpointOffset,
			"attempted_checkpoint_event_id": payload.CheckpointEvent,
			"reason":                        err.Error(),
		})
		writeJSON(w, status, map[string]any{"error": err.Error(), "lease": lease, "sequence_bridge": subscriberLeaseSequenceBridgePayload(lease)})
		return
	}
	s.publishSubscriberLeaseEvent(domain.EventSubscriberCheckpointCommitted, now, lease, map[string]any{
		"committed_by": payload.ConsumerID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"lease": lease, "sequence_bridge": subscriberLeaseSequenceBridgePayload(lease)})
}

func (s *Server) handleSubscriberGroupLeaseStatus(w http.ResponseWriter, r *http.Request) {
	if s.SubscriberLeases == nil {
		http.Error(w, "subscriber lease coordinator unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/subscriber-groups/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[1] != "subscribers" || parts[0] == "" || parts[2] == "" {
		http.Error(w, "expected /subscriber-groups/{group_id}/subscribers/{subscriber_id}", http.StatusBadRequest)
		return
	}
	lease, ok := s.SubscriberLeases.Get(parts[0], parts[2])
	if !ok {
		http.Error(w, "subscriber lease not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lease": lease, "sequence_bridge": subscriberLeaseSequenceBridgePayload(lease)})
}

func (s *Server) handleDebugTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, map[string]any{"traces": s.Recorder.TraceSummaries(limit)})
}

func (s *Server) handleDebugTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	traceID := strings.TrimPrefix(r.URL.Path, "/debug/traces/")
	if traceID == "" {
		http.Error(w, "missing trace id", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	summary, ok := s.Recorder.TraceSummary(traceID)
	if !ok {
		http.Error(w, "trace not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trace":  summary,
		"events": s.Recorder.EventsByTrace(traceID, limit),
	})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, fmt.Sprintf("decode task: %v", err), http.StatusBadRequest)
		return
	}
	now := s.Now()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", now.UnixNano())
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}
	if task.Title == "" {
		task.Title = task.ID
	}
	if task.TraceID == "" {
		task.TraceID = task.ID
	}
	task.State = domain.TaskQueued
	if err := s.Queue.Enqueue(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf("enqueue task: %v", err), http.StatusInternalServerError)
		return
	}
	s.Recorder.StoreTask(task)
	s.publish(domain.Event{ID: task.ID + "-queued", Type: domain.EventTaskQueued, TaskID: task.ID, TraceID: task.TraceID, Timestamp: now, Payload: map[string]any{"executor": task.RequiredExecutor, "title": task.Title}})
	writeJSON(w, http.StatusAccepted, map[string]any{"task": task, "state": string(task.State)})
}

func (s *Server) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	events := s.Recorder.EventsByTask(taskID, 100)
	task, hasTask := s.Recorder.Task(taskID)
	if len(events) == 0 && !hasTask {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	latest, hasLatest := s.Recorder.LatestByTask(taskID)
	traceID := task.TraceID
	state := string(task.State)
	if traceID == "" && hasLatest {
		traceID = latest.TraceID
	}
	if state == "" && hasLatest {
		state = eventState(latest.Type)
	}
	if state == "" {
		state = "unknown"
	}
	payload := map[string]any{
		"task_id":  taskID,
		"trace_id": traceID,
		"state":    state,
		"events":   events,
	}
	if hasLatest {
		payload["latest_event"] = latest
	}
	if hasTask {
		payload["task"] = task
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleDeadLetters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Queue.ListDeadLetters(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("list dead letters: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dead_letters": items})
}

func (s *Server) handleDeadLetterAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/deadletters/")
	if !strings.HasSuffix(path, "/replay") {
		http.Error(w, "unsupported action", http.StatusNotFound)
		return
	}
	taskID := strings.TrimSuffix(path, "/replay")
	taskID = strings.TrimSuffix(taskID, "/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	if err := s.Queue.ReplayDeadLetter(r.Context(), taskID); err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(err.Error(), "not dead-lettered"):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, fmt.Sprintf("replay dead letter: %v", err), http.StatusInternalServerError)
		}
		return
	}
	now := s.Now()
	s.syncTaskState(taskID, domain.TaskQueued, now)
	traceID := s.traceIDForTask(taskID)
	s.publish(domain.Event{ID: taskID + "-replayed", Type: domain.EventTaskQueued, TaskID: taskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"replayed": true}})
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": taskID, "state": string(domain.TaskQueued), "replayed": true})
}

func (s *Server) handleStreamEventCheckpoint(w http.ResponseWriter, r *http.Request) {
	store := s.checkpointStore()
	if store == nil {
		http.Error(w, "checkpoint store unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/stream/events/checkpoints/")
	historyRequest := false
	if strings.HasSuffix(path, "/history") {
		historyRequest = true
		path = strings.TrimSuffix(path, "/history")
	}
	subscriberID := strings.Trim(path, "/")
	if subscriberID == "" {
		http.Error(w, "missing subscriber id", http.StatusBadRequest)
		return
	}
	if historyRequest {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		historyProvider := s.checkpointResetHistoryProvider()
		if historyProvider == nil {
			http.Error(w, "checkpoint reset history unavailable", http.StatusServiceUnavailable)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		history, err := historyProvider.CheckpointResetHistory(subscriberID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"history": history,
			"backend": s.eventLogBackend(),
			"durable": s.eventLogDurable(),
		})
		return
	}
	switch r.Method {
	case http.MethodGet:
		checkpoint, err := store.Checkpoint(subscriberID)
		if err != nil {
			if events.IsNoEventLog(err) {
				http.Error(w, "checkpoint not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		diagnostics := s.checkpointDiagnosticsForCheckpoint(subscriberID, checkpoint)
		payload := map[string]any{
			"checkpoint":      checkpoint,
			"sequence_bridge": checkpointSequenceBridgePayload(checkpoint),
			"backend":         s.eventLogBackend(),
			"durable":         s.eventLogDurable(),
		}
		if diagnostics != nil {
			payload["checkpoint_diagnostics"] = diagnostics
		}
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPost:
		var payload struct {
			EventID string `json:"event_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode checkpoint ack: %v", err), http.StatusBadRequest)
			return
		}
		checkpoint, err := store.Acknowledge(subscriberID, strings.TrimSpace(payload.EventID), s.Now())
		if err != nil {
			if events.IsNoEventLog(err) {
				http.Error(w, "event not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"checkpoint":      checkpoint,
			"sequence_bridge": checkpointSequenceBridgePayload(checkpoint),
			"backend":         s.eventLogBackend(),
			"durable":         s.eventLogDurable(),
		})
	case http.MethodDelete:
		resetter := s.checkpointResetter()
		if resetter == nil {
			http.Error(w, "checkpoint reset unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := resetter.ResetCheckpoint(subscriberID); err != nil {
			if events.IsNoEventLog(err) {
				http.Error(w, "checkpoint not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payload := map[string]any{
			"subscriber_id": subscriberID,
			"reset":         true,
			"backend":       s.eventLogBackend(),
			"durable":       s.eventLogDurable(),
		}
		if historyProvider := s.checkpointResetHistoryProvider(); historyProvider != nil {
			history, err := historyProvider.CheckpointResetHistory(subscriberID, 1)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(history) > 0 {
				payload["reset_audit"] = history[0]
			}
		}
		writeJSON(w, http.StatusOK, payload)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStreamEvents(w http.ResponseWriter, r *http.Request) {
	if s.Bus == nil {
		http.Error(w, "event bus unavailable", http.StatusServiceUnavailable)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	taskID := r.URL.Query().Get("task_id")
	traceID := r.URL.Query().Get("trace_id")
	subscriberID := strings.TrimSpace(r.URL.Query().Get("subscriber_id"))
	afterID, err := s.resolveAfterID(subscriberID, replayCursorFromRequest(r))
	if err != nil {
		var expired checkpointExpiredError
		if errors.As(err, &expired) {
			writeJSON(w, http.StatusConflict, map[string]any{"code": "checkpoint_expired", "error": "checkpoint expired", "checkpoint_diagnostics": expired.Diagnostics})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eventTypes := parseEventTypes(r.URL.Query()["event_type"])
	eventTypeSet := toEventTypeSet(eventTypes)
	replay := r.URL.Query().Get("replay") == "1" || afterID != ""
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := s.Bus.SubscribeTopic(128, events.SubscriptionFilter{TaskID: taskID, TraceID: traceID, EventTypes: eventTypeSet})
	defer cancel()
	delivered := make(map[string]struct{})
	if replay {
		var history []domain.Event
		if s.EventLog == nil {
			busLimit := limit
			if len(eventTypes) > 0 {
				busLimit = 0
			}
			var cursor events.ReplayCursorStatus
			history, cursor = s.Bus.ReplayWindow(busLimit, afterID, taskID, traceID)
			history = limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit)
			writeReplayCursorHeaders(w, cursor)
		} else {
			var err error
			history, _, err = s.queryEvents(taskID, traceID, afterID, eventTypes, limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			history = events.WithDeliveryBatch(history, domain.EventDeliveryModeReplay)
		}
		for _, event := range history {
			if !markStreamEventDelivered(delivered, event) {
				continue
			}
			if err := writeSSEEvent(w, flusher, event); err != nil {
				return
			}
		}
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if !matchesEventStreamFilter(event, taskID, traceID, eventTypeSet) {
				continue
			}
			if !markStreamEventDelivered(delivered, event) {
				continue
			}
			if err := writeSSEEvent(w, flusher, event); err != nil {
				return
			}
		}
	}
}

func (s *Server) queryEvents(taskID, traceID, afterID string, eventTypes []domain.EventType, limit int) ([]domain.Event, string, error) {
	backendLimit := limit
	if len(eventTypes) > 0 {
		backendLimit = 0
	}
	if s.EventLog != nil {
		var history []domain.Event
		var err error
		if afterID != "" {
			switch {
			case taskID != "":
				history, err = s.EventLog.EventsByTaskAfter(taskID, afterID, backendLimit)
			case traceID != "":
				history, err = s.EventLog.EventsByTraceAfter(traceID, afterID, backendLimit)
			default:
				history, err = s.EventLog.ReplayAfter(afterID, backendLimit)
			}
		} else {
			switch {
			case taskID != "":
				history, err = s.EventLog.EventsByTask(taskID, backendLimit)
			case traceID != "":
				history, err = s.EventLog.EventsByTrace(traceID, backendLimit)
			default:
				history, err = s.EventLog.Replay(backendLimit)
			}
		}
		if err != nil {
			return nil, "sqlite", err
		}
		return limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit), s.eventLogBackend(), nil
	}
	history := s.queryMemoryEvents(taskID, traceID, afterID, backendLimit)
	return limitQueriedEvents(filterEventsByType(history, eventTypes), afterID, limit), "memory", nil
}

func (s *Server) queryMemoryEvents(taskID, traceID, afterID string, limit int) []domain.Event {
	if s.Recorder == nil {
		return nil
	}
	logs := s.Recorder.Logs()
	if afterID == "" {
		filtered := make([]domain.Event, 0, len(logs))
		for _, event := range logs {
			if !matchesEventStreamFilter(event, taskID, traceID, nil) {
				continue
			}
			filtered = append(filtered, event)
		}
		if limit > 0 && len(filtered) > limit {
			filtered = filtered[len(filtered)-limit:]
		}
		out := make([]domain.Event, len(filtered))
		copy(out, filtered)
		return out
	}
	start := 0
	for index, event := range logs {
		if event.ID == afterID {
			start = index + 1
		}
	}
	filtered := make([]domain.Event, 0, len(logs)-start)
	for _, event := range logs[start:] {
		if !matchesEventStreamFilter(event, taskID, traceID, nil) {
			continue
		}
		filtered = append(filtered, event)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func (s *Server) eventLogCapabilities(ctx context.Context) events.BackendCapabilities {
	if s.Bus != nil {
		return s.Bus.Capabilities(ctx)
	}
	return events.UnavailableCapabilities()
}

func replayCursorFromRequest(r *http.Request) string {
	if afterID := strings.TrimSpace(r.URL.Query().Get("after_id")); afterID != "" {
		return afterID
	}
	return strings.TrimSpace(r.Header.Get("Last-Event-ID"))
}

func writeReplayCursorHeaders(w http.ResponseWriter, status events.ReplayCursorStatus) {
	w.Header().Set("X-Replay-Cursor-Status", status.Status)
	w.Header().Set("X-Replay-Fallback", status.Fallback)
	if status.RequestedAfterID != "" {
		w.Header().Set("X-Replay-Requested-After-ID", status.RequestedAfterID)
	}
	if status.OldestEventID != "" {
		w.Header().Set("X-Replay-Oldest-Event-ID", status.OldestEventID)
	}
	if status.NewestEventID != "" {
		w.Header().Set("X-Replay-Newest-Event-ID", status.NewestEventID)
	}
	if status.HistoryTruncated {
		w.Header().Set("X-Replay-History-Truncated", "true")
	}
}

func (s *Server) checkpointStore() events.CheckpointStore {
	if store, ok := s.EventLog.(events.CheckpointStore); ok {
		return store
	}
	return nil
}

func (s *Server) checkpointResetter() events.CheckpointResetter {
	if resetter, ok := s.EventLog.(events.CheckpointResetter); ok {
		return resetter
	}
	return nil
}

func (s *Server) checkpointResetHistoryProvider() events.CheckpointResetHistoryProvider {
	if provider, ok := s.EventLog.(events.CheckpointResetHistoryProvider); ok {
		return provider
	}
	return nil
}

func (s *Server) checkpointResetRecentProvider() events.RecentCheckpointResetProvider {
	if provider, ok := s.EventLog.(events.RecentCheckpointResetProvider); ok {
		return provider
	}
	return nil
}

func (s *Server) logServiceStore() events.LogServiceStore {
	if store, ok := s.EventLog.(events.LogServiceStore); ok {
		return store
	}
	return nil
}

func (s *Server) eventLogBackend() string {
	type backendInfo interface{ Backend() string }
	if store, ok := s.EventLog.(backendInfo); ok {
		return store.Backend()
	}
	if s.EventLog != nil {
		return "sqlite"
	}
	return "memory"
}

func (s *Server) eventLogDurable() bool {
	return s.EventLog != nil
}

func (s *Server) retentionWatermark() any {
	if s.EventLog != nil {
		if provider, ok := s.EventLog.(events.RetentionWatermarkProvider); ok {
			watermark, err := provider.RetentionWatermark()
			if err == nil {
				return watermark
			}
		}
	}
	if s.Bus != nil {
		if provider, ok := any(s.Bus).(events.RetentionWatermarkProvider); ok {
			watermark, err := provider.RetentionWatermark()
			if err == nil {
				return watermark
			}
		}
	}
	return nil
}

func (s *Server) resolveAfterID(subscriberID string, afterID string) (string, error) {
	if afterID != "" || subscriberID == "" {
		return afterID, nil
	}
	store := s.checkpointStore()
	if store == nil {
		return "", nil
	}
	checkpoint, err := store.Checkpoint(subscriberID)
	if err != nil {
		if events.IsNoEventLog(err) {
			return "", nil
		}
		return "", err
	}
	if diagnostics := s.checkpointDiagnosticsForCheckpoint(subscriberID, checkpoint); diagnostics != nil && diagnostics.Status == "expired" {
		return "", checkpointExpiredError{Diagnostics: *diagnostics}
	}
	return checkpoint.EventID, nil
}

func (s *Server) checkpointDiagnosticsForCheckpoint(subscriberID string, checkpoint events.SubscriberCheckpoint) *checkpointDiagnostics {
	diagnostics := &checkpointDiagnostics{
		SubscriberID:    subscriberID,
		Status:          "ok",
		Checkpoint:      &checkpoint,
		SuggestedAction: "resume replay using the saved checkpoint",
	}
	if watermark := s.typedRetentionWatermark(); watermark != nil {
		diagnostics.RetentionWatermark = watermark
		if watermark.TrimmedThroughSequence > 0 && checkpoint.EventSequence > 0 && checkpoint.EventSequence <= watermark.TrimmedThroughSequence {
			diagnostics.Status = "expired"
			diagnostics.Reason = "checkpoint_before_retention_boundary"
			diagnostics.SuggestedAction = "DELETE /stream/events/checkpoints/{subscriber_id} and resume from the earliest retained event"
		}
	}
	return diagnostics
}

func (s *Server) typedRetentionWatermark() *events.RetentionWatermark {
	value := s.retentionWatermark()
	if value == nil {
		return nil
	}
	watermark, ok := value.(events.RetentionWatermark)
	if !ok {
		return nil
	}
	return &watermark
}

func (s *Server) checkpointResetAuditSnapshot(limit int) *checkpointResetSnapshot {
	provider := s.checkpointResetRecentProvider()
	if provider == nil {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	recent, err := provider.RecentCheckpointResets(limit)
	if err != nil {
		return nil
	}
	return &checkpointResetSnapshot{
		RecentCount:  len(recent),
		BySubscriber: summarizeCheckpointResetFacet(recent, func(entry events.CheckpointResetAudit) string { return entry.SubscriberID }),
		ByReason:     summarizeCheckpointResetFacet(recent, func(entry events.CheckpointResetAudit) string { return entry.Reason }),
		Recent:       recent,
	}
}

func summarizeCheckpointResetFacet(entries []events.CheckpointResetAudit, keyFn func(events.CheckpointResetAudit) string) []checkpointResetFacetCount {
	counts := make(map[string]int)
	for _, entry := range entries {
		key := strings.TrimSpace(keyFn(entry))
		if key == "" {
			key = "unknown"
		}
		counts[key]++
	}
	out := make([]checkpointResetFacetCount, 0, len(counts))
	for key, count := range counts {
		out = append(out, checkpointResetFacetCount{Key: key, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func parseEventTypes(values []string) []domain.EventType {
	seen := make(map[domain.EventType]struct{})
	out := make([]domain.EventType, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			eventType := domain.EventType(part)
			if _, ok := seen[eventType]; ok {
				continue
			}
			seen[eventType] = struct{}{}
			out = append(out, eventType)
		}
	}
	return out
}

func toEventTypeSet(eventTypes []domain.EventType) map[domain.EventType]struct{} {
	if len(eventTypes) == 0 {
		return nil
	}
	set := make(map[domain.EventType]struct{}, len(eventTypes))
	for _, eventType := range eventTypes {
		set[eventType] = struct{}{}
	}
	return set
}

func filterEventsByType(events []domain.Event, eventTypes []domain.EventType) []domain.Event {
	if len(eventTypes) == 0 {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	allowed := toEventTypeSet(eventTypes)
	filtered := make([]domain.Event, 0, len(events))
	for _, event := range events {
		if _, ok := allowed[event.Type]; ok {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func limitQueriedEvents(events []domain.Event, afterID string, limit int) []domain.Event {
	if limit <= 0 || len(events) <= limit {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	if afterID != "" {
		out := make([]domain.Event, limit)
		copy(out, events[:limit])
		return out
	}
	out := make([]domain.Event, limit)
	copy(out, events[len(events)-limit:])
	return out
}

func nextAfterID(events []domain.Event, fallback string) string {
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].ID != "" {
			return events[index].ID
		}
	}
	return fallback
}

func markStreamEventDelivered(delivered map[string]struct{}, event domain.Event) bool {
	key := streamEventKey(event)
	if _, ok := delivered[key]; ok {
		return false
	}
	delivered[key] = struct{}{}
	return true
}

func streamEventKey(event domain.Event) string {
	if event.ID != "" {
		return "id:" + event.ID
	}
	return fmt.Sprintf("anon:%s:%s:%s:%s:%d", event.Type, event.TaskID, event.TraceID, event.RunID, event.Timestamp.UnixNano())
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n", payload); err != nil {
		return err
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func matchesEventStreamFilter(event domain.Event, taskID string, traceID string, eventTypes map[domain.EventType]struct{}) bool {
	if taskID != "" && event.TaskID != taskID {
		return false
	}
	if traceID != "" && event.TraceID != traceID {
		return false
	}
	if len(eventTypes) > 0 {
		if _, ok := eventTypes[event.Type]; !ok {
			return false
		}
	}
	return true
}

func (s *Server) publish(event domain.Event) {
	if s.Bus != nil {
		s.Bus.Publish(event)
		return
	}
	if s.Recorder != nil {
		s.Recorder.Record(event)
	}
}

func (s *Server) executorNames() []string {
	names := make([]string, 0, len(s.Executors))
	for _, executor := range s.Executors {
		names = append(names, string(executor))
	}
	sort.Strings(names)
	return names
}

func eventState(eventType domain.EventType) string {
	if state, ok := domain.TaskStateFromEventType(eventType); ok {
		return string(state)
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
