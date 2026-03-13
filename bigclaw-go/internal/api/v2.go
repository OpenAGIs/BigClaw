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
	"bigclaw-go/internal/risk"
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

type dashboardTrendPoint struct {
	Start            time.Time `json:"start"`
	End              time.Time `json:"end"`
	Label            string    `json:"label"`
	TotalTasks       int       `json:"total_tasks"`
	ActiveRuns       int       `json:"active_runs"`
	Blockers         int       `json:"blockers"`
	PremiumRuns      int       `json:"premium_runs"`
	SLARiskRuns      int       `json:"sla_risk_runs"`
	BudgetCentsTotal int64     `json:"budget_cents_total"`
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
	Risk      risk.Score         `json:"risk_score"`
	Takeover  *control.Takeover  `json:"takeover,omitempty"`
	Latest    *domain.Event      `json:"latest_event,omitempty"`
	Drilldown dashboardDrilldown `json:"drilldown"`
}

type operationsSummary struct {
	TotalRuns         int            `json:"total_runs"`
	ActiveRuns        int            `json:"active_runs"`
	BlockedRuns       int            `json:"blocked_runs"`
	SLARiskRuns       int            `json:"sla_risk_runs"`
	OverdueRuns       int            `json:"overdue_runs"`
	BudgetCentsTotal  int64          `json:"budget_cents_total"`
	StateDistribution map[string]int `json:"state_distribution"`
	RiskDistribution  map[string]int `json:"risk_distribution"`
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
	Team      string       `json:"team,omitempty"`
	Project   string       `json:"project,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Reason    string       `json:"reason,omitempty"`
	Note      string       `json:"note,omitempty"`
	Event     domain.Event `json:"event"`
}

type auditFacetCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type controlAuditSummary struct {
	Total      int               `json:"total"`
	ByAction   []auditFacetCount `json:"by_action"`
	ByActor    []auditFacetCount `json:"by_actor"`
	ByRole     []auditFacetCount `json:"by_role"`
	ByTeam     []auditFacetCount `json:"by_team"`
	ByProject  []auditFacetCount `json:"by_project"`
	NotesCount int               `json:"notes_count"`
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
	Task          domain.Task               `json:"task"`
	Policy        policy.Summary            `json:"policy"`
	Risk          risk.Score                `json:"risk_score"`
	Takeover      *control.Takeover         `json:"takeover,omitempty"`
	Latest        *domain.Event             `json:"latest_event,omitempty"`
	RecentActions []controlActionAuditEntry `json:"recent_actions,omitempty"`
}

type queueTaskOverview struct {
	QueueTask      queue.TaskSnapshot        `json:"queue_task"`
	EffectiveState domain.TaskState          `json:"effective_state"`
	Policy         policy.Summary            `json:"policy"`
	Risk           risk.Score                `json:"risk_score"`
	Takeover       *control.Takeover         `json:"takeover,omitempty"`
	Drilldown      dashboardDrilldown        `json:"drilldown"`
	RecentActions  []controlActionAuditEntry `json:"recent_actions,omitempty"`
}

type runValidationSummary struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
	Status             string   `json:"status"`
	Checks             int      `json:"checks"`
}

type runArtifactRef struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	URI     string `json:"uri"`
	Source  string `json:"source"`
	EventID string `json:"event_id,omitempty"`
}

type runToolTrace struct {
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Status    string    `json:"status,omitempty"`
	Executor  string    `json:"executor,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	EventID   string    `json:"event_id,omitempty"`
	Artifacts []string  `json:"artifacts,omitempty"`
}

type runReportLink struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Format   string `json:"format"`
	Download bool   `json:"download"`
}

type runDetailResponse struct {
	Task          domain.Task                 `json:"task"`
	State         string                      `json:"state"`
	Policy        policy.Summary              `json:"policy"`
	Risk          risk.Score                  `json:"risk_score"`
	Collaboration *control.Takeover           `json:"collaboration,omitempty"`
	Trace         *observability.TraceSummary `json:"trace,omitempty"`
	FailureReason string                      `json:"failure_reason,omitempty"`
	Events        []domain.Event              `json:"events"`
	Timeline      []domain.Event              `json:"timeline"`
	Validation    runValidationSummary        `json:"validation"`
	Artifacts     map[string]string           `json:"artifacts"`
	ArtifactRefs  []runArtifactRef            `json:"artifact_refs,omitempty"`
	ToolTraces    []runToolTrace              `json:"tool_traces,omitempty"`
	AuditSummary  controlAuditSummary         `json:"audit_summary"`
	RecentActions []controlActionAuditEntry   `json:"recent_actions,omitempty"`
	NotesTimeline []controlActionAuditEntry   `json:"notes_timeline,omitempty"`
	Reports       []runReportLink             `json:"reports,omitempty"`
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
	bucket, err := parseDashboardBucket(r.URL.Query().Get("bucket"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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
	bucket = normalizeDashboardBucket(bucket, since, until, tasks)
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
			"bucket":      bucket,
		},
		"summary":                summary,
		"ticket_to_merge_funnel": funnel,
		"project_breakdown":      sortedDashboardBreakdowns(projectBreakdown),
		"team_breakdown":         sortedDashboardBreakdowns(teamBreakdown),
		"trend":                  buildDashboardTrend(tasks, since, until, bucket),
		"blocked_tasks":          limitDashboardTasks(blockedTasks, limit),
		"high_risk_tasks":        limitDashboardTasks(highRiskTasks, limit),
		"tasks":                  overviews,
	})
}

func (s *Server) handleV2OperationsDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	bucket, err := parseDashboardBucket(r.URL.Query().Get("bucket"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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
	bucket = normalizeDashboardBucket(bucket, since, until, tasks)
	summary := operationsSummary{StateDistribution: make(map[string]int), RiskDistribution: make(map[string]int)}
	projectBreakdown := make(map[string]*dashboardBreakdown)
	teamBreakdown := make(map[string]*dashboardBreakdown)
	overviews := make([]dashboardTaskOverview, 0, len(tasks))
	slaRiskTasks := make([]dashboardTaskOverview, 0)
	overdueTasks := make([]dashboardTaskOverview, 0)
	blockedTasks := make([]dashboardTaskOverview, 0)
	now := s.Now()
	for _, task := range tasks {
		policySummary := policy.Resolve(task)
		overview := s.dashboardTaskOverview(task, policySummary)
		overviews = append(overviews, overview)
		summary.TotalRuns++
		summary.BudgetCentsTotal += task.BudgetCents
		summary.StateDistribution[string(task.State)]++
		risk := string(task.RiskLevel)
		if risk == "" {
			risk = "unspecified"
		}
		summary.RiskDistribution[risk]++
		if domain.IsActiveTaskState(task.State) {
			summary.ActiveRuns++
		}
		if task.State == domain.TaskBlocked || strings.EqualFold(strings.TrimSpace(task.Metadata["blocked"]), "true") {
			summary.BlockedRuns++
			blockedTasks = append(blockedTasks, overview)
		}
		if isHighRiskTask(task) {
			summary.SLARiskRuns++
			slaRiskTasks = append(slaRiskTasks, overview)
		}
		if isOverdueTask(task, now) {
			summary.OverdueRuns++
			overdueTasks = append(overdueTasks, overview)
		}
		accumulateDashboardBreakdown(projectBreakdown, strings.TrimSpace(task.Metadata["project"]), task, policySummary)
		accumulateDashboardBreakdown(teamBreakdown, strings.TrimSpace(task.Metadata["team"]), task, policySummary)
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
			"bucket":      bucket,
		},
		"summary":           summary,
		"project_breakdown": sortedDashboardBreakdowns(projectBreakdown),
		"team_breakdown":    sortedDashboardBreakdowns(teamBreakdown),
		"trend":             buildDashboardTrend(tasks, since, until, bucket),
		"sla_risk_tasks":    limitDashboardTasks(slaRiskTasks, limit),
		"overdue_tasks":     limitDashboardTasks(overdueTasks, limit),
		"blocked_tasks":     limitDashboardTasks(blockedTasks, limit),
		"tasks":             overviews,
	})
}

func isOverdueTask(task domain.Task, now time.Time) bool {
	if strings.EqualFold(strings.TrimSpace(task.Metadata["sla_breach"]), "true") {
		return true
	}
	status := strings.ToLower(strings.TrimSpace(task.Metadata["sla_status"]))
	if status == "breach" || status == "overdue" {
		return true
	}
	dueAtRaw := firstNonEmpty(task.Metadata["sla_due_at"], task.Metadata["due_at"], task.Metadata["deadline_at"])
	if dueAtRaw == "" {
		return false
	}
	dueAt, err := time.Parse(time.RFC3339, dueAtRaw)
	if err != nil {
		return false
	}
	if task.State == domain.TaskSucceeded || task.State == domain.TaskCancelled {
		return false
	}
	return dueAt.Before(now)
}

func parseDashboardBucket(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "", "auto", "hour", "day":
		return raw, nil
	default:
		return "", fmt.Errorf("invalid bucket value")
	}
}

func normalizeDashboardBucket(bucket string, since, until time.Time, tasks []domain.Task) string {
	if bucket != "" && bucket != "auto" {
		return bucket
	}
	windowStart, windowEnd := dashboardWindowBounds(tasks, since, until)
	if !windowStart.IsZero() && !windowEnd.IsZero() && windowEnd.Sub(windowStart) <= 72*time.Hour {
		return "hour"
	}
	return "day"
}

func dashboardWindowBounds(tasks []domain.Task, since, until time.Time) (time.Time, time.Time) {
	if !since.IsZero() && !until.IsZero() {
		return since, until
	}
	var minAnchor time.Time
	var maxAnchor time.Time
	for _, task := range tasks {
		anchor := taskAnchorTime(task)
		if anchor.IsZero() {
			continue
		}
		if minAnchor.IsZero() || anchor.Before(minAnchor) {
			minAnchor = anchor
		}
		if maxAnchor.IsZero() || anchor.After(maxAnchor) {
			maxAnchor = anchor
		}
	}
	if since.IsZero() {
		since = minAnchor
	}
	if until.IsZero() {
		until = maxAnchor
	}
	return since, until
}

func buildDashboardTrend(tasks []domain.Task, since, until time.Time, bucket string) []dashboardTrendPoint {
	windowStart, windowEnd := dashboardWindowBounds(tasks, since, until)
	if windowStart.IsZero() || windowEnd.IsZero() {
		return []dashboardTrendPoint{}
	}
	step := 24 * time.Hour
	if bucket == "hour" {
		step = time.Hour
	}
	windowStart = truncateDashboardTime(windowStart, bucket)
	windowEnd = truncateDashboardTime(windowEnd, bucket)
	if !windowEnd.After(windowStart) {
		windowEnd = windowStart.Add(step)
	} else {
		windowEnd = windowEnd.Add(step)
	}
	points := make([]dashboardTrendPoint, 0)
	index := make(map[time.Time]int)
	for cursor := windowStart; cursor.Before(windowEnd); cursor = cursor.Add(step) {
		point := dashboardTrendPoint{
			Start: cursor,
			End:   cursor.Add(step),
			Label: formatDashboardTrendLabel(cursor, bucket),
		}
		index[cursor] = len(points)
		points = append(points, point)
	}
	for _, task := range tasks {
		anchor := truncateDashboardTime(taskAnchorTime(task), bucket)
		position, ok := index[anchor]
		if !ok {
			continue
		}
		point := &points[position]
		point.TotalTasks++
		point.BudgetCentsTotal += task.BudgetCents
		if domain.IsActiveTaskState(task.State) {
			point.ActiveRuns++
		}
		if task.State == domain.TaskBlocked || strings.EqualFold(strings.TrimSpace(task.Metadata["blocked"]), "true") {
			point.Blockers++
		}
		if isHighRiskTask(task) {
			point.SLARiskRuns++
		}
		if policy.Resolve(task).Plan == "premium" {
			point.PremiumRuns++
		}
	}
	return points
}

func truncateDashboardTime(value time.Time, bucket string) time.Time {
	if value.IsZero() {
		return value
	}
	if bucket == "hour" {
		return value.UTC().Truncate(time.Hour)
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func formatDashboardTrendLabel(value time.Time, bucket string) string {
	if bucket == "hour" {
		return value.UTC().Format(time.RFC3339)
	}
	return value.UTC().Format("2006-01-02")
}

func (s *Server) dashboardTaskOverview(task domain.Task, policySummary policy.Summary) dashboardTaskOverview {
	overview := dashboardTaskOverview{
		Task:      task,
		Policy:    policySummary,
		Risk:      s.riskScore(task),
		Drilldown: drilldownForTask(task),
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

func drilldownForTask(task domain.Task) dashboardDrilldown {
	return dashboardDrilldown{
		Run:               fmt.Sprintf("/v2/runs/%s", task.ID),
		Events:            fmt.Sprintf("/events?task_id=%s&limit=%d", task.ID, 200),
		Replay:            fmt.Sprintf("/replay/%s", task.ID),
		IssueKey:          firstNonEmpty(task.Metadata["issue_key"], task.Metadata["issue_id"], task.Metadata["ticket_id"], task.Metadata["linear_issue"], task.Metadata["jira_issue"]),
		IssueURL:          firstNonEmpty(task.Metadata["issue_url"], task.Metadata["linear_url"], task.Metadata["jira_url"]),
		PullRequestURL:    firstNonEmpty(task.Metadata["pr_url"], task.Metadata["pull_request_url"]),
		PullRequestStatus: firstNonEmpty(task.Metadata["pr_status"], task.Metadata["pull_request_status"]),
		Workpad:           task.Metadata["workpad"],
	}
}

func accumulateQueueBreakdown(breakdowns map[string]*dashboardBreakdown, item queueTaskOverview) {
	accumulateDashboardBreakdown(breakdowns, item.QueueTask.Task.Metadata["project"], item.QueueTask.Task, item.Policy)
}

func accumulateQueueBreakdownByTeam(breakdowns map[string]*dashboardBreakdown, item queueTaskOverview) {
	accumulateDashboardBreakdown(breakdowns, item.QueueTask.Task.Metadata["team"], item.QueueTask.Task, item.Policy)
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
		overview := taskOverview{Task: task, Policy: policy.Resolve(task), Risk: s.riskScore(task), RecentActions: s.recentControlActionsForTask(task.ID, 3, authorization)}
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
	queueByProject := make(map[string]*dashboardBreakdown)
	queueByTeam := make(map[string]*dashboardBreakdown)
	for _, item := range queueTasks {
		accumulateQueueBreakdown(queueByProject, item)
		accumulateQueueBreakdownByTeam(queueByTeam, item)
	}
	auditEntries := s.controlActionAuditEntries(filters.AuditLimit, filters.TaskID, filters.Action, filters.Actor, filters.Team, authorization)
	auditSummary := summarizeControlAudit(auditEntries)
	notesTimeline := auditNotesTimeline(auditEntries, filters.AuditLimit)
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
		"queue_by_project": sortedDashboardBreakdowns(queueByProject),
		"queue_by_team":    sortedDashboardBreakdowns(queueByTeam),
		"dead_letters":     limitTasks(filteredDeadLetters, filters.Limit),
		"active_takeovers": s.filteredActiveTakeovers(filters),
		"recent_tasks":     overviews,
		"audit":            auditEntries,
		"audit_summary":    auditSummary,
		"notes_timeline":   notesTimeline,
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
		"audit":          entries,
		"audit_summary":  summarizeControlAudit(entries),
		"notes_timeline": auditNotesTimeline(entries, filters.AuditLimit),
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
		s.publish(domain.Event{ID: fmt.Sprintf("control-pause-%d", now.UnixNano()), Type: domain.EventControlPaused, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reason": request.Reason, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
		s.publish(domain.Event{ID: fmt.Sprintf("%s-replayed-%d", request.TaskID, now.UnixNano()), Type: domain.EventTaskQueued, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "replayed": true, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
		s.publish(domain.Event{ID: fmt.Sprintf("%s-cancel-%d", request.TaskID, now.UnixNano()), Type: domain.EventTaskCancelled, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reason": request.Reason, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
		s.publish(domain.Event{ID: fmt.Sprintf("%s-takeover-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunTakeover, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "reviewer": request.Reviewer, "note": note, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
		s.publish(domain.Event{ID: fmt.Sprintf("%s-release-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunReleased, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "note": request.Note, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
		s.publish(domain.Event{ID: fmt.Sprintf("%s-annotate-%d", request.TaskID, now.UnixNano()), Type: domain.EventRunAnnotated, TaskID: request.TaskID, TraceID: traceID, Timestamp: now, Payload: map[string]any{"actor": actor, "role": string(authorization.Role), "note": request.Note, "team": s.taskTeam(request.TaskID), "project": s.taskProject(request.TaskID)}})
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
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v2/runs/"), "/")
	if path == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	switch {
	case strings.HasSuffix(path, "/audit"):
		s.handleV2RunAudit(w, r, strings.TrimSuffix(path, "/audit"))
		return
	case strings.HasSuffix(path, "/report"):
		s.handleV2RunReport(w, r, strings.TrimSuffix(path, "/report"))
		return
	}
	taskID := path
	authorization := parseControlAuthorization(r, "", "", "")
	limit := parseRunDetailLimit(r.URL.Query().Get("limit"))
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err := s.authorizeTaskAccess(authorization, task); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, s.buildRunDetailResponse(task, limit, authorization))
}

func parseRunDetailLimit(raw string) int {
	limit, _ := strconv.Atoi(raw)
	if limit <= 0 {
		return 200
	}
	return limit
}

func (s *Server) handleV2RunAudit(w http.ResponseWriter, r *http.Request, taskID string) {
	taskID = strings.Trim(taskID, "/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	limit := parseRunDetailLimit(r.URL.Query().Get("limit"))
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err := s.authorizeTaskAccess(authorization, task); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	entries := s.recentControlActionsForTask(taskID, limit, authorization)
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":        taskID,
		"recent_actions": entries,
		"audit_summary":  summarizeControlAudit(entries),
		"notes_timeline": auditNotesTimeline(entries, limit),
	})
}

func (s *Server) handleV2RunReport(w http.ResponseWriter, r *http.Request, taskID string) {
	taskID = strings.Trim(taskID, "/")
	if taskID == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	limit := parseRunDetailLimit(r.URL.Query().Get("limit"))
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if err := s.authorizeTaskAccess(authorization, task); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	response := s.buildRunDetailResponse(task, limit, authorization)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", taskID+"-run-report.md"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderRunDetailMarkdown(response)))
}

func (s *Server) buildRunDetailResponse(task domain.Task, limit int, authorization ControlAuthorization) runDetailResponse {
	events := s.Recorder.EventsByTask(task.ID, limit)
	var traceSummary *observability.TraceSummary
	if task.TraceID != "" {
		if summary, ok := s.Recorder.TraceSummary(task.TraceID); ok {
			traceSummary = &summary
		}
	}
	auditEntries := s.recentControlActionsForTask(task.ID, limit, authorization)
	artifacts := map[string]string{
		"replay": fmt.Sprintf("/replay/%s", task.ID),
		"events": fmt.Sprintf("/events?task_id=%s&limit=%d", task.ID, limit),
		"audit":  fmt.Sprintf("/v2/runs/%s/audit?limit=%d", task.ID, limit),
		"report": fmt.Sprintf("/v2/runs/%s/report?limit=%d", task.ID, limit),
	}
	if task.TraceID != "" {
		artifacts["trace"] = fmt.Sprintf("/debug/traces/%s?limit=%d", task.TraceID, limit)
	}
	if workpad := strings.TrimSpace(task.Metadata["workpad"]); workpad != "" {
		artifacts["workpad"] = workpad
	}
	response := runDetailResponse{
		Task:          task,
		State:         string(task.State),
		Policy:        policy.Resolve(task),
		Risk:          s.riskScore(task),
		Trace:         traceSummary,
		FailureReason: runFailureReason(task, events),
		Events:        events,
		Timeline:      events,
		Validation:    buildRunValidation(task),
		Artifacts:     artifacts,
		ArtifactRefs:  collectRunArtifactRefs(task, events),
		ToolTraces:    collectRunToolTraces(task, events),
		AuditSummary:  summarizeControlAudit(auditEntries),
		RecentActions: auditEntries,
		NotesTimeline: auditNotesTimeline(auditEntries, limit),
		Reports: []runReportLink{{
			Name:     "run_report",
			URL:      fmt.Sprintf("/v2/runs/%s/report?limit=%d", task.ID, limit),
			Format:   "markdown",
			Download: true,
		}},
		Workpad: task.Metadata["workpad"],
	}
	if s.Control != nil {
		if takeover, ok := s.Control.TakeoverStatus(task.ID); ok {
			copy := takeover
			response.Collaboration = &copy
		}
	}
	return response
}

func buildRunValidation(task domain.Task) runValidationSummary {
	return runValidationSummary{
		AcceptanceCriteria: append([]string(nil), task.AcceptanceCriteria...),
		ValidationPlan:     append([]string(nil), task.ValidationPlan...),
		Status:             runValidationStatus(task.State),
		Checks:             len(task.AcceptanceCriteria) + len(task.ValidationPlan),
	}
}

func runValidationStatus(state domain.TaskState) string {
	switch state {
	case domain.TaskSucceeded:
		return "passed"
	case domain.TaskCancelled, domain.TaskFailed, domain.TaskDeadLetter:
		return "failed"
	case domain.TaskBlocked:
		return "blocked"
	case domain.TaskQueued, domain.TaskLeased, domain.TaskRunning, domain.TaskRetrying:
		return "pending"
	default:
		return "unknown"
	}
}

func runFailureReason(task domain.Task, events []domain.Event) string {
	if task.State == domain.TaskSucceeded {
		return ""
	}
	for index := len(events) - 1; index >= 0; index-- {
		switch events[index].Type {
		case domain.EventTaskCancelled, domain.EventTaskDeadLetter, domain.EventTaskRetried:
			if message := runEventMessage(events[index]); message != "" {
				return message
			}
		}
	}
	return firstNonEmpty(task.Metadata["failure_reason"], task.Metadata["blocked_reason"], task.Metadata["cancel_reason"])
}

func collectRunArtifactRefs(task domain.Task, events []domain.Event) []runArtifactRef {
	refs := make([]runArtifactRef, 0)
	seen := make(map[string]struct{})
	appendRef := func(name, kind, uri, source, eventID string) {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			return
		}
		key := kind + "|" + uri
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, runArtifactRef{Name: name, Kind: kind, URI: uri, Source: source, EventID: eventID})
	}
	for _, event := range events {
		for index, uri := range stringSliceFromAny(event.Payload["artifacts"]) {
			appendRef(fmt.Sprintf("artifact_%d", index+1), "executor_artifact", uri, string(event.Type), event.ID)
		}
	}
	appendRef("workpad", "workpad", task.Metadata["workpad"], "task_metadata", "")
	appendRef("issue", "issue", firstNonEmpty(task.Metadata["issue_url"], task.Metadata["linear_url"], task.Metadata["jira_url"]), "task_metadata", "")
	appendRef("pull_request", "pull_request", firstNonEmpty(task.Metadata["pr_url"], task.Metadata["pull_request_url"]), "task_metadata", "")
	return refs
}

func collectRunToolTraces(task domain.Task, events []domain.Event) []runToolTrace {
	traces := make([]runToolTrace, 0, len(task.RequiredTools)+len(events))
	seenDeclared := make(map[string]struct{})
	for _, tool := range task.RequiredTools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		if _, ok := seenDeclared[tool]; ok {
			continue
		}
		seenDeclared[tool] = struct{}{}
		traces = append(traces, runToolTrace{Name: tool, Source: "declared", Status: "required"})
	}
	for _, event := range events {
		status, ok := runToolTraceStatus(event.Type)
		if !ok {
			continue
		}
		executorName := eventStringValue(event.Payload, "executor")
		name := firstNonEmpty(executorName, "executor")
		if event.Type == domain.EventSchedulerRouted {
			name = "scheduler"
		}
		traces = append(traces, runToolTrace{
			Name:      name,
			Source:    "event",
			Status:    status,
			Executor:  executorName,
			Message:   runEventMessage(event),
			Timestamp: event.Timestamp,
			EventID:   event.ID,
			Artifacts: stringSliceFromAny(event.Payload["artifacts"]),
		})
	}
	return traces
}

func runToolTraceStatus(eventType domain.EventType) (string, bool) {
	switch eventType {
	case domain.EventSchedulerRouted:
		return "routed", true
	case domain.EventTaskStarted:
		return "started", true
	case domain.EventTaskCompleted:
		return "completed", true
	case domain.EventTaskDeadLetter:
		return "dead_lettered", true
	case domain.EventTaskRetried:
		return "retried", true
	case domain.EventTaskCancelled:
		return "cancelled", true
	default:
		return "", false
	}
}

func runEventMessage(event domain.Event) string {
	return firstNonEmpty(
		eventStringValue(event.Payload, "message"),
		eventStringValue(event.Payload, "reason"),
		eventStringValue(event.Payload, "note"),
	)
}

func eventStringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	return stringValue(payload[key])
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case domain.ExecutorKind:
		return strings.TrimSpace(string(typed))
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		if text := stringValue(typed); text != "" {
			return []string{text}
		}
		return nil
	}
}

func renderRunDetailMarkdown(detail runDetailResponse) string {
	var builder strings.Builder
	builder.WriteString("# BigClaw Run Report\n\n")
	fmt.Fprintf(&builder, "- Task ID: %s\n", detail.Task.ID)
	fmt.Fprintf(&builder, "- Title: %s\n", firstNonEmpty(detail.Task.Title, detail.Task.ID))
	fmt.Fprintf(&builder, "- State: %s\n", detail.State)
	fmt.Fprintf(&builder, "- Trace ID: %s\n", detail.Task.TraceID)
	fmt.Fprintf(&builder, "- Plan: %s\n", detail.Policy.Plan)
	if detail.FailureReason != "" {
		fmt.Fprintf(&builder, "- Failure Reason: %s\n", detail.FailureReason)
	}
	if detail.Workpad != "" {
		fmt.Fprintf(&builder, "- Workpad: %s\n", detail.Workpad)
	}
	builder.WriteString("\n## Validation\n\n")
	fmt.Fprintf(&builder, "- Status: %s\n", detail.Validation.Status)
	fmt.Fprintf(&builder, "- Checks: %d\n", detail.Validation.Checks)
	for _, item := range detail.Validation.AcceptanceCriteria {
		fmt.Fprintf(&builder, "- Acceptance: %s\n", item)
	}
	for _, item := range detail.Validation.ValidationPlan {
		fmt.Fprintf(&builder, "- Validation Step: %s\n", item)
	}
	builder.WriteString("\n## Tool Trace\n\n")
	if len(detail.ToolTraces) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, trace := range detail.ToolTraces {
			line := fmt.Sprintf("- %s [%s]", trace.Name, firstNonEmpty(trace.Status, "observed"))
			if trace.Executor != "" && trace.Name != trace.Executor {
				line += fmt.Sprintf(" via %s", trace.Executor)
			}
			if trace.Message != "" {
				line += fmt.Sprintf(": %s", trace.Message)
			}
			if len(trace.Artifacts) > 0 {
				line += fmt.Sprintf(" (artifacts=%d)", len(trace.Artifacts))
			}
			builder.WriteString(line + "\n")
		}
	}
	builder.WriteString("\n## Artifacts\n\n")
	if len(detail.ArtifactRefs) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range detail.ArtifactRefs {
			fmt.Fprintf(&builder, "- %s (%s): %s\n", item.Name, item.Kind, item.URI)
		}
	}
	builder.WriteString("\n## Audit\n\n")
	if len(detail.RecentActions) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range detail.RecentActions {
			line := fmt.Sprintf("- %s by %s", item.Action, firstNonEmpty(item.Actor, "system"))
			if item.Note != "" {
				line += fmt.Sprintf(": %s", item.Note)
			} else if item.Reason != "" {
				line += fmt.Sprintf(": %s", item.Reason)
			}
			builder.WriteString(line + "\n")
		}
	}
	builder.WriteString("\n## Timeline\n\n")
	if len(detail.Timeline) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, event := range detail.Timeline {
			line := fmt.Sprintf("- %s %s", event.Timestamp.UTC().Format(time.RFC3339), event.Type)
			if message := runEventMessage(event); message != "" {
				line += fmt.Sprintf(": %s", message)
			}
			builder.WriteString(line + "\n")
		}
	}
	return builder.String()
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
		anchor := taskAnchorTime(task)
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

func taskAnchorTime(task domain.Task) time.Time {
	anchor := task.UpdatedAt
	if anchor.IsZero() {
		anchor = task.CreatedAt
	}
	return anchor
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

func (s *Server) taskTeam(taskID string) string {
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		return ""
	}
	return strings.TrimSpace(task.Metadata["team"])
}

func (s *Server) taskProject(taskID string) string {
	task, ok := s.taskSnapshot(taskID)
	if !ok {
		return ""
	}
	return strings.TrimSpace(task.Metadata["project"])
}

func (s *Server) riskScore(task domain.Task) risk.Score {
	return risk.ScoreTask(task, s.Recorder.EventsByTask(task.ID, 0))
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
		out = append(out, queueTaskOverview{QueueTask: snapshot, EffectiveState: effective, Policy: policy.Resolve(snapshot.Task), Risk: s.riskScore(snapshot.Task), Takeover: takeover, Drilldown: drilldownForTask(snapshot.Task), RecentActions: s.recentControlActionsForTask(snapshot.Task.ID, 3, ControlAuthorization{})})
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
	team, _ := event.Payload["team"].(string)
	project, _ := event.Payload["project"].(string)
	return controlActionAuditEntry{
		Action:    action,
		Actor:     actor,
		Role:      role,
		TaskID:    event.TaskID,
		Team:      team,
		Project:   project,
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

func (s *Server) recentControlActionsForTask(taskID string, limit int, authorization ControlAuthorization) []controlActionAuditEntry {
	if limit <= 0 {
		return nil
	}
	entries := s.controlActionAuditEntries(limit, taskID, "", "", "", authorization)
	return entries
}

func summarizeControlAudit(entries []controlActionAuditEntry) controlAuditSummary {
	summary := controlAuditSummary{
		Total:    len(entries),
		ByAction: summarizeAuditFacet(entries, func(entry controlActionAuditEntry) string { return entry.Action }),
		ByActor:  summarizeAuditFacet(entries, func(entry controlActionAuditEntry) string { return entry.Actor }),
		ByRole:   summarizeAuditFacet(entries, func(entry controlActionAuditEntry) string { return entry.Role }),
		ByTeam: summarizeAuditFacet(entries, func(entry controlActionAuditEntry) string {
			return firstNonEmpty(entry.Team, entry.EventStringField("team"), "unassigned")
		}),
		ByProject: summarizeAuditFacet(entries, func(entry controlActionAuditEntry) string {
			return firstNonEmpty(entry.Project, entry.EventStringField("project"), "unassigned")
		}),
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Note) != "" || strings.TrimSpace(entry.Reason) != "" {
			summary.NotesCount++
		}
	}
	return summary
}

func summarizeAuditFacet(entries []controlActionAuditEntry, keyFn func(controlActionAuditEntry) string) []auditFacetCount {
	counts := make(map[string]int)
	for _, entry := range entries {
		key := strings.TrimSpace(keyFn(entry))
		if key == "" {
			key = "unknown"
		}
		counts[key]++
	}
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

func auditNotesTimeline(entries []controlActionAuditEntry, limit int) []controlActionAuditEntry {
	out := make([]controlActionAuditEntry, 0)
	for _, entry := range entries {
		if strings.TrimSpace(entry.Note) == "" && strings.TrimSpace(entry.Reason) == "" {
			continue
		}
		out = append(out, entry)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (entry controlActionAuditEntry) EventStringField(key string) string {
	if entry.Event.Payload == nil {
		return ""
	}
	value, _ := entry.Event.Payload[key].(string)
	return value
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
