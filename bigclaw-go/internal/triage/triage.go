package triage

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/risk"
)

type Record struct {
	Task   domain.Task
	Events []domain.Event
}

type SimilarCase struct {
	TaskID   string  `json:"task_id"`
	TraceID  string  `json:"trace_id,omitempty"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
	Owner    string  `json:"owner"`
	Severity string  `json:"severity"`
}

type Finding struct {
	TaskID            string        `json:"task_id"`
	TraceID           string        `json:"trace_id,omitempty"`
	Source            string        `json:"source,omitempty"`
	State             string        `json:"state"`
	Severity          string        `json:"severity"`
	Owner             string        `json:"owner"`
	Reason            string        `json:"reason"`
	NextAction        string        `json:"next_action"`
	SuggestedWorkflow string        `json:"suggested_workflow"`
	SuggestedPriority string        `json:"suggested_priority"`
	SuggestedOwner    string        `json:"suggested_owner"`
	SuggestedAction   string        `json:"suggested_action"`
	Confidence        float64       `json:"confidence"`
	Risk              risk.Score    `json:"risk_score"`
	SimilarCases      []SimilarCase `json:"similar_cases,omitempty"`
}

type Cluster struct {
	Reason   string   `json:"reason"`
	Count    int      `json:"count"`
	TaskIDs  []string `json:"task_ids"`
	States   []string `json:"states"`
	Owners   []string `json:"owners"`
	Workflow string   `json:"workflow"`
}

type Center struct {
	FlaggedRuns    int            `json:"flagged_runs"`
	InboxSize      int            `json:"inbox_size"`
	Recommendation string         `json:"recommendation"`
	SeverityCounts map[string]int `json:"severity_counts"`
	OwnerCounts    map[string]int `json:"owner_counts"`
	Findings       []Finding      `json:"findings"`
	Clusters       []Cluster      `json:"clusters"`
}

func Build(records []Record) Center {
	center := Center{
		SeverityCounts: map[string]int{"critical": 0, "high": 0, "medium": 0},
		OwnerCounts:    map[string]int{"security": 0, "engineering": 0, "operations": 0},
	}
	findings := make([]Finding, 0)
	for _, record := range records {
		if !requiresTriage(record) {
			continue
		}
		riskScore := risk.ScoreTask(record.Task, record.Events)
		severity := severityFor(record, riskScore)
		owner := ownerFor(record, riskScore)
		reason := reasonFor(record, riskScore)
		workflow := workflowFor(record, severity, owner)
		nextAction := nextActionFor(severity, owner, workflow)
		finding := Finding{
			TaskID:            record.Task.ID,
			TraceID:           record.Task.TraceID,
			Source:            firstNonEmpty(record.Task.Source, record.Task.Metadata["source"]),
			State:             string(record.Task.State),
			Severity:          severity,
			Owner:             owner,
			Reason:            reason,
			NextAction:        nextAction,
			SuggestedWorkflow: workflow,
			SuggestedPriority: priorityFor(severity),
			SuggestedOwner:    owner,
			SuggestedAction:   nextAction,
			Confidence:        confidenceFor(record, severity, owner, records),
			Risk:              riskScore,
			SimilarCases:      similarCases(record, severity, owner, records),
		}
		findings = append(findings, finding)
		center.SeverityCounts[severity]++
		center.OwnerCounts[owner]++
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if severityRank(findings[i].Severity) == severityRank(findings[j].Severity) {
			if findings[i].Owner == findings[j].Owner {
				return findings[i].TaskID < findings[j].TaskID
			}
			return findings[i].Owner < findings[j].Owner
		}
		return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
	})
	center.Findings = findings
	center.FlaggedRuns = len(findings)
	center.InboxSize = len(findings)
	center.Recommendation = recommendationFor(center.SeverityCounts, center.OwnerCounts)
	center.Clusters = buildClusters(findings)
	return center
}

func requiresTriage(record Record) bool {
	if isFailureState(record.Task.State) || record.Task.State == domain.TaskBlocked || record.Task.State == domain.TaskRetrying {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(record.Task.Metadata["ci_status"]), "failed") {
		return true
	}
	riskScore := risk.ScoreTask(record.Task, record.Events)
	if riskScore.RequiresApproval {
		return true
	}
	for _, event := range record.Events {
		if event.Type == domain.EventTaskDeadLetter || event.Type == domain.EventTaskRetried || event.Type == domain.EventTaskCancelled {
			return true
		}
	}
	return false
}

func severityFor(record Record, riskScore risk.Score) string {
	if isFailureState(record.Task.State) || strings.EqualFold(strings.TrimSpace(record.Task.Metadata["ci_status"]), "failed") {
		return "critical"
	}
	for _, event := range record.Events {
		if event.Type == domain.EventTaskDeadLetter || event.Type == domain.EventTaskCancelled {
			return "critical"
		}
	}
	if record.Task.State == domain.TaskBlocked || record.Task.State == domain.TaskRetrying || riskScore.RequiresApproval {
		return "high"
	}
	return "medium"
}

func ownerFor(record Record, riskScore risk.Score) string {
	evidence := strings.ToLower(strings.Join([]string{
		record.Task.Title,
		record.Task.Description,
		strings.Join(record.Task.Labels, " "),
		strings.Join(record.Task.RequiredTools, " "),
		record.Task.Metadata["policy_approval_flow"],
		record.Task.Metadata["approval_flow"],
		record.Task.Metadata["ci_status"],
		reasonFor(record, riskScore),
	}, " "))
	if riskScore.RequiresApproval || strings.Contains(evidence, "security") || strings.Contains(evidence, "compliance") {
		return "security"
	}
	if containsAny(record.Task.RequiredTools, "browser", "github", "deploy", "terminal", "shell") || strings.Contains(evidence, "pr") || strings.Contains(evidence, "issue") || strings.Contains(evidence, "ci") {
		return "engineering"
	}
	return "operations"
}

func workflowFor(record Record, severity string, owner string) string {
	if strings.EqualFold(strings.TrimSpace(record.Task.Metadata["ci_status"]), "failed") {
		return "ci-failure-review"
	}
	if owner == "security" {
		return "security-review"
	}
	if severity == "critical" {
		return "run-replay"
	}
	if record.Task.State == domain.TaskBlocked || record.Task.State == domain.TaskRetrying {
		return "queue-gate-review"
	}
	return "owner-routing"
}

func nextActionFor(severity string, owner string, workflow string) string {
	switch workflow {
	case "security-review":
		return "request approval and queue security review"
	case "ci-failure-review":
		return "inspect CI failure evidence and route to owning team"
	case "run-replay":
		if owner == "engineering" {
			return "replay run and inspect tool failures"
		}
		return "open incident review and coordinate response"
	case "queue-gate-review":
		return "confirm owner and clear pending workflow gate"
	default:
		if severity == "critical" {
			return "open incident review and coordinate response"
		}
		return "confirm owner and route follow-up workflow"
	}
}

func priorityFor(severity string) string {
	switch severity {
	case "critical":
		return "P0"
	case "high":
		return "P1"
	default:
		return "P2"
	}
}

func confidenceFor(record Record, severity string, owner string, records []Record) float64 {
	base := 0.55
	if severity == "critical" {
		base = 0.7
	} else if severity == "high" {
		base = 0.62
	}
	matches := similarCases(record, severity, owner, records)
	if len(matches) > 0 && matches[0].Score > base {
		base = matches[0].Score
	}
	if base > 0.95 {
		base = 0.95
	}
	return round2(base)
}

func similarCases(record Record, severity string, owner string, records []Record) []SimilarCase {
	matches := make([]SimilarCase, 0)
	for _, candidate := range records {
		if candidate.Task.ID == record.Task.ID {
			continue
		}
		score := similarityScore(record, candidate, owner)
		if score < 0.35 {
			continue
		}
		candidateRisk := risk.ScoreTask(candidate.Task, candidate.Events)
		matches = append(matches, SimilarCase{
			TaskID:   candidate.Task.ID,
			TraceID:  candidate.Task.TraceID,
			Score:    round2(score),
			Reason:   similarityReason(record, candidate, owner),
			Owner:    ownerFor(candidate, candidateRisk),
			Severity: severityFor(candidate, candidateRisk),
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].TaskID < matches[j].TaskID
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > 2 {
		matches = matches[:2]
	}
	return matches
}

func similarityScore(record Record, candidate Record, owner string) float64 {
	score := 0.0
	reason := reasonFor(record, risk.ScoreTask(record.Task, record.Events))
	candidateReason := reasonFor(candidate, risk.ScoreTask(candidate.Task, candidate.Events))
	if reason != "" && reason == candidateReason {
		score += 0.35
	}
	if ownerFor(candidate, risk.ScoreTask(candidate.Task, candidate.Events)) == owner {
		score += 0.1
	}
	if record.Task.State == candidate.Task.State {
		score += 0.15
	}
	titleOverlap := overlapScore(tokenize(record.Task.Title+" "+record.Task.Description), tokenize(candidate.Task.Title+" "+candidate.Task.Description))
	score += titleOverlap * 0.25
	labelOverlap := overlapScore(tokenize(strings.Join(record.Task.Labels, " ")+" "+strings.Join(record.Task.RequiredTools, " ")), tokenize(strings.Join(candidate.Task.Labels, " ")+" "+strings.Join(candidate.Task.RequiredTools, " ")))
	score += labelOverlap * 0.15
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func similarityReason(record Record, candidate Record, owner string) string {
	reasons := make([]string, 0)
	if record.Task.State == candidate.Task.State {
		reasons = append(reasons, fmt.Sprintf("shared state %s", record.Task.State))
	}
	if ownerFor(candidate, risk.ScoreTask(candidate.Task, candidate.Events)) == owner {
		reasons = append(reasons, fmt.Sprintf("shared owner %s", owner))
	}
	reason := reasonFor(record, risk.ScoreTask(record.Task, record.Events))
	candidateReason := reasonFor(candidate, risk.ScoreTask(candidate.Task, candidate.Events))
	if reason != "" && reason == candidateReason {
		reasons = append(reasons, "matching primary reason")
	}
	if len(reasons) == 0 {
		return "similar execution trail"
	}
	return strings.Join(reasons, ", ")
}

func buildClusters(findings []Finding) []Cluster {
	clusters := make(map[string]*Cluster)
	for _, finding := range findings {
		cluster := clusters[finding.Reason]
		if cluster == nil {
			cluster = &Cluster{Reason: finding.Reason, Workflow: finding.SuggestedWorkflow}
			clusters[finding.Reason] = cluster
		}
		cluster.Count++
		appendUnique(&cluster.TaskIDs, finding.TaskID)
		appendUnique(&cluster.States, finding.State)
		appendUnique(&cluster.Owners, finding.Owner)
	}
	out := make([]Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, *cluster)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Reason < out[j].Reason
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func reasonFor(record Record, riskScore risk.Score) string {
	if strings.EqualFold(strings.TrimSpace(record.Task.Metadata["ci_status"]), "failed") {
		return firstNonEmpty(record.Task.Metadata["ci_failure_reason"], "ci failed")
	}
	for index := len(record.Events) - 1; index >= 0; index-- {
		if message := eventMessage(record.Events[index]); message != "" {
			return message
		}
	}
	if riskScore.RequiresApproval {
		return firstNonEmpty(record.Task.Metadata["approval_reason"], "requires approval for high-risk task")
	}
	if record.Task.State == domain.TaskBlocked {
		return firstNonEmpty(record.Task.Metadata["blocked_reason"], "task blocked pending operator action")
	}
	return firstNonEmpty(record.Task.Metadata["summary"], record.Task.Title, string(record.Task.State))
}

func recommendationFor(severityCounts map[string]int, ownerCounts map[string]int) string {
	if severityCounts["critical"] > 0 {
		return "immediate-attention"
	}
	if severityCounts["high"] > 0 && ownerCounts["security"] > 0 {
		return "approval-review"
	}
	return "queue-review"
}

func eventMessage(event domain.Event) string {
	if event.Payload == nil {
		return ""
	}
	for _, key := range []string{"reason", "message", "note"} {
		if value, ok := event.Payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	default:
		return 2
	}
}

func containsAny(values []string, wants ...string) bool {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}
	for _, want := range wants {
		if _, ok := set[strings.ToLower(strings.TrimSpace(want))]; ok {
			return true
		}
	}
	return false
}

func tokenize(value string) []string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ", ":", " ", ",", " ", ".", " ")
	value = replacer.Replace(value)
	parts := strings.Fields(value)
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func overlapScore(left []string, right []string) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(left))
	for _, item := range left {
		set[item] = struct{}{}
	}
	shared := 0
	for _, item := range right {
		if _, ok := set[item]; ok {
			shared++
		}
	}
	denom := len(left)
	if len(right) > denom {
		denom = len(right)
	}
	return float64(shared) / float64(denom)
}

func appendUnique(target *[]string, value string) {
	for _, existing := range *target {
		if existing == value {
			return
		}
	}
	*target = append(*target, value)
}

func isFailureState(state domain.TaskState) bool {
	switch state {
	case domain.TaskFailed, domain.TaskDeadLetter, domain.TaskCancelled:
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
