package reporting

import (
	"html"
	"path/filepath"
	"strconv"
	"strings"
)

type NarrativeSection struct {
	Heading  string   `json:"heading"`
	Body     string   `json:"body"`
	Evidence []string `json:"evidence,omitempty"`
	Callouts []string `json:"callouts,omitempty"`
}

func (s NarrativeSection) Ready() bool {
	return strings.TrimSpace(s.Heading) != "" && strings.TrimSpace(s.Body) != ""
}

type ReportStudio struct {
	Name          string             `json:"name"`
	IssueID       string             `json:"issue_id"`
	Audience      string             `json:"audience"`
	Period        string             `json:"period"`
	Summary       string             `json:"summary"`
	Sections      []NarrativeSection `json:"sections,omitempty"`
	ActionItems   []string           `json:"action_items,omitempty"`
	SourceReports []string           `json:"source_reports,omitempty"`
}

func (s ReportStudio) Ready() bool {
	if strings.TrimSpace(s.Summary) == "" || len(s.Sections) == 0 {
		return false
	}
	for _, section := range s.Sections {
		if !section.Ready() {
			return false
		}
	}
	return true
}

func (s ReportStudio) Recommendation() string {
	if s.Ready() {
		return "publish"
	}
	return "draft"
}

func (s ReportStudio) ExportSlug() string {
	slug := slugify(s.Name)
	if slug == "" {
		return "report-studio"
	}
	return slug
}

type ReportStudioArtifacts struct {
	RootDir      string `json:"root_dir"`
	MarkdownPath string `json:"markdown_path"`
	HTMLPath     string `json:"html_path"`
	TextPath     string `json:"text_path"`
}

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{
		"# Report Studio",
		"",
		"- Name: " + studio.Name,
		"- Issue ID: " + studio.IssueID,
		"- Audience: " + studio.Audience,
		"- Period: " + studio.Period,
		"- Sections: " + itoaReporting(len(studio.Sections)),
		"- Recommendation: " + studio.Recommendation(),
		"",
		"## Narrative Summary",
		"",
		firstNonEmptyReporting(studio.Summary, "No summary drafted."),
		"",
		"## Sections",
		"",
	}
	if len(studio.Sections) == 0 {
		lines = append(lines, "- None", "")
	} else {
		for _, section := range studio.Sections {
			lines = append(lines,
				"### "+section.Heading,
				"",
				firstNonEmptyReporting(section.Body, "No narrative drafted."),
				"",
				"- Evidence: "+joinOrNone(section.Evidence),
				"- Callouts: "+joinOrNone(section.Callouts),
				"",
			)
		}
	}
	lines = append(lines, "## Action Items", "")
	if len(studio.ActionItems) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range studio.ActionItems {
			lines = append(lines, "- "+item)
		}
	}
	lines = append(lines, "", "## Sources", "")
	if len(studio.SourceReports) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, source := range studio.SourceReports {
			lines = append(lines, "- "+source)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	lines := []string{
		studio.Name + " (" + studio.IssueID + ")",
		"Audience: " + studio.Audience,
		"Period: " + studio.Period,
		"Recommendation: " + studio.Recommendation(),
		"",
		firstNonEmptyReporting(studio.Summary, "No summary drafted."),
		"",
	}
	for _, section := range studio.Sections {
		lines = append(lines, strings.ToUpper(section.Heading))
		lines = append(lines, firstNonEmptyReporting(section.Body, "No narrative drafted."))
		if len(section.Callouts) > 0 {
			lines = append(lines, "Callouts: "+strings.Join(section.Callouts, "; "))
		}
		if len(section.Evidence) > 0 {
			lines = append(lines, "Evidence: "+strings.Join(section.Evidence, "; "))
		}
		lines = append(lines, "")
	}
	if len(studio.ActionItems) > 0 {
		lines = append(lines, "Action Items:")
		for _, item := range studio.ActionItems {
			lines = append(lines, "- "+item)
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func RenderReportStudioHTML(studio ReportStudio) string {
	var sectionBuilder strings.Builder
	for _, section := range studio.Sections {
		sectionBuilder.WriteString(`
        <section class="section">
          <h2>` + html.EscapeString(section.Heading) + `</h2>
          <p>` + html.EscapeString(section.Body) + `</p>
          <p class="meta">Evidence: ` + html.EscapeString(joinOrNone(section.Evidence)) + `</p>
          <p class="meta">Callouts: ` + html.EscapeString(joinOrNone(section.Callouts)) + `</p>
        </section>
        `)
	}
	sectionHTML := sectionBuilder.String()
	if sectionHTML == "" {
		sectionHTML = `<section class="section"><p>No sections drafted.</p></section>`
	}
	actionHTML := listItemsHTML(studio.ActionItems)
	sourceHTML := listItemsHTML(studio.SourceReports)
	return `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>` + html.EscapeString(studio.Name) + `</title>
    <style>
      body { font-family: Georgia, 'Times New Roman', serif; margin: 40px auto; max-width: 840px; color: #1f2933; line-height: 1.6; }
      h1, h2 { font-family: 'Avenir Next', 'Segoe UI', sans-serif; }
      .meta { color: #52606d; font-size: 0.95rem; }
      .summary { padding: 16px 20px; background: #f7f3e8; border-left: 4px solid #c58b32; }
      .section { margin-top: 28px; }
    </style>
  </head>
  <body>
    <header>
      <p class="meta">` + html.EscapeString(studio.IssueID) + ` · ` + html.EscapeString(studio.Audience) + ` · ` + html.EscapeString(studio.Period) + `</p>
      <h1>` + html.EscapeString(studio.Name) + `</h1>
      <p class="meta">Recommendation: ` + html.EscapeString(studio.Recommendation()) + `</p>
    </header>
    <section class="summary">
      <h2>Narrative Summary</h2>
      <p>` + html.EscapeString(firstNonEmptyReporting(studio.Summary, "No summary drafted.")) + `</p>
    </section>
    ` + sectionHTML + `
    <section class="section">
      <h2>Action Items</h2>
      <ul>` + actionHTML + `</ul>
    </section>
    <section class="section">
      <h2>Sources</h2>
      <ul>` + sourceHTML + `</ul>
    </section>
  </body>
</html>
`
}

func WriteReportStudioBundle(rootDir string, studio ReportStudio) (ReportStudioArtifacts, error) {
	markdownPath := filepath.Join(rootDir, studio.ExportSlug()+".md")
	htmlPath := filepath.Join(rootDir, studio.ExportSlug()+".html")
	textPath := filepath.Join(rootDir, studio.ExportSlug()+".txt")
	if err := WriteReport(markdownPath, RenderReportStudioReport(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(htmlPath, RenderReportStudioHTML(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(textPath, RenderReportStudioPlainText(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	return ReportStudioArtifacts{
		RootDir:      rootDir,
		MarkdownPath: markdownPath,
		HTMLPath:     htmlPath,
		TextPath:     textPath,
	}, nil
}

func listItemsHTML(items []string) string {
	if len(items) == 0 {
		return "<li>None</li>"
	}
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString("<li>" + html.EscapeString(item) + "</li>")
	}
	return builder.String()
}

func firstNonEmptyReporting(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func slugify(value string) string {
	var parts []string
	var current strings.Builder
	for _, ch := range strings.TrimSpace(strings.ToLower(value)) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			current.WriteRune(ch)
			continue
		}
		if current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return strings.Join(parts, "-")
}

func itoaReporting(value int) string {
	return strconv.Itoa(value)
}
