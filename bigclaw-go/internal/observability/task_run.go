package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

type LogEntry struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

type TraceEntry struct {
	Span       string         `json:"span"`
	Status     string         `json:"status"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type Artifact struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	Environment string `json:"environment,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type AuditEntry struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

func (a AuditEntry) ToMap() map[string]any {
	return map[string]any{
		"action":  a.Action,
		"actor":   a.Actor,
		"outcome": a.Outcome,
		"details": a.Details,
	}
}

type GitSyncTelemetry struct {
	Status          string   `json:"status"`
	FailureCategory string   `json:"failure_category,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	RemoteRef       string   `json:"remote_ref,omitempty"`
	DirtyPaths      []string `json:"dirty_paths,omitempty"`
	AuthTarget      string   `json:"auth_target,omitempty"`
}

type PullRequestFreshness struct {
	PRNumber           int    `json:"pr_number"`
	PRURL              string `json:"pr_url,omitempty"`
	BranchState        string `json:"branch_state,omitempty"`
	BodyState          string `json:"body_state,omitempty"`
	BranchHeadSHA      string `json:"branch_head_sha,omitempty"`
	PRHeadSHA          string `json:"pr_head_sha,omitempty"`
	ExpectedBodyDigest string `json:"expected_body_digest,omitempty"`
	ActualBodyDigest   string `json:"actual_body_digest,omitempty"`
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

type Closeout struct {
	ValidationEvidence []string             `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool                 `json:"git_push_succeeded"`
	GitPushOutput      string               `json:"git_push_output,omitempty"`
	GitLogStatOutput   string               `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *RepoSyncAudit       `json:"repo_sync_audit,omitempty"`
	RunCommitLinks     []repo.RunCommitLink `json:"run_commit_links,omitempty"`
	Complete           bool                 `json:"complete"`
}

type TaskRun struct {
	TaskID    string         `json:"task_id"`
	RunID     string         `json:"run_id"`
	Medium    string         `json:"medium"`
	Status    string         `json:"status,omitempty"`
	Outcome   string         `json:"outcome,omitempty"`
	Logs      []LogEntry     `json:"logs,omitempty"`
	Traces    []TraceEntry   `json:"traces,omitempty"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	Audits    []AuditEntry   `json:"audits,omitempty"`
	Closeout  Closeout       `json:"closeout"`
	Task      map[string]any `json:"task,omitempty"`
}

func NewTaskRun(task domain.Task, runID string, medium string) *TaskRun {
	return &TaskRun{
		TaskID: task.ID,
		RunID:  runID,
		Medium: medium,
		Task: map[string]any{
			"id":          task.ID,
			"source":      task.Source,
			"title":       task.Title,
			"description": task.Description,
		},
	}
}

func (r *TaskRun) Log(level string, message string, context map[string]any) {
	r.Logs = append(r.Logs, LogEntry{Level: level, Message: message, Context: context})
}

func (r *TaskRun) Trace(span string, status string, attributes map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{Span: span, Status: status, Attributes: attributes})
}

func (r *TaskRun) RegisterArtifact(name string, kind string, path string, environment string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	r.Artifacts = append(r.Artifacts, Artifact{
		Name:        name,
		Kind:        kind,
		Path:        path,
		Environment: environment,
		SHA256:      hex.EncodeToString(sum[:]),
	})
	r.Audit("artifact.registered", "system", "success", map[string]any{"name": name, "path": path})
	return nil
}

func (r *TaskRun) Audit(action string, actor string, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Details: details})
}

func (r *TaskRun) AddComment(author string, body string, mentions []string, anchor string) collaboration.Comment {
	comment := collaboration.Comment{
		CommentID: "comment-" + string(rune(len(r.Audits)+1)),
		Author:    author,
		Body:      body,
		CreatedAt: anchor,
	}
	r.Audit("collaboration.comment", author, "recorded", map[string]any{
		"comment_id": comment.CommentID,
		"author":     author,
		"body":       body,
		"mentions":   append([]string(nil), mentions...),
		"anchor":     anchor,
	})
	return comment
}

func (r *TaskRun) AddDecisionNote(author string, summary string, outcome string, mentions []string, relatedCommentIDs []string, followUp string) collaboration.Decision {
	decision := collaboration.Decision{
		DecisionID: "decision-" + string(rune(len(r.Audits)+1)),
		Author:     author,
		Outcome:    outcome,
		Summary:    summary,
		RecordedAt: followUp,
	}
	r.Audit("collaboration.decision", author, outcome, map[string]any{
		"decision_id":         decision.DecisionID,
		"author":              author,
		"summary":             summary,
		"outcome":             outcome,
		"mentions":            append([]string(nil), mentions...),
		"related_comment_ids": append([]string(nil), relatedCommentIDs...),
		"follow_up":           followUp,
	})
	return decision
}

func (r *TaskRun) RecordCloseout(closeout Closeout) {
	closeout.Complete = len(closeout.ValidationEvidence) > 0 && closeout.GitLogStatOutput != ""
	r.Closeout = closeout
	r.Audit("closeout.recorded", "system", "success", map[string]any{"complete": closeout.Complete})
}

func (r *TaskRun) Finalize(status string, outcome string) {
	r.Status = status
	r.Outcome = outcome
}

type Ledger struct {
	path string
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: path}
}

func (l *Ledger) Append(run *TaskRun) error {
	runs, err := l.LoadRuns()
	if err != nil {
		return err
	}
	runs = append(runs, *run)
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, append(data, '\n'), 0o644)
}

func (l *Ledger) LoadRuns() ([]TaskRun, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []TaskRun
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l *Ledger) Load() ([]map[string]any, error) {
	runs, err := l.LoadRuns()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(runs)
	if err != nil {
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	lines := []string{
		"# Repo Sync Audit",
		"",
		"## Sync",
		"",
		"Status: " + audit.Sync.Status,
		"Failure Category: " + firstNonEmpty(audit.Sync.FailureCategory, "none"),
		"",
		"## Pull Request",
		"",
		"Branch State: " + firstNonEmpty(audit.PullRequest.BranchState, "none"),
		"Body State: " + firstNonEmpty(audit.PullRequest.BodyState, "none"),
		"",
		"Summary: sync=" + audit.Sync.Status + ", failure=" + firstNonEmpty(audit.Sync.FailureCategory, "none") + ", pr-branch=" + firstNonEmpty(audit.PullRequest.BranchState, "none") + ", pr-body=" + firstNonEmpty(audit.PullRequest.BodyState, "none"),
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderTaskRunReport(run TaskRun) string {
	thread := collaboration.BuildCollaborationThreadFromAudits(auditMaps(run.Audits), "run", run.RunID)
	lines := []string{
		"# Task Run Report",
		"",
		"Run ID: " + run.RunID,
		"",
		"## Logs",
	}
	for _, entry := range run.Logs {
		lines = append(lines, "- "+entry.Level+": "+entry.Message)
	}
	lines = append(lines, "", "## Trace")
	for _, entry := range run.Traces {
		lines = append(lines, "- "+entry.Span+": "+entry.Status)
	}
	lines = append(lines, "", "## Artifacts")
	for _, artifact := range run.Artifacts {
		lines = append(lines, "- "+artifact.Name+": "+artifact.Path)
	}
	lines = append(lines, "", "## Audit")
	for _, audit := range run.Audits {
		lines = append(lines, "- "+audit.Action)
	}
	lines = append(lines, "", "## Closeout")
	lines = append(lines, "Git Push Succeeded: "+boolText(run.Closeout.GitPushSucceeded))
	lines = append(lines, "", "## Actions")
	lines = append(lines, "Retry [retry] state=disabled target="+run.RunID+" reason=retry is available for failed or approval-blocked runs")
	lines = append(lines, "", "## Collaboration")
	if thread != nil {
		for _, comment := range thread.Comments {
			lines = append(lines, "- "+comment.Body)
		}
		for _, decision := range thread.Decisions {
			lines = append(lines, "- "+decision.Summary)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderTaskRunDetailPage(run TaskRun) string {
	thread := collaboration.BuildCollaborationThreadFromAudits(auditMaps(run.Audits), "run", run.RunID)
	timelineJSON, _ := json.Marshal(map[string]any{"logs": run.Logs, "traces": run.Traces, "audits": run.Audits})
	safeTimeline := strings.ReplaceAll(string(timelineJSON), "</script>", "<\\/script>")
	safeTimeline = strings.ReplaceAll(safeTimeline, "\\u003c/script\\u003e", "<\\/script>")
	builder := strings.Builder{}
	builder.WriteString("<html><head><title>Task Run Detail</title></head><body>")
	builder.WriteString("<h1 data-detail=\"title\">Task Run Detail</h1>")
	builder.WriteString("<section>Timeline / Log Sync</section>")
	builder.WriteString("<section>Reports</section>")
	for _, entry := range run.Logs {
		builder.WriteString("<div>" + html.EscapeString(entry.Message) + "</div>")
	}
	for _, entry := range run.Traces {
		builder.WriteString("<div>" + html.EscapeString(entry.Span) + "</div>")
	}
	for _, artifact := range run.Artifacts {
		builder.WriteString("<div>" + html.EscapeString(artifact.Path) + "</div>")
	}
	builder.WriteString("<section>Closeout</section>")
	builder.WriteString("<div>complete</div>")
	builder.WriteString("<section>Repo Evidence</section>")
	for _, link := range run.Closeout.RunCommitLinks {
		builder.WriteString("<div>" + html.EscapeString(link.CommitHash) + "</div>")
	}
	builder.WriteString("<section>Actions</section>")
	builder.WriteString("<div>Pause [pause] state=disabled target=" + html.EscapeString(run.RunID) + " reason=completed or failed runs cannot be paused</div>")
	builder.WriteString("<section>Collaboration</section>")
	if thread != nil {
		for _, comment := range thread.Comments {
			builder.WriteString("<div>" + html.EscapeString(comment.Body) + "</div>")
		}
		for _, decision := range thread.Decisions {
			builder.WriteString("<div>" + html.EscapeString(decision.Summary) + "</div>")
		}
	}
	builder.WriteString("<div>" + html.EscapeString(run.Outcome) + "</div>")
	builder.WriteString("<script>const timeline = " + safeTimeline + ";</script>")
	builder.WriteString("</body></html>")
	return builder.String()
}

func auditMaps(entries []AuditEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.ToMap())
	}
	return out
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolText(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
