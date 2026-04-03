package uireview

import (
	"fmt"
	"sort"
	"strings"
)

type set[T comparable] map[T]struct{}

func (s set[T]) add(value T) {
	s[value] = struct{}{}
}

func (s set[T]) has(value T) bool {
	_, ok := s[value]
	return ok
}

type blockerIndex map[string][]ReviewBlocker

func (b blockerIndex) hasBlocker(blockerID string) bool {
	for _, blockers := range b {
		for _, blocker := range blockers {
			if blocker.BlockerID == blockerID {
				return true
			}
		}
	}
	return false
}

func surfaceMap(items []WireframeSurface) map[string]WireframeSurface {
	out := map[string]WireframeSurface{}
	for _, item := range items {
		out[item.SurfaceID] = item
	}
	return out
}

func checklistBySurface(items []ReviewerChecklistItem) map[string][]ReviewerChecklistItem {
	out := map[string][]ReviewerChecklistItem{}
	for _, item := range items {
		out[item.SurfaceID] = append(out[item.SurfaceID], item)
	}
	return out
}

func decisionsBySurface(items []ReviewDecision) map[string][]ReviewDecision {
	out := map[string][]ReviewDecision{}
	for _, item := range items {
		out[item.SurfaceID] = append(out[item.SurfaceID], item)
	}
	return out
}

func assignmentsBySurface(items []ReviewRoleAssignment) map[string][]ReviewRoleAssignment {
	out := map[string][]ReviewRoleAssignment{}
	for _, item := range items {
		out[item.SurfaceID] = append(out[item.SurfaceID], item)
	}
	return out
}

func signoffsBySurface(items []ReviewSignoff) map[string][]ReviewSignoff {
	out := map[string][]ReviewSignoff{}
	for _, item := range items {
		out[item.SurfaceID] = append(out[item.SurfaceID], item)
	}
	return out
}

func blockersBySurface(items []ReviewBlocker) map[string][]ReviewBlocker {
	out := map[string][]ReviewBlocker{}
	for _, item := range items {
		out[item.SurfaceID] = append(out[item.SurfaceID], item)
	}
	return out
}

func blockersBySignoff(items []ReviewBlocker) blockerIndex {
	out := blockerIndex{}
	for _, item := range items {
		out[item.SignoffID] = append(out[item.SignoffID], item)
	}
	return out
}

func timelineByBlocker(items []ReviewBlockerEvent) map[string][]ReviewBlockerEvent {
	out := map[string][]ReviewBlockerEvent{}
	for _, item := range items {
		out[item.BlockerID] = append(out[item.BlockerID], item)
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool {
			if out[key][i].Timestamp == out[key][j].Timestamp {
				return out[key][i].EventID < out[key][j].EventID
			}
			return out[key][i].Timestamp < out[key][j].Timestamp
		})
	}
	return out
}

func setFromAssignments(items []ReviewRoleAssignment) set[string] {
	out := set[string]{}
	for _, item := range items {
		out.add(item.AssignmentID)
	}
	return out
}

func setFromSignoffs(items []ReviewSignoff) set[string] {
	out := set[string]{}
	for _, item := range items {
		out.add(item.SignoffID)
	}
	return out
}

func setFromChecklist(items []ReviewerChecklistItem) set[string] {
	out := set[string]{}
	for _, item := range items {
		out.add(item.ItemID)
	}
	return out
}

func setFromDecisions(items []ReviewDecision) set[string] {
	out := set[string]{}
	for _, item := range items {
		out.add(item.DecisionID)
	}
	return out
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func hasResolutionEvent(items []ReviewBlockerEvent) bool {
	for _, item := range items {
		if item.Status == "resolved" || item.Status == "closed" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func countMapString(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, m[key]))
	}
	return strings.Join(parts, " ")
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ",")
}

func sanitizeSlug(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), " ", "-")
}
