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
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/policy"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/worker"
)

type dashboardSummary struct {
	TotalTasks        int            `json:"total_tasks"`
	ActiveRuns        int            `json:"active_runs"`
	Blockers          int            `json:"blockers"`
	PremiumRuns       int            `json:"premium_runs"`
	SLARiskRuns       int            `json:"sla_risk_runs"`
	BudgetCentsTotal  int64          `json:"budget_cents_total"`
	StateDistribution map[string]int `json:"state_distribution"`
}

type funnelSummary struct {
	Tickets   int `json:"tickets"`
	PROpened  int `json:"prs_opened"`
	MergedPRs int `json:"merged_prs"`
}

type dashboardBreakdown struct {
	Key              string `json:"key"`
	TotalTasks       int    `json:"total_tasks"`
	ActiveRuns       int    `json:"active_runs"`
	Blockers         int    `json:"blockers"`
	PremiumRuns      int    `json:"premium_runs"`
	SLARiskRuns      int    `json:"sla_risk_runs"`
	BudgetCentsTotal int64  `json:"budget_cents_total"`
	MergedPRs        int    `json:"merged_prs"`
}

type dashboardDrilldown struct {
	Run               string `json:"run"`
	Events            string `json:"events"`
	Replay            string `json:"replay"`
	IssueKey          string `json:"issue_key,omitempty"`
	IssueURL          string `json:"issue_url,omitempty"`
	PullRequestURL    string `json:"pull_request_url,omitempty"`
	PullRequestStatus string `json:"pull_request_status,omitempty"`
	Workpad           string `json:"workpad,omitempty"`
}

type dashboardTaskOverview struct {
	Task      domain.Task        `json:"task"`
	Policy    policy.Summary     `json:"policy"`
	Takeover  *control.Takeover  `json:"takeover,omitempty"`
	Latest    *domain.Event      `json:"latest_event,omitempty"`
	Drilldown dashboardDrilldown `json:"drilldown"`
}

type controlCenterSummary struct {
	QueueDepth            int            `json:"queue_depth"`
	LeasedRuns            int            `json:"leased_runs"`
	BlockedRuns           int            `json:"blocked_runs"`
	PremiumRuns           int            `json:"premium_runs"`
	HighRiskRuns          int            `json:"high_risk_runs"`
	QueueBudgetCentsTotal int64          `json:"queue_budget_cents_total"`
	StateDistribution     map[string]int `json:"state_distribution"`
	RiskDistribution      map[string]int `json:"risk_distribution"`
	PriorityDistribution  map[string]int `json:"priority_distribution"`
	DeadLetters           int            `json:"dead_letters"`
	ActiveTakeovers       int            `json:"active_takeovers"`
}

type workerPoolSummary struct {
	TotalWorkers  int             `json:"total_workers"`
	ActiveWorkers int             `json:"active_workers"`
	IdleWorkers   int             `json:"idle_workers"`
	Workers       []worker.Status `json:"workers"`
}

type controlActionAuditEntry struct {
	Action    string       `json:"action"`
	Actor     string       `json:"actor,omitempty"`
	Role      string       `json:"role,omitempty"`
	TaskID    string       `json:"task_id,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Reason    string       `json:"reason,omitempty"`
	Note      string       `json:"note,omitempty"`
	Event     domain.Event `json:"event"`
}

type controlCenterFilters struct {
	Team       string
	Project    string
	TaskID     string
	State      string
	RiskLevel  string
	Actor      string
	Action     string
	Priority   *int
	Limit      int
	AuditLimit int
}

type taskOverview struct {
	Task     domain.Task       `json:"task"`
	Policy   policy.Summary    `json:"policy"`
	Takeover *control.Takeover `json:"takeover,omitempty"`
	Latest   *domain.Event     `json:"latest_event,omitempty"`
}

type queueTaskOverview struct {
	QueueTask      queue.TaskSnapshot `json:"queue_task"`
	EffectiveState domain.TaskState   `json:"effective_state"`
	Policy         policy.Summary     `json:"policy"`
	Takeover       *control.Takeover  `json:"takeover,omitempty"`
}

type runDetailResponse struct {
	Task          domain.Task                 `json:"task"`
	State         string                      `json:"state"`
	Policy        policy.Summary              `json:"policy"`
	Collaboration *control.Takeover           `json:"collaboration,omitempty"`
	Trace         *observability.TraceSummary `json:"trace,omitempty"`
	Events        []domain.Event              `json:"events"`
	Timeline      []domain.Event              `json:"timeline"`
	Validation    map[string]any              `json:"validation"`
	Artifacts     map[string]string           `json:"artifacts"`
	Workpad       string                      `json:"workpad,omitempty"`
}

type controlActionRequest struct {
	Action     string `json:"action"`
	TaskID     string `json:"task_id,omitempty"`
	Actor      string `json:"actor,omitempty"`
	Role       string `json:"role,omitempty"`
	ViewerTeam string `json:"viewer_team,omitempty"`
	Reviewer   string `json:"reviewer,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Note       string `json:"note,omitempty"`
}

func (s *Server) handleV2EngineeringDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	since, err := parseOptionalTime(r.URL.Query().Get("since"))
	if err != nil {
		http.Error(w, "invalid since value, expected RFC3339", http.StatusBadRequest)
		return
	}
	until, err := parseOptionalTime(r.URL.Query().Get("until"))
	if err != nil {
		http.Error(w, "invalid until value, expected RFC3339", http.StatusBadRequest)
		return
	}
	if err := enforceScopedTeamFilter(authorization, &team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	tasks := s.filteredTasks(team, project, tenantID, since, until)
	summary := dashboardSummary{StateDistribution: make(map[string]int)}
	funnel := funnelSummary{}
	overviews := make([]dashboardTaskOverview, 0, len(tasks))
	projectBreakdown := make(map[string]*dashboardBreakdown)
	teamBreakdown := make(map[string]*dashboardBreakdown)
	blockedTasks := make([]dashboardTaskOverview, 0)
	highRiskTasks := make([]dashboardTaskOverview, 0)
	for _, task := range tasks {
		summary.TotalTasks++
		summary.BudgetCentsTotal += task.BudgetCents
		summary.StateDistribution[string(task.State)]++
		if domain.IsActiveTaskState(task.State) {
			summary.ActiveRuns++
		}
		if task.State == domain.TaskBlocked || strings.EqualFold(strings.TrimSpace(task.Metadata["blocked"]), "true") {
			summary.Blockers++
		}
		if isHighRiskTask(task) {
			summary.SLARiskRuns++
		}
		policySummary := policy.Resolve(task)
		if policySummary.Plan == "premium" {
			summary.PremiumRuns++
		}
		funnel.Tickets++
		prStatus := strings.ToLower(strings.TrimSpace(task.Metadata["pr_status"]))
		merged := strings.EqualFold(strings.TrimSpace(task.Metadata["merged"]), "true") || prStatus == "merged"
		if prStatus != "" || merged {
			funnel.PROpened++
		}
		if merged {
			funnel.MergedPRs++
		}
		overview := s.dashboardTaskOverview(task, policySummary)
		overviews = append(overviews, overview)
		accumulateDashboardBreakdown(projectBreakdown, strings.TrimSpace(task.Metadata["project"]), task, policySummary)
		accumulateDashboardBreakdown(teamBreakdown, strings.TrimSpace(task.Metadata["team"]), task, policySummary)
		if task.State == domain.TaskBlocked || strings.EqualFold(strings.TrimSpace(task.Metadata["blocked"]), "true") {
			blockedTasks = append(blockedTasks, overview)
		}
		if isHighRiskTask(task) {
			highRiskTasks = append(highRiskTasks, overview)
		}
	}
	if limit > 0 && len(overviews) > limit {
		overviews = overviews[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authorization": authorization,
		"filters": map[string]any{
			"team":        team,
			"project":     project,
			"tenant_id":   tenantID,
			"viewer_team": authorization.ViewerTeam,
			"since":       since,
			"until":       until,
			"limit":       limit,
		},
		"summary":                summary,
		"ticket_to_merge_funnel": funnel,
		"project_breakdown":      sortedDashboardBreakdowns(projectBreakdown),
		"team_breakdown":         sortedDashboardBreakdowns(teamBreakdown),
		"blocked_tasks":          limitDashboardTasks(blockedTasks, limit),
		"high_risk_tasks":        limitDashboardTasks(highRiskTasks, limit),
		"tasks":                  overviews,
	})
}

func (s *Server) dashboardTaskOverview(task domain.Task, policySummary policy.Summary) dashboardTaskOverview {
	overview := dashboardTaskOverview{
		Task:   task,
		Policy: policySummary,
		Drilldown: dashboardDrilldown{
			Run:               fmt.Sprintf("/v2/runs/%s", task.ID),
			Events:            fmt.Sprintf("/events?task_id=%s&limit=%d", task.ID, 200),
			Replay:            fmt.Sprintf("/replay/%s", task.ID),
			IssueKey:          firstNonEmpty(task.Metadata["issue_key"], task.Metadata["issue_id"], task.Metadata["ticket_id"], task.Metadata["linear_issue"], task.Metadata["jira_issue"]),
			IssueURL:          firstNonEmpty(task.Metadata["issue_url"], task.Metadata["linear_url"], task.Metadata["jira_url"]),
			PullRequestURL:    firstNonEmpty(task.Metadata["pr_url"], task.Metadata["pull_request_url"]),
			PullRequestStatus: firstNonEmpty(task.Metadata["pr_status"], task.Metadata["pull_request_status"]),
			Workpad:           task.Metadata["workpad"],
		},
	}
	if latest, ok := s.Recorder.LatestByTask(task.ID); ok {
		copy := latest
		overview.Latest = &copy
	}
	if takeover, ok := s.Control.TakeoverStatus(task.ID); ok {
		copy := takeover
		overview.Takeover = &copy
	}
	return overview
}

func accumulateDashboardBreakdown(breakdowns map[string]*dashboardBreakdown, key string, task domain.Task, policySummary policy.Summary) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unassigned"
	}
	entry, ok := breakdowns[key]
	if !ok {
		entry = &dashboardBreakdown{Key: key}
		breakdowns[key] = entry
	}
	entry.TotalTasks++
	entry.BudgetCentsTotal += task.BudgetCents
	if domain.IsActiveTaskState(task.State) {
		entry.ActiveRuns++
	}
	if task.State == domain.TaskBlocked || strings.EqualFold(strings.TrimSpace(task.Metadata["blocked"]), "true") {
		entry.Blockers++
	}
	if isHighRiskTask(task) {
		entry.SLARiskRuns++
	}
	if policySummary.Plan == "premium" {
		entry.PremiumRuns++
	}
	prStatus := strings.ToLower(strings.TrimSpace(task.Metadata["pr_status"]))
	merged := strings.EqualFold(strings.TrimSpace(task.Metadata["merged"]), "true") || prStatus == "merged"
	if merged {
		entry.MergedPRs++
	}
}

func sortedDashboardBreakdowns(breakdowns map[string]*dashboardBreakdown) []dashboardBreakdown {
	out := make([]dashboardBreakdown, 0, len(breakdowns))
	for _, entry := range breakdowns {
		out = append(out, *entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Blockers == out[j].Blockers {
			if out[i].ActiveRuns == out[j].ActiveRuns {
				if out[i].BudgetCentsTotal == out[j].BudgetCentsTotal {
					return out[i].Key < out[j].Key
				}
				return out[i].BudgetCentsTotal > out[j].BudgetCentsTotal
			}
			return out[i].ActiveRuns > out[j].ActiveRuns
		}
		return out[i].Blockers > out[j].Blockers
	})
	return out
}

func limitDashboardTasks(tasks []dashboardTaskOverview, limit int) []dashboardTaskOverview {
	if limit <= 0 || len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *Server) handleV2ControlCenter(w http.ResponseWriter, r *http.Request) {
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
	deadLetters, err := s.Queue.ListDeadLetters(r.Context(), 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("list dead letters: %v", err), http.StatusInternalServerError)
		return
	}
	filteredDeadLetters := filterTasks(deadLetters, filters)

	recentTasks := filterTasks(s.Recorder.Tasks(0), filters)
	recentTasks = limitTasks(recentTasks, filters.Limit)
	overviews := make([]taskOverview, 0, len(recentTasks))
	for _, task := range recentTasks {
		overview := taskOverview{Task: task, Policy: policy.Resolve(task)}
		if latest, ok := s.Recorder.LatestByTask(task.ID); ok {
			copy := latest
			overview.Latest = &copy
		}
		if takeover, ok := s.Control.TakeoverStatus(task.ID); ok {
			copy := takeover
			overview.Takeover = &copy
		}
		overviews = append(overviews, overview)
	}

	queueTasks, err := s.filteredQueueTasks(r.Context(), filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("list queue tasks: %v", err), http.StatusInternalServerError)
		return
	}
	returnedQueueTasks := queueTasks
	if filters.Limit > 0 && len(returnedQueueTasks) > filters.Limit {
		returnedQueueTasks = returnedQueueTasks[:filters.Limit]
	}
	auditEntries := s.controlActionAuditEntries(filters.AuditLimit, filters.TaskID, filters.Action, filters.Actor, filters.Team, authorization)
	response := map[string]any{
		"authorization": authorization,
		"filters": map[string]any{
			"team":        filters.Team,
			"project":     filters.Project,
			"task_id":     filters.TaskID,
			"state":       filters.State,
			"risk_level":  filters.RiskLevel,
			"priority":    filters.Priority,
			"limit":       filters.Limit,
			"audit_limit": filters.AuditLimit,
		},
		"control":          s.Control.Snapshot(),
		"summary":          summarizeControlCenter(queueTasks, filteredDeadLetters),
		"queue":            map[string]any{"size": s.Queue.Size(context.Background()), "filtered_size": len(queueTasks), "dead_letters": len(filteredDeadLetters), "tasks": returnedQueueTasks, "cancellable": supportsQueueCancel(s.Queue)},
		"dead_letters":     limitTasks(filteredDeadLetters, filters.Limit),
		"active_takeovers": s.filteredActiveTakeovers(filters),
		"recent_tasks":     overviews,
		"audit":            auditEntries,
	}
	if pool := s.workerPoolSummary(); pool != nil {
		response["worker_pool"] = pool
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleV2ControlCenterAudit(w http.ResponseWriter, r *http.Request) {
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
	entries := s.controlActionAuditEntries(filters.AuditLimit, filters.TaskID, filters.Action, filters.Actor, filters.Team, authorization)
	writeJSON(w, http.StatusOK, map[string]any{
		"authorization": authorization,
		"filters": map[string]any{
			"task_id": filters.TaskID,
			"team":    filters.Team,
			"action":  filters.Action,
			"actor":   filters.Actor,
			"limit":   filters.AuditLimit,
		},
		"audit": entries,
	})
}

func (s *Server) handleV2ControlCenterAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request controlActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode action: %v", err), http.StatusBadRequest)
		return
	}
	now := s.Now()
	authorization := parseControlAuthorization(r, request.Actor, request.Role, request.ViewerTeam)
	if err := authorization.validateScope(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	actor := normalizedActor(authorization.Actor)
	action := strings.ToLower(strings.TrimSpace(request.Action))
	if !canPerformControlAction(authorization.Role, action) {
		http.Error(w, fmt.Sprintf("forbidden: role %s cannot perform %s", authorization.Role, normalizeActionName(action)), http.StatusForbidden)
		return
	}
	switch action {
	case "pause":
		snapshot := s.Control.Pause(actor, request.Reason, now)
		s.publish(domain.Event{ID: fmt.Sprintf("control-pause-%d", now.UnixNano()), Type: domain.EventControlPaused, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reason": request.Reason}})
		writeJSON(w, http.StatusOK, map[string]any{"action": "pause", "control": snapshot})
	case "resume":
		snapshot := s.Control.Resume(actor, now)
		s.publish(domain.Event{ID: fmt.Sprintf("control-resume-%d", now.UnixNano()), Type: domain.EventControlResumed, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role)}})
		writeJSON(w, http.StatusOK, map[string]any{"action": "resume", "control": snapshot})
	case "replay_deadletter", "retry":
		if request.TaskID == "" {
			http.Error(w, "missing task_id", http.StatusBadRequest)
			return
		}
		if err := s.authorizeTaskIDAccess(authorization, request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err := s.Queue.ReplayDeadLetter(r.Context(), request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		s.syncTaskState(request.TaskID, domain.TaskQueued, now)
		traceID := s.traceIDForTask(request.TaskID)
		s.publish(domain.Event{ID: fmt.Sprintf("%s-replayed-%d", request.TaskID, now.UnixNano()), Type: domain.EventTaskQueued, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "replayed": true}})
		task, _ := s.Recorder.Task(request.TaskID)
		writeJSON(w, http.StatusOK, map[string]any{"action": "replay_deadletter", "task": task, "replayed": true})
	case "cancel":
		if request.TaskID == "" {
			http.Error(w, "missing task_id", http.StatusBadRequest)
			return
		}
		if err := s.authorizeTaskIDAccess(authorization, request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		controller, ok := s.Queue.(queue.TaskController)
		if !ok {
			http.Error(w, "queue backend does not support cancel", http.StatusNotImplemented)
			return
		}
		snapshot, err := controller.CancelTask(r.Context(), request.TaskID, request.Reason)
		if err != nil {
			switch {
			case errors.Is(err, queue.ErrTaskNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				http.Error(w, err.Error(), http.StatusConflict)
			}
			return
		}
		s.syncTaskState(request.TaskID, domain.TaskCancelled, now)
		traceID := s.traceIDForTask(request.TaskID)
		s.publish(domain.Event{ID: fmt.Sprintf("%s-cancel-%d", request.TaskID, now.UnixNano()), Type: domain.EventTaskCancelled, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reason": request.Reason}})
		task, _ := s.Recorder.Task(request.TaskID)
		writeJSON(w, http.StatusOK, map[string]any{"action": "cancel", "task": task, "queue_task": snapshot, "cancelled": true})
	case "takeover", "transfer_to_human":
		if request.TaskID == "" {
			http.Error(w, "missing task_id", http.StatusBadRequest)
			return
		}
		if err := s.authorizeTaskIDAccess(authorization, request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		note := strings.TrimSpace(request.Note)
		if note == "" {
			note = strings.TrimSpace(request.Reason)
		}
		takeover := s.Control.Takeover(request.TaskID, actor, request.Reviewer, note, now)
		s.syncTaskState(request.TaskID, domain.TaskBlocked, now)
		traceID := s.traceIDForTask(request.TaskID)
		s.publish(domain.Event{ID: fmt.Sprintf("%s-takeover-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunTakeover, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reviewer": request.Reviewer, "note": note}})
		writeJSON(w, http.StatusOK, map[string]any{"action": "takeover", "takeover": takeover})
	case "release_takeover":
		if request.TaskID == "" {
			http.Error(w, "missing task_id", http.StatusBadRequest)
			return
		}
		if err := s.authorizeTaskIDAccess(authorization, request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		takeover, ok := s.Control.Release(request.TaskID, actor, request.Note, now)
		if !ok {
			http.Error(w, "takeover not found", http.StatusNotFound)
			return
		}
		s.syncTaskState(request.TaskID, domain.TaskQueued, now)
		traceID := s.traceIDForTask(request.TaskID)
		s.publish(domain.Event{ID: fmt.Sprintf("%s-release-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunReleased, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "note": request.Note}})
		writeJSON(w, http.StatusOK, map[string]any{"action": "release_takeover", "takeover": takeover})
	case "annotate":
		if request.TaskID == "" {
			http.Error(w, "missing task_id", http.StatusBadRequest)
			return
		}
		if err := s.authorizeTaskIDAccess(authorization, request.TaskID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		takeover := s.Control.Annotate(request.TaskID, actor, request.Note, now)
		traceID := s.traceIDForTask(request.TaskID)
		s.publish(domain.Event{ID: fmt.Sprintf("%s-annotate-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunAnnotated, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "note": request.Note}})
		writeJSON(w, http.StatusOK, map[string]any{"action": "annotate", "takeover": takeover})
	default:
		http.Error(w, "unsupported action", http.StatusBadRequest)
	}
}

func (s *Server) handleV2RunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, "/v2/runs/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 200
	}
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err := s.authorizeTaskAccess(authorization, task); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	events := s.Recorder.EventsByTask(taskID, limit)
	var traceSummary *observability.TraceSummary
	if task.TraceID != "" {
		if summary, ok := s.Recorder.TraceSummary(task.TraceID); ok {
			traceSummary = &summary
		}
	}
	response := runDetailResponse{
		Task:       task,
		State:      string(task.State),
		Policy:     policy.Resolve(task),
		Events:     events,
		Timeline:   events,
		Validation: map[string]any{"acceptance_criteria": task.AcceptanceCriteria, "validation_plan": task.ValidationPlan},
		Artifacts: map[string]string{
			"replay": fmt.Sprintf("/replay/%s", taskID),
			"events": fmt.Sprintf("/events?task_id=%s&limit=%d", taskID, limit),
		},
		Workpad: task.Metadata["workpad"],
		Trace:   traceSummary,
	}
	if takeover, ok := s.Control.TakeoverStatus(taskID); ok {
		copy := takeover
		response.Collaboration = &copy
	}
	writeJSON(w, http.StatusOK, response)
}

func enforceScopedTeamFilter(authorization ControlAuthorization, team *string) error {
	if !authorization.teamScoped() {
		return nil
	}
	if err := authorization.validateScope(); err != nil {
		return err
	}
	requested := strings.TrimSpace(*team)
	if requested == "" {
		*team = authorization.ViewerTeam
		return nil
	}
	if !authorization.permitsTeam(requested) {
		return fmt.Errorf("forbidden: role %s cannot access team %s", authorization.Role, requested)
	}
	*team = authorization.ViewerTeam
	return nil
}

func (s *Server) authorizeTaskAccess(authorization ControlAuthorization, task domain.Task) error {
	if !authorization.teamScoped() {
		return nil
	}
	if err := authorization.validateScope(); err != nil {
		return err
	}
	team := strings.TrimSpace(task.Metadata["team"])
	if !authorization.permitsTeam(team) {
		return fmt.Errorf("forbidden: role %s cannot access team %s", authorization.Role, team)
	}
	return nil
}

func (s *Server) authorizeTaskIDAccess(authorization ControlAuthorization, taskID string) error {
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		return nil
	}
	return s.authorizeTaskAccess(authorization, task)
}

func (s *Server) filteredTasks(team, project, tenantID string, since, until time.Time) []domain.Task {
	tasks := s.Recorder.Tasks(0)
	filtered := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if team != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["team"]), team) {
			continue
		}
		if project != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["project"]), project) {
			continue
		}
		if tenantID != "" && !strings.EqualFold(strings.TrimSpace(task.TenantID), tenantID) {
			continue
		}
		anchor := task.UpdatedAt
		if anchor.IsZero() {
			anchor = task.CreatedAt
		}
		if !since.IsZero() && anchor.Before(since) {
			continue
		}
		if !until.IsZero() && anchor.After(until) {
			continue
		}
		filtered = append(filtered, task)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].UpdatedAt.Equal(filtered[j].UpdatedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	return filtered
}

func (s *Server) taskSnapshot(taskID string) (domain.Task, bool) {
	task, ok := s.Recorder.Task(taskID)
	if ok {
		if task.TraceID == "" {
			task.TraceID = s.traceIDForTask(taskID)
		}
		if task.State == "" {
			if latest, found := s.Recorder.LatestByTask(taskID); found {
				if state, ok := domain.TaskStateFromEventType(latest.Type); ok {
					task.State = state
				}
			}
		}
		return task, true
	}
	if latest, ok := s.Recorder.LatestByTask(taskID); ok {
		task = domain.Task{ID: taskID, TraceID: latest.TraceID, UpdatedAt: latest.Timestamp}
		if state, ok := domain.TaskStateFromEventType(latest.Type); ok {
			task.State = state
		}
		return task, true
	}
	return domain.Task{}, false
}

func (s *Server) syncTaskState(taskID string, state domain.TaskState, updatedAt time.Time) {
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		task = domain.Task{ID: taskID, TraceID: s.traceIDForTask(taskID), CreatedAt: updatedAt}
	}
	task.State = state
	task.UpdatedAt = updatedAt
	if task.TraceID == "" {
		task.TraceID = taskID
	}
	s.Recorder.StoreTask(task)
}

func (s *Server) traceIDForTask(taskID string) string {
	if task, ok := s.Recorder.Task(taskID); ok && task.TraceID != "" {
		return task.TraceID
	}
	if latest, ok := s.Recorder.LatestByTask(taskID); ok && latest.TraceID != "" {
		return latest.TraceID
	}
	return taskID
}

func parseOptionalTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func normalizedActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}

func supportsQueueCancel(q queue.Queue) bool {
	_, ok := q.(queue.TaskController)
	return ok
}

func parseControlCenterFilters(r *http.Request) (controlCenterFilters, error) {
	priorityRaw := strings.TrimSpace(r.URL.Query().Get("priority"))
	var priority *int
	if priorityRaw != "" {
		value, err := strconv.Atoi(priorityRaw)
		if err != nil {
			return controlCenterFilters{}, fmt.Errorf("invalid priority value")
		}
		priority = &value
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	auditLimit, _ := strconv.Atoi(r.URL.Query().Get("audit_limit"))
	if auditLimit <= 0 {
		auditLimit = limit
	}
	return controlCenterFilters{
		Team:       strings.TrimSpace(r.URL.Query().Get("team")),
		Project:    strings.TrimSpace(r.URL.Query().Get("project")),
		TaskID:     strings.TrimSpace(r.URL.Query().Get("task_id")),
		State:      strings.ToLower(strings.TrimSpace(r.URL.Query().Get("state"))),
		RiskLevel:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("risk_level"))),
		Actor:      strings.TrimSpace(r.URL.Query().Get("actor")),
		Action:     normalizeActionName(r.URL.Query().Get("action")),
		Priority:   priority,
		Limit:      limit,
		AuditLimit: auditLimit,
	}, nil
}

func filterTasks(tasks []domain.Task, filters controlCenterFilters) []domain.Task {
	filtered := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if !matchesTaskFilters(task, task.State, filters) {
			continue
		}
		filtered = append(filtered, task)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].UpdatedAt.Equal(filtered[j].UpdatedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	return filtered
}

func limitTasks(tasks []domain.Task, limit int) []domain.Task {
	if limit <= 0 || len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

func (s *Server) filteredQueueTasks(ctx context.Context, filters controlCenterFilters) ([]queueTaskOverview, error) {
	inspector, ok := s.Queue.(queue.TaskInspector)
	if !ok {
		return nil, nil
	}
	snapshots, err := inspector.ListTasks(ctx, 0)
	if err != nil {
		return nil, err
	}
	out := make([]queueTaskOverview, 0, len(snapshots))
	for _, snapshot := range snapshots {
		var takeover *control.Takeover
		if current, ok := s.Control.TakeoverStatus(snapshot.Task.ID); ok {
			copy := current
			takeover = &copy
		}
		effective := effectiveTaskState(snapshot.Task.State, takeover)
		if !matchesTaskFilters(snapshot.Task, effective, filters) {
			continue
		}
		out = append(out, queueTaskOverview{QueueTask: snapshot, EffectiveState: effective, Policy: policy.Resolve(snapshot.Task), Takeover: takeover})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rankForControlState(out[i].EffectiveState) == rankForControlState(out[j].EffectiveState) {
			if out[i].QueueTask.Task.Priority == out[j].QueueTask.Task.Priority {
				return out[i].QueueTask.Task.UpdatedAt.After(out[j].QueueTask.Task.UpdatedAt)
			}
			return out[i].QueueTask.Task.Priority < out[j].QueueTask.Task.Priority
		}
		return rankForControlState(out[i].EffectiveState) < rankForControlState(out[j].EffectiveState)
	})
	return out, nil
}

func summarizeControlCenter(queueTasks []queueTaskOverview, deadLetters []domain.Task) controlCenterSummary {
	summary := controlCenterSummary{
		QueueDepth:           len(queueTasks),
		StateDistribution:    make(map[string]int),
		RiskDistribution:     make(map[string]int),
		PriorityDistribution: make(map[string]int),
		DeadLetters:          len(deadLetters),
	}
	for _, item := range queueTasks {
		task := item.QueueTask.Task
		summary.QueueBudgetCentsTotal += task.BudgetCents
		summary.StateDistribution[string(item.EffectiveState)]++
		risk := string(task.RiskLevel)
		if risk == "" {
			risk = "unspecified"
		}
		summary.RiskDistribution[risk]++
		summary.PriorityDistribution[fmt.Sprintf("p%d", task.Priority)]++
		if item.QueueTask.Leased {
			summary.LeasedRuns++
		}
		if item.EffectiveState == domain.TaskBlocked {
			summary.BlockedRuns++
		}
		if isHighRiskTask(task) {
			summary.HighRiskRuns++
		}
		if item.Policy.Plan == "premium" {
			summary.PremiumRuns++
		}
		if item.Takeover != nil && item.Takeover.Active {
			summary.ActiveTakeovers++
		}
	}
	return summary
}

func (s *Server) filteredActiveTakeovers(filters controlCenterFilters) []control.Takeover {
	takeovers := s.Control.ActiveTakeovers()
	if filters.Team == "" && filters.Project == "" && filters.TaskID == "" && filters.State == "" && filters.RiskLevel == "" && filters.Priority == nil {
		return takeovers
	}
	filtered := make([]control.Takeover, 0, len(takeovers))
	for _, takeover := range takeovers {
		task, ok := s.taskSnapshot(takeover.TaskID)
		if !ok {
			continue
		}
		if !matchesTaskFilters(task, effectiveTaskState(task.State, &takeover), filters) {
			continue
		}
		filtered = append(filtered, takeover)
	}
	return filtered
}

func (s *Server) controlActionAuditEntries(limit int, taskID string, action string, actor string, team string, authorization ControlAuthorization) []controlActionAuditEntry {
	logs := s.Recorder.Logs()
	out := make([]controlActionAuditEntry, 0)
	for index := len(logs) - 1; index >= 0; index-- {
		entry, ok := controlActionEntry(logs[index])
		if !ok {
			continue
		}
		if taskID != "" && entry.TaskID != taskID {
			continue
		}
		if action != "" && entry.Action != action {
			continue
		}
		if actor != "" && !strings.EqualFold(entry.Actor, actor) {
			continue
		}
		if team != "" || authorization.teamScoped() {
			if entry.TaskID == "" {
				continue
			}
			task, ok := s.taskSnapshot(entry.TaskID)
			if !ok {
				continue
			}
			if team != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["team"]), team) {
				continue
			}
			if err := s.authorizeTaskAccess(authorization, task); err != nil {
				continue
			}
		}
		out = append(out, entry)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func controlActionEntry(event domain.Event) (controlActionAuditEntry, bool) {
	action, ok := controlActionName(event)
	if !ok {
		return controlActionAuditEntry{}, false
	}
	actor, _ := event.Payload["actor"].(string)
	role, _ := event.Payload["role"].(string)
	reason, _ := event.Payload["reason"].(string)
	note, _ := event.Payload["note"].(string)
	if note == "" {
		note, _ = event.Payload["message"].(string)
	}
	return controlActionAuditEntry{
		Action:    action,
		Actor:     actor,
		Role:      role,
		TaskID:    event.TaskID,
		Timestamp: event.Timestamp,
		Reason:    reason,
		Note:      note,
		Event:     event,
	}, true
}

func controlActionName(event domain.Event) (string, bool) {
	switch event.Type {
	case domain.EventControlPaused:
		return "pause", true
	case domain.EventControlResumed:
		return "resume", true
	case domain.EventRunTakeover:
		return "takeover", true
	case domain.EventRunReleased:
		return "release_takeover", true
	case domain.EventRunAnnotated:
		return "annotate", true
	case domain.EventTaskCancelled:
		if _, ok := event.Payload["actor"]; ok {
			return "cancel", true
		}
		if _, ok := event.Payload["reason"]; ok {
			return "cancel", true
		}
		return "", false
	case domain.EventTaskQueued:
		if replayed, ok := event.Payload["replayed"].(bool); ok && replayed {
			return "retry", true
		}
		return "", false
	default:
		return "", false
	}
}

func matchesTaskFilters(task domain.Task, effectiveState domain.TaskState, filters controlCenterFilters) bool {
	if filters.TaskID != "" && task.ID != filters.TaskID {
		return false
	}
	if filters.Team != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["team"]), filters.Team) {
		return false
	}
	if filters.Project != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["project"]), filters.Project) {
		return false
	}
	if filters.State != "" && !strings.EqualFold(string(effectiveState), filters.State) {
		return false
	}
	if filters.RiskLevel != "" && !strings.EqualFold(string(task.RiskLevel), filters.RiskLevel) {
		return false
	}
	if filters.Priority != nil && task.Priority != *filters.Priority {
		return false
	}
	return true
}

func effectiveTaskState(current domain.TaskState, takeover *control.Takeover) domain.TaskState {
	if takeover != nil && takeover.Active {
		return domain.TaskBlocked
	}
	return current
}

func rankForControlState(state domain.TaskState) int {
	switch state {
	case domain.TaskBlocked:
		return 0
	case domain.TaskLeased:
		return 1
	case domain.TaskRunning:
		return 2
	case domain.TaskQueued:
		return 3
	case domain.TaskRetrying:
		return 4
	case domain.TaskCancelled:
		return 5
	case domain.TaskDeadLetter:
		return 6
	default:
		return 7
	}
}

func isHighRiskTask(task domain.Task) bool {
	return task.RiskLevel == domain.RiskHigh || strings.EqualFold(strings.TrimSpace(task.Metadata["sla_risk"]), "true") || strings.EqualFold(strings.TrimSpace(task.Metadata["sla_risk"]), "high")
}

func (s *Server) workerPoolSummary() *workerPoolSummary {
	if s.Worker == nil {
		return nil
	}
	snapshot := s.Worker.Snapshot()
	active := 0
	if snapshot.State == "leased" || snapshot.State == "running" {
		active = 1
	}
	idle := 1
	if active == 1 {
		idle = 0
	}
	return &workerPoolSummary{
		TotalWorkers:  1,
		ActiveWorkers: active,
		IdleWorkers:   idle,
		Workers:       []worker.Status{snapshot},
	}
}
