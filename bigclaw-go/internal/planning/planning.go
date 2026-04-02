package planning

import (
	"fmt"
	"slices"
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
	Theme             string         `json:"theme,omitempty"`
	Priority          string         `json:"priority,omitempty"`
	Owner             string         `json:"owner,omitempty"`
	Outcome           string         `json:"outcome,omitempty"`
	ValidationCommand string         `json:"validation_command,omitempty"`
	Capabilities      []string       `json:"capabilities,omitempty"`
	Evidence          []string       `json:"evidence,omitempty"`
	Blockers          []string       `json:"blockers,omitempty"`
	EvidenceLinks     []EvidenceLink `json:"evidence_links,omitempty"`
}

func (c CandidateEntry) Ready() bool {
	return len(c.Blockers) == 0
}

func (c CandidateEntry) Score() int {
	if c.Ready() {
		return 100
	}
	score := 100 - (len(c.Blockers) * 25)
	if score < 0 {
		score = 0
	}
	return score
}

type CandidateBacklog struct {
	EpicID     string           `json:"epic_id"`
	Title      string           `json:"title"`
	Version    string           `json:"version"`
	Candidates []CandidateEntry `json:"candidates,omitempty"`
}

func (b CandidateBacklog) RankedCandidates() []CandidateEntry {
	out := append([]CandidateEntry(nil), b.Candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Ready() != out[j].Ready() {
			return out[i].Ready()
		}
		return false
	})
	return out
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

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baselineAudit *governance.ScopeFreezeAudit) EntryGateDecision {
	decision := EntryGateDecision{GateID: gate.GateID}
	readyCapabilities := make(map[string]struct{})
	readyEvidence := make(map[string]struct{})

	for _, candidate := range backlog.RankedCandidates() {
		if candidate.Ready() {
			decision.ReadyCandidateIDs = append(decision.ReadyCandidateIDs, candidate.CandidateID)
			for _, capability := range candidate.Capabilities {
				readyCapabilities[strings.TrimSpace(capability)] = struct{}{}
			}
			for _, evidence := range candidate.Evidence {
				readyEvidence[strings.TrimSpace(evidence)] = struct{}{}
			}
			continue
		}
		decision.BlockedCandidateIDs = append(decision.BlockedCandidateIDs, candidate.CandidateID)
	}

	for _, capability := range gate.RequiredCapabilities {
		if _, ok := readyCapabilities[strings.TrimSpace(capability)]; !ok {
			decision.MissingCapabilities = append(decision.MissingCapabilities, capability)
		}
	}
	for _, evidence := range gate.RequiredEvidence {
		if _, ok := readyEvidence[strings.TrimSpace(evidence)]; !ok {
			decision.MissingEvidence = append(decision.MissingEvidence, evidence)
		}
	}

	decision.BaselineReady = baselineAudit != nil && baselineAudit.ReleaseReady()
	switch {
	case strings.TrimSpace(gate.RequiredBaselineVersion) == "":
		decision.BaselineReady = true
	case baselineAudit == nil:
		decision.BaselineFindings = []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
	case !baselineAudit.ReleaseReady():
		decision.BaselineFindings = []string{fmt.Sprintf("baseline %s is not release ready (%.1f)", gate.RequiredBaselineVersion, baselineAudit.ReadinessScore())}
	}

	decision.BlockerCount = len(decision.BlockedCandidateIDs)
	decision.Passed = len(decision.ReadyCandidateIDs) >= gate.MinReadyCandidates &&
		len(decision.MissingCapabilities) == 0 &&
		len(decision.MissingEvidence) == 0 &&
		decision.BaselineReady &&
		len(decision.BaselineFindings) == 0
	return decision
}

func RenderCandidateBacklogReport(backlog CandidateBacklog, gate EntryGate, decision EntryGateDecision) string {
	lines := []string{
		"# V3 Candidate Backlog Report",
		"",
		fmt.Sprintf("- Epic: %s %s", backlog.EpicID, backlog.Title),
		fmt.Sprintf("- Version: %s", backlog.Version),
		fmt.Sprintf("- Gate: %s %s", gate.GateID, gate.Name),
		fmt.Sprintf("- Decision: %s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d", passString(decision.Passed), len(decision.ReadyCandidateIDs), len(decision.BlockedCandidateIDs), len(decision.MissingCapabilities), len(decision.MissingEvidence), len(decision.BaselineFindings)),
		"",
		"## Candidates",
	}
	for _, candidate := range backlog.RankedCandidates() {
		lines = append(lines,
			fmt.Sprintf("- %s: %s priority=%s owner=%s score=%d ready=%s", candidate.CandidateID, candidate.Title, candidate.Priority, candidate.Owner, candidate.Score(), boolString(candidate.Ready())),
			fmt.Sprintf("validation=%s", candidate.ValidationCommand),
		)
		if len(candidate.EvidenceLinks) == 0 {
			lines = append(lines, "- evidence links: none")
			continue
		}
		for _, link := range candidate.EvidenceLinks {
			line := fmt.Sprintf("- %s -> %s capability=%s", link.Label, link.Target, link.Capability)
			if strings.TrimSpace(link.Note) != "" {
				line += " note=" + link.Note
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines,
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(decision.MissingCapabilities)),
		fmt.Sprintf("- Missing evidence: %s", joinOrNone(decision.MissingEvidence)),
		fmt.Sprintf("- Baseline ready: %s", boolString(decision.BaselineReady)),
		fmt.Sprintf("- Baseline findings: %s", joinOrNone(decision.BaselineFindings)),
	)
	return strings.Join(lines, "\n") + "\n"
}

type WeeklyGoal struct {
	GoalID        string `json:"goal_id"`
	Title         string `json:"title"`
	Owner         string `json:"owner,omitempty"`
	Status        string `json:"status,omitempty"`
	SuccessMetric string `json:"success_metric,omitempty"`
	TargetValue   string `json:"target_value,omitempty"`
}

type WeeklyExecutionPlan struct {
	WeekNumber int          `json:"week_number"`
	Theme      string       `json:"theme"`
	Objective  string       `json:"objective"`
	Goals      []WeeklyGoal `json:"goals,omitempty"`
}

func (w WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	out := make([]string, 0)
	for _, goal := range w.Goals {
		switch strings.TrimSpace(goal.Status) {
		case "at-risk", "blocked":
			out = append(out, goal.GoalID)
		}
	}
	return out
}

func (w WeeklyExecutionPlan) completedGoals() int {
	count := 0
	for _, goal := range w.Goals {
		if strings.TrimSpace(goal.Status) == "done" {
			count++
		}
	}
	return count
}

type FourWeekExecutionPlan struct {
	PlanID    string                `json:"plan_id"`
	Title     string                `json:"title"`
	Owner     string                `json:"owner,omitempty"`
	StartDate string                `json:"start_date,omitempty"`
	Weeks     []WeeklyExecutionPlan `json:"weeks,omitempty"`
}

func (p FourWeekExecutionPlan) Validate() error {
	if len(p.Weeks) != 4 {
		return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for index, week := range p.Weeks {
		if week.WeekNumber != index+1 {
			return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
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
		total += week.completedGoals()
	}
	return total
}

func (p FourWeekExecutionPlan) OverallProgressPercent() int {
	total := p.TotalGoals()
	if total == 0 {
		return 0
	}
	return int(float64(p.CompletedGoals()) / float64(total) * 100)
}

func (p FourWeekExecutionPlan) AtRiskWeeks() []int {
	out := make([]int, 0)
	for _, week := range p.Weeks {
		if len(week.AtRiskGoalIDs()) > 0 {
			out = append(out, week.WeekNumber)
		}
	}
	return out
}

func (p FourWeekExecutionPlan) GoalStatusCounts() map[string]int {
	counts := map[string]int{}
	for _, week := range p.Weeks {
		for _, goal := range week.Goals {
			counts[goal.Status]++
		}
	}
	return counts
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
	}
	for _, week := range plan.Weeks {
		total := len(week.Goals)
		completed := week.completedGoals()
		progress := 0
		if total > 0 {
			progress = int(float64(completed) / float64(total) * 100)
		}
		lines = append(lines, fmt.Sprintf("- Week %d: %s progress=%d/%d (%d%%)", week.WeekNumber, week.Theme, completed, total, progress))
		for _, goal := range week.Goals {
			lines = append(lines, fmt.Sprintf("- %s: %s owner=%s status=%s", goal.GoalID, goal.Title, goal.Owner, goal.Status))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildBig4701ExecutionPlan() FourWeekExecutionPlan {
	return FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "4周执行计划与周目标",
		Owner:     "execution-office",
		StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{
				WeekNumber: 1,
				Theme:      "Alignment and scope",
				Objective:  "Freeze the initial migration scope and working agreements.",
				Goals: []WeeklyGoal{
					{GoalID: "w1-scope-freeze", Title: "Freeze tranche scope", Owner: "execution-office", Status: "done"},
					{GoalID: "w1-migration-map", Title: "Map Python test coverage to Go packages", Owner: "platform-core", Status: "done"},
				},
			},
			{
				WeekNumber: 2,
				Theme:      "Build and integration",
				Objective:  "Land high-risk integration work.",
				Goals: []WeeklyGoal{
					{GoalID: "w2-handoff-sync", Title: "Resolve orchestration and console handoff dependencies", Owner: "orchestration-office", Status: "at-risk"},
					{GoalID: "w2-worker-paths", Title: "Wire worker/reporting integration", Owner: "runtime", Status: "not-started"},
				},
			},
			{
				WeekNumber: 3,
				Theme:      "Reporting and rollout",
				Objective:  "Consolidate governance evidence and launch review inputs.",
				Goals: []WeeklyGoal{
					{GoalID: "w3-evidence", Title: "Publish rollout evidence digest", Owner: "ops-platform", Status: "on-track"},
					{GoalID: "w3-closeout", Title: "Complete closeout parity review", Owner: "ops-platform", Status: "not-started"},
				},
			},
			{
				WeekNumber: 4,
				Theme:      "Validation and release",
				Objective:  "Validate the tranche and prepare release handoff.",
				Goals: []WeeklyGoal{
					{GoalID: "w4-validation", Title: "Run targeted validation", Owner: "quality", Status: "not-started"},
					{GoalID: "w4-release", Title: "Prepare release handoff packet", Owner: "execution-office", Status: "not-started"},
				},
			},
		},
	}
}

func BuildV3CandidateBacklog() CandidateBacklog {
	return CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "ops-platform",
				Outcome:           "Package command-center and approval surfaces with linked evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/reporting ./internal/worker ./internal/scheduler",
				Capabilities:      []string{"ops-control", "saved-views"},
				Evidence:          []string{"weekly-review", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/operations.py", Capability: "ops-control"},
					{Label: "control-center-tests", Target: "tests/test_control_center.py", Capability: "ops-control"},
					{Label: "operations-tests", Target: "tests/test_operations.py", Capability: "ops-control"},
					{Label: "execution-contract-src", Target: "src/bigclaw/execution_contract.py", Capability: "ops-control"},
					{Label: "workflow-src", Target: "src/bigclaw/workflow.py", Capability: "ops-control"},
					{Label: "workflow-engine-tests", Target: "bigclaw-go/internal/workflow/engine_test.go", Capability: "ops-control"},
					{Label: "worker-runtime-tests", Target: "bigclaw-go/internal/worker/runtime_test.go", Capability: "ops-control"},
					{Label: "saved-views-src", Target: "src/bigclaw/saved_views.py", Capability: "saved-views"},
					{Label: "saved-views-tests", Target: "tests/test_saved_views.py", Capability: "saved-views"},
					{Label: "evaluation-src", Target: "src/bigclaw/evaluation.py", Capability: "saved-views"},
					{Label: "evaluation-tests", Target: "tests/test_evaluation.py", Capability: "saved-views"},
				},
			},
			{
				CandidateID:       "candidate-orchestration-rollout",
				Title:             "Orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P1",
				Owner:             "orchestration",
				Outcome:           "Promote cross-team orchestration with commercialization visibility.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/workflow ./internal/worker",
				Capabilities:      []string{"commercialization", "handoff"},
				Evidence:          []string{"pilot-evidence"},
				EvidenceLinks: []EvidenceLink{
					{Label: "report-studio-tests", Target: "tests/test_reports.py", Capability: "commercialization"},
				},
			},
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "ui-acceptance", Target: "tests/test_design_system.py", Capability: "release-gate"},
				},
			},
		},
	}
}

func BuildV3EntryGate() EntryGate {
	return EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      3,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"acceptance-suite", "pilot-evidence", "validation-report"},
		RequiredBaselineVersion: "v2.0",
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

func boolString(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func passString(value bool) string {
	if value {
		return "PASS"
	}
	return "HOLD"
}

func containsAll(values []string, wants []string) bool {
	for _, want := range wants {
		if !slices.Contains(values, want) {
			return false
		}
	}
	return true
}
