package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/billing"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/flow"
	"bigclaw-go/internal/intake"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/prd"
	"bigclaw-go/internal/product"
	"bigclaw-go/internal/reporting"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/workflow"
)

type flowTemplateRequest struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name"`
	Summary string      `json:"summary,omitempty"`
	Nodes   []flow.Node `json:"nodes"`
}

type flowLaunchRequest struct {
	Actor   string `json:"actor,omitempty"`
	Team    string `json:"team,omitempty"`
	Project string `json:"project,omitempty"`
	Title   string `json:"title,omitempty"`
}

type prdIntakeRequest struct {
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

type workflowDefinitionRenderRequest struct {
	Definition workflow.Definition `json:"definition"`
	Task       domain.Task         `json:"task"`
	RunID      string              `json:"run_id"`
}

type workflowRunRequest struct {
	Definition         workflow.Definition          `json:"definition"`
	Task               domain.Task                  `json:"task"`
	RunID              string                       `json:"run_id"`
	ValidationEvidence []string                     `json:"validation_evidence,omitempty"`
	Approvals          []string                     `json:"approvals,omitempty"`
	GitPushSucceeded   bool                         `json:"git_push_succeeded"`
	GitLogStatOutput   string                       `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *observability.RepoSyncAudit `json:"repo_sync_audit,omitempty"`
	Quota              scheduler.QuotaSnapshot      `json:"quota,omitempty"`
}

func (s *Server) handleV2WeeklyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, start, end, err := parseWeeklyFilters(r, s.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := reporting.Build(s.filteredTasks(team, project, "", start, end), s.Recorder.EventsByTask("", 0), start, end)
	writeJSON(w, http.StatusOK, map[string]any{
		"filters":        map[string]any{"team": team, "project": project, "week_start": start, "week_end": end},
		"summary":        report.Summary,
		"team_breakdown": report.TeamBreakdown,
		"highlights":     report.Highlights,
		"actions":        report.Actions,
		"report":         map[string]any{"markdown": report.Markdown, "export_url": weeklyExportURL(team, project, start, end)},
	})
}

func (s *Server) handleV2WeeklyReportExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, start, end, err := parseWeeklyFilters(r, s.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := reporting.Build(s.filteredTasks(team, project, "", start, end), s.Recorder.EventsByTask("", 0), start, end)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fmt.Sprintf("bigclaw-weekly-report-%s.md", start.Format("2006-01-02"))))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report.Markdown))
}

func (s *Server) handleV2FlowTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"templates": s.FlowStore.List()})
	case http.MethodPost:
		var request flowTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("decode template: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.Name) == "" || len(request.Nodes) == 0 {
			http.Error(w, "missing name or nodes", http.StatusBadRequest)
			return
		}
		template := s.FlowStore.Save(flow.Template{ID: request.ID, Name: request.Name, Summary: request.Summary, Nodes: request.Nodes}, s.Now())
		writeJSON(w, http.StatusOK, map[string]any{"template": template})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleV2FlowTemplateAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v2/flows/templates/"), "/")
	if !strings.HasSuffix(path, "/launch") {
		http.Error(w, "unsupported action", http.StatusNotFound)
		return
	}
	templateID := strings.TrimSuffix(path, "/launch")
	templateID = strings.TrimSuffix(templateID, "/")
	template, ok := s.FlowStore.Get(templateID)
	if !ok {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	var request flowLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode launch request: %v", err), http.StatusBadRequest)
		return
	}
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		actor = "system"
	}
	flowID := fmt.Sprintf("flow-%d", s.Now().UnixNano())
	tasks := flow.LaunchTasks(template, flowID, request.Title, request.Team, request.Project, actor, s.Now())
	for _, task := range tasks {
		if err := s.Queue.Enqueue(r.Context(), task); err != nil {
			http.Error(w, fmt.Sprintf("launch flow task: %v", err), http.StatusInternalServerError)
			return
		}
		s.Recorder.StoreTask(task)
		s.publish(domain.Event{ID: task.ID + "-queued", Type: domain.EventTaskQueued, TaskID: task.ID, TraceID: task.TraceID, Timestamp: s.Now(), Payload: map[string]any{"actor": actor, "flow_id": flowID, "template_id": template.ID, "department": task.Metadata["department"]}})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"flow_id": flowID, "template": template, "launched_tasks": tasks})
}

func (s *Server) handleV2FlowOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	tasks := s.filteredTasks(team, project, "", time.Time{}, time.Time{})
	if flowID != "" {
		filtered := make([]domain.Task, 0)
		for _, task := range tasks {
			if strings.EqualFold(strings.TrimSpace(task.Metadata["flow_id"]), flowID) {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": flow.BuildOverview(tasks)})
}

func (s *Server) handleV2PRDIntake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request prdIntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode prd intake: %v", err), http.StatusBadRequest)
		return
	}
	result := prd.Build(request.Title, request.Body, request.AcceptanceCriteria, s.Now())
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleV2IntakeConnectorIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v2/intake/connectors/"), "/")
	if !strings.HasSuffix(path, "/issues") {
		http.Error(w, "unsupported intake route", http.StatusNotFound)
		return
	}
	name := strings.TrimSuffix(path, "/issues")
	name = strings.TrimSuffix(name, "/")
	connector, ok := intake.ConnectorByName(name)
	if !ok {
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if project == "" {
		http.Error(w, "missing project", http.StatusBadRequest)
		return
	}
	states := splitCSV(r.URL.Query().Get("states"))
	issues, err := connector.FetchIssues(project, states)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch connector issues: %v", err), http.StatusInternalServerError)
		return
	}
	mapped := make([]domain.Task, 0, len(issues))
	for _, issue := range issues {
		mapped = append(mapped, intake.MapSourceIssueToTask(issue))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"connector":    connector.Name(),
		"project":      project,
		"states":       states,
		"issues":       issues,
		"mapped_tasks": mapped,
	})
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (s *Server) handleV2IntakeIssueMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var issue intake.SourceIssue
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		http.Error(w, fmt.Sprintf("decode intake issue: %v", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": intake.MapSourceIssueToTask(issue)})
}

func (s *Server) handleV2LaunchChecklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	if flowID == "" {
		http.Error(w, "missing flow_id", http.StatusBadRequest)
		return
	}
	checklist := flow.BuildLaunchChecklist(s.Recorder.Tasks(0), flowID)
	writeJSON(w, http.StatusOK, checklist)
}

func (s *Server) handleV2SupportHandoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	if flowID == "" {
		http.Error(w, "missing flow_id", http.StatusBadRequest)
		return
	}
	handoff := flow.BuildSupportHandoff(s.Recorder.Tasks(0), flowID)
	writeJSON(w, http.StatusOK, handoff)
}

func (s *Server) handleV2WorkflowDefinitionRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request workflowDefinitionRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode workflow definition render request: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Definition.Name) == "" {
		http.Error(w, "missing definition.name", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Task.ID) == "" {
		http.Error(w, "missing task.id", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.RunID) == "" {
		http.Error(w, "missing run_id", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"definition": request.Definition,
		"rendered": map[string]any{
			"report_path":  request.Definition.RenderReportPath(request.Task, request.RunID),
			"journal_path": request.Definition.RenderJournalPath(request.Task, request.RunID),
		},
	})
}

func (s *Server) handleV2WorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runtime := s.workflowRuntime()
	if runtime == nil {
		http.Error(w, "workflow runtime not configured", http.StatusServiceUnavailable)
		return
	}
	var request workflowRunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode workflow run request: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Definition.Name) == "" {
		http.Error(w, "missing definition.name", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Task.ID) == "" {
		http.Error(w, "missing task.id", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.RunID) == "" {
		http.Error(w, "missing run_id", http.StatusBadRequest)
		return
	}
	engine := &workflow.Engine{
		Runtime:  runtime,
		Recorder: s.Recorder,
		Queue:    s.Queue,
		Quota:    request.Quota,
		Now:      s.Now,
	}
	result, err := engine.RunDefinition(r.Context(), request.Task, request.Definition, request.RunID, workflow.RunOptions{
		ValidationEvidence: request.ValidationEvidence,
		Approvals:          request.Approvals,
		GitPushSucceeded:   request.GitPushSucceeded,
		GitLogStatOutput:   request.GitLogStatOutput,
		RepoSyncAudit:      request.RepoSyncAudit,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("run workflow definition: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

func (s *Server) handleV2Navigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := normalizeHomeRole(r)
	writeJSON(w, http.StatusOK, map[string]any{"role": role, "sections": product.Navigation()})
}

func (s *Server) handleV2Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := normalizeHomeRole(r)
	home := product.HomeForRole(role, s.Recorder.Tasks(0))
	writeJSON(w, http.StatusOK, map[string]any{"home": home, "summary": product.SummaryText(home)})
}

func (s *Server) handleV2DesignSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, product.DefaultDesignSystem())
}

func (s *Server) handleV2BillingUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	organization := strings.TrimSpace(r.URL.Query().Get("organization"))
	tier := strings.TrimSpace(r.URL.Query().Get("tier"))
	tasks := s.filteredTasks(team, project, "", time.Time{}, time.Time{})
	writeJSON(w, http.StatusOK, billing.BuildUsage(tasks, organization, tier))
}

func (s *Server) handleV2BillingEntitlements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, billing.EntitlementsForTier(r.URL.Query().Get("tier")))
}

func parseWeeklyFilters(r *http.Request, now time.Time) (string, string, time.Time, time.Time, error) {
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	start, err := parseOptionalTime(r.URL.Query().Get("week_start"))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid week_start value, expected RFC3339")
	}
	end, err := parseOptionalTime(r.URL.Query().Get("week_end"))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid week_end value, expected RFC3339")
	}
	if start.IsZero() {
		endOfWindow := now.UTC()
		start = endOfWindow.Add(-7 * 24 * time.Hour)
		if end.IsZero() {
			end = endOfWindow
		}
	}
	if end.IsZero() {
		end = start.Add(7*24*time.Hour - time.Second)
	}
	return team, project, start, end, nil
}

func weeklyExportURL(team string, project string, start time.Time, end time.Time) string {
	parts := make([]string, 0)
	if team != "" {
		parts = append(parts, "team="+team)
	}
	if project != "" {
		parts = append(parts, "project="+project)
	}
	parts = append(parts, "week_start="+start.Format(time.RFC3339))
	parts = append(parts, "week_end="+end.Format(time.RFC3339))
	return "/v2/reports/weekly/export?" + strings.Join(parts, "&")
}

func normalizeHomeRole(r *http.Request) string {
	if role := strings.TrimSpace(r.URL.Query().Get("role")); role != "" {
		return strings.ToLower(role)
	}
	authorization := parseControlAuthorization(r, "", "", "")
	return strings.ToLower(string(authorization.Role))
}

func renderJSONBody(body any) *bytes.Reader {
	payload, _ := json.Marshal(body)
	return bytes.NewReader(payload)
}

func collectFlowTasks(ctx context.Context, qTasks []domain.Task) []domain.Task {
	out := append([]domain.Task(nil), qTasks...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	_ = ctx
	return out
}
