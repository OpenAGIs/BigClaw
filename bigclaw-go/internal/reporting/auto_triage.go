package reporting

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/observability"
)

type TriageFinding struct {
	RunID      string          `json:"run_id"`
	TaskID     string          `json:"task_id"`
	Source     string          `json:"source"`
	Severity   string          `json:"severity"`
	Owner      string          `json:"owner"`
	Status     string          `json:"status"`
	Reason     string          `json:"reason"`
	NextAction string          `json:"next_action"`
	Actions    []ConsoleAction `json:"actions,omitempty"`
}

type TriageSimilarityEvidence struct {
	RelatedRunID  string  `json:"related_run_id"`
	RelatedTaskID string  `json:"related_task_id"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
}

type TriageSuggestion struct {
	Label          string                     `json:"label"`
	Action         string                     `json:"action"`
	Owner          string                     `json:"owner"`
	Confidence     float64                    `json:"confidence"`
	Evidence       []TriageSimilarityEvidence `json:"evidence,omitempty"`
	FeedbackStatus string                     `json:"feedback_status"`
}

type TriageInboxItem struct {
	RunID       string             `json:"run_id"`
	TaskID      string             `json:"task_id"`
	Source      string             `json:"source"`
	Status      string             `json:"status"`
	Severity    string             `json:"severity"`
	Owner       string             `json:"owner"`
	Summary     string             `json:"summary"`
	SubmittedAt string             `json:"submitted_at"`
	Suggestions []TriageSuggestion `json:"suggestions,omitempty"`
}

type AutoTriageCenter struct {
	Name     string                 `json:"name"`
	Period   string                 `json:"period"`
	Findings []TriageFinding        `json:"findings,omitempty"`
	Inbox    []TriageInboxItem      `json:"inbox,omitempty"`
	Feedback []TriageFeedbackRecord `json:"feedback,omitempty"`
}

func (c AutoTriageCenter) FlaggedRuns() int { return len(c.Findings) }
func (c AutoTriageCenter) InboxSize() int   { return len(c.Inbox) }

func (c AutoTriageCenter) SeverityCounts() map[string]int {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0}
	for _, finding := range c.Findings {
		counts[finding.Severity]++
	}
	return counts
}

func (c AutoTriageCenter) OwnerCounts() map[string]int {
	counts := map[string]int{"security": 0, "engineering": 0, "operations": 0}
	for _, finding := range c.Findings {
		counts[finding.Owner]++
	}
	return counts
}

func (c AutoTriageCenter) FeedbackCounts() map[string]int {
	counts := map[string]int{"accepted": 0, "rejected": 0, "pending": 0}
	for _, record := range c.Feedback {
		counts[record.Decision]++
	}
	pending := 0
	for _, item := range c.Inbox {
		for _, suggestion := range item.Suggestions {
			if suggestion.FeedbackStatus == "pending" {
				pending++
			}
		}
	}
	counts["pending"] = pending
	return counts
}

func (c AutoTriageCenter) Recommendation() string {
	severity := c.SeverityCounts()
	feedback := c.FeedbackCounts()
	if severity["critical"] > 0 {
		return "immediate-attention"
	}
	if feedback["rejected"] > feedback["accepted"] {
		return "retune-suggestions"
	}
	if severity["high"] > 0 {
		return "review-queue"
	}
	return "monitor"
}

func BuildAutoTriageCenter(runs []observability.TaskRun, name string, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	findings := make([]TriageFinding, 0)
	inbox := make([]TriageInboxItem, 0)
	for _, run := range runs {
		if !runRequiresTriage(run) {
			continue
		}
		severity := triageSeverity(run)
		owner := triageOwner(run)
		reason := triageReason(run)
		nextAction := triageNextAction(severity, owner)
		suggestions := buildTriageSuggestions(run, runs, severity, owner, feedback)
		findings = append(findings, TriageFinding{
			RunID:      run.RunID,
			TaskID:     run.TaskID,
			Source:     run.Source,
			Severity:   severity,
			Owner:      owner,
			Status:     run.Status,
			Reason:     reason,
			NextAction: nextAction,
			Actions: workflowConsoleActions(
				run.RunID,
				severity == "critical" && owner != "security",
				"retry available after owner review",
				run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
				"completed or failed runs cannot be paused",
				owner != "security",
				"security-owned findings stay with the security queue",
				true,
				"",
			),
		})
		inbox = append(inbox, TriageInboxItem{
			RunID:       run.RunID,
			TaskID:      run.TaskID,
			Source:      run.Source,
			Status:      run.Status,
			Severity:    severity,
			Owner:       owner,
			Summary:     reason,
			SubmittedAt: firstNonEmptyString(run.EndedAt, run.StartedAt),
			Suggestions: suggestions,
		})
	}
	severityRank := map[string]int{"critical": 0, "high": 1, "medium": 2}
	sort.Slice(findings, func(i, j int) bool {
		if severityRank[findings[i].Severity] == severityRank[findings[j].Severity] {
			if findings[i].Owner == findings[j].Owner {
				return findings[i].RunID < findings[j].RunID
			}
			return findings[i].Owner < findings[j].Owner
		}
		return severityRank[findings[i].Severity] < severityRank[findings[j].Severity]
	})
	sort.Slice(inbox, func(i, j int) bool {
		if severityRank[inbox[i].Severity] == severityRank[inbox[j].Severity] {
			if inbox[i].Owner == inbox[j].Owner {
				return inbox[i].RunID < inbox[j].RunID
			}
			return inbox[i].Owner < inbox[j].Owner
		}
		return severityRank[inbox[i].Severity] < severityRank[inbox[j].Severity]
	})
	return AutoTriageCenter{Name: name, Period: period, Findings: findings, Inbox: inbox, Feedback: feedback}
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns *int, view *SharedViewContext) string {
	total := center.FlaggedRuns()
	if totalRuns != nil {
		total = *totalRuns
	}
	severity := center.SeverityCounts()
	owners := center.OwnerCounts()
	feedback := center.FeedbackCounts()
	lines := []string{
		"# Auto Triage Center",
		"",
		"- Center: " + center.Name,
		"- Period: " + center.Period,
		fmt.Sprintf("- Flagged Runs: %d", center.FlaggedRuns()),
		fmt.Sprintf("- Inbox Size: %d", center.InboxSize()),
		fmt.Sprintf("- Total Runs: %d", total),
		"- Recommendation: " + center.Recommendation(),
		fmt.Sprintf("- Severity Mix: critical=%d high=%d medium=%d", severity["critical"], severity["high"], severity["medium"]),
		fmt.Sprintf("- Owner Mix: security=%d engineering=%d operations=%d", owners["security"], owners["engineering"], owners["operations"]),
		fmt.Sprintf("- Feedback Loop: accepted=%d rejected=%d pending=%d", feedback["accepted"], feedback["rejected"], feedback["pending"]),
		"",
		"## Queue",
		"",
	}
	lines = append(lines, RenderSharedViewContext(view)...)
	if len(center.Findings) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, finding := range center.Findings {
			lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s task=%s reason=%s next=%s actions=%s", finding.RunID, finding.Severity, finding.Owner, finding.Status, finding.TaskID, finding.Reason, finding.NextAction, RenderConsoleActions(finding.Actions)))
		}
	}
	lines = append(lines, "", "## Inbox", "")
	if len(center.Inbox) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, item := range center.Inbox {
		suggestionSummary := "none"
		if len(item.Suggestions) > 0 {
			parts := make([]string, 0, len(item.Suggestions))
			for _, suggestion := range item.Suggestions {
				parts = append(parts, fmt.Sprintf("%s(%s, confidence=%.2f)", suggestion.Action, suggestion.FeedbackStatus, suggestion.Confidence))
			}
			suggestionSummary = strings.Join(parts, "; ")
		}
		evidenceSummary := "none"
		if len(item.Suggestions) > 0 {
			parts := make([]string, 0)
			for _, suggestion := range item.Suggestions {
				for _, evidence := range suggestion.Evidence {
					parts = append(parts, fmt.Sprintf("%s:%.2f", evidence.RelatedRunID, evidence.Score))
				}
			}
			if len(parts) > 0 {
				evidenceSummary = strings.Join(parts, ", ")
			}
		}
		lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s summary=%s suggestions=%s similar=%s", item.RunID, item.Severity, item.Owner, item.Status, item.Summary, suggestionSummary, evidenceSummary))
	}
	return strings.Join(lines, "\n") + "\n"
}

func runRequiresTriage(run observability.TaskRun) bool {
	if run.Status == "failed" || run.Status == "needs-approval" {
		return true
	}
	for _, trace := range run.Traces {
		if trace.Status == "pending" || trace.Status == "error" || trace.Status == "failed" {
			return true
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "pending" || audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return true
		}
	}
	return false
}

func triageSeverity(run observability.TaskRun) string {
	if run.Status == "failed" {
		return "critical"
	}
	for _, trace := range run.Traces {
		if trace.Status == "error" || trace.Status == "failed" {
			return "critical"
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return "critical"
		}
	}
	if run.Status == "needs-approval" {
		return "high"
	}
	for _, trace := range run.Traces {
		if trace.Status == "pending" {
			return "high"
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "pending" {
			return "high"
		}
	}
	return "medium"
}

func triageOwner(run observability.TaskRun) string {
	parts := []string{run.Summary, run.Title, run.Source, run.Medium}
	for _, trace := range run.Traces {
		parts = append(parts, trace.Status, trace.Span)
	}
	for _, audit := range run.Audits {
		parts = append(parts, audit.Outcome, stringValue(audit.Details["reason"]), fmt.Sprint(audit.Details["approvals"]))
	}
	evidence := strings.ToLower(strings.Join(parts, " "))
	if strings.Contains(evidence, "security") || strings.Contains(evidence, "high-risk") || strings.Contains(evidence, "security-review") {
		return "security"
	}
	if run.Medium == "browser" {
		return "engineering"
	}
	for _, artifact := range run.Artifacts {
		if artifact.Kind == "page" {
			return "engineering"
		}
	}
	return "operations"
}

func triageReason(run observability.TaskRun) string {
	for _, audit := range run.Audits {
		if (audit.Outcome == "failed" || audit.Outcome == "rejected" || audit.Outcome == "pending") && stringValue(audit.Details["reason"]) != "" {
			return stringValue(audit.Details["reason"])
		}
	}
	for _, trace := range run.Traces {
		if trace.Status == "error" || trace.Status == "failed" || trace.Status == "pending" {
			return fmt.Sprintf("%s is %s", trace.Span, trace.Status)
		}
	}
	return firstNonEmptyString(run.Summary, run.Status)
}

func triageNextAction(severity, owner string) string {
	if severity == "critical" {
		if owner == "engineering" {
			return "replay run and inspect tool failures"
		}
		if owner == "security" {
			return "page security reviewer and block rollout"
		}
		return "open incident review and coordinate response"
	}
	if owner == "security" {
		return "request approval and queue security review"
	}
	if owner == "engineering" {
		return "inspect execution evidence and retry when safe"
	}
	return "confirm owner and clear pending workflow gate"
}

func buildTriageSuggestions(run observability.TaskRun, runs []observability.TaskRun, severity, owner string, feedback []TriageFeedbackRecord) []TriageSuggestion {
	action := triageNextAction(severity, owner)
	evidence := similarityEvidence(run, runs, 2)
	return []TriageSuggestion{{
		Label:          triageSuggestionLabel(run, severity, owner),
		Action:         action,
		Owner:          owner,
		Confidence:     triageSuggestionConfidence(run, evidence),
		Evidence:       evidence,
		FeedbackStatus: feedbackStatus(run.RunID, action, feedback),
	}}
}

func triageSuggestionLabel(run observability.TaskRun, severity, owner string) string {
	if severity == "critical" && owner == "engineering" {
		return "replay candidate"
	}
	if owner == "security" {
		return "approval review"
	}
	if run.Status == "failed" {
		return "incident review"
	}
	return "workflow follow-up"
}

func triageSuggestionConfidence(run observability.TaskRun, evidence []TriageSimilarityEvidence) float64 {
	base := 0.45
	if run.Status == "needs-approval" || run.Status == "failed" {
		base = 0.55
	}
	if len(evidence) > 0 {
		base = maxFloat(base, minFloat(0.95, 0.45+evidence[0].Score/2))
	}
	return round2(base)
}

func feedbackStatus(runID, action string, feedback []TriageFeedbackRecord) string {
	for i := len(feedback) - 1; i >= 0; i-- {
		if feedback[i].RunID == runID && feedback[i].Action == action {
			return feedback[i].Decision
		}
	}
	return "pending"
}

func similarityEvidence(run observability.TaskRun, runs []observability.TaskRun, limit int) []TriageSimilarityEvidence {
	type scored struct {
		score float64
		run   observability.TaskRun
	}
	scoredMatches := make([]scored, 0)
	for _, candidate := range runs {
		if candidate.RunID == run.RunID {
			continue
		}
		score := runSimilarityScore(run, candidate)
		if score < 0.35 {
			continue
		}
		scoredMatches = append(scoredMatches, scored{score: score, run: candidate})
	}
	sort.Slice(scoredMatches, func(i, j int) bool {
		if scoredMatches[i].score == scoredMatches[j].score {
			return scoredMatches[i].run.RunID < scoredMatches[j].run.RunID
		}
		return scoredMatches[i].score > scoredMatches[j].score
	})
	if limit > 0 && len(scoredMatches) > limit {
		scoredMatches = scoredMatches[:limit]
	}
	evidence := make([]TriageSimilarityEvidence, 0, len(scoredMatches))
	for _, item := range scoredMatches {
		evidence = append(evidence, TriageSimilarityEvidence{
			RelatedRunID:  item.run.RunID,
			RelatedTaskID: item.run.TaskID,
			Score:         round2(item.score),
			Reason:        similarityReason(run, item.run),
		})
	}
	return evidence
}

func runSimilarityScore(run, candidate observability.TaskRun) float64 {
	left := buildSimilarityText(run)
	right := buildSimilarityText(candidate)
	score := tokenSimilarity(left, right)
	if run.Status == candidate.Status {
		score += 0.15
	}
	if triageOwner(run) == triageOwner(candidate) {
		score += 0.10
	}
	return minFloat(1.0, score)
}

func buildSimilarityText(run observability.TaskRun) string {
	parts := []string{run.Title, run.Summary}
	traceSpans := make([]string, 0, len(run.Traces))
	for _, trace := range run.Traces {
		traceSpans = append(traceSpans, trace.Span)
	}
	parts = append(parts, strings.Join(traceSpans, " "))
	auditOutcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		auditOutcomes = append(auditOutcomes, audit.Outcome)
	}
	parts = append(parts, strings.Join(auditOutcomes, " "))
	return strings.ToLower(strings.Join(parts, " "))
}

func tokenSimilarity(left, right string) float64 {
	leftTokens := strings.Fields(left)
	rightTokens := strings.Fields(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}
	leftCounts := make(map[string]int)
	rightCounts := make(map[string]int)
	for _, token := range leftTokens {
		leftCounts[token]++
	}
	for _, token := range rightTokens {
		rightCounts[token]++
	}
	shared := 0
	for token, count := range leftCounts {
		if other, ok := rightCounts[token]; ok {
			if count < other {
				shared += count
			} else {
				shared += other
			}
		}
	}
	return (2 * float64(shared)) / float64(len(leftTokens)+len(rightTokens))
}

func similarityReason(run, candidate observability.TaskRun) string {
	reasons := make([]string, 0)
	if run.Status == candidate.Status {
		reasons = append(reasons, "shared status "+run.Status)
	}
	if triageOwner(run) == triageOwner(candidate) {
		reasons = append(reasons, "shared owner "+triageOwner(run))
	}
	if triageReason(run) == triageReason(candidate) {
		reasons = append(reasons, "matching failure reason")
	}
	if len(reasons) == 0 {
		return "similar execution trail"
	}
	return strings.Join(reasons, ", ")
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
