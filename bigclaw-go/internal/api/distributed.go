package api

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
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
	Executor             string  `json:"executor"`
	MaxConcurrency       int     `json:"max_concurrency"`
	AvailableConcurrency int     `json:"available_concurrency"`
	ActiveWorkers        int     `json:"active_workers"`
	RoutedDecisions      int     `json:"routed_decisions"`
	StartedRuns          int     `json:"started_runs"`
	CompletedRuns        int     `json:"completed_runs"`
	DeadLetters          int     `json:"dead_letters"`
	SaturationPercent    float64 `json:"saturation_percent"`
	SupportsGPU          bool    `json:"supports_gpu"`
	SupportsBrowser      bool    `json:"supports_browser"`
	SupportsShell        bool    `json:"supports_shell"`
	Health               string  `json:"health"`
}

type clusterHealthRollup struct {
	HealthyExecutors  int            `json:"healthy_executors"`
	DegradedExecutors int            `json:"degraded_executors"`
	IdleExecutors     int            `json:"idle_executors"`
	WorkerStates      map[string]int `json:"worker_states"`
	Notes             []string       `json:"notes"`
}

type distributedDiagnosticsReport struct {
	Markdown  string `json:"markdown"`
	ExportURL string `json:"export_url"`
}

type distributedDiagnostics struct {
	Summary          distributedDiagnosticsSummary `json:"summary"`
	RoutingReasons   []routingReasonSummary        `json:"routing_reasons"`
	ExecutorCapacity []executorCapacityView        `json:"executor_capacity"`
	ClusterHealth    clusterHealthRollup           `json:"cluster_health"`
	RolloutReport    distributedDiagnosticsReport  `json:"rollout_report"`
}

type executorDiagnosticsCounters struct {
	Routed     int
	Started    int
	Completed  int
	DeadLetter int
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
			"team":     filters.Team,
			"project":  filters.Project,
			"task_id":  filters.TaskID,
			"limit":    filters.Limit,
			"priority": filters.Priority,
		},
		"summary":           diagnostics.Summary,
		"routing_reasons":   diagnostics.RoutingReasons,
		"executor_capacity": diagnostics.ExecutorCapacity,
		"cluster_health":    diagnostics.ClusterHealth,
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
	countersByExecutor := make(map[domain.ExecutorKind]*executorDiagnosticsCounters)
	routingIndex := make(map[string]*routingReasonSummary)
	totalRouted := 0
	for _, event := range s.Recorder.EventsByTask("", 0) {
		if event.TaskID == "" {
			continue
		}
		task, ok := s.taskSnapshot(event.TaskID)
		if !ok {
			continue
		}
		takeover, hasTakeover := s.Control.TakeoverStatus(task.ID)
		if !matchesTaskFilters(task, effectiveTaskState(task.State, takeoverOrNil(takeover, hasTakeover)), filters) {
			continue
		}
		executorKind := eventExecutorKind(event, task)
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
	saturatedExecutors := 0
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
		if view.SaturationPercent >= 80 || (view.MaxConcurrency > 0 && view.AvailableConcurrency == 0 && view.RoutedDecisions > 0) {
			saturatedExecutors++
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

	summary := distributedDiagnosticsSummary{
		RegisteredExecutors:  len(capabilities),
		ActiveExecutors:      activeExecutors,
		TotalTasks:           len(tasks),
		ActiveRuns:           countActiveRuns(tasks),
		TotalRoutedDecisions: totalRouted,
		ActiveWorkers:        activeWorkers,
		IdleWorkers:          idleWorkers,
		SaturatedExecutors:   saturatedExecutors,
		ActiveTakeovers:      len(s.filteredActiveTakeovers(filters)),
	}
	clusterHealth := clusterHealthRollup{
		HealthyExecutors:  healthyExecutors,
		DegradedExecutors: degradedExecutors,
		IdleExecutors:     idleExecutors,
		WorkerStates:      workerStates,
		Notes:             diagnosticsNotes(summary, executorCapacity, s.Control.Snapshot()),
	}
	diagnostics := distributedDiagnostics{
		Summary:          summary,
		RoutingReasons:   routingReasons,
		ExecutorCapacity: executorCapacity,
		ClusterHealth:    clusterHealth,
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

func diagnosticsHealth(view executorCapacityView) string {
	if view.DeadLetters > 0 && view.CompletedRuns == 0 {
		return "degraded"
	}
	if view.ActiveWorkers > 0 || view.RoutedDecisions > 0 || view.CompletedRuns > 0 {
		return "healthy"
	}
	return "idle"
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
		if item.SaturationPercent >= 80 {
			notes = append(notes, fmt.Sprintf("%s executor is above 80%% worker saturation", item.Executor))
		}
	}
	if len(notes) == 0 {
		notes = append(notes, "distributed control plane looks healthy for the current slice")
	}
	return notes
}

func countActiveRuns(tasks []domain.Task) int {
	count := 0
	for _, task := range tasks {
		if domain.IsActiveTaskState(task.State) {
			count++
		}
	}
	return count
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
		fmt.Sprintf("Filters: team=%s project=%s task_id=%s", firstNonEmpty(filters.Team, "all"), firstNonEmpty(filters.Project, "all"), firstNonEmpty(filters.TaskID, "all")),
		"",
		"## Summary",
		fmt.Sprintf("- Registered executors: %d", diagnostics.Summary.RegisteredExecutors),
		fmt.Sprintf("- Active executors: %d", diagnostics.Summary.ActiveExecutors),
		fmt.Sprintf("- Total tasks: %d", diagnostics.Summary.TotalTasks),
		fmt.Sprintf("- Active runs: %d", diagnostics.Summary.ActiveRuns),
		fmt.Sprintf("- Routed decisions: %d", diagnostics.Summary.TotalRoutedDecisions),
		fmt.Sprintf("- Active workers: %d", diagnostics.Summary.ActiveWorkers),
		fmt.Sprintf("- Idle workers: %d", diagnostics.Summary.IdleWorkers),
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
		lines = append(lines, fmt.Sprintf("- %s: health=%s active_workers=%d max_concurrency=%d available=%d saturation=%.1f%% routed=%d completed=%d dead_letters=%d", item.Executor, item.Health, item.ActiveWorkers, item.MaxConcurrency, item.AvailableConcurrency, item.SaturationPercent, item.RoutedDecisions, item.CompletedRuns, item.DeadLetters))
	}
	lines = append(lines,
		"",
		"## Cluster Health",
		fmt.Sprintf("- Healthy executors: %d", diagnostics.ClusterHealth.HealthyExecutors),
		fmt.Sprintf("- Degraded executors: %d", diagnostics.ClusterHealth.DegradedExecutors),
		fmt.Sprintf("- Idle executors: %d", diagnostics.ClusterHealth.IdleExecutors),
	)
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
	lines = append(lines, "", "## Notes")
	for _, note := range diagnostics.ClusterHealth.Notes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func sanitizeReportName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "all"
	}
	return value
}
