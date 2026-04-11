package flow

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type Node struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Department string   `json:"department"`
	Owner      string   `json:"owner,omitempty"`
	SLA        string   `json:"sla,omitempty"`
	Validation string   `json:"validation,omitempty"`
	Approval   string   `json:"approval,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	DependsOn  []string `json:"depends_on,omitempty"`
}

type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Summary   string    `json:"summary,omitempty"`
	Nodes     []Node    `json:"nodes"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Store struct {
	mu        sync.Mutex
	templates map[string]Template
}

func NewStore() *Store {
	return &Store{templates: make(map[string]Template)}
}

func (s *Store) Save(template Template, now time.Time) Template {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	template.ID = strings.TrimSpace(template.ID)
	if template.ID == "" {
		template.ID = slug(template.Name)
	}
	if template.ID == "" {
		template.ID = fmt.Sprintf("template-%d", now.UnixNano())
	}
	if existing, ok := s.templates[template.ID]; ok {
		template.CreatedAt = existing.CreatedAt
	}
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now
	template.Nodes = cloneNodes(template.Nodes)
	s.templates[template.ID] = template
	return cloneTemplate(template)
}

func (s *Store) Get(id string) (Template, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	template, ok := s.templates[strings.TrimSpace(id)]
	if !ok {
		return Template{}, false
	}
	return cloneTemplate(template), true
}

func (s *Store) List() []Template {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Template, 0, len(s.templates))
	for _, template := range s.templates {
		out = append(out, cloneTemplate(template))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

type DepartmentSummary struct {
	Key           string `json:"key"`
	TotalTasks    int    `json:"total_tasks"`
	Completed     int    `json:"completed"`
	Blocked       int    `json:"blocked"`
	Active        int    `json:"active"`
	Pending       int    `json:"pending"`
	Owner         string `json:"owner,omitempty"`
	NextGate      string `json:"next_gate,omitempty"`
	OverallStatus string `json:"overall_status"`
}

type Overview struct {
	FlowID        string              `json:"flow_id"`
	Title         string              `json:"title"`
	TemplateID    string              `json:"template_id,omitempty"`
	Team          string              `json:"team,omitempty"`
	Project       string              `json:"project,omitempty"`
	OverallStatus string              `json:"overall_status"`
	NextGate      string              `json:"next_gate,omitempty"`
	Owner         string              `json:"owner,omitempty"`
	Blockers      int                 `json:"blockers"`
	Departments   []DepartmentSummary `json:"departments"`
}

type ChecklistItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	TaskID      string `json:"task_id,omitempty"`
	Department  string `json:"department,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type Checklist struct {
	FlowID string          `json:"flow_id"`
	Items  []ChecklistItem `json:"items"`
	Ready  bool            `json:"ready"`
}

type SupportHandoff struct {
	FlowID         string   `json:"flow_id"`
	ReleaseSummary string   `json:"release_summary"`
	KnownIssues    []string `json:"known_issues"`
	FAQDraft       []string `json:"faq_draft"`
	TicketTemplate string   `json:"ticket_template"`
	Status         string   `json:"status"`
}

func LaunchTasks(template Template, flowID string, title string, team string, project string, actor string, now time.Time) []domain.Task {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if strings.TrimSpace(flowID) == "" {
		flowID = fmt.Sprintf("flow-%d", now.UnixNano())
	}
	baseTitle := strings.TrimSpace(title)
	if baseTitle == "" {
		baseTitle = template.Name
	}
	tasks := make([]domain.Task, 0, len(template.Nodes))
	for index, node := range template.Nodes {
		identifier := strings.TrimSpace(node.ID)
		if identifier == "" {
			identifier = fmt.Sprintf("node-%d", index+1)
		}
		metadata := map[string]string{
			"flow_id":          flowID,
			"flow_template_id": template.ID,
			"department":       strings.TrimSpace(node.Department),
			"owner":            strings.TrimSpace(node.Owner),
			"team":             team,
			"project":          project,
			"sla":              strings.TrimSpace(node.SLA),
			"approval":         strings.TrimSpace(node.Approval),
			"workflow":         firstNonEmpty(strings.TrimSpace(node.Kind), strings.TrimSpace(node.Department), identifier),
			"created_by":       actor,
		}
		if len(node.DependsOn) > 0 {
			metadata["depends_on"] = strings.Join(node.DependsOn, ",")
		}
		tasks = append(tasks, domain.Task{
			ID:                 fmt.Sprintf("%s-%s", flowID, identifier),
			TraceID:            fmt.Sprintf("%s-%s", flowID, identifier),
			Title:              fmt.Sprintf("%s / %s", baseTitle, firstNonEmpty(node.Name, identifier)),
			Priority:           priorityForDepartment(node.Department),
			State:              domain.TaskQueued,
			AcceptanceCriteria: stringSlice(node.Validation),
			ValidationPlan:     stringSlice(node.Validation),
			Metadata:           metadata,
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}
	return tasks
}

func BuildOverview(tasks []domain.Task) []Overview {
	grouped := make(map[string][]domain.Task)
	for _, task := range tasks {
		flowID := strings.TrimSpace(task.Metadata["flow_id"])
		if flowID == "" {
			continue
		}
		grouped[flowID] = append(grouped[flowID], task)
	}
	out := make([]Overview, 0, len(grouped))
	for flowID, items := range grouped {
		out = append(out, buildOverview(flowID, items))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OverallStatus == out[j].OverallStatus {
			return out[i].FlowID < out[j].FlowID
		}
		return statusRank(out[i].OverallStatus) < statusRank(out[j].OverallStatus)
	})
	return out
}

func BuildLaunchChecklist(tasks []domain.Task, flowID string) Checklist {
	items := filterFlowTasks(tasks, flowID)
	findByDepartment := func(dept string) (domain.Task, bool) {
		for _, task := range items {
			if strings.EqualFold(strings.TrimSpace(task.Metadata["department"]), dept) {
				return task, true
			}
		}
		return domain.Task{}, false
	}
	checklist := Checklist{FlowID: flowID}
	for _, spec := range []struct {
		key, label, department string
		required               bool
		description            string
	}{
		{"engineering_validation", "Engineering validation", "engineering", true, "Engineering task must complete and validation evidence must exist."},
		{"docs_ready", "Documentation ready", "docs", true, "Docs node should be completed before launch."},
		{"release_gate", "Release gate", "release", true, "Release checklist and approvals should be complete."},
		{"support_ready", "Support handoff", "support", true, "Support packet should be ready before public launch."},
	} {
		task, ok := findByDepartment(spec.department)
		status := "missing"
		if ok {
			status = checklistStatus(task.State)
		}
		checklist.Items = append(checklist.Items, ChecklistItem{
			Key:         spec.key,
			Label:       spec.label,
			Status:      status,
			TaskID:      task.ID,
			Department:  spec.department,
			Required:    spec.required,
			Description: spec.description,
		})
	}
	checklist.Ready = true
	for _, item := range checklist.Items {
		if item.Required && item.Status != "completed" {
			checklist.Ready = false
			break
		}
	}
	return checklist
}

func BuildSupportHandoff(tasks []domain.Task, flowID string) SupportHandoff {
	items := filterFlowTasks(tasks, flowID)
	handoff := SupportHandoff{FlowID: flowID, Status: "draft"}
	knownIssues := make([]string, 0)
	faq := make([]string, 0)
	for _, task := range items {
		department := strings.ToLower(strings.TrimSpace(task.Metadata["department"]))
		if department == "release" && handoff.ReleaseSummary == "" {
			handoff.ReleaseSummary = fmt.Sprintf("Release node %s is %s.", task.Title, task.State)
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			knownIssues = append(knownIssues, fmt.Sprintf("%s: %s", task.ID, firstNonEmpty(task.Metadata["blocked_reason"], task.Metadata["failure_reason"], string(task.State))))
		}
		if department == "docs" || department == "support" {
			faq = append(faq, fmt.Sprintf("How do we operate %s? -> See %s status %s.", flowID, department, task.State))
		}
	}
	if handoff.ReleaseSummary == "" {
		handoff.ReleaseSummary = fmt.Sprintf("Flow %s release summary is pending.", flowID)
	}
	if len(knownIssues) == 0 {
		knownIssues = append(knownIssues, "No blocking issues reported in the current flow window.")
	}
	if len(faq) == 0 {
		faq = append(faq, fmt.Sprintf("What changed in %s? -> Review engineering and release tasks for the latest artifacts.", flowID))
	}
	handoff.KnownIssues = knownIssues
	handoff.FAQDraft = faq
	handoff.TicketTemplate = fmt.Sprintf("Support ticket for %s\n- Customer impact:\n- Workaround:\n- Escalation owner:\n", flowID)
	if allFlowTasksCompleted(items) {
		handoff.Status = "ready"
	}
	return handoff
}

func buildOverview(flowID string, tasks []domain.Task) Overview {
	overview := Overview{FlowID: flowID}
	departments := make(map[string]*DepartmentSummary)
	for _, task := range tasks {
		if overview.Title == "" {
			overview.Title = strings.TrimSpace(task.Metadata["flow_title"])
			if overview.Title == "" {
				overview.Title = strings.TrimSpace(task.Title)
			}
		}
		overview.TemplateID = firstNonEmpty(overview.TemplateID, task.Metadata["flow_template_id"])
		overview.Team = firstNonEmpty(overview.Team, task.Metadata["team"])
		overview.Project = firstNonEmpty(overview.Project, task.Metadata["project"])
		dept := firstNonEmpty(task.Metadata["department"], "unassigned")
		entry := departments[dept]
		if entry == nil {
			entry = &DepartmentSummary{Key: dept}
			departments[dept] = entry
		}
		entry.TotalTasks++
		entry.Owner = firstNonEmpty(entry.Owner, task.Metadata["owner"])
		entry.NextGate = firstNonEmpty(entry.NextGate, task.Metadata["approval"], task.Metadata["sla"])
		switch task.State {
		case domain.TaskSucceeded:
			entry.Completed++
		case domain.TaskBlocked, domain.TaskDeadLetter, domain.TaskFailed:
			entry.Blocked++
		case domain.TaskRunning, domain.TaskLeased, domain.TaskRetrying:
			entry.Active++
		default:
			entry.Pending++
		}
	}
	for _, entry := range departments {
		entry.OverallStatus = summarizeDepartmentStatus(*entry)
		overview.Departments = append(overview.Departments, *entry)
		if entry.Blocked > 0 {
			overview.Blockers += entry.Blocked
		}
	}
	sort.SliceStable(overview.Departments, func(i, j int) bool { return overview.Departments[i].Key < overview.Departments[j].Key })
	overview.Owner = flowOwner(overview.Departments)
	overview.OverallStatus = summarizeFlowStatus(overview.Departments)
	overview.NextGate = nextGate(overview.Departments)
	return overview
}

func filterFlowTasks(tasks []domain.Task, flowID string) []domain.Task {
	out := make([]domain.Task, 0)
	for _, task := range tasks {
		if strings.EqualFold(strings.TrimSpace(task.Metadata["flow_id"]), strings.TrimSpace(flowID)) {
			out = append(out, task)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func summarizeDepartmentStatus(item DepartmentSummary) string {
	if item.Blocked > 0 {
		return "blocked"
	}
	if item.Active > 0 {
		return "active"
	}
	if item.Pending > 0 {
		return "pending"
	}
	if item.TotalTasks > 0 && item.Completed == item.TotalTasks {
		return "completed"
	}
	return "unknown"
}

func summarizeFlowStatus(items []DepartmentSummary) string {
	status := "completed"
	for _, item := range items {
		switch item.OverallStatus {
		case "blocked":
			return "blocked"
		case "active":
			status = "active"
		case "pending":
			if status != "active" {
				status = "pending"
			}
		}
	}
	return status
}

func nextGate(items []DepartmentSummary) string {
	for _, item := range items {
		if item.OverallStatus != "completed" {
			return item.Key
		}
	}
	return "done"
}

func flowOwner(items []DepartmentSummary) string {
	for _, item := range items {
		if strings.TrimSpace(item.Owner) != "" {
			return item.Owner
		}
	}
	return ""
}

func allFlowTasksCompleted(tasks []domain.Task) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, task := range tasks {
		if task.State != domain.TaskSucceeded {
			return false
		}
	}
	return true
}

func checklistStatus(state domain.TaskState) string {
	switch state {
	case domain.TaskSucceeded:
		return "completed"
	case domain.TaskBlocked, domain.TaskDeadLetter, domain.TaskFailed:
		return "blocked"
	case domain.TaskRunning, domain.TaskLeased, domain.TaskRetrying:
		return "active"
	case domain.TaskQueued:
		return "pending"
	default:
		return "missing"
	}
}

func priorityForDepartment(department string) int {
	switch strings.ToLower(strings.TrimSpace(department)) {
	case "engineering", "release":
		return 1
	case "docs", "support":
		return 2
	default:
		return 3
	}
}

func statusRank(status string) int {
	switch status {
	case "blocked":
		return 0
	case "active":
		return 1
	case "pending":
		return 2
	case "completed":
		return 3
	default:
		return 4
	}
}

func stringSlice(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []string{value}
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}

func cloneTemplate(template Template) Template {
	copy := template
	copy.Nodes = cloneNodes(template.Nodes)
	return copy
}

func cloneNodes(nodes []Node) []Node {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]Node, len(nodes))
	copy(out, nodes)
	for index := range out {
		out[index].DependsOn = append([]string(nil), out[index].DependsOn...)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
