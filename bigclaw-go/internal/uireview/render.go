package uireview

import (
	"fmt"
	"strings"
)

func RenderUIReviewPackReport(pack UIReviewPack, audit UIReviewPackAudit) string {
	lines := []string{
		"# UI Review Pack",
		"",
		fmt.Sprintf("- Issue: %s %s", pack.IssueID, pack.Title),
		fmt.Sprintf("- Version: %s", pack.Version),
		fmt.Sprintf("- Audit: %s", audit.Summary()),
		"",
		"## Objectives",
		"",
	}
	if len(pack.Objectives) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, objective := range pack.Objectives {
			objective = objective.withDefaults()
			lines = append(lines, fmt.Sprintf("- %s: %s persona=%s priority=%s", objective.ObjectiveID, objective.Title, objective.Persona, objective.Priority))
		}
	}

	lines = append(lines, "", "## Wireframes", "")
	if len(pack.Wireframes) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, wireframe := range pack.Wireframes {
			lines = append(lines, fmt.Sprintf("- %s: %s device=%s entry=%s", wireframe.SurfaceID, wireframe.Name, wireframe.Device, wireframe.EntryPoint))
		}
	}

	lines = append(lines, "", "## Interactions", "")
	if len(pack.Interactions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, interaction := range pack.Interactions {
			lines = append(lines, fmt.Sprintf("- %s: %s states=%s", interaction.FlowID, interaction.Name, joinOrNone(interaction.States)))
		}
	}

	lines = append(lines, "", "## Open Questions", "")
	if len(pack.OpenQuestions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, question := range pack.OpenQuestions {
			question = question.withDefaults()
			lines = append(lines, fmt.Sprintf("- %s: owner=%s theme=%s status=%s", question.QuestionID, question.Owner, question.Theme, question.Status))
		}
	}

	lines = append(lines, "", "## Findings", "")
	lines = append(lines, fmt.Sprintf("- Missing sections: %s", joinOrNone(audit.MissingSections)))
	lines = append(lines, fmt.Sprintf("- Objectives missing signals: %s", joinOrNone(audit.ObjectivesMissingSignals)))
	lines = append(lines, fmt.Sprintf("- Wireframes missing blocks: %s", joinOrNone(audit.WireframesMissingBlocks)))
	lines = append(lines, fmt.Sprintf("- Interactions missing states: %s", joinOrNone(audit.InteractionsMissingStates)))
	lines = append(lines, fmt.Sprintf("- Unresolved questions: %s", joinOrNone(audit.UnresolvedQuestionIDs)))
	return strings.Join(lines, "\n") + "\n"
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
