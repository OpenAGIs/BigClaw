package observabilitysurface

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"

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
	Label       string `json:"label"`
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

type Comment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
}

type DecisionNote struct {
	Author            string   `json:"author"`
	Summary           string   `json:"summary"`
	Outcome           string   `json:"outcome"`
	Mentions          []string `json:"mentions,omitempty"`
	RelatedCommentIDs []string `json:"related_comment_ids,omitempty"`
	FollowUp          string   `json:"follow_up,omitempty"`
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
	PRNumber           int    `json:"pr_number,omitempty"`
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
	TaskID        string         `json:"task_id"`
	Source        string         `json:"source,omitempty"`
	Title         string         `json:"title"`
	RunID         string         `json:"run_id"`
	Medium        string         `json:"medium"`
	Status        string         `json:"status,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	Logs          []LogEntry     `json:"logs,omitempty"`
	Traces        []TraceEntry   `json:"traces,omitempty"`
	Artifacts     []Artifact     `json:"artifacts,omitempty"`
	Audits        []AuditEntry   `json:"audits,omitempty"`
	Comments      []Comment      `json:"comments,omitempty"`
	DecisionNotes []DecisionNote `json:"decision_notes,omitempty"`
	Closeout      Closeout       `json:"closeout"`
}

func NewTaskRun(task domain.Task, runID, medium string) *TaskRun {
	return &TaskRun{
		TaskID: task.ID,
		Source: task.Source,
		Title:  task.Title,
		RunID:  runID,
		Medium: medium,
	}
}

func (r *TaskRun) Log(level, message string, context map[string]any) {
	r.Logs = append(r.Logs, LogEntry{Level: level, Message: message, Context: cloneMap(context)})
}

func (r *TaskRun) Trace(span, status string, attributes map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{Span: span, Status: status, Attributes: cloneMap(attributes)})
}

func (r *TaskRun) RegisterArtifact(label, kind, path, environment string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	r.Artifacts = append(r.Artifacts, Artifact{
		Label:       label,
		Kind:        kind,
		Path:        path,
		Environment: environment,
		SHA256:      hex.EncodeToString(sum[:]),
	})
	r.Audits = append(r.Audits, AuditEntry{Action: "artifact.registered", Actor: "task-run", Outcome: "recorded", Details: map[string]any{"label": label, "path": path}})
	return nil
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Details: cloneMap(details)})
}

func (r *TaskRun) AddComment(author, body string, mentions []string, anchor string) Comment {
	comment := Comment{
		CommentID: "comment-" + strconvItoa(len(r.Comments)+1),
		Author:    author,
		Body:      body,
		Mentions:  append([]string(nil), mentions...),
		Anchor:    anchor,
	}
	r.Comments = append(r.Comments, comment)
	r.Audits = append(r.Audits, AuditEntry{Action: "comment.added", Actor: author, Outcome: "recorded", Details: map[string]any{"comment_id": comment.CommentID, "body": body, "mentions": comment.Mentions, "anchor": anchor}})
	return comment
}

func (r *TaskRun) AddDecisionNote(author, summary, outcome string, mentions, relatedCommentIDs []string, followUp string) {
	note := DecisionNote{
		Author:            author,
		Summary:           summary,
		Outcome:           outcome,
		Mentions:          append([]string(nil), mentions...),
		RelatedCommentIDs: append([]string(nil), relatedCommentIDs...),
		FollowUp:          followUp,
	}
	r.DecisionNotes = append(r.DecisionNotes, note)
	r.Audits = append(r.Audits, AuditEntry{Action: "decision.note_added", Actor: author, Outcome: outcome, Details: map[string]any{"summary": summary, "mentions": note.Mentions, "follow_up": followUp}})
}

func (r *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit, runCommitLinks []repo.RunCommitLink) {
	r.Closeout = Closeout{
		ValidationEvidence: append([]string(nil), validationEvidence...),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		RunCommitLinks:     append([]repo.RunCommitLink(nil), runCommitLinks...),
		Complete:           len(validationEvidence) > 0,
	}
	r.Audits = append(r.Audits, AuditEntry{Action: "closeout.recorded", Actor: "task-run", Outcome: "recorded", Details: map[string]any{"complete": r.Closeout.Complete}})
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

type ObservabilityLedger struct {
	path string
}

func NewObservabilityLedger(path string) *ObservabilityLedger {
	return &ObservabilityLedger{path: path}
}

func (l *ObservabilityLedger) Append(run *TaskRun) error {
	entries, err := l.LoadRuns()
	if err != nil {
		return err
	}
	entries = append(entries, *run)
	return l.write(entries)
}

func (l *ObservabilityLedger) Load() ([]map[string]any, error) {
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *ObservabilityLedger) LoadRuns() ([]TaskRun, error) {
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []TaskRun
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *ObservabilityLedger) write(entries []TaskRun) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

type CollaborationThread struct {
	MentionCount int
	Comments     []Comment
}

func BuildCollaborationThreadFromAudits(audits []AuditEntry) *CollaborationThread {
	thread := &CollaborationThread{}
	for _, audit := range audits {
		if audit.Action != "comment.added" {
			continue
		}
		comment := Comment{
			CommentID: stringValue(audit.Details["comment_id"]),
			Body:      stringValue(audit.Details["body"]),
			Anchor:    stringValue(audit.Details["anchor"]),
			Author:    audit.Actor,
			Mentions:  stringSliceValue(audit.Details["mentions"]),
		}
		thread.MentionCount += len(comment.Mentions)
		thread.Comments = append(thread.Comments, comment)
	}
	if len(thread.Comments) == 0 {
		return nil
	}
	return thread
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	return strings.Join([]string{
		"# Repo Sync Audit",
		"",
		"- Failure Category: " + audit.Sync.FailureCategory,
		"- Branch State: " + audit.PullRequest.BranchState,
		"- Body State: " + audit.PullRequest.BodyState,
		"",
		"sync=" + audit.Sync.Status + ", failure=" + audit.Sync.FailureCategory + ", pr-branch=" + audit.PullRequest.BranchState + ", pr-body=" + audit.PullRequest.BodyState,
	}, "\n")
}

func RenderTaskRunReport(run TaskRun) string {
	lines := []string{
		"# Task Run Report",
		"",
		"Run ID: " + run.RunID,
		"## Logs",
		formatLogs(run.Logs),
		"## Trace",
		formatTraces(run.Traces),
		"## Artifacts",
		formatArtifacts(run.Artifacts),
		"## Audit",
		formatAudits(run.Audits),
		"## Closeout",
		"Git Push Succeeded: " + boolString(run.Closeout.GitPushSucceeded),
		"## Actions",
		"Retry [retry] state=disabled target=" + run.RunID + " reason=retry is available for failed or approval-blocked runs",
		"## Collaboration",
	}
	for _, comment := range run.Comments {
		lines = append(lines, comment.Body)
	}
	for _, note := range run.DecisionNotes {
		lines = append(lines, note.Summary)
	}
	return strings.Join(lines, "\n")
}

func RenderTaskRunDetailPage(run TaskRun) string {
	repoEvidence := "n/a"
	if len(run.Closeout.RunCommitLinks) > 0 {
		parts := make([]string, 0, len(run.Closeout.RunCommitLinks))
		for _, link := range run.Closeout.RunCommitLinks {
			parts = append(parts, link.CommitHash)
		}
		repoEvidence = strings.Join(parts, ", ")
	}
	lines := []string{
		"<title>Task Run Detail</title>",
		"<h1>Task Run Detail</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<div data-detail=\"title\">" + html.EscapeString(run.Title) + "</div>",
		"<h2>Reports</h2>",
		"<div>" + escapedJSONValue(formatLogs(run.Logs)) + "</div>",
		"<div>" + escapedJSONValue(formatTraces(run.Traces)) + "</div>",
		"<div>" + html.EscapeString(formatArtifacts(run.Artifacts)) + "</div>",
		"<div>" + html.EscapeString(run.Summary) + "</div>",
		"<h2>Closeout</h2>",
		"<div>complete</div>",
		"<h2>Repo Evidence</h2>",
		"<div>" + html.EscapeString(repoEvidence) + "</div>",
		"<h2>Actions</h2>",
		"<div>Pause [pause] state=disabled target=" + html.EscapeString(run.RunID) + " reason=completed or failed runs cannot be paused</div>",
		"<h2>Collaboration</h2>",
	}
	for _, comment := range run.Comments {
		lines = append(lines, "<div>"+html.EscapeString(comment.Body)+"</div>")
	}
	for _, note := range run.DecisionNotes {
		lines = append(lines, "<div>"+html.EscapeString(note.Summary)+"</div>")
	}
	return strings.Join(lines, "\n")
}

func formatLogs(entries []LogEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Message)
	}
	return strings.Join(parts, "\n")
}

func formatTraces(entries []TraceEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Span)
	}
	return strings.Join(parts, "\n")
}

func formatArtifacts(entries []Artifact) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Path)
	}
	return strings.Join(parts, "\n")
}

func formatAudits(entries []AuditEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Action)
	}
	return strings.Join(parts, "\n")
}

func escapedJSONValue(value string) string {
	return strings.ReplaceAll(value, "</script>", "<\\/script>")
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func boolString(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
