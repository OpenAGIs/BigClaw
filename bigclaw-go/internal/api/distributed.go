package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/risk"
)

type distributedDiagnosticsSummary struct {
	RegisteredExecutors  int `json:"registered_executors"`
	ActiveExecutors      int `json:"active_executors"`
	TotalTasks           int `json:"total_tasks"`
	ActiveRuns           int `json:"active_runs"`
	TotalRoutedDecisions int `json:"total_routed_decisions"`
	ActiveWorkers        int `json:"active_workers"`
	IdleWorkers          int `json:"idle_workers"`
	LeaseRenewalFailures int `json:"lease_renewal_failures"`
	LeaseLostRuns        int `json:"lease_lost_runs"`
	SaturatedExecutors   int `json:"saturated_executors"`
	ActiveTakeovers      int `json:"active_takeovers"`
}

type routingReasonSummary struct {
	Executor string `json:"executor"`
	Reason   string `json:"reason"`
	Count    int    `json:"count"`
}

type executorCapacityView struct {
	Executor             string                 `json:"executor"`
	MaxConcurrency       int                    `json:"max_concurrency"`
	AvailableConcurrency int                    `json:"available_concurrency"`
	ActiveWorkers        int                    `json:"active_workers"`
	QueuedTasks          int                    `json:"queued_tasks"`
	ActiveTasks          int                    `json:"active_tasks"`
	RoutedDecisions      int                    `json:"routed_decisions"`
	StartedRuns          int                    `json:"started_runs"`
	CompletedRuns        int                    `json:"completed_runs"`
	DeadLetters          int                    `json:"dead_letters"`
	SaturationPercent    float64                `json:"saturation_percent"`
	SupportsGPU          bool                   `json:"supports_gpu"`
	SupportsBrowser      bool                   `json:"supports_browser"`
	SupportsShell        bool                   `json:"supports_shell"`
	Health               string                 `json:"health"`
	TeamBreakdown        []auditFacetCount      `json:"team_breakdown,omitempty"`
	ProjectBreakdown     []auditFacetCount      `json:"project_breakdown,omitempty"`
	TopRoutingReasons    []routingReasonSummary `json:"top_routing_reasons,omitempty"`
	HealthSummary        []string               `json:"health_summary,omitempty"`
	SampleTasks          []string               `json:"sample_tasks,omitempty"`
}

type clusterHealthRollup struct {
	HealthyExecutors   int               `json:"healthy_executors"`
	DegradedExecutors  int               `json:"degraded_executors"`
	IdleExecutors      int               `json:"idle_executors"`
	WorkerStates       map[string]int    `json:"worker_states"`
	TeamBreakdown      []auditFacetCount `json:"team_breakdown,omitempty"`
	ProjectBreakdown   []auditFacetCount `json:"project_breakdown,omitempty"`
	TakeoverOwners     []auditFacetCount `json:"takeover_owners,omitempty"`
	SaturatedExecutors []string          `json:"saturated_executors,omitempty"`
	Notes              []string          `json:"notes"`
}

type recoveryDiagnostics struct {
	ActiveTakeovers          int `json:"active_takeovers"`
	TakeoverEvents           int `json:"takeover_events"`
	ReleaseEvents            int `json:"release_events"`
	RetriedRuns              int `json:"retried_runs"`
	DeadLetterRuns           int `json:"dead_letter_runs"`
	LeaseExpiredEvents       int `json:"lease_expired_events"`
	CheckpointRejectedEvents int `json:"checkpoint_rejected_events"`
	UnreleasedTakeovers      int `json:"unreleased_takeovers"`
}

type fairnessExecutorShare struct {
	Executor        string  `json:"executor"`
	RoutedDecisions int     `json:"routed_decisions"`
	ExpectedShare   float64 `json:"expected_share"`
	ActualShare     float64 `json:"actual_share"`
	ShareDelta      float64 `json:"share_delta"`
	Status          string  `json:"status"`
}

type fairnessDiagnostics struct {
	TotalRoutedDecisions int                     `json:"total_routed_decisions"`
	CapacityWeightTotal  int                     `json:"capacity_weight_total"`
	ImbalanceScore       float64                 `json:"imbalance_score"`
	ExecutorShares       []fairnessExecutorShare `json:"executor_shares"`
	Notes                []string                `json:"notes,omitempty"`
}

type distributedDiagnosticsReport struct {
	Markdown  string `json:"markdown"`
	ExportURL string `json:"export_url"`
}

type sharedQueueCoordinationDiagnostics struct {
	DeadLetterBacklog           int   `json:"dead_letter_backlog"`
	DeadLetterEvents            int   `json:"dead_letter_events"`
	ReplayedQueueEvents         int   `json:"replayed_queue_events"`
	LeaseAcquiredEvents         int   `json:"lease_acquired_events"`
	LeaseRejectedEvents         int   `json:"lease_rejected_events"`
	LeaseExpiredEvents          int   `json:"lease_expired_events"`
	TakeoverSucceededEvents     int   `json:"takeover_succeeded_events"`
	CheckpointCommittedEvents   int   `json:"checkpoint_committed_events"`
	CheckpointRejectedEvents    int   `json:"checkpoint_rejected_events"`
	LeaseFencedEvents           int   `json:"lease_fenced_events"`
	CheckpointResetsRecent      int   `json:"checkpoint_resets_recent"`
	RetentionWatermarkAvailable bool  `json:"retention_watermark_available"`
	RetentionTrimmedThroughSeq  int64 `json:"retention_trimmed_through_sequence,omitempty"`
	RetentionHistoryTruncated   bool  `json:"retention_history_truncated"`
}

type brokerProofReference struct {
	Path       string   `json:"path"`
	ScenarioID string   `json:"scenario_id"`
	Outcomes   []string `json:"outcomes,omitempty"`
}

type brokerReviewPack struct {
	Status                string               `json:"status"`
	SummaryPath           string               `json:"summary_path"`
	ReportPath            string               `json:"report_path"`
	ValidationPackPath    string               `json:"validation_pack_path"`
	ArtifactDirectory     string               `json:"artifact_directory"`
	ReviewerLinks         []string             `json:"reviewer_links,omitempty"`
	AmbiguousPublishProof brokerProofReference `json:"ambiguous_publish_proof"`
}

type traceExportBundleSummary struct {
	TotalTraces             int                      `json:"total_traces"`
	TracesWithTerminalState int                      `json:"traces_with_terminal_state"`
	RecentTraces            []traceExportBundleTrace `json:"recent_traces,omitempty"`
	ValidationArtifacts     []string                 `json:"validation_artifacts,omitempty"`
	ReviewerNavigation      []string                 `json:"reviewer_navigation,omitempty"`
	BackendLimitations      []string                 `json:"backend_limitations,omitempty"`
	AmbiguousPublishProof   brokerProofReference     `json:"ambiguous_publish_proof"`
}

type traceExportBundleTrace struct {
	TraceID         string           `json:"trace_id"`
	TaskID          string           `json:"task_id"`
	Executor        string           `json:"executor"`
	State           string           `json:"state"`
	EventCount      int              `json:"event_count"`
	LatestEventType domain.EventType `json:"latest_event_type"`
	DurationSeconds float64          `json:"duration_seconds"`
	TraceURL        string           `json:"trace_url"`
	EventURL        string           `json:"event_url"`
}

type distributedDiagnostics struct {
	Summary               distributedDiagnosticsSummary            `json:"summary"`
	RoutingReasons        []routingReasonSummary                   `json:"routing_reasons"`
	ExecutorCapacity      []executorCapacityView                   `json:"executor_capacity"`
	ClusterHealth         clusterHealthRollup                      `json:"cluster_health"`
	Recovery              recoveryDiagnostics                      `json:"recovery"`
	Fairness              fairnessDiagnostics                      `json:"fairness"`
	CoordinationLeader    any                                      `json:"coordination_leader_election,omitempty"`
	SharedQueue           sharedQueueCoordinationDiagnostics       `json:"shared_queue_diagnostics"`
	LiveShadowMirror      liveShadowMirrorSurface                  `json:"live_shadow_mirror_scorecard"`
	BrokerReviewPack      brokerReviewPack                         `json:"broker_review_pack"`
	BrokerReviewBundle    brokerReviewBundleSurface                `json:"broker_review_bundle"`
	MigrationReviewPack   migrationReviewPack                      `json:"migration_review_pack"`
	BrokerFanoutIsolation brokerStubFanoutIsolationEvidencePack    `json:"broker_stub_fanout_isolation"`
	ProviderLiveHandoff   providerLiveHandoffIsolationEvidencePack `json:"provider_live_handoff_isolation"`
	BrokerBootstrap       brokerBootstrapSurface                   `json:"broker_bootstrap_surface"`
	DeliveryAckReadiness  deliveryAckReadinessSurface              `json:"delivery_ack_readiness"`
	PublishAckOutcomes    publishAckOutcomeSurface                 `json:"publish_ack_outcomes"`
	SequenceBridge        sequenceBridgeSurface                    `json:"sequence_bridge_surface"`
	RetentionExpiry       retentionExpirySurface                   `json:"retention_expiry_surface"`
	ContinuationGate      validationBundleContinuationGateSurface  `json:"validation_bundle_continuation"`
	TraceBundle           traceExportBundleSummary                 `json:"trace_export_bundle"`
	RolloutReport         distributedDiagnosticsReport             `json:"rollout_report"`
}

type executorDiagnosticsCounters struct {
	Routed     int
	Started    int
	Completed  int
	DeadLetter int
}

type distributedTaskAssignment struct {
	Task           domain.Task
	EffectiveState domain.TaskState
	Executor       domain.ExecutorKind
}

type distributedWorkerRollup struct {
	WorkerStates         map[string]int
	ActiveByExecutor     map[domain.ExecutorKind]int
	ActiveWorkers        int
	IdleWorkers          int
	LeaseRenewalFailures int
	LeaseLostRuns        int
}

type distributedTaskRollup struct {
	Tasks            []domain.Task
	Takeovers        []control.Takeover
	Assignments      []distributedTaskAssignment
	AssignmentByTask map[string]distributedTaskAssignment
	TasksByExecutor  map[domain.ExecutorKind][]distributedTaskAssignment
	Recovery         recoveryDiagnostics
}

type distributedEventRollup struct {
	CountersByExecutor map[domain.ExecutorKind]*executorDiagnosticsCounters
	RoutingReasons     []routingReasonSummary
	TotalRouted        int
	Recovery           recoveryDiagnostics
}

type executorCapacityRollup struct {
	ExecutorCapacity  []executorCapacityView
	HealthyExecutors  int
	DegradedExecutors int
	IdleExecutors     int
	ActiveExecutors   int
	Saturated         []string
}

func (s *Server) handleV2DistributedReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	filters, err := parseControlCenterFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := enforceScopedTeamFilter(authorization, &filters.Team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	diagnostics := s.buildDistributedDiagnostics(filters)
	writeJSON(w, http.StatusOK, map[string]any{
		"authorization": authorization,
		"filters": map[string]any{
			"team":       filters.Team,
			"project":    filters.Project,
			"task_id":    filters.TaskID,
			"state":      filters.State,
			"risk_level": filters.RiskLevel,
			"since":      filters.Since,
			"until":      filters.Until,
			"limit":      filters.Limit,
			"priority":   filters.Priority,
		},
		"event_durability":                s.EventPlan,
		"summary":                         diagnostics.Summary,
		"routing_reasons":                 diagnostics.RoutingReasons,
		"executor_capacity":               diagnostics.ExecutorCapacity,
		"cluster_health":                  diagnostics.ClusterHealth,
		"recovery":                        diagnostics.Recovery,
		"fairness":                        diagnostics.Fairness,
		"coordination_leader_election":    diagnostics.CoordinationLeader,
		"shared_queue_diagnostics":        diagnostics.SharedQueue,
		"live_shadow_mirror_scorecard":    diagnostics.LiveShadowMirror,
		"broker_review_pack":              diagnostics.BrokerReviewPack,
		"broker_review_bundle":            diagnostics.BrokerReviewBundle,
		"broker_stub_fanout_isolation":    diagnostics.BrokerFanoutIsolation,
		"provider_live_handoff_isolation": diagnostics.ProviderLiveHandoff,
		"broker_bootstrap_surface":        diagnostics.BrokerBootstrap,
		"trace_export_bundle":             diagnostics.TraceBundle,
		"migration_review_pack":           diagnostics.MigrationReviewPack,
		"delivery_ack_readiness":          diagnostics.DeliveryAckReadiness,
		"publish_ack_outcomes":            diagnostics.PublishAckOutcomes,
		"sequence_bridge_surface":         diagnostics.SequenceBridge,
		"retention_expiry_surface":        diagnostics.RetentionExpiry,
		"validation_bundle_continuation":  diagnostics.ContinuationGate,
		"report":                          diagnostics.RolloutReport,
	})
}

func (s *Server) handleV2DistributedEvidenceBundles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, parallelDiagnosticsEvidenceBundleIndexPayload())
}

func (s *Server) handleV2DistributedEvidenceBundleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, err := parsePositiveIntQuery(r.URL.Query().Get("limit"), 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, parallelDiagnosticsEvidenceSearchPayload(
		r.URL.Query().Get("q"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("lane"),
		r.URL.Query().Get("path"),
		limit,
	))
}

func (s *Server) handleV2DistributedReportExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	filters, err := parseControlCenterFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := enforceScopedTeamFilter(authorization, &filters.Team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	diagnostics := s.buildDistributedDiagnostics(filters)
	filenameScope := firstNonEmpty(filters.Team, filters.Project, filters.TaskID, "all")
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", fmt.Sprintf("bigclaw-distributed-diagnostics-%s.md", sanitizeReportName(filenameScope))),
	)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(diagnostics.RolloutReport.Markdown))
}

func (s *Server) buildDistributedDiagnostics(filters controlCenterFilters) distributedDiagnostics {
	capabilities := s.executorCapabilities()
	worker := s.distributedWorkerRollup()
	taskRollup := s.distributedTaskRollup(filters)
	eventRollup := s.distributedEventRollup(taskRollup.AssignmentByTask, taskRollup.Recovery)
	capacity := buildExecutorCapacityRollup(capabilities, taskRollup.TasksByExecutor, eventRollup.CountersByExecutor, eventRollup.RoutingReasons, worker.ActiveByExecutor)

	summary := distributedDiagnosticsSummary{
		RegisteredExecutors:  len(capabilities),
		ActiveExecutors:      capacity.ActiveExecutors,
		TotalTasks:           len(taskRollup.Assignments),
		ActiveRuns:           countActiveAssignments(taskRollup.Assignments),
		TotalRoutedDecisions: eventRollup.TotalRouted,
		ActiveWorkers:        worker.ActiveWorkers,
		IdleWorkers:          worker.IdleWorkers,
		LeaseRenewalFailures: worker.LeaseRenewalFailures,
		LeaseLostRuns:        worker.LeaseLostRuns,
		SaturatedExecutors:   len(capacity.Saturated),
		ActiveTakeovers:      len(taskRollup.Takeovers),
	}
	clusterHealth := clusterHealthRollup{
		HealthyExecutors:  capacity.HealthyExecutors,
		DegradedExecutors: capacity.DegradedExecutors,
		IdleExecutors:     capacity.IdleExecutors,
		WorkerStates:      worker.WorkerStates,
		TeamBreakdown: facetCountsFromAssignments(taskRollup.Assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["team"]), "unassigned")
		}),
		ProjectBreakdown: facetCountsFromAssignments(taskRollup.Assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["project"]), "unassigned")
		}),
		TakeoverOwners:     facetCountsFromTakeovers(taskRollup.Takeovers, func(item control.Takeover) string { return firstNonEmpty(strings.TrimSpace(item.Owner), "unassigned") }),
		SaturatedExecutors: capacity.Saturated,
		Notes:              diagnosticsNotes(summary, capacity.ExecutorCapacity, s.Control.Snapshot()),
	}
	fairness := buildFairnessDiagnostics(capabilities, eventRollup.CountersByExecutor, eventRollup.TotalRouted)
	diagnostics := distributedDiagnostics{
		Summary:               summary,
		RoutingReasons:        eventRollup.RoutingReasons,
		ExecutorCapacity:      capacity.ExecutorCapacity,
		ClusterHealth:         clusterHealth,
		Recovery:              eventRollup.Recovery,
		Fairness:              fairness,
		CoordinationLeader:    s.coordinationLeaderElectionPayload(),
		SharedQueue:           s.sharedQueueCoordinationDiagnostics(),
		LiveShadowMirror:      liveShadowMirrorPayload(),
		BrokerReviewPack:      buildBrokerReviewPack(),
		BrokerReviewBundle:    brokerReviewBundleSurfacePayload(),
		MigrationReviewPack:   buildMigrationReviewPack(),
		BrokerFanoutIsolation: brokerStubFanoutIsolationPayload(),
		ProviderLiveHandoff:   providerLiveHandoffIsolationPayload(),
		BrokerBootstrap:       brokerBootstrapSurfacePayload(),
		DeliveryAckReadiness:  deliveryAckReadinessPayload(),
		PublishAckOutcomes:    publishAckOutcomeSurfacePayload(),
		SequenceBridge:        sequenceBridgeSurfacePayload(),
		RetentionExpiry:       retentionExpirySurfacePayload(),
		ContinuationGate:      validationBundleContinuationGatePayload(),
		TraceBundle:           buildTraceExportBundle(taskRollup.Assignments, s.Recorder.TraceSummaries(5)),
	}
	diagnostics.RolloutReport = distributedDiagnosticsReport{
		Markdown:  renderDistributedDiagnosticsMarkdown(diagnostics, filters),
		ExportURL: distributedExportURL(filters),
	}
	return diagnostics
}

func (s *Server) distributedWorkerRollup() distributedWorkerRollup {
	rollup := distributedWorkerRollup{
		WorkerStates:     make(map[string]int),
		ActiveByExecutor: make(map[domain.ExecutorKind]int),
	}
	pool := s.workerPoolSummary()
	if pool == nil {
		return rollup
	}
	rollup.ActiveWorkers = pool.ActiveWorkers
	rollup.IdleWorkers = pool.IdleWorkers
	for _, workerStatus := range pool.Workers {
		state := strings.TrimSpace(workerStatus.State)
		if state == "" {
			state = "idle"
		}
		rollup.WorkerStates[state]++
		rollup.LeaseRenewalFailures += workerStatus.LeaseRenewalFailures
		rollup.LeaseLostRuns += workerStatus.LeaseLostRuns
		if workerStatus.CurrentExecutor != "" && (state == "leased" || state == "running") {
			rollup.ActiveByExecutor[workerStatus.CurrentExecutor]++
		}
	}
	return rollup
}

func (s *Server) distributedTaskRollup(filters controlCenterFilters) distributedTaskRollup {
	tasks := filterTasks(s.Recorder.Tasks(0), filters)
	takeovers := s.filteredActiveTakeovers(filters)
	takeoverByTask := make(map[string]*control.Takeover, len(takeovers))
	for index := range takeovers {
		copy := takeovers[index]
		takeoverByTask[copy.TaskID] = &copy
	}
	assignments := make([]distributedTaskAssignment, 0, len(tasks))
	assignmentByTask := make(map[string]distributedTaskAssignment, len(tasks))
	tasksByExecutor := make(map[domain.ExecutorKind][]distributedTaskAssignment)
	for _, task := range tasks {
		assignment := distributedTaskAssignment{
			Task:           task,
			EffectiveState: effectiveTaskState(task.State, takeoverByTask[task.ID]),
			Executor:       s.distributedTaskExecutor(task),
		}
		assignments = append(assignments, assignment)
		assignmentByTask[task.ID] = assignment
		tasksByExecutor[assignment.Executor] = append(tasksByExecutor[assignment.Executor], assignment)
	}
	return distributedTaskRollup{
		Tasks:            tasks,
		Takeovers:        takeovers,
		Assignments:      assignments,
		AssignmentByTask: assignmentByTask,
		TasksByExecutor:  tasksByExecutor,
		Recovery: recoveryDiagnostics{
			ActiveTakeovers: len(takeovers),
		},
	}
}

func (s *Server) distributedEventRollup(assignmentByTask map[string]distributedTaskAssignment, base recoveryDiagnostics) distributedEventRollup {
	countersByExecutor := make(map[domain.ExecutorKind]*executorDiagnosticsCounters)
	routingIndex := make(map[string]*routingReasonSummary)
	totalRouted := 0
	recovery := base
	for _, event := range s.Recorder.EventsByTask("", 0) {
		assignment, ok := assignmentByTask[event.TaskID]
		if !ok {
			continue
		}
		executorKind := eventExecutorKind(event, assignment.Task)
		if executorKind == "" {
			executorKind = assignment.Executor
		}
		if executorKind == "" {
			continue
		}
		entry := countersByExecutor[executorKind]
		if entry == nil {
			entry = &executorDiagnosticsCounters{}
			countersByExecutor[executorKind] = entry
		}
		switch event.Type {
		case domain.EventSchedulerRouted:
			entry.Routed++
			totalRouted++
			reason := firstNonEmpty(strings.TrimSpace(eventStringValue(event.Payload, "reason")), "unknown")
			key := string(executorKind) + "\n" + reason
			item := routingIndex[key]
			if item == nil {
				item = &routingReasonSummary{Executor: string(executorKind), Reason: reason}
				routingIndex[key] = item
			}
			item.Count++
		case domain.EventTaskStarted:
			entry.Started++
		case domain.EventTaskCompleted:
			entry.Completed++
		case domain.EventTaskRetried:
			recovery.RetriedRuns++
		case domain.EventTaskDeadLetter:
			entry.DeadLetter++
			recovery.DeadLetterRuns++
		case domain.EventRunTakeover:
			recovery.TakeoverEvents++
		case domain.EventRunReleased:
			recovery.ReleaseEvents++
		case domain.EventSubscriberLeaseExpired:
			recovery.LeaseExpiredEvents++
		case domain.EventSubscriberCheckpointRejected:
			recovery.CheckpointRejectedEvents++
		}
	}
	if recovery.TakeoverEvents > recovery.ReleaseEvents {
		recovery.UnreleasedTakeovers = recovery.TakeoverEvents - recovery.ReleaseEvents
	}
	return distributedEventRollup{
		CountersByExecutor: countersByExecutor,
		RoutingReasons:     sortedRoutingReasons(routingIndex),
		TotalRouted:        totalRouted,
		Recovery:           recovery,
	}
}

func buildExecutorCapacityRollup(capabilities []executor.Capability, tasksByExecutor map[domain.ExecutorKind][]distributedTaskAssignment, countersByExecutor map[domain.ExecutorKind]*executorDiagnosticsCounters, routingReasons []routingReasonSummary, activeByExecutor map[domain.ExecutorKind]int) executorCapacityRollup {
	executorCapacity := make([]executorCapacityView, 0, len(capabilities))
	rollup := executorCapacityRollup{
		Saturated: make([]string, 0),
	}
	for _, capability := range capabilities {
		assignments := tasksByExecutor[capability.Kind]
		counts := countersByExecutor[capability.Kind]
		view := executorCapacityView{
			Executor:        string(capability.Kind),
			MaxConcurrency:  capability.MaxConcurrency,
			ActiveWorkers:   activeByExecutor[capability.Kind],
			SupportsGPU:     capability.SupportsGPU,
			SupportsBrowser: capability.SupportsBrowser,
			SupportsShell:   capability.SupportsShell,
		}
		if counts != nil {
			view.RoutedDecisions = counts.Routed
			view.StartedRuns = counts.Started
			view.CompletedRuns = counts.Completed
			view.DeadLetters = counts.DeadLetter
		}
		for _, assignment := range assignments {
			if assignment.EffectiveState == domain.TaskQueued || assignment.EffectiveState == domain.TaskRetrying {
				view.QueuedTasks++
			}
			if domain.IsActiveTaskState(assignment.EffectiveState) {
				view.ActiveTasks++
			}
		}
		view.TeamBreakdown = facetCountsFromAssignments(assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["team"]), "unassigned")
		})
		view.ProjectBreakdown = facetCountsFromAssignments(assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["project"]), "unassigned")
		})
		view.TopRoutingReasons = routingReasonsForExecutor(routingReasons, capability.Kind, 3)
		view.SampleTasks = sampleTaskIDs(assignments, 3)
		if capability.MaxConcurrency > 0 {
			view.AvailableConcurrency = capability.MaxConcurrency - view.ActiveWorkers
			if view.AvailableConcurrency < 0 {
				view.AvailableConcurrency = 0
			}
			view.SaturationPercent = float64(view.ActiveWorkers) / float64(capability.MaxConcurrency) * 100
		}
		view.Health = diagnosticsHealth(view)
		view.HealthSummary = executorHealthSummary(view)
		if view.Health != "idle" {
			rollup.ActiveExecutors++
		}
		if executorIsSaturated(view) {
			rollup.Saturated = append(rollup.Saturated, view.Executor)
		}
		switch view.Health {
		case "healthy":
			rollup.HealthyExecutors++
		case "degraded":
			rollup.DegradedExecutors++
		default:
			rollup.IdleExecutors++
		}
		executorCapacity = append(executorCapacity, view)
	}
	sort.SliceStable(executorCapacity, func(i, j int) bool { return executorCapacity[i].Executor < executorCapacity[j].Executor })
	sort.Strings(rollup.Saturated)
	rollup.ExecutorCapacity = executorCapacity
	return rollup
}

func sortedRoutingReasons(index map[string]*routingReasonSummary) []routingReasonSummary {
	reasons := make([]routingReasonSummary, 0, len(index))
	for _, item := range index {
		reasons = append(reasons, *item)
	}
	sort.SliceStable(reasons, func(i, j int) bool {
		if reasons[i].Count == reasons[j].Count {
			if reasons[i].Executor == reasons[j].Executor {
				return reasons[i].Reason < reasons[j].Reason
			}
			return reasons[i].Executor < reasons[j].Executor
		}
		return reasons[i].Count > reasons[j].Count
	})
	return reasons
}

func (s *Server) executorCapabilities() []executor.Capability {
	out := make([]executor.Capability, 0, len(s.Executors))
	for _, kind := range s.Executors {
		out = append(out, executor.CapabilityForKind(kind))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}

func (s *Server) distributedTaskExecutor(task domain.Task) domain.ExecutorKind {
	if latest, ok := s.Recorder.LatestByTask(task.ID); ok {
		if executorKind := eventExecutorKind(latest, task); executorKind != "" {
			return executorKind
		}
	}
	events := s.Recorder.EventsByTask(task.ID, 0)
	for index := len(events) - 1; index >= 0; index-- {
		if executorKind := eventExecutorKind(events[index], task); executorKind != "" {
			return executorKind
		}
	}
	if task.RequiredExecutor != "" {
		return task.RequiredExecutor
	}
	score := risk.ScoreTask(task, events)
	if taskRequiresTool(task, "gpu") {
		return domain.ExecutorRay
	}
	if taskRequiresTool(task, "browser") {
		return domain.ExecutorKubernetes
	}
	if score.Level == domain.RiskHigh {
		return domain.ExecutorKubernetes
	}
	return domain.ExecutorLocal
}

func diagnosticsHealth(view executorCapacityView) string {
	if view.DeadLetters > 0 && view.CompletedRuns == 0 {
		return "degraded"
	}
	if view.ActiveWorkers > 0 || view.ActiveTasks > 0 || view.RoutedDecisions > 0 || view.CompletedRuns > 0 {
		return "healthy"
	}
	return "idle"
}

func executorIsSaturated(view executorCapacityView) bool {
	if view.MaxConcurrency <= 0 {
		return false
	}
	if view.SaturationPercent >= 80 {
		return true
	}
	return view.AvailableConcurrency == 0 && (view.ActiveTasks > 0 || view.RoutedDecisions > 0)
}

func diagnosticsNotes(summary distributedDiagnosticsSummary, capacity []executorCapacityView, snapshot control.Snapshot) []string {
	notes := make([]string, 0)
	if snapshot.Paused {
		notes = append(notes, fmt.Sprintf("control plane paused by %s", firstNonEmpty(snapshot.PauseActor, "system")))
	}
	if summary.ActiveTakeovers > 0 {
		notes = append(notes, fmt.Sprintf("%d runs currently require human takeover coverage", summary.ActiveTakeovers))
	}
	if summary.TotalRoutedDecisions == 0 {
		notes = append(notes, "no scheduler routing evidence captured for the current filter set")
	}
	for _, item := range capacity {
		if item.Health == "degraded" {
			notes = append(notes, fmt.Sprintf("%s executor shows dead-letter activity without offsetting completions", item.Executor))
		}
		if executorIsSaturated(item) {
			notes = append(notes, fmt.Sprintf("%s executor is above 80%% worker saturation or has no spare capacity", item.Executor))
		}
		if item.QueuedTasks > item.ActiveWorkers && item.QueuedTasks > 0 {
			notes = append(notes, fmt.Sprintf("%s executor has %d queued tasks waiting behind %d active workers", item.Executor, item.QueuedTasks, item.ActiveWorkers))
		}
	}
	if len(notes) == 0 {
		notes = append(notes, "distributed control plane looks healthy for the current slice")
	}
	return notes
}

func executorHealthSummary(view executorCapacityView) []string {
	summary := make([]string, 0, 3)
	if view.Health == "degraded" {
		summary = append(summary, "dead-letter activity without offsetting completions")
	}
	if executorIsSaturated(view) {
		summary = append(summary, "worker saturation above 80% or no spare capacity")
	}
	if view.QueuedTasks > view.ActiveWorkers && view.QueuedTasks > 0 {
		summary = append(summary, fmt.Sprintf("%d queued tasks waiting behind %d active workers", view.QueuedTasks, view.ActiveWorkers))
	}
	if len(summary) == 0 {
		summary = append(summary, "no immediate capacity or recovery risk signals")
	}
	return summary
}

func buildFairnessDiagnostics(capabilities []executor.Capability, countersByExecutor map[domain.ExecutorKind]*executorDiagnosticsCounters, totalRouted int) fairnessDiagnostics {
	shares := make([]fairnessExecutorShare, 0, len(capabilities))
	weightTotal := 0
	for _, capability := range capabilities {
		weight := capability.MaxConcurrency
		if weight <= 0 {
			weight = 1
		}
		weightTotal += weight
	}
	if weightTotal <= 0 {
		weightTotal = len(capabilities)
		if weightTotal == 0 {
			weightTotal = 1
		}
	}

	maxDelta := 0.0
	for _, capability := range capabilities {
		weight := capability.MaxConcurrency
		if weight <= 0 {
			weight = 1
		}
		expectedShare := float64(weight) / float64(weightTotal)
		routed := 0
		if counters := countersByExecutor[capability.Kind]; counters != nil {
			routed = counters.Routed
		}
		actualShare := 0.0
		if totalRouted > 0 {
			actualShare = float64(routed) / float64(totalRouted)
		}
		delta := actualShare - expectedShare
		absDelta := math.Abs(delta)
		if absDelta > maxDelta {
			maxDelta = absDelta
		}
		status := "balanced"
		switch {
		case totalRouted == 0:
			status = "no-traffic"
		case absDelta <= 0.15:
			status = "balanced"
		case delta > 0:
			status = "over-assigned"
		default:
			status = "under-assigned"
		}
		shares = append(shares, fairnessExecutorShare{
			Executor:        string(capability.Kind),
			RoutedDecisions: routed,
			ExpectedShare:   expectedShare,
			ActualShare:     actualShare,
			ShareDelta:      delta,
			Status:          status,
		})
	}
	sort.SliceStable(shares, func(i, j int) bool { return shares[i].Executor < shares[j].Executor })

	notes := make([]string, 0, 2)
	if totalRouted == 0 {
		notes = append(notes, "no routed decisions available for fairness analysis")
	} else if maxDelta > 0.15 {
		notes = append(notes, "routing distribution is imbalanced relative to executor capacity weights")
	} else {
		notes = append(notes, "routing distribution is within fairness tolerance")
	}

	return fairnessDiagnostics{
		TotalRoutedDecisions: totalRouted,
		CapacityWeightTotal:  weightTotal,
		ImbalanceScore:       maxDelta,
		ExecutorShares:       shares,
		Notes:                notes,
	}
}

func countActiveAssignments(assignments []distributedTaskAssignment) int {
	count := 0
	for _, item := range assignments {
		if domain.IsActiveTaskState(item.EffectiveState) {
			count++
		}
	}
	return count
}

func facetCountsFromAssignments(assignments []distributedTaskAssignment, valueFn func(distributedTaskAssignment) string) []auditFacetCount {
	counts := make(map[string]int)
	for _, item := range assignments {
		key := firstNonEmpty(strings.TrimSpace(valueFn(item)), "unknown")
		counts[key]++
	}
	return sortFacetCounts(counts)
}

func facetCountsFromTakeovers(takeovers []control.Takeover, valueFn func(control.Takeover) string) []auditFacetCount {
	counts := make(map[string]int)
	for _, item := range takeovers {
		key := firstNonEmpty(strings.TrimSpace(valueFn(item)), "unknown")
		counts[key]++
	}
	return sortFacetCounts(counts)
}

func sortFacetCounts(counts map[string]int) []auditFacetCount {
	out := make([]auditFacetCount, 0, len(counts))
	for key, count := range counts {
		out = append(out, auditFacetCount{Key: key, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func routingReasonsForExecutor(reasons []routingReasonSummary, executorKind domain.ExecutorKind, limit int) []routingReasonSummary {
	out := make([]routingReasonSummary, 0)
	for _, item := range reasons {
		if item.Executor != string(executorKind) {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func sampleTaskIDs(assignments []distributedTaskAssignment, limit int) []string {
	if len(assignments) == 0 {
		return nil
	}
	sorted := append([]distributedTaskAssignment(nil), assignments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i].Task.UpdatedAt
		right := sorted[j].Task.UpdatedAt
		if left.Equal(right) {
			return sorted[i].Task.ID < sorted[j].Task.ID
		}
		return left.After(right)
	})
	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	out := make([]string, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, item.Task.ID)
	}
	return out
}

func takeoverOrNil(takeover control.Takeover, ok bool) *control.Takeover {
	if !ok {
		return nil
	}
	copy := takeover
	return &copy
}

func eventExecutorKind(event domain.Event, task domain.Task) domain.ExecutorKind {
	if executorName := strings.TrimSpace(eventStringValue(event.Payload, "executor")); executorName != "" {
		return domain.ExecutorKind(executorName)
	}
	if task.RequiredExecutor != "" {
		return task.RequiredExecutor
	}
	return ""
}

func eventBoolValue(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func taskRequiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if item == tool {
			return true
		}
	}
	return false
}

func distributedExportURL(filters controlCenterFilters) string {
	values := url.Values{}
	if filters.Team != "" {
		values.Set("team", filters.Team)
	}
	if filters.Project != "" {
		values.Set("project", filters.Project)
	}
	if filters.TaskID != "" {
		values.Set("task_id", filters.TaskID)
	}
	if filters.State != "" {
		values.Set("state", filters.State)
	}
	if filters.RiskLevel != "" {
		values.Set("risk_level", filters.RiskLevel)
	}
	if !filters.Since.IsZero() {
		values.Set("since", filters.Since.UTC().Format(time.RFC3339))
	}
	if !filters.Until.IsZero() {
		values.Set("until", filters.Until.UTC().Format(time.RFC3339))
	}
	if filters.Priority != nil {
		values.Set("priority", fmt.Sprintf("%d", *filters.Priority))
	}
	if filters.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", filters.Limit))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/v2/reports/distributed/export"
	}
	return "/v2/reports/distributed/export?" + encoded
}

func renderDistributedDiagnosticsMarkdown(diagnostics distributedDiagnostics, filters controlCenterFilters) string {
	lines := []string{
		"# BigClaw Distributed Diagnostics Report",
		"",
		fmt.Sprintf(
			"Filters: team=%s project=%s task_id=%s state=%s risk_level=%s priority=%s since=%s until=%s",
			firstNonEmpty(filters.Team, "all"),
			firstNonEmpty(filters.Project, "all"),
			firstNonEmpty(filters.TaskID, "all"),
			firstNonEmpty(filters.State, "all"),
			firstNonEmpty(filters.RiskLevel, "all"),
			formatOptionalPriority(filters.Priority),
			formatOptionalFilterTime(filters.Since),
			formatOptionalFilterTime(filters.Until),
		),
		"",
		"## Summary",
		fmt.Sprintf("- Registered executors: %d", diagnostics.Summary.RegisteredExecutors),
		fmt.Sprintf("- Active executors: %d", diagnostics.Summary.ActiveExecutors),
		fmt.Sprintf("- Total tasks: %d", diagnostics.Summary.TotalTasks),
		fmt.Sprintf("- Active runs: %d", diagnostics.Summary.ActiveRuns),
		fmt.Sprintf("- Routed decisions: %d", diagnostics.Summary.TotalRoutedDecisions),
		fmt.Sprintf("- Active workers: %d", diagnostics.Summary.ActiveWorkers),
		fmt.Sprintf("- Idle workers: %d", diagnostics.Summary.IdleWorkers),
		fmt.Sprintf("- Lease renewal failures: %d", diagnostics.Summary.LeaseRenewalFailures),
		fmt.Sprintf("- Lease lost runs: %d", diagnostics.Summary.LeaseLostRuns),
		fmt.Sprintf("- Saturated executors: %d", diagnostics.Summary.SaturatedExecutors),
		fmt.Sprintf("- Active takeovers: %d", diagnostics.Summary.ActiveTakeovers),
		"",
		"## Routing Reasons",
	}
	if len(diagnostics.RoutingReasons) == 0 {
		lines = append(lines, "- No routing decisions captured")
	} else {
		for _, item := range diagnostics.RoutingReasons {
			lines = append(lines, fmt.Sprintf("- %s: %s (%d)", item.Executor, item.Reason, item.Count))
		}
	}
	lines = append(lines, "", "## Executor Capacity")
	for _, item := range diagnostics.ExecutorCapacity {
		lines = append(lines, fmt.Sprintf("- %s: health=%s active_workers=%d queued_tasks=%d active_tasks=%d max_concurrency=%d available=%d saturation=%.1f%% routed=%d completed=%d dead_letters=%d", item.Executor, item.Health, item.ActiveWorkers, item.QueuedTasks, item.ActiveTasks, item.MaxConcurrency, item.AvailableConcurrency, item.SaturationPercent, item.RoutedDecisions, item.CompletedRuns, item.DeadLetters))
		if len(item.TeamBreakdown) > 0 {
			lines = append(lines, "  - teams: "+formatFacetCounts(item.TeamBreakdown))
		}
		if len(item.ProjectBreakdown) > 0 {
			lines = append(lines, "  - projects: "+formatFacetCounts(item.ProjectBreakdown))
		}
		if len(item.TopRoutingReasons) > 0 {
			parts := make([]string, 0, len(item.TopRoutingReasons))
			for _, reason := range item.TopRoutingReasons {
				parts = append(parts, fmt.Sprintf("%s (%d)", reason.Reason, reason.Count))
			}
			lines = append(lines, "  - top routing reasons: "+strings.Join(parts, ", "))
		}
		if len(item.HealthSummary) > 0 {
			lines = append(lines, "  - health summary: "+strings.Join(item.HealthSummary, "; "))
		}
		if len(item.SampleTasks) > 0 {
			lines = append(lines, "  - sample tasks: "+strings.Join(item.SampleTasks, ", "))
		}
	}
	lines = append(lines,
		"",
		"## Cluster Health",
		fmt.Sprintf("- Healthy executors: %d", diagnostics.ClusterHealth.HealthyExecutors),
		fmt.Sprintf("- Degraded executors: %d", diagnostics.ClusterHealth.DegradedExecutors),
		fmt.Sprintf("- Idle executors: %d", diagnostics.ClusterHealth.IdleExecutors),
	)
	if len(diagnostics.ClusterHealth.SaturatedExecutors) > 0 {
		lines = append(lines, "- Saturated executors: "+strings.Join(diagnostics.ClusterHealth.SaturatedExecutors, ", "))
	}
	if len(diagnostics.ClusterHealth.WorkerStates) > 0 {
		stateKeys := make([]string, 0, len(diagnostics.ClusterHealth.WorkerStates))
		for key := range diagnostics.ClusterHealth.WorkerStates {
			stateKeys = append(stateKeys, key)
		}
		sort.Strings(stateKeys)
		parts := make([]string, 0, len(stateKeys))
		for _, key := range stateKeys {
			parts = append(parts, fmt.Sprintf("%s=%d", key, diagnostics.ClusterHealth.WorkerStates[key]))
		}
		lines = append(lines, "- Worker states: "+strings.Join(parts, ", "))
	}
	if len(diagnostics.ClusterHealth.TeamBreakdown) > 0 {
		lines = append(lines, "- Team breakdown: "+formatFacetCounts(diagnostics.ClusterHealth.TeamBreakdown))
	}
	if len(diagnostics.ClusterHealth.ProjectBreakdown) > 0 {
		lines = append(lines, "- Project breakdown: "+formatFacetCounts(diagnostics.ClusterHealth.ProjectBreakdown))
	}
	if len(diagnostics.ClusterHealth.TakeoverOwners) > 0 {
		lines = append(lines, "- Takeover owners: "+formatFacetCounts(diagnostics.ClusterHealth.TakeoverOwners))
	}
	if leader, ok := diagnostics.CoordinationLeader.(coordinationLeaderElectionSurface); ok {
		lines = append(lines,
			"",
			"## Coordination Leader Election",
			fmt.Sprintf("- Endpoint: %s", leader.Endpoint),
			fmt.Sprintf("- Status: %s", firstNonEmpty(leader.Status, "unknown")),
			fmt.Sprintf("- Backend: %s", firstNonEmpty(leader.Backend, "unknown")),
			fmt.Sprintf("- Leader present: %t", leader.LeaderPresent),
		)
		if leader.Lease != nil {
			lines = append(lines, fmt.Sprintf("- Lease owner: %s (epoch=%d token=%s)", firstNonEmpty(leader.Lease.ConsumerID, "unknown"), leader.Lease.LeaseEpoch, firstNonEmpty(leader.Lease.LeaseToken, "unknown")))
		}
		if leader.RemainingTTLSeconds > 0 {
			lines = append(lines, fmt.Sprintf("- Remaining TTL seconds: %d", leader.RemainingTTLSeconds))
		}
		if len(leader.Notes) > 0 {
			lines = append(lines, "- Notes: "+strings.Join(leader.Notes, "; "))
		}
	}
	lines = append(lines,
		"",
		"## Shared Queue Coordination",
		fmt.Sprintf("- Dead-letter backlog: %d", diagnostics.SharedQueue.DeadLetterBacklog),
		fmt.Sprintf("- Dead-letter events: %d", diagnostics.SharedQueue.DeadLetterEvents),
		fmt.Sprintf("- Replayed queue events: %d", diagnostics.SharedQueue.ReplayedQueueEvents),
		fmt.Sprintf("- Lease acquired events: %d", diagnostics.SharedQueue.LeaseAcquiredEvents),
		fmt.Sprintf("- Lease rejected events: %d", diagnostics.SharedQueue.LeaseRejectedEvents),
		fmt.Sprintf("- Lease expired events: %d", diagnostics.SharedQueue.LeaseExpiredEvents),
		fmt.Sprintf("- Takeover succeeded events: %d", diagnostics.SharedQueue.TakeoverSucceededEvents),
		fmt.Sprintf("- Checkpoint committed events: %d", diagnostics.SharedQueue.CheckpointCommittedEvents),
		fmt.Sprintf("- Checkpoint rejected events: %d", diagnostics.SharedQueue.CheckpointRejectedEvents),
		fmt.Sprintf("- Lease fenced events: %d", diagnostics.SharedQueue.LeaseFencedEvents),
		fmt.Sprintf("- Checkpoint resets (recent): %d", diagnostics.SharedQueue.CheckpointResetsRecent),
		fmt.Sprintf("- Retention watermark visible: %t", diagnostics.SharedQueue.RetentionWatermarkAvailable),
	)
	if diagnostics.SharedQueue.RetentionTrimmedThroughSeq > 0 {
		lines = append(lines, fmt.Sprintf("- Retention trimmed through sequence: %d", diagnostics.SharedQueue.RetentionTrimmedThroughSeq))
	}
	if diagnostics.SharedQueue.RetentionHistoryTruncated {
		lines = append(lines, "- Retention history truncated: true")
	}
	lines = append(lines,
		"",
		"## Recovery Signals",
		fmt.Sprintf("- Active takeovers: %d", diagnostics.Recovery.ActiveTakeovers),
		fmt.Sprintf("- Takeover events: %d", diagnostics.Recovery.TakeoverEvents),
		fmt.Sprintf("- Release events: %d", diagnostics.Recovery.ReleaseEvents),
		fmt.Sprintf("- Unreleased takeovers: %d", diagnostics.Recovery.UnreleasedTakeovers),
		fmt.Sprintf("- Retried runs: %d", diagnostics.Recovery.RetriedRuns),
		fmt.Sprintf("- Dead-letter runs: %d", diagnostics.Recovery.DeadLetterRuns),
		fmt.Sprintf("- Lease expired events: %d", diagnostics.Recovery.LeaseExpiredEvents),
		fmt.Sprintf("- Checkpoint rejected events: %d", diagnostics.Recovery.CheckpointRejectedEvents),
		"",
		"## Fairness",
		fmt.Sprintf("- Total routed decisions: %d", diagnostics.Fairness.TotalRoutedDecisions),
		fmt.Sprintf("- Capacity weight total: %d", diagnostics.Fairness.CapacityWeightTotal),
		fmt.Sprintf("- Imbalance score: %.3f", diagnostics.Fairness.ImbalanceScore),
	)
	for _, item := range diagnostics.Fairness.ExecutorShares {
		lines = append(lines, fmt.Sprintf("- %s: routed=%d expected_share=%.3f actual_share=%.3f delta=%.3f status=%s", item.Executor, item.RoutedDecisions, item.ExpectedShare, item.ActualShare, item.ShareDelta, item.Status))
	}
	if len(diagnostics.Fairness.Notes) > 0 {
		lines = append(lines, "- Notes: "+strings.Join(diagnostics.Fairness.Notes, "; "))
	}
	lines = append(lines,
		"",
		"## Trace Export Bundle",
		fmt.Sprintf("- Total traces: %d", diagnostics.TraceBundle.TotalTraces),
		fmt.Sprintf("- Traces with terminal state: %d", diagnostics.TraceBundle.TracesWithTerminalState),
	)
	if len(diagnostics.TraceBundle.RecentTraces) == 0 {
		lines = append(lines, "- No trace summaries captured")
	} else {
		for _, item := range diagnostics.TraceBundle.RecentTraces {
			lines = append(lines, fmt.Sprintf("- %s: task=%s executor=%s state=%s events=%d latest=%s duration=%.3fs trace=%s events=%s", item.TraceID, item.TaskID, firstNonEmpty(item.Executor, "unknown"), firstNonEmpty(item.State, "unknown"), item.EventCount, firstNonEmpty(string(item.LatestEventType), "unknown"), item.DurationSeconds, item.TraceURL, item.EventURL))
		}
	}
	if len(diagnostics.TraceBundle.ValidationArtifacts) > 0 {
		lines = append(lines, "- Validation artifacts: "+strings.Join(diagnostics.TraceBundle.ValidationArtifacts, ", "))
	}
	if diagnostics.TraceBundle.AmbiguousPublishProof.Path != "" {
		lines = append(lines, fmt.Sprintf("- Ambiguous publish proof: %s (%s: %s)", diagnostics.TraceBundle.AmbiguousPublishProof.Path, diagnostics.TraceBundle.AmbiguousPublishProof.ScenarioID, strings.Join(diagnostics.TraceBundle.AmbiguousPublishProof.Outcomes, ", ")))
	}
	if len(diagnostics.TraceBundle.ReviewerNavigation) > 0 {
		lines = append(lines, "- Reviewer navigation: "+strings.Join(diagnostics.TraceBundle.ReviewerNavigation, ", "))
	}
	if len(diagnostics.TraceBundle.BackendLimitations) > 0 {
		lines = append(lines, "- Backend limitations: "+strings.Join(diagnostics.TraceBundle.BackendLimitations, "; "))
	}
	lines = append(lines,
		"",
		"## Live Shadow Mirror Scorecard",
		fmt.Sprintf("- Canonical summary: %s", diagnostics.LiveShadowMirror.CanonicalSummaryPath),
		fmt.Sprintf("- Scorecard report: %s", diagnostics.LiveShadowMirror.ReportPath),
		fmt.Sprintf("- Status: %s", diagnostics.LiveShadowMirror.Status),
		fmt.Sprintf("- Severity: %s", firstNonEmpty(diagnostics.LiveShadowMirror.Severity, "none")),
		fmt.Sprintf("- Latest evidence timestamp: %s", firstNonEmpty(diagnostics.LiveShadowMirror.LatestEvidenceTimestamp, "unknown")),
		fmt.Sprintf("- Evidence runs: %d", diagnostics.LiveShadowMirror.Summary.TotalEvidenceRuns),
		fmt.Sprintf("- Parity OK: %d", diagnostics.LiveShadowMirror.Summary.ParityOKCount),
		fmt.Sprintf("- Drift detected: %d", diagnostics.LiveShadowMirror.Summary.DriftDetectedCount),
		fmt.Sprintf("- Matrix mismatched: %d", diagnostics.LiveShadowMirror.Summary.MatrixMismatched),
		fmt.Sprintf("- Fresh inputs: %d", diagnostics.LiveShadowMirror.Summary.FreshInputs),
		fmt.Sprintf("- Stale inputs: %d", diagnostics.LiveShadowMirror.Summary.StaleInputs),
	)
	if len(diagnostics.LiveShadowMirror.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.LiveShadowMirror.ReviewerLinks, ", "))
	}
	for _, checkpoint := range diagnostics.LiveShadowMirror.CutoverCheckpoints {
		lines = append(lines, fmt.Sprintf("- Cutover check %s: passed=%t detail=%s", checkpoint.Name, checkpoint.Passed, firstNonEmpty(checkpoint.Detail, "n/a")))
	}
	lines = append(lines,
		"",
		"## Broker Failover Review Pack",
		fmt.Sprintf("- Status: %s", diagnostics.BrokerReviewPack.Status),
		fmt.Sprintf("- Canonical summary: %s", diagnostics.BrokerReviewPack.SummaryPath),
		fmt.Sprintf("- Stub report: %s", diagnostics.BrokerReviewPack.ReportPath),
		fmt.Sprintf("- Validation pack: %s", diagnostics.BrokerReviewPack.ValidationPackPath),
		fmt.Sprintf("- Artifact directory: %s", diagnostics.BrokerReviewPack.ArtifactDirectory),
	)
	if diagnostics.BrokerReviewPack.AmbiguousPublishProof.Path != "" {
		lines = append(lines, fmt.Sprintf("- Ambiguous publish proof: %s (%s: %s)", diagnostics.BrokerReviewPack.AmbiguousPublishProof.Path, diagnostics.BrokerReviewPack.AmbiguousPublishProof.ScenarioID, strings.Join(diagnostics.BrokerReviewPack.AmbiguousPublishProof.Outcomes, ", ")))
	}
	if len(diagnostics.BrokerReviewPack.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.BrokerReviewPack.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Broker Review Bundle",
		fmt.Sprintf("- Canonical summary: %s", diagnostics.BrokerReviewBundle.CanonicalSummaryPath),
		fmt.Sprintf("- Canonical bootstrap summary: %s", diagnostics.BrokerReviewBundle.CanonicalBootstrapSummaryPath),
		fmt.Sprintf("- Validation pack: %s", diagnostics.BrokerReviewBundle.ValidationPackPath),
		fmt.Sprintf("- Stub report: %s", diagnostics.BrokerReviewBundle.StubReportPath),
		fmt.Sprintf("- Artifact directory: %s", diagnostics.BrokerReviewBundle.ArtifactDirectory),
		fmt.Sprintf("- Review readiness: %s", diagnostics.BrokerReviewBundle.ReviewReadinessPath),
		fmt.Sprintf("- Live validation index: %s", diagnostics.BrokerReviewBundle.LiveValidationIndexPath),
		fmt.Sprintf("- Operator guide: %s", diagnostics.BrokerReviewBundle.OperatorGuidePath),
		fmt.Sprintf("- Runtime posture: %s", firstNonEmpty(diagnostics.BrokerReviewBundle.RuntimePosture, "unknown")),
		fmt.Sprintf("- Bootstrap ready: %t", diagnostics.BrokerReviewBundle.BootstrapReady),
		fmt.Sprintf("- Live adapter implemented: %t", diagnostics.BrokerReviewBundle.LiveAdapterImplemented),
	)
	if diagnostics.BrokerReviewBundle.ProofBoundary != "" {
		lines = append(lines, "- Proof boundary: "+diagnostics.BrokerReviewBundle.ProofBoundary)
	}
	if len(diagnostics.BrokerReviewBundle.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer bundle links: "+strings.Join(diagnostics.BrokerReviewBundle.ReviewerLinks, ", "))
	}
	if diagnostics.BrokerReviewBundle.AmbiguousPublishProof.Path != "" {
		lines = append(lines, fmt.Sprintf("- Ambiguous publish proof: %s (%s: %s)", diagnostics.BrokerReviewBundle.AmbiguousPublishProof.Path, diagnostics.BrokerReviewBundle.AmbiguousPublishProof.ScenarioID, strings.Join(diagnostics.BrokerReviewBundle.AmbiguousPublishProof.Outcomes, ", ")))
	}
	lines = append(lines,
		"",
		"## Broker Bootstrap Readiness",
		fmt.Sprintf("- Canonical report: %s", diagnostics.BrokerBootstrap.ReportPath),
		fmt.Sprintf("- Canonical summary: %s", diagnostics.BrokerBootstrap.CanonicalSummaryPath),
		fmt.Sprintf("- Canonical bootstrap summary: %s", diagnostics.BrokerBootstrap.CanonicalBootstrapSummaryPath),
		fmt.Sprintf("- Validation pack: %s", diagnostics.BrokerBootstrap.ValidationPackPath),
		fmt.Sprintf("- Configuration state: %s", firstNonEmpty(diagnostics.BrokerBootstrap.ConfigurationState, "unknown")),
		fmt.Sprintf("- Runtime posture: %s", firstNonEmpty(diagnostics.BrokerBootstrap.RuntimePosture, "unknown")),
		fmt.Sprintf("- Runtime gate: requested=%t fail_closed=%t contract_only=%t stub_driver_only=%t safe_for_live_traffic=%t", diagnostics.BrokerBootstrap.RuntimeGate.Requested, diagnostics.BrokerBootstrap.RuntimeGate.FailClosed, diagnostics.BrokerBootstrap.RuntimeGate.ContractOnly, diagnostics.BrokerBootstrap.RuntimeGate.StubDriverOnly, diagnostics.BrokerBootstrap.RuntimeGate.SafeForLiveTraffic),
		fmt.Sprintf("- Bootstrap ready: %t", diagnostics.BrokerBootstrap.BootstrapReady),
		fmt.Sprintf("- Live adapter implemented: %t", diagnostics.BrokerBootstrap.LiveAdapterImplemented),
	)
	if diagnostics.BrokerBootstrap.ProofBoundary != "" {
		lines = append(lines, "- Proof boundary: "+diagnostics.BrokerBootstrap.ProofBoundary)
	}
	if diagnostics.BrokerBootstrap.RuntimeGate.OperatorMessage != "" {
		lines = append(lines, "- Runtime gate message: "+diagnostics.BrokerBootstrap.RuntimeGate.OperatorMessage)
	}
	lines = append(lines, fmt.Sprintf("- Config completeness: driver=%t urls=%t topic=%t consumer_group=%t", diagnostics.BrokerBootstrap.ConfigCompleteness.Driver, diagnostics.BrokerBootstrap.ConfigCompleteness.URLs, diagnostics.BrokerBootstrap.ConfigCompleteness.Topic, diagnostics.BrokerBootstrap.ConfigCompleteness.ConsumerGroup))
	if len(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingFields) > 0 {
		lines = append(lines, "- Missing fields: "+strings.Join(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingFields, ", "))
	}
	if len(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingRequiredEnv) > 0 {
		lines = append(lines, "- Missing required env: "+strings.Join(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingRequiredEnv, ", "))
	}
	if len(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingAdvisoryEnv) > 0 {
		lines = append(lines, "- Missing advisory env: "+strings.Join(diagnostics.BrokerBootstrap.ConfigDiagnostics.MissingAdvisoryEnv, ", "))
	}
	if len(diagnostics.BrokerBootstrap.ValidationErrors) > 0 {
		lines = append(lines, "- Validation errors: "+strings.Join(diagnostics.BrokerBootstrap.ValidationErrors, "; "))
	}
	if diagnostics.BrokerBootstrap.BootstrapSummary.BrokerBootstrap != nil {
		bootstrap := diagnostics.BrokerBootstrap.BootstrapSummary.BrokerBootstrap
		lines = append(lines, fmt.Sprintf("- Broker timing knobs: publish_timeout=%s replay_limit=%d checkpoint_interval=%s", bootstrap.PublishTimeout, bootstrap.ReplayLimit, bootstrap.CheckpointInterval))
	}
	if len(diagnostics.BrokerBootstrap.ConfigDiagnostics.NextActions) > 0 {
		lines = append(lines, "- Next actions: "+strings.Join(diagnostics.BrokerBootstrap.ConfigDiagnostics.NextActions, " ; "))
	}
	if len(diagnostics.BrokerBootstrap.ConfigDiagnostics.ReferenceDocs) > 0 {
		lines = append(lines, "- Reference docs: "+strings.Join(diagnostics.BrokerBootstrap.ConfigDiagnostics.ReferenceDocs, ", "))
	}
	lines = append(lines,
		"",
		"## Migration Readiness Review Pack",
		fmt.Sprintf("- Status: %s", diagnostics.MigrationReviewPack.Status),
		fmt.Sprintf("- Readiness report: %s", diagnostics.MigrationReviewPack.ReadinessReportPath),
		fmt.Sprintf("- Live shadow scorecard: %s", diagnostics.MigrationReviewPack.ScorecardPath),
		fmt.Sprintf("- Canonical summary: %s", diagnostics.MigrationReviewPack.CanonicalSummaryPath),
		fmt.Sprintf("- Run summary: %s", diagnostics.MigrationReviewPack.SummaryPath),
		fmt.Sprintf("- Live shadow index: %s", diagnostics.MigrationReviewPack.IndexPath),
		fmt.Sprintf("- Follow-up digest: %s", diagnostics.MigrationReviewPack.FollowUpDigestPath),
		fmt.Sprintf("- Rollback trigger surface: %s", diagnostics.MigrationReviewPack.RollbackTriggerPath),
		fmt.Sprintf("- Parity OK runs: %d", diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.ParityOKCount),
		fmt.Sprintf("- Drift detected runs: %d", diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.DriftDetectedCount),
		fmt.Sprintf("- Matrix mismatches: %d", diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.MatrixMismatched),
		fmt.Sprintf("- Corpus coverage present: %t", diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.CorpusCoveragePresent),
		fmt.Sprintf("- Rollback automation boundary: %s", diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.RollbackTriggerSurface.AutomationBoundary),
	)
	if len(diagnostics.MigrationReviewPack.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.MigrationReviewPack.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Rollback Trigger Surface",
		fmt.Sprintf("- Canonical report: %s", diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReportPath),
		fmt.Sprintf("- Issue: %s / %s", firstNonEmpty(diagnostics.MigrationReviewPack.RollbackTriggerSurface.Issue.ID, "unknown"), firstNonEmpty(diagnostics.MigrationReviewPack.RollbackTriggerSurface.Issue.Slug, "unknown")),
		fmt.Sprintf("- Status: %s", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Status),
		fmt.Sprintf("- Automation boundary: %s", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.AutomationBoundary),
		fmt.Sprintf("- Automated rollback trigger: %t", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.AutomatedRollbackTrigger),
		fmt.Sprintf("- Cutover gate: %s", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.CutoverGate),
		fmt.Sprintf("- Blockers: %d", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.Blockers),
		fmt.Sprintf("- Warnings: %d", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.Warnings),
		fmt.Sprintf("- Manual-only paths: %d", diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.ManualOnlyPaths),
	)
	if len(diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Broker Stub Live Fanout Isolation",
		fmt.Sprintf("- Canonical report: %s", diagnostics.BrokerFanoutIsolation.ReportPath),
		fmt.Sprintf("- Scenario count: %d", diagnostics.BrokerFanoutIsolation.Summary.ScenarioCount),
		fmt.Sprintf("- Isolated scenarios: %d", diagnostics.BrokerFanoutIsolation.Summary.IsolatedScenarios),
		fmt.Sprintf("- Stalled scenarios: %d", diagnostics.BrokerFanoutIsolation.Summary.StalledScenarios),
		fmt.Sprintf("- Replay backlog: %d events", diagnostics.BrokerFanoutIsolation.Summary.ReplayBacklogEvents),
		fmt.Sprintf("- Replay step delay: %dms", diagnostics.BrokerFanoutIsolation.Summary.ReplayStepDelayMS),
		fmt.Sprintf("- Live delivery deadline: %dms", diagnostics.BrokerFanoutIsolation.Summary.LiveDeliveryDeadlineMS),
	)
	for _, scenario := range diagnostics.BrokerFanoutIsolation.Scenarios {
		lines = append(lines, fmt.Sprintf("- %s: status=%s replay=%s live=%s backlog=%d replay_delay=%dms live_deadline=%dms replay_after_live=%t", scenario.Name, scenario.Status, firstNonEmpty(scenario.ReplayPath, "unknown"), firstNonEmpty(scenario.LivePath, "unknown"), scenario.ReplayBacklogEvents, scenario.ReplayStepDelayMS, scenario.LiveDeliveryDeadlineMS, scenario.ReplayDrainsAfterLive))
		if len(scenario.SourceTests) > 0 {
			lines = append(lines, "  - source tests: "+strings.Join(scenario.SourceTests, ", "))
		}
	}
	if len(diagnostics.BrokerFanoutIsolation.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.BrokerFanoutIsolation.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Delivery Acknowledgement Readiness",
		fmt.Sprintf("- Canonical report: %s", diagnostics.DeliveryAckReadiness.ReportPath),
		fmt.Sprintf("- Explicit ACK backends: %d", diagnostics.DeliveryAckReadiness.Summary.ExplicitAckBackends),
		fmt.Sprintf("- Durable ACK backends: %d", diagnostics.DeliveryAckReadiness.Summary.DurableAckBackends),
		fmt.Sprintf("- Best-effort backends: %d", diagnostics.DeliveryAckReadiness.Summary.BestEffortBackends),
		fmt.Sprintf("- Contract-only backends: %d", diagnostics.DeliveryAckReadiness.Summary.ContractOnlyBackends),
	)
	for _, backend := range diagnostics.DeliveryAckReadiness.Backends {
		lines = append(lines, fmt.Sprintf("- %s: class=%s explicit_ack=%t durable_ack=%t readiness=%s", backend.Backend, backend.AcknowledgementClass, backend.ExplicitAcknowledgement, backend.DurableAcknowledgement, backend.RuntimeReadiness))
		if len(backend.SourceReportLinks) > 0 {
			lines = append(lines, "  - sources: "+strings.Join(backend.SourceReportLinks, ", "))
		}
	}
	if len(diagnostics.DeliveryAckReadiness.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.DeliveryAckReadiness.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Publish Acknowledgement Outcome Ledger",
		fmt.Sprintf("- Canonical report: %s", diagnostics.PublishAckOutcomes.ReportPath),
		fmt.Sprintf("- Scenario: %s", firstNonEmpty(diagnostics.PublishAckOutcomes.Summary.ScenarioID, "unknown")),
		fmt.Sprintf("- Proof status: %s", firstNonEmpty(diagnostics.PublishAckOutcomes.Summary.ProofStatus, "unknown")),
		fmt.Sprintf("- committed=%d rejected=%d unknown_commit=%d", diagnostics.PublishAckOutcomes.Summary.CommittedCount, diagnostics.PublishAckOutcomes.Summary.RejectedCount, diagnostics.PublishAckOutcomes.Summary.UnknownCommitCount),
	)
	if len(diagnostics.PublishAckOutcomes.Summary.RequiredOutcomes) > 0 {
		lines = append(lines, "- Required outcomes: "+strings.Join(diagnostics.PublishAckOutcomes.Summary.RequiredOutcomes, ", "))
	}
	for _, outcome := range diagnostics.PublishAckOutcomes.Outcomes {
		lines = append(lines, fmt.Sprintf("- %s: %s", outcome.Outcome, firstNonEmpty(outcome.ProofRule, "n/a")))
		if len(outcome.RequiredEvidence) > 0 {
			lines = append(lines, "  - evidence: "+strings.Join(outcome.RequiredEvidence, ", "))
		}
		if outcome.OperatorAction != "" {
			lines = append(lines, "  - operator action: "+outcome.OperatorAction)
		}
	}
	if len(diagnostics.PublishAckOutcomes.Limitations) > 0 {
		lines = append(lines, "- Limitations: "+strings.Join(diagnostics.PublishAckOutcomes.Limitations, "; "))
	}
	if len(diagnostics.PublishAckOutcomes.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.PublishAckOutcomes.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Durable Sequence Bridge",
		fmt.Sprintf("- Canonical report: %s", diagnostics.SequenceBridge.ReportPath),
		fmt.Sprintf("- Backends: %d", diagnostics.SequenceBridge.Summary.BackendCount),
		fmt.Sprintf("- Live-proven backends: %d", diagnostics.SequenceBridge.Summary.LiveProvenBackends),
		fmt.Sprintf("- Harness-proven backends: %d", diagnostics.SequenceBridge.Summary.HarnessProvenBackends),
		fmt.Sprintf("- Contract-only backends: %d", diagnostics.SequenceBridge.Summary.ContractOnlyBackends),
		fmt.Sprintf("- One-to-one mappings: %d", diagnostics.SequenceBridge.Summary.OneToOneMappings),
		fmt.Sprintf("- Epoch-bridged backends: %d", diagnostics.SequenceBridge.Summary.ProviderEpochBridgedBackends),
	)
	for _, backend := range diagnostics.SequenceBridge.Backends {
		lines = append(lines, fmt.Sprintf("- %s: readiness=%s contract=%s", backend.Backend, backend.RuntimeReadiness, firstNonEmpty(backend.MappingContract, "n/a")))
		if backend.PortableSequenceSource != "" {
			lines = append(lines, "  - portable sequence: "+backend.PortableSequenceSource)
		}
		if backend.ProviderOffsetSource != "" {
			lines = append(lines, "  - provider offset: "+backend.ProviderOffsetSource)
		}
		if backend.OwnershipEpochSource != "" {
			lines = append(lines, "  - ownership epoch: "+backend.OwnershipEpochSource)
		}
	}
	if len(diagnostics.SequenceBridge.CurrentCeiling) > 0 {
		lines = append(lines, "- Current ceiling: "+strings.Join(diagnostics.SequenceBridge.CurrentCeiling, "; "))
	}
	if len(diagnostics.SequenceBridge.NextRuntimeHooks) > 0 {
		lines = append(lines, "- Next runtime hooks: "+strings.Join(diagnostics.SequenceBridge.NextRuntimeHooks, "; "))
	}
	lines = append(lines,
		"",
		"## Retention Watermark & Expiry",
		fmt.Sprintf("- Canonical report: %s", diagnostics.RetentionExpiry.ReportPath),
		fmt.Sprintf("- Runtime-visible backends: %d", diagnostics.RetentionExpiry.Summary.RuntimeVisibleBackends),
		fmt.Sprintf("- Persisted-boundary backends: %d", diagnostics.RetentionExpiry.Summary.PersistedBoundaryBackends),
		fmt.Sprintf("- Fail-closed expiry backends: %d", diagnostics.RetentionExpiry.Summary.FailClosedExpiryBackends),
		fmt.Sprintf("- Contract-only backends: %d", diagnostics.RetentionExpiry.Summary.ContractOnlyBackends),
	)
	for _, backend := range diagnostics.RetentionExpiry.Backends {
		lines = append(lines, fmt.Sprintf("- %s: readiness=%s visible=%t persisted=%t fail_closed=%t", backend.Backend, backend.RuntimeReadiness, backend.RetainedBoundaryVisible, backend.PersistedBoundaries, backend.FailClosedExpiry))
		if backend.ReplayBoundarySource != "" {
			lines = append(lines, "  - replay boundary: "+backend.ReplayBoundarySource)
		}
		if backend.CheckpointExpiryHandling != "" {
			lines = append(lines, "  - expiry handling: "+backend.CheckpointExpiryHandling)
		}
		if backend.CheckpointCleanupPolicy != "" {
			lines = append(lines, "  - checkpoint cleanup: "+backend.CheckpointCleanupPolicy)
		}
	}
	if len(diagnostics.RetentionExpiry.PolicySplit) > 0 {
		lines = append(lines, "- Policy split: "+strings.Join(diagnostics.RetentionExpiry.PolicySplit, "; "))
	}
	if len(diagnostics.RetentionExpiry.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.RetentionExpiry.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Provider-backed Live Handoff Isolation",
		fmt.Sprintf("- Canonical report: %s", diagnostics.ProviderLiveHandoff.ReportPath),
		fmt.Sprintf("- Backend: %s", firstNonEmpty(diagnostics.ProviderLiveHandoff.Backend, "unknown")),
		fmt.Sprintf("- Validation lane: %s", firstNonEmpty(diagnostics.ProviderLiveHandoff.ValidationLane, "unknown")),
		fmt.Sprintf("- Isolated scenarios: %d", diagnostics.ProviderLiveHandoff.Summary.IsolatedScenarios),
		fmt.Sprintf("- Stalled scenarios: %d", diagnostics.ProviderLiveHandoff.Summary.StalledScenarios),
		fmt.Sprintf("- Replay backlog events: %d", diagnostics.ProviderLiveHandoff.Summary.ReplayBacklogEvents),
		fmt.Sprintf("- Live delivery deadline: %dms", diagnostics.ProviderLiveHandoff.Summary.LiveDeliveryDeadlineMS),
	)
	for _, scenario := range diagnostics.ProviderLiveHandoff.Scenarios {
		lines = append(lines, fmt.Sprintf("- %s: status=%s replay_drains_after_live=%t", scenario.Name, scenario.Status, scenario.ReplayDrainsAfterLive))
		if scenario.ReplayPath != "" {
			lines = append(lines, "  - replay path: "+scenario.ReplayPath)
		}
		if scenario.LivePath != "" {
			lines = append(lines, "  - live path: "+scenario.LivePath)
		}
	}
	if len(diagnostics.ProviderLiveHandoff.Limitations) > 0 {
		lines = append(lines, "- Limitations: "+strings.Join(diagnostics.ProviderLiveHandoff.Limitations, "; "))
	}
	if len(diagnostics.ProviderLiveHandoff.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.ProviderLiveHandoff.ReviewerLinks, ", "))
	}
	lines = append(lines,
		"",
		"## Validation Bundle Continuation Gate",
		fmt.Sprintf("- Canonical report: %s", diagnostics.ContinuationGate.ReportPath),
		fmt.Sprintf("- Scorecard: %s", diagnostics.ContinuationGate.ScorecardPath),
		fmt.Sprintf("- Recommendation: %s", diagnostics.ContinuationGate.Recommendation),
		fmt.Sprintf("- Status: %s", diagnostics.ContinuationGate.Status),
		fmt.Sprintf("- Latest run: %s", firstNonEmpty(diagnostics.ContinuationGate.Summary.LatestRunID, "unknown")),
		fmt.Sprintf("- Latest bundle age: %.2f hours", diagnostics.ContinuationGate.Summary.LatestBundleAgeHours),
		fmt.Sprintf("- Recent bundle count: %d", diagnostics.ContinuationGate.Summary.RecentBundleCount),
		fmt.Sprintf("- Repeated executor coverage: %t", diagnostics.ContinuationGate.Summary.AllExecutorTracksHaveRepeatedRecentCoverage),
		fmt.Sprintf("- Shared queue companion available: %t", diagnostics.ContinuationGate.Summary.SharedQueueCompanionAvailable),
		fmt.Sprintf("- Cross-node completions: %d", diagnostics.ContinuationGate.Summary.CrossNodeCompletions),
	)
	if diagnostics.ContinuationGate.DigestPath != "" {
		lines = append(lines, "- Reviewer digest: "+diagnostics.ContinuationGate.DigestPath)
	}
	for _, check := range diagnostics.ContinuationGate.PolicyChecks {
		lines = append(lines, fmt.Sprintf("- Policy check %s: passed=%t detail=%s", check.Name, check.Passed, firstNonEmpty(check.Detail, "n/a")))
	}
	for _, lane := range diagnostics.ContinuationGate.ExecutorLanes {
		lines = append(lines, fmt.Sprintf("- Lane %s: latest_status=%s latest_enabled=%t enabled_runs=%d succeeded_runs=%d consecutive_successes=%d all_recent_runs_succeeded=%t", lane.Lane, firstNonEmpty(lane.LatestStatus, "unknown"), lane.LatestEnabled, lane.EnabledRuns, lane.SucceededRuns, lane.ConsecutiveSuccesses, lane.AllRecentRunsSucceeded))
	}
	if len(diagnostics.ContinuationGate.CurrentCeiling) > 0 {
		lines = append(lines, "- Current ceiling: "+strings.Join(diagnostics.ContinuationGate.CurrentCeiling, "; "))
	}
	if len(diagnostics.ContinuationGate.NextActions) > 0 {
		lines = append(lines, "- Next actions: "+strings.Join(diagnostics.ContinuationGate.NextActions, "; "))
	}
	if len(diagnostics.ContinuationGate.ReviewerLinks) > 0 {
		lines = append(lines, "- Reviewer links: "+strings.Join(diagnostics.ContinuationGate.ReviewerLinks, ", "))
	}
	lines = append(lines, "", "## Notes")
	for _, note := range diagnostics.ClusterHealth.Notes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func formatFacetCounts(items []auditFacetCount) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s=%d", item.Key, item.Count))
	}
	return strings.Join(parts, ", ")
}

func (s *Server) sharedQueueCoordinationDiagnostics() sharedQueueCoordinationDiagnostics {
	diagnostics := sharedQueueCoordinationDiagnostics{}
	if s.Queue != nil {
		if deadLetters, err := s.Queue.ListDeadLetters(context.Background(), 0); err == nil {
			diagnostics.DeadLetterBacklog = len(deadLetters)
		}
	}
	for _, event := range s.Recorder.EventsByTask("", 0) {
		switch event.Type {
		case domain.EventTaskDeadLetter:
			diagnostics.DeadLetterEvents++
		case domain.EventTaskQueued:
			if eventBoolValue(event.Payload, "replayed") {
				diagnostics.ReplayedQueueEvents++
			}
		case domain.EventSubscriberLeaseAcquired:
			diagnostics.LeaseAcquiredEvents++
		case domain.EventSubscriberLeaseRejected:
			diagnostics.LeaseRejectedEvents++
		case domain.EventSubscriberLeaseExpired:
			diagnostics.LeaseExpiredEvents++
		case domain.EventSubscriberTakeoverSucceeded:
			diagnostics.TakeoverSucceededEvents++
		case domain.EventSubscriberCheckpointCommitted:
			diagnostics.CheckpointCommittedEvents++
		case domain.EventSubscriberCheckpointRejected:
			diagnostics.CheckpointRejectedEvents++
			if strings.Contains(strings.ToLower(eventStringValue(event.Payload, "reason")), "fenced") {
				diagnostics.LeaseFencedEvents++
			}
		}
	}
	if checkpointResets := s.checkpointResetAuditSnapshot(20); checkpointResets != nil {
		diagnostics.CheckpointResetsRecent = checkpointResets.RecentCount
	}
	if watermark := s.typedRetentionWatermark(); watermark != nil {
		diagnostics.RetentionWatermarkAvailable = true
		diagnostics.RetentionTrimmedThroughSeq = watermark.TrimmedThroughSequence
		diagnostics.RetentionHistoryTruncated = watermark.HistoryTruncated
	}
	return diagnostics
}

func sanitizeReportName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "all"
	}
	return value
}

func buildTraceExportBundle(assignments []distributedTaskAssignment, summaries []observability.TraceSummary) traceExportBundleSummary {
	ambiguousPublishProof := buildAmbiguousPublishProofReference()
	assignmentByTrace := make(map[string]distributedTaskAssignment, len(assignments))
	tracesWithTerminalState := 0
	for _, assignment := range assignments {
		if assignment.Task.TraceID == "" {
			continue
		}
		assignmentByTrace[assignment.Task.TraceID] = assignment
		if isTerminalState(assignment.EffectiveState) {
			tracesWithTerminalState++
		}
	}
	recent := make([]traceExportBundleTrace, 0, len(summaries))
	for _, summary := range summaries {
		assignment, ok := assignmentByTrace[summary.TraceID]
		if !ok {
			continue
		}
		recent = append(recent, traceExportBundleTrace{
			TraceID:         summary.TraceID,
			TaskID:          assignment.Task.ID,
			Executor:        string(assignment.Executor),
			State:           string(assignment.EffectiveState),
			EventCount:      summary.EventCount,
			LatestEventType: summary.LatestEventType,
			DurationSeconds: summary.DurationSeconds,
			TraceURL:        fmt.Sprintf("/debug/traces/%s?limit=%d", summary.TraceID, 200),
			EventURL:        fmt.Sprintf("/events?trace_id=%s&limit=%d", summary.TraceID, 200),
		})
	}
	return traceExportBundleSummary{
		TotalTraces:             len(assignmentByTrace),
		TracesWithTerminalState: tracesWithTerminalState,
		RecentTraces:            recent,
		ValidationArtifacts: []string{
			"docs/reports/live-validation-index.md",
			"docs/reports/live-validation-summary.json",
			"docs/reports/go-control-plane-observability-report.md",
			"docs/reports/broker-validation-summary.json",
			"docs/reports/broker-failover-stub-report.json",
			"docs/reports/broker-failover-stub-artifacts",
			ambiguousPublishProof.Path,
		},
		ReviewerNavigation: []string{
			"/v2/reports/distributed/export",
			"/debug/traces",
			"/events?trace_id=<trace_id>&limit=200",
			"docs/reports/review-readiness.md",
			"docs/reports/broker-failover-fault-injection-validation-pack.md",
			ambiguousPublishProof.Path,
		},
		BackendLimitations: []string{
			"no external tracing backend or OTLP/Jaeger/Tempo/Zipkin export path",
			"no cross-process span propagation beyond in-memory trace_id grouping",
			"validation evidence is workflow-exported and repo-native, not a continuously indexed trace service",
		},
		AmbiguousPublishProof: ambiguousPublishProof,
	}
}

func buildBrokerReviewPack() brokerReviewPack {
	return brokerReviewPack{
		Status:             "checked_in_stub_evidence",
		SummaryPath:        "docs/reports/broker-validation-summary.json",
		ReportPath:         "docs/reports/broker-failover-stub-report.json",
		ValidationPackPath: "docs/reports/broker-failover-fault-injection-validation-pack.md",
		ArtifactDirectory:  "docs/reports/broker-failover-stub-artifacts",
		ReviewerLinks: []string{
			"docs/reports/live-validation-index.json",
			"docs/reports/review-readiness.md",
		},
		AmbiguousPublishProof: buildAmbiguousPublishProofReference(),
	}
}

func buildAmbiguousPublishProofReference() brokerProofReference {
	return brokerProofReference{
		Path:       "docs/reports/ambiguous-publish-outcome-proof-summary.json",
		ScenarioID: "BF-05",
		Outcomes:   []string{"committed", "rejected", "unknown_commit"},
	}
}

func isTerminalState(state domain.TaskState) bool {
	switch state {
	case domain.TaskSucceeded, domain.TaskFailed, domain.TaskCancelled, domain.TaskDeadLetter:
		return true
	default:
		return false
	}
}
