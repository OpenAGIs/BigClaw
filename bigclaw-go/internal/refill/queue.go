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

var terminalStateSet = map[string]struct{}{
	"archived":  {},
	"canceled":  {},
	"cancelled": {},
	"closed":    {},
	"done":      {},
	"duplicate": {},
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
	count := 0
	for _, identifier := range issueOrder {
		status := statusNormalize(states[identifier])
		if status == "" || !isTerminalStatus(status) {
			count++
		}
	}
	return count
}

func statusNormalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasSuffix(value, ".") {
		value = strings.TrimSuffix(value, ".")
	}
	return strings.ToLower(value)
}

func isTerminalStatus(status string) bool {
	if normalized := statusNormalize(status); normalized != "" {
		_, ok := terminalStateSet[normalized]
		return ok
	}
	return false
}

func (q *ParallelIssueQueue) normalizedRefillStateSet() map[string]struct{} {
	result := map[string]struct{}{}
	for state := range q.RefillStates() {
		if normalized := statusNormalize(state); normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
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
	refillStates := q.normalizedRefillStateSet()
	for _, identifier := range q.IssueOrder() {
		if needed == 0 {
			break
		}
		if _, ok := activeIdentifiers[identifier]; ok {
			continue
		}
		if _, ok := refillStates[statusNormalize(issueStates[identifier])]; ok {
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

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (q *ParallelIssueQueue) RefreshRecentBatchesFromStates(states map[string]string) bool {
	if len(states) == 0 {
		return false
	}
	active, completed, standby := q.classifyRecentBatches(states)
	if equalStringSlices(q.payload.RecentBatches.Active, active) &&
		equalStringSlices(q.payload.RecentBatches.Completed, completed) &&
		equalStringSlices(q.payload.RecentBatches.Standby, standby) {
		return false
	}
	q.payload.RecentBatches.Active = active
	q.payload.RecentBatches.Completed = completed
	q.payload.RecentBatches.Standby = standby
	return true
}

func (q *ParallelIssueQueue) classifyRecentBatches(states map[string]string) ([]string, []string, []string) {
	active := []string{}
	completed := []string{}
	standby := []string{}
	activeState := statusNormalize(q.ActivateStateName())
	for _, identifier := range q.IssueOrder() {
		status := statusNormalize(states[identifier])
		if status == "" {
			continue
		}
		switch {
		case status == activeState:
			active = append(active, identifier)
		case isTerminalStatus(status):
			completed = append(completed, identifier)
		default:
			standby = append(standby, identifier)
		}
	}
	return active, completed, standby
}
