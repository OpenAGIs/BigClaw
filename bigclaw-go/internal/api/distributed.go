package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
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

type distributedDiagnosticsReport struct {
	Markdown  string `json:"markdown"`
	ExportURL string `json:"export_url"`
}

type distributedRayEvidence struct {
	SummaryPath         string `json:"summary_path,omitempty"`
	CanonicalReportPath string `json:"canonical_report_path,omitempty"`
	BundleReportPath    string `json:"bundle_report_path,omitempty"`
	BundlePath          string `json:"bundle_path,omitempty"`
	ServiceLogPath      string `json:"service_log_path,omitempty"`
	AuditLogPath        string `json:"audit_log_path,omitempty"`
}

type distributedRayReadiness struct {
	Configured                 bool                   `json:"configured"`
	Address                    string                 `json:"address,omitempty"`
	ValidationStatus           string                 `json:"validation_status,omitempty"`
	LocalValidationStatus      string                 `json:"local_validation_status,omitempty"`
	KubernetesValidationStatus string                 `json:"kubernetes_validation_status,omitempty"`
	LatestRunID                string                 `json:"latest_run_id,omitempty"`
	GeneratedAt                string                 `json:"generated_at,omitempty"`
	TaskID                     string                 `json:"task_id,omitempty"`
	LatestEventType            string                 `json:"latest_event_type,omitempty"`
	LatestEventAt              string                 `json:"latest_event_at,omitempty"`
	JobArtifacts               []string               `json:"job_artifacts,omitempty"`
	Evidence                   distributedRayEvidence `json:"evidence,omitempty"`
	Notes                      []string               `json:"notes,omitempty"`
}

type distributedDiagnostics struct {
	Summary          distributedDiagnosticsSummary `json:"summary"`
	RoutingReasons   []routingReasonSummary        `json:"routing_reasons"`
	ExecutorCapacity []executorCapacityView        `json:"executor_capacity"`
	ClusterHealth    clusterHealthRollup           `json:"cluster_health"`
	RayReadiness     distributedRayReadiness       `json:"ray_readiness"`
	RolloutReport    distributedDiagnosticsReport  `json:"rollout_report"`
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

type liveValidationArtifactSummary struct {
	RunID       string                         `json:"run_id"`
	GeneratedAt string                         `json:"generated_at"`
	Status      string                         `json:"status"`
	BundlePath  string                         `json:"bundle_path"`
	Local       liveValidationComponentSummary `json:"local"`
	Kubernetes  liveValidationComponentSummary `json:"kubernetes"`
	Ray         liveValidationComponentSummary `json:"ray"`
}

type liveValidationComponentSummary struct {
	Enabled             bool                     `json:"enabled"`
	BundleReportPath    string                   `json:"bundle_report_path"`
	CanonicalReportPath string                   `json:"canonical_report_path"`
	Status              string                   `json:"status"`
	TaskID              string                   `json:"task_id"`
	ServiceLogPath      string                   `json:"service_log_path"`
	AuditLogPath        string                   `json:"audit_log_path"`
	Report              liveValidationReportBody `json:"report"`
}

type liveValidationReportBody struct {
	Status liveValidationReportStatus `json:"status"`
}

type liveValidationReportStatus struct {
	State       string                  `json:"state"`
	TaskID      string                  `json:"task_id"`
	LatestEvent liveValidationEvent     `json:"latest_event"`
	Task        liveValidationTaskState `json:"task"`
}

type liveValidationTaskState struct {
	ID string `json:"id"`
}

type liveValidationEvent struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
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
			"limit":      filters.Limit,
			"priority":   filters.Priority,
		},
		"summary":           diagnostics.Summary,
		"routing_reasons":   diagnostics.RoutingReasons,
		"executor_capacity": diagnostics.ExecutorCapacity,
		"cluster_health":    diagnostics.ClusterHealth,
		"ray_readiness":     diagnostics.RayReadiness,
		"report":            diagnostics.RolloutReport,
	})
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
	pool := s.workerPoolSummary()
	workerStates := make(map[string]int)
	activeByExecutor := make(map[domain.ExecutorKind]int)
	activeWorkers := 0
	idleWorkers := 0
	if pool != nil {
		activeWorkers = pool.ActiveWorkers
		idleWorkers = pool.IdleWorkers
		for _, workerStatus := range pool.Workers {
			state := strings.TrimSpace(workerStatus.State)
			if state == "" {
				state = "idle"
			}
			workerStates[state]++
			if workerStatus.CurrentExecutor != "" && (state == "leased" || state == "running") {
				activeByExecutor[workerStatus.CurrentExecutor]++
			}
		}
	}

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

	countersByExecutor := make(map[domain.ExecutorKind]*executorDiagnosticsCounters)
	routingIndex := make(map[string]*routingReasonSummary)
	totalRouted := 0
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
		case domain.EventTaskDeadLetter:
			entry.DeadLetter++
		}
	}

	routingReasons := make([]routingReasonSummary, 0, len(routingIndex))
	for _, item := range routingIndex {
		routingReasons = append(routingReasons, *item)
	}
	sort.SliceStable(routingReasons, func(i, j int) bool {
		if routingReasons[i].Count == routingReasons[j].Count {
			if routingReasons[i].Executor == routingReasons[j].Executor {
				return routingReasons[i].Reason < routingReasons[j].Reason
			}
			return routingReasons[i].Executor < routingReasons[j].Executor
		}
		return routingReasons[i].Count > routingReasons[j].Count
	})

	executorCapacity := make([]executorCapacityView, 0, len(capabilities))
	healthyExecutors := 0
	degradedExecutors := 0
	idleExecutors := 0
	activeExecutors := 0
	saturatedExecutors := make([]string, 0)
	for _, capability := range capabilities {
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
		for _, assignment := range tasksByExecutor[capability.Kind] {
			if assignment.EffectiveState == domain.TaskQueued || assignment.EffectiveState == domain.TaskRetrying {
				view.QueuedTasks++
			}
			if domain.IsActiveTaskState(assignment.EffectiveState) {
				view.ActiveTasks++
			}
		}
		view.TeamBreakdown = facetCountsFromAssignments(tasksByExecutor[capability.Kind], func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["team"]), "unassigned")
		})
		view.ProjectBreakdown = facetCountsFromAssignments(tasksByExecutor[capability.Kind], func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["project"]), "unassigned")
		})
		view.TopRoutingReasons = routingReasonsForExecutor(routingReasons, capability.Kind, 3)
		view.SampleTasks = sampleTaskIDs(tasksByExecutor[capability.Kind], 3)
		if capability.MaxConcurrency > 0 {
			view.AvailableConcurrency = capability.MaxConcurrency - view.ActiveWorkers
			if view.AvailableConcurrency < 0 {
				view.AvailableConcurrency = 0
			}
			view.SaturationPercent = float64(view.ActiveWorkers) / float64(capability.MaxConcurrency) * 100
		}
		view.Health = diagnosticsHealth(view)
		if view.Health != "idle" {
			activeExecutors++
		}
		if executorIsSaturated(view) {
			saturatedExecutors = append(saturatedExecutors, view.Executor)
		}
		switch view.Health {
		case "healthy":
			healthyExecutors++
		case "degraded":
			degradedExecutors++
		default:
			idleExecutors++
		}
		executorCapacity = append(executorCapacity, view)
	}
	sort.SliceStable(executorCapacity, func(i, j int) bool { return executorCapacity[i].Executor < executorCapacity[j].Executor })
	sort.Strings(saturatedExecutors)

	summary := distributedDiagnosticsSummary{
		RegisteredExecutors:  len(capabilities),
		ActiveExecutors:      activeExecutors,
		TotalTasks:           len(assignments),
		ActiveRuns:           countActiveAssignments(assignments),
		TotalRoutedDecisions: totalRouted,
		ActiveWorkers:        activeWorkers,
		IdleWorkers:          idleWorkers,
		SaturatedExecutors:   len(saturatedExecutors),
		ActiveTakeovers:      len(takeovers),
	}
	clusterHealth := clusterHealthRollup{
		HealthyExecutors:  healthyExecutors,
		DegradedExecutors: degradedExecutors,
		IdleExecutors:     idleExecutors,
		WorkerStates:      workerStates,
		TeamBreakdown: facetCountsFromAssignments(assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["team"]), "unassigned")
		}),
		ProjectBreakdown: facetCountsFromAssignments(assignments, func(item distributedTaskAssignment) string {
			return firstNonEmpty(strings.TrimSpace(item.Task.Metadata["project"]), "unassigned")
		}),
		TakeoverOwners:     facetCountsFromTakeovers(takeovers, func(item control.Takeover) string { return firstNonEmpty(strings.TrimSpace(item.Owner), "unassigned") }),
		SaturatedExecutors: saturatedExecutors,
		Notes:              diagnosticsNotes(summary, executorCapacity, s.Control.Snapshot()),
	}
	diagnostics := distributedDiagnostics{
		Summary:          summary,
		RoutingReasons:   routingReasons,
		ExecutorCapacity: executorCapacity,
		ClusterHealth:    clusterHealth,
		RayReadiness:     s.buildRayReadiness(),
	}
	diagnostics.RolloutReport = distributedDiagnosticsReport{
		Markdown:  renderDistributedDiagnosticsMarkdown(diagnostics, filters),
		ExportURL: distributedExportURL(filters),
	}
	return diagnostics
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

func taskRequiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if item == tool {
			return true
		}
	}
	return false
}

func (s *Server) buildRayReadiness() distributedRayReadiness {
	readiness := distributedRayReadiness{
		Configured: strings.TrimSpace(s.RuntimeConfig.RayAddress) != "",
		Address:    strings.TrimSpace(s.RuntimeConfig.RayAddress),
		Evidence: distributedRayEvidence{
			SummaryPath: "docs/reports/live-validation-summary.json",
		},
	}
	if readiness.Configured {
		readiness.Notes = append(readiness.Notes, fmt.Sprintf("ray executor configured for %s", readiness.Address))
	} else {
		readiness.Notes = append(readiness.Notes, "ray executor address is not configured")
	}
	summary, err := s.loadLiveValidationSummary()
	if err != nil {
		report, reportErr := s.loadRayValidationReport()
		if reportErr != nil {
			readiness.Notes = append(readiness.Notes, "ray live-validation artifacts not found")
			return readiness
		}
		readiness.Evidence.CanonicalReportPath = "docs/reports/ray-live-smoke-report.json"
		populateRayReadinessFromComponent(&readiness, liveValidationComponentSummary{
			CanonicalReportPath: readiness.Evidence.CanonicalReportPath,
			Report:              report,
		})
		readiness.Notes = append(readiness.Notes, "loaded ray readiness from canonical smoke report without live-validation summary")
		return readiness
	}
	readiness.LatestRunID = summary.RunID
	readiness.GeneratedAt = summary.GeneratedAt
	readiness.LocalValidationStatus = firstNonEmpty(summary.Local.Status, summary.Local.Report.Status.State)
	readiness.KubernetesValidationStatus = firstNonEmpty(summary.Kubernetes.Status, summary.Kubernetes.Report.Status.State)
	readiness.Evidence.BundlePath = summary.BundlePath
	populateRayReadinessFromComponent(&readiness, summary.Ray)
	switch {
	case readiness.ValidationStatus == "succeeded":
		readiness.Notes = append(readiness.Notes, "latest Ray live-validation bundle succeeded")
	case readiness.ValidationStatus != "":
		readiness.Notes = append(readiness.Notes, fmt.Sprintf("latest Ray live-validation bundle status is %s", readiness.ValidationStatus))
	default:
		readiness.Notes = append(readiness.Notes, "Ray live-validation bundle does not include a status")
	}
	if readiness.LocalValidationStatus != "" || readiness.KubernetesValidationStatus != "" {
		readiness.Notes = append(readiness.Notes, fmt.Sprintf("companion validation states local=%s kubernetes=%s", firstNonEmpty(readiness.LocalValidationStatus, "unknown"), firstNonEmpty(readiness.KubernetesValidationStatus, "unknown")))
	}
	return readiness
}

func populateRayReadinessFromComponent(readiness *distributedRayReadiness, component liveValidationComponentSummary) {
	readiness.ValidationStatus = firstNonEmpty(component.Status, component.Report.Status.State)
	readiness.TaskID = firstNonEmpty(component.TaskID, component.Report.Status.TaskID, component.Report.Status.Task.ID)
	readiness.LatestEventType = component.Report.Status.LatestEvent.Type
	readiness.LatestEventAt = component.Report.Status.LatestEvent.Timestamp
	readiness.JobArtifacts = stringSliceValue(component.Report.Status.LatestEvent.Payload["artifacts"])
	readiness.Evidence.CanonicalReportPath = firstNonEmpty(readiness.Evidence.CanonicalReportPath, component.CanonicalReportPath)
	readiness.Evidence.BundleReportPath = component.BundleReportPath
	readiness.Evidence.ServiceLogPath = component.ServiceLogPath
	readiness.Evidence.AuditLogPath = component.AuditLogPath
}

func (s *Server) loadLiveValidationSummary() (liveValidationArtifactSummary, error) {
	var summary liveValidationArtifactSummary
	content, err := os.ReadFile(filepath.Join(s.reportsDir(), "live-validation-summary.json"))
	if err != nil {
		return summary, err
	}
	if err := json.Unmarshal(content, &summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func (s *Server) loadRayValidationReport() (liveValidationReportBody, error) {
	var report liveValidationReportBody
	content, err := os.ReadFile(filepath.Join(s.reportsDir(), "ray-live-smoke-report.json"))
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(content, &report); err != nil {
		return report, err
	}
	return report, nil
}

func (s *Server) reportsDir() string {
	if strings.TrimSpace(s.ReportsDir) != "" {
		return s.ReportsDir
	}
	return filepath.Join("docs", "reports")
}

func stringSliceValue(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if ok && strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out
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
		fmt.Sprintf("Filters: team=%s project=%s task_id=%s state=%s risk_level=%s", firstNonEmpty(filters.Team, "all"), firstNonEmpty(filters.Project, "all"), firstNonEmpty(filters.TaskID, "all"), firstNonEmpty(filters.State, "all"), firstNonEmpty(filters.RiskLevel, "all")),
		"",
		"## Summary",
		fmt.Sprintf("- Registered executors: %d", diagnostics.Summary.RegisteredExecutors),
		fmt.Sprintf("- Active executors: %d", diagnostics.Summary.ActiveExecutors),
		fmt.Sprintf("- Total tasks: %d", diagnostics.Summary.TotalTasks),
		fmt.Sprintf("- Active runs: %d", diagnostics.Summary.ActiveRuns),
		fmt.Sprintf("- Routed decisions: %d", diagnostics.Summary.TotalRoutedDecisions),
		fmt.Sprintf("- Active workers: %d", diagnostics.Summary.ActiveWorkers),
		fmt.Sprintf("- Idle workers: %d", diagnostics.Summary.IdleWorkers),
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
	lines = append(lines, "", "## Notes")
	for _, note := range diagnostics.ClusterHealth.Notes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Ray Readiness")
	lines = append(lines, fmt.Sprintf("- Configured: %t", diagnostics.RayReadiness.Configured))
	if diagnostics.RayReadiness.Address != "" {
		lines = append(lines, "- Address: "+diagnostics.RayReadiness.Address)
	}
	if diagnostics.RayReadiness.ValidationStatus != "" {
		lines = append(lines, "- Validation status: "+diagnostics.RayReadiness.ValidationStatus)
	}
	if diagnostics.RayReadiness.LocalValidationStatus != "" || diagnostics.RayReadiness.KubernetesValidationStatus != "" {
		lines = append(lines, fmt.Sprintf("- Companion validation: local=%s kubernetes=%s", firstNonEmpty(diagnostics.RayReadiness.LocalValidationStatus, "unknown"), firstNonEmpty(diagnostics.RayReadiness.KubernetesValidationStatus, "unknown")))
	}
	if diagnostics.RayReadiness.LatestRunID != "" {
		lines = append(lines, "- Latest run: "+diagnostics.RayReadiness.LatestRunID)
	}
	if diagnostics.RayReadiness.GeneratedAt != "" {
		lines = append(lines, "- Generated at: "+diagnostics.RayReadiness.GeneratedAt)
	}
	if diagnostics.RayReadiness.TaskID != "" {
		lines = append(lines, "- Task ID: "+diagnostics.RayReadiness.TaskID)
	}
	if diagnostics.RayReadiness.LatestEventType != "" {
		lines = append(lines, fmt.Sprintf("- Latest event: %s @ %s", diagnostics.RayReadiness.LatestEventType, firstNonEmpty(diagnostics.RayReadiness.LatestEventAt, "unknown")))
	}
	if len(diagnostics.RayReadiness.JobArtifacts) > 0 {
		lines = append(lines, "- Job artifacts: "+strings.Join(diagnostics.RayReadiness.JobArtifacts, ", "))
	}
	if diagnostics.RayReadiness.Evidence.SummaryPath != "" {
		lines = append(lines, "- Summary JSON: "+diagnostics.RayReadiness.Evidence.SummaryPath)
	}
	if diagnostics.RayReadiness.Evidence.CanonicalReportPath != "" {
		lines = append(lines, "- Canonical report: "+diagnostics.RayReadiness.Evidence.CanonicalReportPath)
	}
	if diagnostics.RayReadiness.Evidence.BundleReportPath != "" {
		lines = append(lines, "- Bundle report: "+diagnostics.RayReadiness.Evidence.BundleReportPath)
	}
	if diagnostics.RayReadiness.Evidence.BundlePath != "" {
		lines = append(lines, "- Bundle path: "+diagnostics.RayReadiness.Evidence.BundlePath)
	}
	if diagnostics.RayReadiness.Evidence.ServiceLogPath != "" {
		lines = append(lines, "- Service log: "+diagnostics.RayReadiness.Evidence.ServiceLogPath)
	}
	if diagnostics.RayReadiness.Evidence.AuditLogPath != "" {
		lines = append(lines, "- Audit log: "+diagnostics.RayReadiness.Evidence.AuditLogPath)
	}
	for _, note := range diagnostics.RayReadiness.Notes {
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

func sanitizeReportName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "all"
	}
	return value
}
