package workflowexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
	"bigclaw-go/internal/workflow"
)

type GitSyncTelemetry struct {
	Status          string   `json:"status"`
	FailureCategory string   `json:"failure_category,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	Remote          string   `json:"remote,omitempty"`
	RemoteRef       string   `json:"remote_ref,omitempty"`
	AheadBy         int      `json:"ahead_by,omitempty"`
	BehindBy        int      `json:"behind_by,omitempty"`
	DirtyPaths      []string `json:"dirty_paths,omitempty"`
	AuthTarget      string   `json:"auth_target,omitempty"`
	Timestamp       string   `json:"timestamp,omitempty"`
}

type PullRequestFreshness struct {
	PRNumber           *int   `json:"pr_number,omitempty"`
	PRURL              string `json:"pr_url,omitempty"`
	BranchState        string `json:"branch_state"`
	BodyState          string `json:"body_state"`
	BranchHeadSHA      string `json:"branch_head_sha,omitempty"`
	PRHeadSHA          string `json:"pr_head_sha,omitempty"`
	ExpectedBodyDigest string `json:"expected_body_digest,omitempty"`
	ActualBodyDigest   string `json:"actual_body_digest,omitempty"`
	CheckedAt          string `json:"checked_at,omitempty"`
}

func (p PullRequestFreshness) Fresh() bool {
	return p.BranchState == "in-sync" && p.BodyState == "fresh"
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

func (a RepoSyncAudit) Summary() string {
	parts := []string{fmt.Sprintf("sync=%s", a.Sync.Status)}
	if strings.TrimSpace(a.Sync.FailureCategory) != "" {
		parts = append(parts, fmt.Sprintf("failure=%s", a.Sync.FailureCategory))
	}
	parts = append(parts, fmt.Sprintf("pr-branch=%s", a.PullRequest.BranchState))
	parts = append(parts, fmt.Sprintf("pr-body=%s", a.PullRequest.BodyState))
	return strings.Join(parts, ", ")
}

type PilotMetric struct {
	Name           string  `json:"name"`
	Baseline       float64 `json:"baseline"`
	Current        float64 `json:"current"`
	Target         float64 `json:"target"`
	Unit           string  `json:"unit,omitempty"`
	HigherIsBetter bool    `json:"higher_is_better,omitempty"`
}

func (m PilotMetric) MetTarget() bool {
	if m.HigherIsBetter {
		return m.Current >= m.Target
	}
	return m.Current <= m.Target
}

func (m PilotMetric) Delta() float64 {
	return m.Current - m.Baseline
}

type PilotScorecard struct {
	IssueID            string        `json:"issue_id"`
	Customer           string        `json:"customer"`
	Period             string        `json:"period"`
	Metrics            []PilotMetric `json:"metrics,omitempty"`
	MonthlyBenefit     float64       `json:"monthly_benefit"`
	MonthlyCost        float64       `json:"monthly_cost"`
	ImplementationCost float64       `json:"implementation_cost"`
	BenchmarkScore     *int          `json:"benchmark_score,omitempty"`
	BenchmarkPassed    *bool         `json:"benchmark_passed,omitempty"`
}

func (p PilotScorecard) MetricsMet() int {
	total := 0
	for _, metric := range p.Metrics {
		if metric.MetTarget() {
			total++
		}
	}
	return total
}

func (p PilotScorecard) MonthlyNetValue() float64 {
	return p.MonthlyBenefit - p.MonthlyCost
}

func (p PilotScorecard) AnnualizedROI() float64 {
	if p.ImplementationCost == 0 {
		return 0
	}
	return ((p.MonthlyNetValue() * 12) / p.ImplementationCost) * 100
}

func (p PilotScorecard) PaybackMonths() *float64 {
	if p.MonthlyNetValue() <= 0 {
		return nil
	}
	value := p.ImplementationCost / p.MonthlyNetValue()
	return &value
}

func (p PilotScorecard) Recommendation() string {
	if p.MonthlyNetValue() <= 0 {
		return "hold"
	}
	if len(p.Metrics) > 0 && p.MetricsMet() == len(p.Metrics) {
		if p.BenchmarkPassed == nil || *p.BenchmarkPassed {
			return "go"
		}
	}
	return "iterate"
}

type RunCloseout struct {
	ValidationEvidence []string       `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool           `json:"git_push_succeeded"`
	GitPushOutput      string         `json:"git_push_output,omitempty"`
	GitLogStatOutput   string         `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *RepoSyncAudit `json:"repo_sync_audit,omitempty"`
	RunCommitLinks     []repo.RunCommitLink `json:"run_commit_links,omitempty"`
	Timestamp          string         `json:"timestamp"`
	Complete           bool           `json:"complete"`
}

type TraceEntry struct {
	Span       string         `json:"span"`
	Status     string         `json:"status"`
	Timestamp  string         `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type ArtifactRecord struct {
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path"`
	SHA256    string         `json:"sha256,omitempty"`
	Timestamp string         `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Outcome   string         `json:"outcome"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	RunID     string           `json:"run_id"`
	TaskID    string           `json:"task_id"`
	Source    string           `json:"source"`
	Title     string           `json:"title"`
	Medium    string           `json:"medium"`
	StartedAt string           `json:"started_at"`
	EndedAt   string           `json:"ended_at,omitempty"`
	Status    string           `json:"status"`
	Summary   string           `json:"summary"`
	Logs      []map[string]any `json:"logs,omitempty"`
	Traces    []TraceEntry     `json:"traces,omitempty"`
	Artifacts []ArtifactRecord `json:"artifacts,omitempty"`
	Audits    []AuditEntry     `json:"audits,omitempty"`
	Comments  []repo.CollaborationComment `json:"comments,omitempty"`
	Decisions []repo.DecisionNote         `json:"decisions,omitempty"`
	Closeout  RunCloseout      `json:"closeout"`
}

func NewTaskRun(task domain.Task, runID, medium string) TaskRun {
	return TaskRun{
		RunID:     strings.TrimSpace(runID),
		TaskID:    task.ID,
		Source:    task.Source,
		Title:     task.Title,
		Medium:    medium,
		StartedAt: nowUTC(),
		Status:    "running",
	}
}

func (r *TaskRun) Log(level, message string, context map[string]any) {
	r.Logs = append(r.Logs, map[string]any{
		"level":     level,
		"message":   message,
		"timestamp": nowUTC(),
		"context":   cloneMap(context),
	})
}

func (r *TaskRun) Trace(span, status string, attributes map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{Span: span, Status: status, Timestamp: nowUTC(), Attributes: cloneMap(attributes)})
}

func (r *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{Action: action, Actor: actor, Outcome: outcome, Timestamp: nowUTC(), Details: cloneMap(details)})
}

func (r *TaskRun) RegisterArtifact(name, kind, path string, metadata map[string]any) error {
	digest := ""
	if strings.TrimSpace(path) != "" {
		body, err := os.ReadFile(path)
		if err == nil {
			sum := sha256.Sum256(body)
			digest = hex.EncodeToString(sum[:])
		}
	}
	r.Artifacts = append(r.Artifacts, ArtifactRecord{
		Name:      name,
		Kind:      kind,
		Path:      path,
		SHA256:    digest,
		Timestamp: nowUTC(),
		Metadata:  cloneMap(metadata),
	})
	r.Audit("artifact.registered", "task-run", "recorded", map[string]any{
		"artifact_name": name,
		"artifact_kind": kind,
		"path":          path,
		"sha256":        digest,
	})
	return nil
}

func (r *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit) {
	complete := len(validationEvidence) > 0 && gitPushSucceeded && strings.TrimSpace(gitLogStatOutput) != ""
	r.Closeout = RunCloseout{
		ValidationEvidence: append([]string(nil), validationEvidence...),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		Timestamp:          nowUTC(),
		Complete:           complete,
	}
	r.Audit("closeout.recorded", "task-run", "recorded", map[string]any{
		"validation_evidence_count": len(validationEvidence),
		"git_push_succeeded":        gitPushSucceeded,
		"git_log_stat_captured":     strings.TrimSpace(gitLogStatOutput) != "",
		"has_repo_sync_audit":       repoSyncAudit != nil,
	})
}

func (r *TaskRun) AddComment(author, body string, mentions []string, anchor string) repo.CollaborationComment {
	comment := repo.CollaborationComment{
		CommentID: fmt.Sprintf("comment-%d", len(r.Comments)+1),
		Author:    strings.TrimSpace(author),
		Body:      strings.TrimSpace(body),
		CreatedAt: nowUTC(),
		Anchor:    strings.TrimSpace(anchor),
		Status:    "open",
	}
	r.Comments = append(r.Comments, comment)
	r.Audit("collaboration.comment", strings.TrimSpace(author), "recorded", map[string]any{
		"comment_id": comment.CommentID,
		"body":       comment.Body,
		"mentions":   append([]string(nil), mentions...),
		"anchor":     comment.Anchor,
	})
	return comment
}

func (r *TaskRun) AddDecisionNote(author, summary, outcome string, mentions []string, relatedCommentIDs []string, followUp string) repo.DecisionNote {
	decision := repo.DecisionNote{
		DecisionID: fmt.Sprintf("decision-%d", len(r.Decisions)+1),
		Author:     strings.TrimSpace(author),
		Outcome:    strings.TrimSpace(outcome),
		Summary:    strings.TrimSpace(summary),
		RecordedAt: nowUTC(),
		Mentions:   append([]string(nil), mentions...),
		FollowUp:   strings.TrimSpace(followUp),
	}
	r.Decisions = append(r.Decisions, decision)
	r.Audit("collaboration.decision", strings.TrimSpace(author), strings.TrimSpace(outcome), map[string]any{
		"decision_id":         decision.DecisionID,
		"summary":             decision.Summary,
		"mentions":            append([]string(nil), mentions...),
		"related_comment_ids": append([]string(nil), relatedCommentIDs...),
		"follow_up":           decision.FollowUp,
	})
	return decision
}

func (r *TaskRun) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
	r.EndedAt = nowUTC()
}

type Ledger struct {
	Path string
}

func (l Ledger) Load() ([]map[string]any, error) {
	body, err := os.ReadFile(l.Path)
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

func (l Ledger) LoadRuns() ([]TaskRun, error) {
	entries, err := l.Load()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	var runs []TaskRun
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l Ledger) Upsert(run TaskRun) error {
	entries, err := l.Load()
	if err != nil {
		return err
	}
	body, err := json.Marshal(run)
	if err != nil {
		return err
	}
	var serialized map[string]any
	if err := json.Unmarshal(body, &serialized); err != nil {
		return err
	}
	updated := false
	for i, entry := range entries {
		if entry["run_id"] == run.RunID {
			entries[i] = serialized
			updated = true
			break
		}
	}
	if !updated {
		entries = append(entries, serialized)
	}
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	output, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, output, 0o644)
}

type CollaborationThread struct {
	Surface      string                      `json:"surface"`
	TargetID     string                      `json:"target_id"`
	Comments     []repo.CollaborationComment `json:"comments,omitempty"`
	Decisions    []repo.DecisionNote         `json:"decisions,omitempty"`
	MentionCount int                         `json:"mention_count"`
}

type Decision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type ExecutionRecord struct {
	Decision            Decision                              `json:"decision"`
	Run                 TaskRun                               `json:"run"`
	OrchestrationPlan   *workflow.OrchestrationPlan           `json:"orchestration_plan,omitempty"`
	OrchestrationPolicy *workflow.OrchestrationPolicyDecision `json:"orchestration_policy,omitempty"`
	HandoffRequest      *workflow.HandoffRequest              `json:"handoff_request,omitempty"`
	ReportPath          string                                `json:"report_path,omitempty"`
}

type RunResult struct {
	Execution               ExecutionRecord             `json:"execution"`
	Acceptance              workflow.AcceptanceDecision `json:"acceptance"`
	Journal                 workflow.WorkpadJournal     `json:"journal"`
	JournalPath             string                      `json:"journal_path,omitempty"`
	OrchestrationReportPath string                      `json:"orchestration_report_path,omitempty"`
	OrchestrationCanvasPath string                      `json:"orchestration_canvas_path,omitempty"`
	PilotReportPath         string                      `json:"pilot_report_path,omitempty"`
	RepoSyncReportPath      string                      `json:"repo_sync_report_path,omitempty"`
}

type Engine struct {
	Gate workflow.AcceptanceGate
	Now  func() time.Time
}

func (e Engine) Run(task domain.Task, runID string, ledger Ledger, reportPath, journalPath string, validationEvidence []string, pilotScorecard *PilotScorecard, pilotReportPath, orchestrationReportPath, orchestrationCanvasPath string, repoSyncAudit *RepoSyncAudit, repoSyncReportPath string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string) (RunResult, error) {
	journal := workflow.WorkpadJournal{TaskID: task.ID, RunID: runID, Now: e.Now}
	journal.Record("intake", "recorded", map[string]any{"source": task.Source})

	decision := decide(task)
	rawPlan := workflow.CrossDepartmentOrchestrator{}.Plan(task)
	orchestrationPlan, policy := workflow.PremiumOrchestrationPolicy{}.Apply(task, rawPlan)
	handoffRequest := buildHandoffRequest(decision, orchestrationPlan, policy)
	run := NewTaskRun(task, runID, decision.Medium)
	run.Log("info", "task received", map[string]any{"source": task.Source})
	run.Trace("scheduler.decide", map[bool]string{true: "ok", false: "pending"}[decision.Approved], map[string]any{"approved": decision.Approved, "medium": decision.Medium})
	run.Trace("orchestration.plan", orchestrationPlan.CollaborationMode, map[string]any{"departments": orchestrationPlan.Departments()})
	if policy.UpgradeRequired {
		run.Trace("orchestration.policy", "upgrade-required", map[string]any{"tier": policy.Tier, "blocked_departments": policy.BlockedDepartments})
	} else {
		run.Trace("orchestration.policy", "ok", map[string]any{"tier": policy.Tier})
	}
	run.Audit("scheduler.decision", "workflow-engine", map[bool]string{true: "approved", false: "pending"}[decision.Approved], map[string]any{"reason": decision.Reason})
	run.Audit("orchestration.plan", "workflow-engine", "recorded", map[string]any{"departments": orchestrationPlan.Departments()})
	run.Audit("orchestration.policy", "workflow-engine", map[bool]string{true: "upgrade-required", false: "enabled"}[policy.UpgradeRequired], map[string]any{"tier": policy.Tier, "entitlement_status": policy.EntitlementStatus})
	if handoffRequest != nil {
		run.Audit("orchestration.handoff", "workflow-engine", handoffRequest.Status, map[string]any{"target_team": handoffRequest.TargetTeam, "reason": handoffRequest.Reason})
	}

	finalStatus := "approved"
	if !decision.Approved {
		finalStatus = "needs-approval"
	}
	run.Finalize(finalStatus, decision.Reason)
	execution := ExecutionRecord{
		Decision:            decision,
		Run:                 run,
		OrchestrationPlan:   &orchestrationPlan,
		OrchestrationPolicy: &policy,
		HandoffRequest:      handoffRequest,
		ReportPath:          reportPath,
	}
	journal.Record("execution", execution.Run.Status, map[string]any{"medium": execution.Decision.Medium, "approved": execution.Decision.Approved})

	var resolvedOrchestrationReport, resolvedCanvas, resolvedPilot, resolvedRepoSync string
	if strings.TrimSpace(orchestrationReportPath) != "" {
		resolvedOrchestrationReport = orchestrationReportPath
		if err := writeReport(resolvedOrchestrationReport, renderOrchestrationPlan(orchestrationPlan, &policy, handoffRequest)); err != nil {
			return RunResult{}, err
		}
		if err := execution.Run.RegisterArtifact("cross-department-orchestration", "report", resolvedOrchestrationReport, map[string]any{"format": "markdown"}); err != nil {
			return RunResult{}, err
		}
		if strings.TrimSpace(orchestrationCanvasPath) != "" {
			resolvedCanvas = orchestrationCanvasPath
			if err := writeReport(resolvedCanvas, renderOrchestrationCanvas(orchestrationPlan, policy, handoffRequest)); err != nil {
				return RunResult{}, err
			}
			if err := execution.Run.RegisterArtifact("orchestration-canvas", "report", resolvedCanvas, map[string]any{"format": "markdown"}); err != nil {
				return RunResult{}, err
			}
		}
		journal.Record("orchestration", orchestrationPlan.CollaborationMode, map[string]any{"departments": orchestrationPlan.Departments(), "handoff_team": teamOrNone(handoffRequest)})
	}

	pilotRecommendation := ""
	if pilotScorecard != nil && strings.TrimSpace(pilotReportPath) != "" {
		resolvedPilot = pilotReportPath
		pilotRecommendation = pilotScorecard.Recommendation()
		if err := writeReport(resolvedPilot, RenderPilotScorecard(*pilotScorecard)); err != nil {
			return RunResult{}, err
		}
		if err := execution.Run.RegisterArtifact("pilot-scorecard", "report", resolvedPilot, map[string]any{"recommendation": pilotRecommendation}); err != nil {
			return RunResult{}, err
		}
		journal.Record("pilot-scorecard", pilotRecommendation, map[string]any{"metrics_met": pilotScorecard.MetricsMet(), "metrics_total": len(pilotScorecard.Metrics), "annualized_roi": round1(pilotScorecard.AnnualizedROI())})
	}

	if repoSyncAudit != nil {
		execution.Run.Audit("repo.sync", "workflow-engine", repoSyncAudit.Sync.Status, map[string]any{
			"failure_category": repoSyncAudit.Sync.FailureCategory,
			"summary":          repoSyncAudit.Sync.Summary,
			"branch":           repoSyncAudit.Sync.Branch,
			"remote_ref":       repoSyncAudit.Sync.RemoteRef,
			"ahead_by":         repoSyncAudit.Sync.AheadBy,
			"behind_by":        repoSyncAudit.Sync.BehindBy,
			"dirty_paths":      repoSyncAudit.Sync.DirtyPaths,
			"auth_target":      repoSyncAudit.Sync.AuthTarget,
		})
		execution.Run.Audit("repo.pr-freshness", "workflow-engine", map[bool]string{true: "fresh", false: "stale"}[repoSyncAudit.PullRequest.Fresh()], map[string]any{
			"pr_number":       repoSyncAudit.PullRequest.PRNumber,
			"pr_url":          repoSyncAudit.PullRequest.PRURL,
			"branch_state":    repoSyncAudit.PullRequest.BranchState,
			"body_state":      repoSyncAudit.PullRequest.BodyState,
			"branch_head_sha": repoSyncAudit.PullRequest.BranchHeadSHA,
			"pr_head_sha":     repoSyncAudit.PullRequest.PRHeadSHA,
		})
		journal.Record("repo-sync", repoSyncAudit.Sync.Status, map[string]any{"failure_category": repoSyncAudit.Sync.FailureCategory, "branch_state": repoSyncAudit.PullRequest.BranchState, "body_state": repoSyncAudit.PullRequest.BodyState})
		if strings.TrimSpace(repoSyncReportPath) != "" {
			resolvedRepoSync = repoSyncReportPath
			if err := writeReport(resolvedRepoSync, RenderRepoSyncAuditReport(*repoSyncAudit)); err != nil {
				return RunResult{}, err
			}
			if err := execution.Run.RegisterArtifact("repo-sync-audit", "report", resolvedRepoSync, map[string]any{"sync_status": repoSyncAudit.Sync.Status}); err != nil {
				return RunResult{}, err
			}
		}
	}

	acceptance := e.Gate.Evaluate(task, workflow.ExecutionOutcome{Approved: decision.Approved, Status: execution.Run.Status}, validationEvidence, nil, pilotRecommendation)
	journal.Record("acceptance", acceptance.Status, map[string]any{"passed": acceptance.Passed, "missing_acceptance_criteria": acceptance.MissingAcceptanceCriteria, "missing_validation_steps": acceptance.MissingValidationSteps})

	execution.Run.RecordCloseout(validationEvidence, gitPushSucceeded, gitPushOutput, gitLogStatOutput, repoSyncAudit)
	if err := ledger.Upsert(execution.Run); err != nil {
		return RunResult{}, err
	}
	journal.Record("closeout", map[bool]string{true: "complete", false: "pending"}[execution.Run.Closeout.Complete], map[string]any{"validation_evidence": append([]string(nil), validationEvidence...)})

	resolvedJournal := ""
	if strings.TrimSpace(journalPath) != "" {
		path, err := journal.Write(journalPath)
		if err != nil {
			return RunResult{}, err
		}
		resolvedJournal = path
	}
	return RunResult{
		Execution:               execution,
		Acceptance:              acceptance,
		Journal:                 journal,
		JournalPath:             resolvedJournal,
		OrchestrationReportPath: resolvedOrchestrationReport,
		OrchestrationCanvasPath: resolvedCanvas,
		PilotReportPath:         resolvedPilot,
		RepoSyncReportPath:      resolvedRepoSync,
	}, nil
}

func RenderPilotScorecard(scorecard PilotScorecard) string {
	lines := []string{
		"# Pilot Scorecard",
		"",
		fmt.Sprintf("- Issue ID: %s", scorecard.IssueID),
		fmt.Sprintf("- Customer: %s", scorecard.Customer),
		fmt.Sprintf("- Period: %s", scorecard.Period),
		fmt.Sprintf("- Recommendation: %s", scorecard.Recommendation()),
		fmt.Sprintf("- Metrics Met: %d/%d", scorecard.MetricsMet(), len(scorecard.Metrics)),
		fmt.Sprintf("- Monthly Net Value: %.2f", scorecard.MonthlyNetValue()),
		fmt.Sprintf("- Annualized ROI: %.1f%%", scorecard.AnnualizedROI()),
	}
	if payback := scorecard.PaybackMonths(); payback == nil {
		lines = append(lines, "- Payback Months: n/a")
	} else {
		lines = append(lines, fmt.Sprintf("- Payback Months: %.1f", *payback))
	}
	if scorecard.BenchmarkScore != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Score: %d", *scorecard.BenchmarkScore))
	}
	if scorecard.BenchmarkPassed != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Passed: %t", *scorecard.BenchmarkPassed))
	}
	lines = append(lines, "", "## KPI Progress", "")
	for _, metric := range scorecard.Metrics {
		comp := ">="
		if !metric.HigherIsBetter {
			comp = "<="
		}
		unit := ""
		if metric.Unit != "" {
			unit = " " + metric.Unit
		}
		lines = append(lines, fmt.Sprintf("- %s: baseline=%.0f%s current=%.0f%s target%s%.0f%s delta=%+.2f%s met=%t", metric.Name, metric.Baseline, unit, metric.Current, unit, comp, metric.Target, unit, metric.Delta(), unit, metric.MetTarget()))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	prNumber := "unknown"
	if audit.PullRequest.PRNumber != nil {
		prNumber = fmt.Sprintf("%d", *audit.PullRequest.PRNumber)
	}
	lines := []string{
		"# Repo Sync Audit",
		"",
		"## Sync Status",
		"",
		fmt.Sprintf("- Status: %s", valueOr(audit.Sync.Status, "unknown")),
		fmt.Sprintf("- Failure Category: %s", valueOr(audit.Sync.FailureCategory, "none")),
		fmt.Sprintf("- Summary: %s", valueOr(audit.Sync.Summary, "none")),
		fmt.Sprintf("- Branch: %s", valueOr(audit.Sync.Branch, "unknown")),
		fmt.Sprintf("- Remote: %s", valueOr(audit.Sync.Remote, "origin")),
		fmt.Sprintf("- Remote Ref: %s", valueOr(audit.Sync.RemoteRef, "unknown")),
		fmt.Sprintf("- Ahead By: %d", audit.Sync.AheadBy),
		fmt.Sprintf("- Behind By: %d", audit.Sync.BehindBy),
		fmt.Sprintf("- Dirty Paths: %s", joinedOrNone(audit.Sync.DirtyPaths)),
		fmt.Sprintf("- Auth Target: %s", valueOr(audit.Sync.AuthTarget, "none")),
		fmt.Sprintf("- Checked At: %s", valueOr(audit.Sync.Timestamp, "unknown")),
		"",
		"## Pull Request Freshness",
		"",
		fmt.Sprintf("- PR Number: %s", prNumber),
		fmt.Sprintf("- PR URL: %s", valueOr(audit.PullRequest.PRURL, "none")),
		fmt.Sprintf("- Branch State: %s", valueOr(audit.PullRequest.BranchState, "unknown")),
		fmt.Sprintf("- Body State: %s", valueOr(audit.PullRequest.BodyState, "unknown")),
		fmt.Sprintf("- Branch Head SHA: %s", valueOr(audit.PullRequest.BranchHeadSHA, "unknown")),
		fmt.Sprintf("- PR Head SHA: %s", valueOr(audit.PullRequest.PRHeadSHA, "unknown")),
		fmt.Sprintf("- Expected Body Digest: %s", valueOr(audit.PullRequest.ExpectedBodyDigest, "unknown")),
		fmt.Sprintf("- Actual Body Digest: %s", valueOr(audit.PullRequest.ActualBodyDigest, "unknown")),
		fmt.Sprintf("- Checked At: %s", valueOr(audit.PullRequest.CheckedAt, "unknown")),
		"",
		"## Summary",
		"",
		fmt.Sprintf("- %s", audit.Summary()),
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildCollaborationThreadFromAudits(audits []AuditEntry, surface, targetID string) *CollaborationThread {
	thread := &CollaborationThread{
		Surface:  strings.TrimSpace(surface),
		TargetID: strings.TrimSpace(targetID),
	}
	mentionSet := map[string]struct{}{}
	for _, audit := range audits {
		switch audit.Action {
		case "collaboration.comment":
			comment := repo.CollaborationComment{
				CommentID: stringValue(audit.Details, "comment_id"),
				Author:    audit.Actor,
				Body:      stringValue(audit.Details, "body"),
				CreatedAt: audit.Timestamp,
				Anchor:    stringValue(audit.Details, "anchor"),
				Status:    "open",
			}
			thread.Comments = append(thread.Comments, comment)
			for _, mention := range stringSliceValue(audit.Details, "mentions") {
				mentionSet[mention] = struct{}{}
			}
		case "collaboration.decision":
			decision := repo.DecisionNote{
				DecisionID: stringValue(audit.Details, "decision_id"),
				Author:     audit.Actor,
				Outcome:    audit.Outcome,
				Summary:    stringValue(audit.Details, "summary"),
				RecordedAt: audit.Timestamp,
				Mentions:   stringSliceValue(audit.Details, "mentions"),
				FollowUp:   stringValue(audit.Details, "follow_up"),
			}
			thread.Decisions = append(thread.Decisions, decision)
			for _, mention := range decision.Mentions {
				mentionSet[mention] = struct{}{}
			}
		}
	}
	sort.Slice(thread.Comments, func(i, j int) bool { return thread.Comments[i].CreatedAt < thread.Comments[j].CreatedAt })
	sort.Slice(thread.Decisions, func(i, j int) bool { return thread.Decisions[i].RecordedAt < thread.Decisions[j].RecordedAt })
	thread.MentionCount = len(mentionSet)
	if len(thread.Comments) == 0 && len(thread.Decisions) == 0 {
		return nil
	}
	return thread
}

func RenderTaskRunReport(run TaskRun) string {
	lines := []string{
		"# Task Run Report",
		"",
		fmt.Sprintf("Run ID: %s", run.RunID),
		fmt.Sprintf("Task ID: %s", run.TaskID),
		fmt.Sprintf("Medium: %s", run.Medium),
		fmt.Sprintf("Status: %s", run.Status),
		fmt.Sprintf("Summary: %s", run.Summary),
		"",
		"## Logs",
	}
	for _, item := range run.Logs {
		lines = append(lines, fmt.Sprintf("- %s: %s", valueOr(anyString(item["level"]), "info"), anyString(item["message"])))
	}
	lines = append(lines, "", "## Trace")
	for _, item := range run.Traces {
		lines = append(lines, fmt.Sprintf("- %s status=%s", item.Span, item.Status))
	}
	lines = append(lines, "", "## Artifacts")
	for _, item := range run.Artifacts {
		lines = append(lines, fmt.Sprintf("- %s kind=%s path=%s", item.Name, item.Kind, item.Path))
	}
	lines = append(lines, "", "## Audit")
	for _, item := range run.Audits {
		lines = append(lines, fmt.Sprintf("- %s outcome=%s", item.Action, item.Outcome))
	}
	lines = append(lines,
		"",
		"## Closeout",
		fmt.Sprintf("- Git Push Succeeded: %t", run.Closeout.GitPushSucceeded),
		fmt.Sprintf("- Complete: %t", run.Closeout.Complete),
		"",
		"## Actions",
		fmt.Sprintf("Retry [retry] state=%s target=%s reason=%s", actionState(run.Status, "retry"), run.RunID, retryReason(run.Status)),
		fmt.Sprintf("Pause [pause] state=%s target=%s reason=%s", actionState(run.Status, "pause"), run.RunID, pauseReason(run.Status)),
	)
	thread := BuildCollaborationThreadFromAudits(run.Audits, "run", run.RunID)
	if thread != nil {
		lines = append(lines, "", "## Collaboration")
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
	var b strings.Builder
	b.WriteString("<html><head><title>Task Run Detail</title></head><body>\n")
	b.WriteString("<h1 data-detail=\"title\">Task Run Detail</h1>\n")
	b.WriteString("<h2>Timeline / Log Sync</h2>\n")
	for _, item := range run.Logs {
		b.WriteString("<p>" + html.EscapeString(escapeScript(anyString(item["message"]))) + "</p>\n")
	}
	for _, item := range run.Traces {
		b.WriteString("<p>" + html.EscapeString(item.Span) + "</p>\n")
	}
	b.WriteString("<h2>Reports</h2>\n")
	for _, item := range run.Artifacts {
		b.WriteString("<p>" + html.EscapeString(item.Path) + "</p>\n")
	}
	b.WriteString("<h2>Closeout</h2>\n")
	b.WriteString("<p>complete</p>\n")
	b.WriteString("<h2>Repo Evidence</h2>\n")
	for _, link := range run.Closeout.RunCommitLinks {
		b.WriteString("<p>" + html.EscapeString(link.CommitHash) + "</p>\n")
	}
	b.WriteString("<h2>Actions</h2>\n")
	b.WriteString("<p>Pause [pause] state=" + html.EscapeString(actionState(run.Status, "pause")) + " target=" + html.EscapeString(run.RunID) + " reason=" + html.EscapeString(pauseReason(run.Status)) + "</p>\n")
	thread := BuildCollaborationThreadFromAudits(run.Audits, "run", run.RunID)
	if thread != nil {
		b.WriteString("<h2>Collaboration</h2>\n")
		for _, comment := range thread.Comments {
			b.WriteString("<p>" + html.EscapeString(comment.Body) + "</p>\n")
		}
		for _, decision := range thread.Decisions {
			b.WriteString("<p>" + html.EscapeString(decision.Summary) + "</p>\n")
		}
	}
	b.WriteString("<p>" + html.EscapeString(run.Summary) + "</p>\n")
	b.WriteString("</body></html>\n")
	return b.String()
}

func renderOrchestrationPlan(plan workflow.OrchestrationPlan, policy *workflow.OrchestrationPolicyDecision, handoff *workflow.HandoffRequest) string {
	lines := []string{
		"# Cross-Department Orchestration Plan",
		"",
		fmt.Sprintf("- Task ID: %s", plan.TaskID),
		fmt.Sprintf("- Collaboration Mode: %s", plan.CollaborationMode),
		fmt.Sprintf("- Departments: %s", joinedOrNone(plan.Departments())),
	}
	if policy != nil {
		lines = append(lines,
			fmt.Sprintf("- Tier: %s", policy.Tier),
			fmt.Sprintf("- Upgrade Required: %t", policy.UpgradeRequired),
			fmt.Sprintf("- Human Handoff Team: %s", teamOrNone(handoff)),
		)
	}
	lines = append(lines, "", "## Handoffs", "")
	for _, handoff := range plan.Handoffs {
		lines = append(lines, fmt.Sprintf("- %s: reason=%s tools=%s approvals=%s", handoff.Department, handoff.Reason, joinedOrNone(handoff.RequiredTools), joinedOrNone(handoff.Approvals)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderOrchestrationCanvas(plan workflow.OrchestrationPlan, policy workflow.OrchestrationPolicyDecision, handoff *workflow.HandoffRequest) string {
	recommendation := "continue"
	if policy.UpgradeRequired {
		recommendation = "resolve-entitlement-gap"
	} else if handoff != nil {
		recommendation = "handoff"
	}
	return strings.Join([]string{
		"# Orchestration Canvas",
		"",
		fmt.Sprintf("- Task ID: %s", plan.TaskID),
		fmt.Sprintf("- Recommendation: %s", recommendation),
		fmt.Sprintf("- Collaboration Mode: %s", plan.CollaborationMode),
		fmt.Sprintf("- Handoff Team: %s", teamOrNone(handoff)),
	}, "\n") + "\n"
}

func decide(task domain.Task) Decision {
	switch {
	case task.RiskLevel == domain.RiskHigh:
		return Decision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	case hasTool(task.RequiredTools, "browser"):
		return Decision{Medium: "browser", Approved: true, Reason: "browser automation task"}
	default:
		return Decision{Medium: "docker", Approved: true, Reason: "default low risk path"}
	}
}

func buildHandoffRequest(decision Decision, plan workflow.OrchestrationPlan, policy workflow.OrchestrationPolicyDecision) *workflow.HandoffRequest {
	if !decision.Approved {
		return &workflow.HandoffRequest{TargetTeam: "security", Reason: decision.Reason, Status: "pending", RequiredApprovals: []string{"security-review"}}
	}
	if policy.UpgradeRequired {
		return &workflow.HandoffRequest{TargetTeam: "operations", Reason: policy.Reason, Status: "pending", RequiredApprovals: []string{"ops-manager"}}
	}
	return nil
}

func writeReport(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func hasTool(tools []string, want string) bool {
	for _, tool := range tools {
		if strings.EqualFold(strings.TrimSpace(tool), want) {
			return true
		}
	}
	return false
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return anyString(values[key])
}

func stringSliceValue(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := anyString(item); value != "" {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

func actionState(status, action string) string {
	status = strings.TrimSpace(status)
	switch action {
	case "retry":
		if status == "failed" || status == "needs-approval" {
			return "enabled"
		}
		return "disabled"
	case "pause":
		if status == "completed" || status == "failed" || status == "approved" || status == "succeeded" {
			return "disabled"
		}
		return "enabled"
	default:
		return "disabled"
	}
}

func retryReason(status string) string {
	if actionState(status, "retry") == "enabled" {
		return "retry available"
	}
	return "retry is available for failed or approval-blocked runs"
}

func pauseReason(status string) string {
	if actionState(status, "pause") == "enabled" {
		return "pause available"
	}
	return "completed or failed runs cannot be paused"
}

func escapeScript(value string) string {
	return strings.ReplaceAll(value, "</script>", "<\\/script>")
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func round1(value float64) float64 {
	if value >= 0 {
		return float64(int(value*10+0.5)) / 10
	}
	return float64(int(value*10-0.5)) / 10
}

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func teamOrNone(request *workflow.HandoffRequest) string {
	if request == nil || strings.TrimSpace(request.TargetTeam) == "" {
		return "none"
	}
	return request.TargetTeam
}
