package refill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (q *ParallelIssueQueue) SaveMarkdown(path string, generatedAt time.Time) (bool, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	body := []byte(q.RenderMarkdown(generatedAt))
	needsWrite, err := markdownNeedsWrite(absolute, body)
	if err != nil {
		return false, err
	}
	if !needsWrite {
		return false, nil
	}

	dir := filepath.Dir(absolute)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(dir, ".parallel-refill-queue-md.*.tmp")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmpName, absolute); err != nil {
		return false, err
	}
	return true, nil
}

func (q *ParallelIssueQueue) MarkdownNeedsWrite(path string, generatedAt time.Time) (bool, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	return markdownNeedsWrite(absolute, []byte(q.RenderMarkdown(generatedAt)))
}

func markdownNeedsWrite(path string, body []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, body) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func (q *ParallelIssueQueue) RenderMarkdown(generatedAt time.Time) string {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	activeStateName := q.ActivateStateName()
	records := q.IssueRecords()
	recordByIdentifier := map[string]IssueRecord{}
	states := issueRecordStateMap(records)
	for _, record := range records {
		recordByIdentifier[record.Identifier] = record
	}

	recent := q.RecentBatchesSnapshot()
	if len(recent.Active) == 0 && len(recent.Completed) == 0 && len(recent.Standby) == 0 {
		active, completed, standby := q.classifyRecentBatches(states)
		recent.Active = active
		recent.Completed = completed
		recent.Standby = standby
	}

	var out strings.Builder
	out.WriteString("# BigClaw v5.3 Go Mainline Refill Queue\n\n")
	out.WriteString("This file is the human-readable companion to `docs/parallel-refill-queue.json`.\n")
	out.WriteString("It records the current Go-mainline cutover backlog slices and the refill order\n")
	out.WriteString("used by the repo-native local tracker in `local-issues.json`.\n\n")
	out.WriteString("Linear issue creation is still blocked by workspace issue limits, but BigClaw no\n")
	out.WriteString("longer waits on Linear to keep issue execution moving.\n\n")

	out.WriteString("## Trigger\n\n")
	out.WriteString("- Manual one-shot refill:\n")
	out.WriteString("  - `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json`\n")
	out.WriteString("- Continuous refill watcher:\n")
	out.WriteString("  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json`\n")
	out.WriteString("- Optional dashboard refresh after promotion:\n")
	out.WriteString("  - `bash scripts/ops/bigclawctl refill --apply --watch --local-issues local-issues.json --refresh-url http://127.0.0.1:4000/api/v1/refresh`\n")
	out.WriteString("- Local issue CLI:\n")
	out.WriteString("  - `bash scripts/ops/bigclaw-issue list`\n")
	out.WriteString(fmt.Sprintf("  - `bash scripts/ops/bigclaw-issue state BIG-GOM-303 %q`\n", activeStateName))
	out.WriteString("- Local dashboard/orchestrator:\n")
	out.WriteString("  - `bash scripts/ops/bigclaw-symphony`\n")
	out.WriteString("  - `bash scripts/ops/bigclaw-panel`\n\n")

	out.WriteString("## Policy\n\n")
	out.WriteString(fmt.Sprintf("- Target: keep `2` issues in `%s` when issue capacity is available again.\n", activeStateName))
	out.WriteString(fmt.Sprintf("- Target: keep `2` issues in `%s` in the local tracker unless a higher\n", activeStateName))
	out.WriteString("  parallelism cap is explicitly chosen for a branch-safe batch.\n")
	out.WriteString("- Promote only issues currently in `Backlog` or `Todo`.\n")
	out.WriteString("- Use the queue order below as the single source of truth for refill priority.\n")
	out.WriteString("- Every substantive code-bearing update must be committed and pushed to GitHub immediately, with local/remote SHA equality verification after each push.\n")
	out.WriteString("- Shared mirror bootstrap remains mandatory so multiple Symphony issues reuse one local mirror/seed cache instead of re-downloading the repo.\n")
	out.WriteString("- `local-issues.json` is the authoritative issue state backend for ongoing work.\n")
	out.WriteString("- Use `docs/go-mainline-cutover-issue-pack.md` as the detailed project brief behind this queue.\n\n")

	out.WriteString("## Repo Validation\n\n")
	out.WriteString("- Current mainline expectation:\n")
	out.WriteString("  - new implementation work lands in `bigclaw-go`\n")
	out.WriteString("  - Python paths are migration-only unless explicitly marked otherwise\n")
	out.WriteString("- Current tracker expectation:\n")
	out.WriteString("  - issue state lives in `local-issues.json`\n")
	out.WriteString("  - queue promotion is handled by `bigclawctl refill`\n")
	out.WriteString("- Repo-native cutover plan:\n")
	out.WriteString("  - `docs/go-mainline-cutover-issue-pack.md`\n\n")

	out.WriteString("## Current batch\n\n")
	out.WriteString(fmt.Sprintf("- Current repo tranche status as of %s:\n", generatedAt.Format("January 2, 2006")))
	writeIssueBucket(&out, "active slices", recent.Active, recordByIdentifier)
	writeIssueBucket(&out, "standby slices", recent.Standby, recordByIdentifier)
	writeIssueBucket(&out, "recently completed slices", tailIdentifiers(recent.Completed, 8), recordByIdentifier)
	out.WriteString(fmt.Sprintf("  - queue status: `queue_runnable=%d`, `target_in_progress=%d`\n", q.RunnableCount(), q.TargetInProgress()))
	out.WriteString("  - run `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status` to keep queue status, recent batches, and this markdown companion aligned after tracker changes\n")
	out.WriteString("- Queue drained recovery:\n")
	out.WriteString("  - if `bigclawctl refill` reports `queue_drained: true`, the queue has no runnable identifiers left in `docs/parallel-refill-queue.json`\n")
	out.WriteString("  - seed the next `BIG-PAR-*` identifier with `bash scripts/ops/bigclawctl refill seed --local-issues local-issues.json --identifier BIG-PAR-XXX --title \"...\" --state Todo --recent-batch standby --json`\n")
	out.WriteString("  - once the next batch exists, run `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json --sync-queue-status` to align queue metadata and this markdown companion with the local tracker state\n")
	out.WriteString("- Completed slices:\n")
	for _, identifier := range q.IssueOrder() {
		if !isTerminalStatus(states[identifier]) {
			continue
		}
		record := recordByIdentifier[identifier]
		out.WriteString(fmt.Sprintf("  - `%s` — %s\n", identifier, record.Title))
	}
	out.WriteString("- Historical first runnable batch once issue creation was available:\n")
	for _, identifier := range headIdentifiers(q.IssueOrder(), 4) {
		record := recordByIdentifier[identifier]
		out.WriteString(fmt.Sprintf("  - `%s` — %s\n", identifier, record.Title))
	}
	out.WriteString("\n## Canonical refill order\n\n")
	for idx, identifier := range q.IssueOrder() {
		out.WriteString(fmt.Sprintf("%d. `%s`\n", idx+1, identifier))
	}
	return out.String()
}

func writeIssueBucket(out *strings.Builder, label string, identifiers []string, records map[string]IssueRecord) {
	if len(identifiers) == 0 {
		out.WriteString(fmt.Sprintf("  - %s: none\n", label))
		return
	}
	items := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		record := records[identifier]
		title := strings.TrimSpace(record.Title)
		if title == "" {
			items = append(items, fmt.Sprintf("`%s`", identifier))
			continue
		}
		items = append(items, fmt.Sprintf("`%s` — %s", identifier, title))
	}
	out.WriteString(fmt.Sprintf("  - %s: %s\n", label, strings.Join(items, "; ")))
}

func tailIdentifiers(identifiers []string, limit int) []string {
	if limit <= 0 || len(identifiers) <= limit {
		return append([]string{}, identifiers...)
	}
	return append([]string{}, identifiers[len(identifiers)-limit:]...)
}

func headIdentifiers(identifiers []string, limit int) []string {
	if limit <= 0 || len(identifiers) <= limit {
		return append([]string{}, identifiers...)
	}
	return append([]string{}, identifiers[:limit]...)
}
