package planning

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/governance"
)

type EvidenceLink struct {
	Label      string `json:"label"`
	Target     string `json:"target"`
	Capability string `json:"capability,omitempty"`
	Note       string `json:"note,omitempty"`
}

type CandidateEntry struct {
	CandidateID       string         `json:"candidate_id"`
	Title             string         `json:"title"`
	Theme             string         `json:"theme"`
	Priority          string         `json:"priority"`
	Owner             string         `json:"owner"`
	Outcome           string         `json:"outcome"`
	ValidationCommand string         `json:"validation_command"`
	Capabilities      []string       `json:"capabilities,omitempty"`
	Evidence          []string       `json:"evidence,omitempty"`
	EvidenceLinks     []EvidenceLink `json:"evidence_links,omitempty"`
	Blockers          []string       `json:"blockers,omitempty"`
}

func (c CandidateEntry) Ready() bool {
	return len(c.Blockers) == 0
}

func (c CandidateEntry) Score() int {
	if !c.Ready() {
		penalty := len(c.Blockers) * 25
		score := 100 - penalty
		if score < 0 {
			return 0
		}
		return score
	}
	return 100
}

type CandidateBacklog struct {
	EpicID     string           `json:"epic_id"`
	Title      string           `json:"title"`
	Version    string           `json:"version"`
	Candidates []CandidateEntry `json:"candidates,omitempty"`
}

func (b CandidateBacklog) RankedCandidates() []CandidateEntry {
	ranked := append([]CandidateEntry(nil), b.Candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		leftReady := ranked[i].Ready()
		rightReady := ranked[j].Ready()
		if leftReady != rightReady {
			return leftReady
		}
		if ranked[i].Score() != ranked[j].Score() {
			return ranked[i].Score() > ranked[j].Score()
		}
		if priorityRank(ranked[i].Priority) != priorityRank(ranked[j].Priority) {
			return priorityRank(ranked[i].Priority) < priorityRank(ranked[j].Priority)
		}
		return ranked[i].CandidateID < ranked[j].CandidateID
	})
	return ranked
}

func (b CandidateBacklog) ReadyCandidateIDs() []string {
	ids := make([]string, 0)
	for _, candidate := range b.RankedCandidates() {
		if candidate.Ready() {
			ids = append(ids, candidate.CandidateID)
		}
	}
	return ids
}

func (b CandidateBacklog) BlockedCandidateIDs() []string {
	ids := make([]string, 0)
	for _, candidate := range b.RankedCandidates() {
		if !candidate.Ready() {
			ids = append(ids, candidate.CandidateID)
		}
	}
	return ids
}

type EntryGate struct {
	GateID                  string   `json:"gate_id"`
	Name                    string   `json:"name"`
	MinReadyCandidates      int      `json:"min_ready_candidates"`
	RequiredCapabilities    []string `json:"required_capabilities,omitempty"`
	RequiredEvidence        []string `json:"required_evidence,omitempty"`
	RequiredBaselineVersion string   `json:"required_baseline_version,omitempty"`
}

type EntryGateDecision struct {
	GateID              string   `json:"gate_id"`
	Passed              bool     `json:"passed"`
	ReadyCandidateIDs   []string `json:"ready_candidate_ids,omitempty"`
	BlockedCandidateIDs []string `json:"blocked_candidate_ids,omitempty"`
	MissingCapabilities []string `json:"missing_capabilities,omitempty"`
	MissingEvidence     []string `json:"missing_evidence,omitempty"`
	BaselineReady       bool     `json:"baseline_ready"`
	BaselineFindings    []string `json:"baseline_findings,omitempty"`
	BlockerCount        int      `json:"blocker_count"`
}

type CandidatePlanner struct{}

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baseline *governance.ScopeFreezeAudit) EntryGateDecision {
	decision := EntryGateDecision{
		GateID:              gate.GateID,
		ReadyCandidateIDs:   backlog.ReadyCandidateIDs(),
		BlockedCandidateIDs: backlog.BlockedCandidateIDs(),
	}
	decision.MissingCapabilities = missingFromCandidates(backlog.Candidates, gate.RequiredCapabilities, func(candidate CandidateEntry) []string {
		return candidate.Capabilities
	})
	decision.MissingEvidence = missingFromCandidates(backlog.Candidates, gate.RequiredEvidence, func(candidate CandidateEntry) []string {
		return candidate.Evidence
	})
	decision.BaselineReady, decision.BaselineFindings = evaluateBaseline(gate.RequiredBaselineVersion, baseline)
	decision.BlockerCount = len(decision.BlockedCandidateIDs) + len(decision.MissingCapabilities) + len(decision.MissingEvidence) + len(decision.BaselineFindings)
	decision.Passed = len(decision.ReadyCandidateIDs) >= gate.MinReadyCandidates &&
		len(decision.MissingCapabilities) == 0 &&
		len(decision.MissingEvidence) == 0 &&
		decision.BaselineReady
	return decision
}

type WeeklyGoal struct {
	GoalID        string `json:"goal_id"`
	Title         string `json:"title"`
	Owner         string `json:"owner"`
	Status        string `json:"status"`
	SuccessMetric string `json:"success_metric"`
	TargetValue   string `json:"target_value"`
}

type WeeklyExecutionPlan struct {
	WeekNumber int          `json:"week_number"`
	Theme      string       `json:"theme"`
	Objective  string       `json:"objective"`
	Goals      []WeeklyGoal `json:"goals,omitempty"`
}

func (w WeeklyExecutionPlan) CompletedGoals() int {
	count := 0
	for _, goal := range w.Goals {
		if goal.Status == "done" {
			count++
		}
	}
	return count
}

func (w WeeklyExecutionPlan) ProgressPercent() int {
	if len(w.Goals) == 0 {
		return 0
	}
	return int(float64(w.CompletedGoals()) / float64(len(w.Goals)) * 100)
}

func (w WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	ids := make([]string, 0)
	for _, goal := range w.Goals {
		if goal.Status == "at-risk" || goal.Status == "blocked" {
			ids = append(ids, goal.GoalID)
		}
	}
	return ids
}

type FourWeekExecutionPlan struct {
	PlanID    string                `json:"plan_id"`
	Title     string                `json:"title"`
	Owner     string                `json:"owner"`
	StartDate string                `json:"start_date"`
	Weeks     []WeeklyExecutionPlan `json:"weeks,omitempty"`
}

func (p FourWeekExecutionPlan) Validate() error {
	if len(p.Weeks) != 4 {
		return errors.New("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for index, week := range p.Weeks {
		if week.WeekNumber != index+1 {
			return errors.New("Four-week execution plans must include weeks 1 through 4 in order")
		}
	}
	return nil
}

func (p FourWeekExecutionPlan) TotalGoals() int {
	total := 0
	for _, week := range p.Weeks {
		total += len(week.Goals)
	}
	return total
}

func (p FourWeekExecutionPlan) CompletedGoals() int {
	total := 0
	for _, week := range p.Weeks {
		total += week.CompletedGoals()
	}
	return total
}

func (p FourWeekExecutionPlan) OverallProgressPercent() int {
	if p.TotalGoals() == 0 {
		return 0
	}
	return int(float64(p.CompletedGoals()) / float64(p.TotalGoals()) * 100)
}

func (p FourWeekExecutionPlan) AtRiskWeeks() []int {
	weeks := make([]int, 0)
	for _, week := range p.Weeks {
		if len(week.AtRiskGoalIDs()) > 0 {
			weeks = append(weeks, week.WeekNumber)
		}
	}
	return weeks
}

func (p FourWeekExecutionPlan) GoalStatusCounts() map[string]int {
	counts := map[string]int{"done": 0, "on-track": 0, "at-risk": 0, "not-started": 0}
	for _, week := range p.Weeks {
		for _, goal := range week.Goals {
			status := normalizeGoalStatus(goal.Status)
			counts[status]++
		}
	}
	return counts
}

func RenderCandidateBacklogReport(backlog CandidateBacklog, gate EntryGate, decision EntryGateDecision) string {
	lines := []string{
		"# V3 Candidate Backlog Report",
		"",
		fmt.Sprintf("- Epic: %s %s", backlog.EpicID, backlog.Title),
		fmt.Sprintf(
			"- Decision: %s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d",
			passFail(decision.Passed),
			len(decision.ReadyCandidateIDs),
			len(decision.BlockedCandidateIDs),
			len(decision.MissingCapabilities),
			len(decision.MissingEvidence),
			len(decision.BaselineFindings),
		),
		fmt.Sprintf("- Gate: %s %s", gate.GateID, gate.Name),
		"",
		"## Candidates",
		"",
	}
	for _, candidate := range backlog.RankedCandidates() {
		lines = append(lines, fmt.Sprintf(
			"- %s: %s priority=%s owner=%s score=%d ready=%t",
			candidate.CandidateID,
			candidate.Title,
			candidate.Priority,
			candidate.Owner,
			candidate.Score(),
			candidate.Ready(),
		))
		lines = append(lines, fmt.Sprintf("  validation=%s", candidate.ValidationCommand))
		for _, link := range candidate.EvidenceLinks {
			lines = append(lines, fmt.Sprintf("  - %s -> %s capability=%s", link.Label, link.Target, link.Capability))
		}
	}
	lines = append(lines, "", fmt.Sprintf("- Missing capabilities: %s", joinOrNone(decision.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Missing evidence: %s", joinOrNone(decision.MissingEvidence)))
	lines = append(lines, fmt.Sprintf("- Baseline ready: %t", decision.BaselineReady))
	lines = append(lines, fmt.Sprintf("- Baseline findings: %s", joinOrNone(decision.BaselineFindings)))
	return strings.Join(lines, "\n") + "\n"
}

func RenderFourWeekExecutionReport(plan FourWeekExecutionPlan) string {
	lines := []string{
		"# Four-Week Execution Plan",
		"",
		fmt.Sprintf("- Plan: %s %s", plan.PlanID, plan.Title),
		fmt.Sprintf("- Overall progress: %d/%d goals complete (%d%%)", plan.CompletedGoals(), plan.TotalGoals(), plan.OverallProgressPercent()),
		fmt.Sprintf("- At-risk weeks: %s", joinInts(plan.AtRiskWeeks())),
		"",
		"## Weeks",
		"",
	}
	for _, week := range plan.Weeks {
		lines = append(lines, fmt.Sprintf(
			"- Week %d: %s progress=%d/%d (%d%%)",
			week.WeekNumber,
			week.Theme,
			week.CompletedGoals(),
			len(week.Goals),
			week.ProgressPercent(),
		))
		for _, goal := range week.Goals {
			lines = append(lines, fmt.Sprintf(
				"  - %s: %s owner=%s status=%s",
				goal.GoalID,
				goal.Title,
				goal.Owner,
				normalizeGoalStatus(goal.Status),
			))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildV3EntryGate() EntryGate {
	return EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      3,
		RequiredCapabilities:    []string{"ops-control", "commercialization", "release-gate"},
		RequiredEvidence:        []string{"validation-report", "weekly-review", "report-studio"},
		RequiredBaselineVersion: "v2.0",
	}
}

func BuildV3CandidateBacklog() CandidateBacklog {
	return CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "ops-platform",
				Outcome:           "Package queue control and approval flows with linked evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/queue ./internal/reporting",
				Capabilities:      []string{"ops-control", "saved-views"},
				Evidence:          []string{"weekly-review", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "bigclaw-go/internal/reporting/reporting.go", Capability: "ops-control"},
					{Label: "command-center-tests", Target: "bigclaw-go/internal/reporting/reporting_test.go", Capability: "ops-control"},
					{Label: "queue-tests", Target: "bigclaw-go/internal/queue/file_queue_test.go", Capability: "ops-control"},
				},
			},
			{
				CandidateID:       "candidate-orchestration-rollout",
				Title:             "Orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P1",
				Owner:             "orchestration",
				Outcome:           "Promote orchestration evidence and commercialization reporting.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/workflow ./internal/reporollout",
				Capabilities:      []string{"commercialization", "handoff"},
				Evidence:          []string{"report-studio", "pilot-evidence"},
				EvidenceLinks: []EvidenceLink{
					{Label: "report-studio-tests", Target: "bigclaw-go/internal/reporollout/rollout_test.go", Capability: "commercialization"},
					{Label: "workflow-tests", Target: "bigclaw-go/internal/workflow/orchestration_test.go", Capability: "handoff"},
				},
			},
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P2",
				Owner:             "platform-ui",
				Outcome:           "Unify release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/reporting",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "release-center-tests", Target: "bigclaw-go/internal/reporting/reporting_test.go", Capability: "release-gate"},
				},
			},
		},
	}
}

func BuildBig4701ExecutionPlan() FourWeekExecutionPlan {
	return FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "Four-week execution plan and weekly goals",
		Owner:     "execution-office",
		StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{
				WeekNumber: 1,
				Theme:      "Alignment and scoping",
				Objective:  "Lock tranche boundaries and validation surfaces.",
				Goals: []WeeklyGoal{
					{GoalID: "w1-scope-freeze", Title: "Freeze tranche scope", Owner: "execution-office", Status: "done", SuccessMetric: "approved boards", TargetValue: "1"},
					{GoalID: "w1-validation-map", Title: "Map validation evidence", Owner: "qa", Status: "done", SuccessMetric: "evidence packs", TargetValue: "1"},
				},
			},
			{
				WeekNumber: 2,
				Theme:      "Build and integration",
				Objective:  "Land high-risk integration work.",
				Goals: []WeeklyGoal{
					{GoalID: "w2-handoff-sync", Title: "Resolve orchestration and console handoff dependencies", Owner: "orchestration-office", Status: "at-risk", SuccessMetric: "handoff blockers", TargetValue: "0"},
					{GoalID: "w2-ops-bundle", Title: "Ship queue and approval bundle", Owner: "ops-platform", Status: "not-started", SuccessMetric: "merged bundles", TargetValue: "1"},
				},
			},
			{
				WeekNumber: 3,
				Theme:      "Readiness evidence",
				Objective:  "Collect release evidence and audits.",
				Goals: []WeeklyGoal{
					{GoalID: "w3-report-pack", Title: "Generate report pack", Owner: "reporting", Status: "on-track", SuccessMetric: "report bundles", TargetValue: "2"},
					{GoalID: "w3-approval-evidence", Title: "Collect approval evidence", Owner: "governance", Status: "not-started", SuccessMetric: "evidence docs", TargetValue: "1"},
				},
			},
			{
				WeekNumber: 4,
				Theme:      "Promotion and closeout",
				Objective:  "Promote ready work and close the tranche.",
				Goals: []WeeklyGoal{
					{GoalID: "w4-rollout", Title: "Promote rollout", Owner: "release", Status: "not-started", SuccessMetric: "rollout rings", TargetValue: "1"},
					{GoalID: "w4-closeout", Title: "Publish closeout pack", Owner: "execution-office", Status: "not-started", SuccessMetric: "closeout packs", TargetValue: "1"},
				},
			},
		},
	}
}

func evaluateBaseline(requiredVersion string, baseline *governance.ScopeFreezeAudit) (bool, []string) {
	if strings.TrimSpace(requiredVersion) == "" {
		return true, nil
	}
	if baseline == nil || strings.TrimSpace(baseline.Version) == "" {
		return false, []string{fmt.Sprintf("missing baseline audit for %s", requiredVersion)}
	}
	if baseline.Version != requiredVersion {
		return false, []string{fmt.Sprintf("missing baseline audit for %s", requiredVersion)}
	}
	if !baseline.ReleaseReady() {
		return false, []string{fmt.Sprintf("baseline %s is not release ready (%.1f)", requiredVersion, baseline.ReadinessScore())}
	}
	return true, nil
}

func missingFromCandidates(candidates []CandidateEntry, required []string, valueFn func(CandidateEntry) []string) []string {
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		if !candidate.Ready() {
			continue
		}
		for _, item := range valueFn(candidate) {
			seen[item] = struct{}{}
		}
	}
	missing := make([]string, 0)
	for _, item := range required {
		if _, ok := seen[item]; !ok {
			missing = append(missing, item)
		}
	}
	sort.Strings(missing)
	return missing
}

func priorityRank(priority string) int {
	switch strings.ToUpper(strings.TrimSpace(priority)) {
	case "P0":
		return 0
	case "P1":
		return 1
	default:
		return 2
	}
}

func normalizeGoalStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "done", "on-track", "at-risk", "not-started", "blocked":
		if strings.EqualFold(status, "blocked") {
			return "at-risk"
		}
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return "not-started"
	}
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func passFail(passed bool) string {
	if passed {
		return "PASS"
	}
	return "HOLD"
}

func roundTrip[T any](value T) (T, error) {
	var restored T
	data, err := json.Marshal(value)
	if err != nil {
		return restored, err
	}
	if err := json.Unmarshal(data, &restored); err != nil {
		return restored, err
	}
	return restored, nil
}
