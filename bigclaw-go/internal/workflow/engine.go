package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type JournalEntry struct {
	Step      string         `json:"step"`
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type WorkpadJournal struct {
	TaskID  string           `json:"task_id"`
	RunID   string           `json:"run_id"`
	Entries []JournalEntry   `json:"entries,omitempty"`
	Now     func() time.Time `json:"-"`
}

type AcceptanceDecision struct {
	Passed                    bool     `json:"passed"`
	Status                    string   `json:"status"`
	Summary                   string   `json:"summary"`
	MissingAcceptanceCriteria []string `json:"missing_acceptance_criteria,omitempty"`
	MissingValidationSteps    []string `json:"missing_validation_steps,omitempty"`
	Approvals                 []string `json:"approvals,omitempty"`
}

type ExecutionOutcome struct {
	Approved bool   `json:"approved"`
	Status   string `json:"status,omitempty"`
}

type AcceptanceGate struct{}

func (j *WorkpadJournal) Record(step, status string, details map[string]any) {
	j.Entries = append(j.Entries, JournalEntry{
		Step:      strings.TrimSpace(step),
		Status:    strings.TrimSpace(status),
		Timestamp: j.now().UTC().Format(time.RFC3339),
		Details:   cloneJournalDetails(details),
	})
}

func (j WorkpadJournal) Write(path string) (string, error) {
	body, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func ReadWorkpadJournal(path string) (WorkpadJournal, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return WorkpadJournal{}, err
	}
	var journal WorkpadJournal
	if err := json.Unmarshal(body, &journal); err != nil {
		return WorkpadJournal{}, err
	}
	return journal, nil
}

func (j WorkpadJournal) Replay() []string {
	replayed := make([]string, 0, len(j.Entries))
	for _, entry := range j.Entries {
		step := strings.TrimSpace(entry.Step)
		status := strings.TrimSpace(entry.Status)
		switch {
		case step == "" && status == "":
			continue
		case step == "":
			replayed = append(replayed, status)
		case status == "":
			replayed = append(replayed, step)
		default:
			replayed = append(replayed, step+":"+status)
		}
	}
	return replayed
}

func (g AcceptanceGate) Evaluate(task domain.Task, outcome ExecutionOutcome, validationEvidence []string, approvals []string, pilotRecommendation string) AcceptanceDecision {
	evidence := make(map[string]struct{}, len(validationEvidence))
	for _, item := range validationEvidence {
		item = strings.TrimSpace(item)
		if item != "" {
			evidence[item] = struct{}{}
		}
	}
	approvalList := compactStrings(approvals)
	missingAcceptance := missingEvidence(task.AcceptanceCriteria, evidence)
	missingValidation := missingEvidence(task.ValidationPlan, evidence)
	needsManualApproval := task.RiskLevel == domain.RiskHigh || !outcome.Approved || strings.EqualFold(strings.TrimSpace(outcome.Status), "needs-approval")
	if needsManualApproval && len(approvalList) == 0 {
		return AcceptanceDecision{
			Passed:                    false,
			Status:                    "needs-approval",
			Summary:                   "manual approval required before acceptance closure",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 approvalList,
		}
	}
	if strings.EqualFold(strings.TrimSpace(pilotRecommendation), "hold") {
		return AcceptanceDecision{
			Passed:                    false,
			Status:                    "rejected",
			Summary:                   "pilot scorecard indicates insufficient ROI or KPI progress",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 approvalList,
		}
	}
	if len(missingAcceptance) > 0 || len(missingValidation) > 0 {
		return AcceptanceDecision{
			Passed:                    false,
			Status:                    "rejected",
			Summary:                   "acceptance evidence incomplete",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 approvalList,
		}
	}
	return AcceptanceDecision{
		Passed:    true,
		Status:    "accepted",
		Summary:   "acceptance criteria and validation plan satisfied",
		Approvals: approvalList,
	}
}

func (j WorkpadJournal) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}

func cloneJournalDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}

func missingEvidence(required []string, evidence map[string]struct{}) []string {
	var missing []string
	for _, item := range required {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := evidence[item]; ok {
			continue
		}
		if !slices.Contains(missing, item) {
			missing = append(missing, item)
		}
	}
	return missing
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}
