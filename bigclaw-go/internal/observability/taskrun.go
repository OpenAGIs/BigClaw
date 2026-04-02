package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
)

type RunLog struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

type RunTrace struct {
	Span       string         `json:"span"`
	Status     string         `json:"status"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type RunArtifact struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

type RunComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
}

type RunDecisionNote struct {
	DecisionID string   `json:"decision_id"`
	Author     string   `json:"author"`
	Summary    string   `json:"summary"`
	Outcome    string   `json:"outcome"`
	Mentions   []string `json:"mentions,omitempty"`
	FollowUp   string   `json:"follow_up,omitempty"`
}

type GitSyncTelemetry struct {
	Status          string   `json:"status,omitempty"`
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

type RunCommitLink struct {
	RunID       string `json:"run_id"`
	CommitHash  string `json:"commit_hash"`
	Role        string `json:"role"`
	RepoSpaceID string `json:"repo_space_id,omitempty"`
}

type RunCloseout struct {
	ValidationEvidence []string        `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool            `json:"git_push_succeeded"`
	GitPushOutput      string          `json:"git_push_output,omitempty"`
	GitLogStatOutput   string          `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *RepoSyncAudit  `json:"repo_sync_audit,omitempty"`
	RunCommitLinks     []RunCommitLink `json:"run_commit_links,omitempty"`
	Complete           bool            `json:"complete"`
}

type AuditItem struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor,omitempty"`
	Outcome string         `json:"outcome,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	Task      domain.Task       `json:"task"`
	RunID     string            `json:"run_id"`
	Medium    string            `json:"medium,omitempty"`
	Status    string            `json:"status"`
	Summary   string            `json:"summary,omitempty"`
	Logs      []RunLog          `json:"logs,omitempty"`
	Traces    []RunTrace        `json:"traces,omitempty"`
	Artifacts []RunArtifact     `json:"artifacts,omitempty"`
	Audits    []AuditItem       `json:"audits,omitempty"`
	Comments  []RunComment      `json:"comments,omitempty"`
	Decisions []RunDecisionNote `json:"decisions,omitempty"`
	Closeout  RunCloseout       `json:"closeout"`
}

func NewTaskRun(task domain.Task, runID string, medium string) *TaskRun {
	return &TaskRun{Task: task, RunID: strings.TrimSpace(runID), Medium: strings.TrimSpace(medium)}
}

func (r *TaskRun) Log(level, message string, context map[string]any) {
	r.Logs = append(r.Logs, RunLog{Level: level, Message: message, Context: cloneAnyMap(context)})
}

func (r *TaskRun) Trace(span, status string, attrs map[string]any) {
	r.Traces = append(r.Traces, RunTrace{Span: span, Status: status, Attributes: cloneAnyMap(attrs)})
}

func (r *TaskRun) RegisterArtifact(name, kind, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	r.Artifacts = append(r.Artifacts, RunArtifact{Name: name, Kind: kind, Path: path, SHA256: hex.EncodeToString(sum[:])})
	r.Audits = append(r.Audits, AuditItem{Action: "artifact.registered", Outcome: "success", Details: map[string]any{"name": name, "kind": kind, "path": path}})
	return nil
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditItem{Action: action, Actor: actor, Outcome: outcome, Details: cloneAnyMap(details)})
}

func (r *TaskRun) AddComment(author, body string, mentions []string, anchor string) RunComment {
	comment := RunComment{
		CommentID: fmt.Sprintf("%s-comment-%d", r.RunID, len(r.Comments)+1),
		Author:    author,
		Body:      body,
		Mentions:  append([]string(nil), mentions...),
		Anchor:    anchor,
	}
	r.Comments = append(r.Comments, comment)
	return comment
}

func (r *TaskRun) AddDecisionNote(author, summary, outcome string, mentions []string, followUp string) {
	r.Decisions = append(r.Decisions, RunDecisionNote{
		DecisionID: fmt.Sprintf("%s-decision-%d", r.RunID, len(r.Decisions)+1),
		Author:     author,
		Summary:    summary,
		Outcome:    outcome,
		Mentions:   append([]string(nil), mentions...),
		FollowUp:   followUp,
	})
}

func (r *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit, runCommitLinks []RunCommitLink) {
	r.Closeout = RunCloseout{
		ValidationEvidence: append([]string(nil), validationEvidence...),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		RunCommitLinks:     append([]RunCommitLink(nil), runCommitLinks...),
		Complete:           true,
	}
	r.Audits = append(r.Audits, AuditItem{Action: "closeout.recorded", Outcome: "success"})
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

type RunLedger struct {
	path string
}

func NewRunLedger(path string) *RunLedger {
	return &RunLedger{path: path}
}

func (l *RunLedger) Append(run *TaskRun) error {
	runs, err := l.LoadRuns()
	if err != nil {
		return err
	}
	runs = append(runs, *run)
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

func (l *RunLedger) LoadRuns() ([]TaskRun, error) {
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var runs []TaskRun
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func RenderTaskRunReport(run TaskRun) string {
	lines := []string{
		"# Task Run Report",
		"",
		fmt.Sprintf("Run ID: %s", run.RunID),
		"## Logs",
	}
	for _, item := range run.Logs {
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Level, item.Message))
	}
	lines = append(lines, "## Trace")
	for _, item := range run.Traces {
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Span, item.Status))
	}
	lines = append(lines, "## Artifacts")
	for _, item := range run.Artifacts {
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Name, item.Path))
	}
	lines = append(lines, "## Audit")
	for _, item := range run.Audits {
		lines = append(lines, fmt.Sprintf("- %s", item.Action))
	}
	lines = append(lines,
		"## Closeout",
		fmt.Sprintf("Git Push Succeeded: %t", run.Closeout.GitPushSucceeded),
		"## Actions",
		fmt.Sprintf("Retry [retry] state=disabled target=%s reason=retry is available for failed or approval-blocked runs", run.RunID),
		"## Collaboration",
	)
	for _, comment := range run.Comments {
		lines = append(lines, comment.Body)
	}
	for _, decision := range run.Decisions {
		lines = append(lines, decision.Summary)
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	return strings.Join([]string{
		"# Repo Sync Audit",
		"",
		fmt.Sprintf("Failure Category: %s", audit.Sync.FailureCategory),
		fmt.Sprintf("Branch State: %s", audit.PullRequest.BranchState),
		fmt.Sprintf("Body State: %s", audit.PullRequest.BodyState),
		fmt.Sprintf("sync=%s, failure=%s, pr-branch=%s, pr-body=%s", audit.Sync.Status, audit.Sync.FailureCategory, audit.PullRequest.BranchState, audit.PullRequest.BodyState),
	}, "\n") + "\n"
}

func RenderTaskRunDetailPage(run TaskRun) string {
	lines := []string{
		"<html><head><title>Task Run Detail</title></head><body>",
		"<h1 data-detail=\"title\">Task Run Detail</h1>",
		"<h2>Timeline / Log Sync</h2>",
		"<h2>Reports</h2>",
	}
	for _, item := range run.Logs {
		lines = append(lines, "<p>"+item.Message+"</p>")
	}
	for _, item := range run.Traces {
		lines = append(lines, "<p>"+item.Span+"</p>")
	}
	for _, item := range run.Artifacts {
		lines = append(lines, "<p>"+item.Path+"</p>")
	}
	lines = append(lines,
		"<h2>Closeout</h2>",
		fmt.Sprintf("<p>%s</p>", run.Summary),
		"<p>complete</p>",
		"<h2>Repo Evidence</h2>",
	)
	for _, link := range run.Closeout.RunCommitLinks {
		lines = append(lines, "<p>"+link.CommitHash+"</p>")
	}
	lines = append(lines,
		"<h2>Actions</h2>",
		fmt.Sprintf("<p>Pause [pause] state=disabled target=%s reason=completed or failed runs cannot be paused</p>", run.RunID),
		"</body></html>",
	)
	return strings.Join(lines, "\n")
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
