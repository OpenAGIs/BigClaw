package intake

import "strings"

type SourceStatus string

const (
	SourceStatusTodo       SourceStatus = "Todo"
	SourceStatusInProgress SourceStatus = "In Progress"
	SourceStatusBlocked    SourceStatus = "Blocked"
	SourceStatusDone       SourceStatus = "Done"
	SourceStatusClosed     SourceStatus = "Closed"
)

func normalizeSourceStatus(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}
