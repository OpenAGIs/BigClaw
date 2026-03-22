package refill

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type QueuePayload struct {
	Project struct {
		Name    string `json:"name"`
		SlugID  string `json:"slug_id"`
		Team    string `json:"team"`
		TeamKey string `json:"team_key"`
		Epic    string `json:"epic"`
	} `json:"project"`
	Policy struct {
		TargetInProgress  int      `json:"target_in_progress"`
		ActivateStateID   string   `json:"activate_state_id"`
		ActivateStateName string   `json:"activate_state_name"`
		RefillStates      []string `json:"refill_states"`
		BlockedReason     string   `json:"blocked_reason,omitempty"`
	} `json:"policy"`
	RecentBatches struct {
		Completed []string `json:"completed"`
		Active    []string `json:"active"`
		Standby   []string `json:"standby"`
	} `json:"recent_batches,omitempty"`
	IssueOrder []string      `json:"issue_order"`
	Issues     []IssueRecord `json:"issues"`
}

type IssueRecord struct {
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Track      string `json:"track"`
	Status     string `json:"status"`
}

type ParallelIssueQueue struct {
	queuePath string
	payload   QueuePayload
}

type TrackedIssue struct {
	ID         string
	Identifier string
	StateName  string
}

func LoadQueue(path string) (*ParallelIssueQueue, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	payload := QueuePayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &ParallelIssueQueue{queuePath: absolute, payload: payload}, nil
}

func (q *ParallelIssueQueue) Save() error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(q.payload); err != nil {
		return err
	}
	dir := filepath.Dir(q.queuePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".parallel-refill-queue.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, q.queuePath)
}

func (q *ParallelIssueQueue) ProjectSlug() string {
	return q.payload.Project.SlugID
}

func (q *ParallelIssueQueue) ActivateStateID() string {
	return q.payload.Policy.ActivateStateID
}

func (q *ParallelIssueQueue) ActivateStateName() string {
	if q.payload.Policy.ActivateStateName != "" {
		return q.payload.Policy.ActivateStateName
	}
	return "In Progress"
}

func (q *ParallelIssueQueue) TargetInProgress() int {
	return q.payload.Policy.TargetInProgress
}

func (q *ParallelIssueQueue) RefillStates() map[string]struct{} {
	result := map[string]struct{}{}
	for _, state := range q.payload.Policy.RefillStates {
		result[state] = struct{}{}
	}
	return result
}

func (q *ParallelIssueQueue) IssueOrder() []string {
	return append([]string{}, q.payload.IssueOrder...)
}

func (q *ParallelIssueQueue) IssueRecords() []IssueRecord {
	return append([]IssueRecord{}, q.payload.Issues...)
}

func (q *ParallelIssueQueue) SyncStatusFromStates(issueStates map[string]string) int {
	updated := 0
	for idx := range q.payload.Issues {
		identifier := strings.TrimSpace(q.payload.Issues[idx].Identifier)
		if identifier == "" {
			continue
		}
		state, ok := issueStates[identifier]
		if !ok {
			continue
		}
		state = strings.TrimSpace(state)
		if state == "" {
			continue
		}
		if strings.TrimSpace(q.payload.Issues[idx].Status) == state {
			continue
		}
		q.payload.Issues[idx].Status = state
		updated++
	}
	return updated
}

func (q *ParallelIssueQueue) IssueIdentifiers() []string {
	records := q.IssueRecords()
	result := make([]string, 0, len(records))
	for _, record := range records {
		result = append(result, record.Identifier)
	}
	return result
}

func (q *ParallelIssueQueue) RunnableCount() int {
	if len(q.payload.IssueOrder) == 0 {
		return 0
	}
	return countRunnable(q.payload.IssueOrder, issueRecordStateMap(q.IssueRecords()))
}

func (q *ParallelIssueQueue) RunnableCountForStates(issueStates map[string]string) int {
	if len(q.payload.IssueOrder) == 0 {
		return 0
	}
	merged := issueRecordStateMap(q.IssueRecords())
	for identifier, state := range issueStates {
		if strings.TrimSpace(identifier) == "" || strings.TrimSpace(state) == "" {
			continue
		}
		merged[identifier] = strings.TrimSpace(state)
	}
	return countRunnable(q.payload.IssueOrder, merged)
}

func issueRecordStateMap(records []IssueRecord) map[string]string {
	statusByIdentifier := map[string]string{}
	for _, record := range records {
		if record.Identifier == "" {
			continue
		}
		statusByIdentifier[record.Identifier] = strings.TrimSpace(record.Status)
	}
	return statusByIdentifier
}

func countRunnable(issueOrder []string, states map[string]string) int {
	terminal := map[string]struct{}{
		"Archived":   {},
		"Canceled":   {},
		"Canceled.":  {},
		"Cancelled":  {},
		"Cancelled.": {},
		"Closed":     {},
		"Closed.":    {},
		"Done":       {},
		"Done.":      {},
		"Duplicate":  {},
	}
	count := 0
	for _, identifier := range issueOrder {
		status, ok := states[identifier]
		if !ok || status == "" {
			count++
			continue
		}
		if _, ok := terminal[status]; ok {
			continue
		}
		count++
	}
	return count
}

func (q *ParallelIssueQueue) SelectCandidates(activeIdentifiers map[string]struct{}, issueStates map[string]string, targetOverride *int) []string {
	target := q.TargetInProgress()
	if targetOverride != nil {
		target = *targetOverride
	}
	needed := target - len(activeIdentifiers)
	if needed <= 0 {
		return []string{}
	}
	candidates := []string{}
	refillStates := q.RefillStates()
	for _, identifier := range q.IssueOrder() {
		if needed == 0 {
			break
		}
		if _, ok := activeIdentifiers[identifier]; ok {
			continue
		}
		if _, ok := refillStates[issueStates[identifier]]; ok {
			candidates = append(candidates, identifier)
			needed--
		}
	}
	return candidates
}

func IssueStateMap(issues []TrackedIssue) map[string]string {
	result := map[string]string{}
	for _, issue := range issues {
		if issue.Identifier != "" && issue.StateName != "" {
			result[issue.Identifier] = issue.StateName
		}
	}
	return result
}

func SortedActive(issues []TrackedIssue) []string {
	active := []string{}
	for _, issue := range issues {
		if issue.StateName == "In Progress" {
			active = append(active, issue.Identifier)
		}
	}
	sort.Strings(active)
	return active
}
