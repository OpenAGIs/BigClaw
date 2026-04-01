package designsystem

import "strings"

type CommandAction struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Section  string `json:"section,omitempty"`
	Shortcut string `json:"shortcut,omitempty"`
}

type ConsoleCommandEntry struct {
	TriggerLabel string          `json:"trigger_label"`
	Placeholder  string          `json:"placeholder"`
	Shortcut     string          `json:"shortcut,omitempty"`
	Commands     []CommandAction `json:"commands,omitempty"`
}

type ConsoleTopBar struct {
	Name                      string              `json:"name"`
	SearchPlaceholder         string              `json:"search_placeholder,omitempty"`
	EnvironmentOptions        []string            `json:"environment_options,omitempty"`
	TimeRangeOptions          []string            `json:"time_range_options,omitempty"`
	AlertChannels             []string            `json:"alert_channels,omitempty"`
	DocumentationComplete     bool                `json:"documentation_complete"`
	AccessibilityRequirements []string            `json:"accessibility_requirements,omitempty"`
	CommandEntry              ConsoleCommandEntry `json:"command_entry"`
}

type ConsoleTopBarAudit struct {
	Name                string   `json:"name"`
	MissingCapabilities []string `json:"missing_capabilities,omitempty"`
	ReleaseReady        bool     `json:"release_ready"`
}

type ConsoleTopBarAuditor struct{}

func (ConsoleTopBarAuditor) Audit(topBar ConsoleTopBar) ConsoleTopBarAudit {
	missing := make([]string, 0)
	if strings.TrimSpace(topBar.SearchPlaceholder) == "" {
		missing = append(missing, "global-search")
	}
	if len(topBar.TimeRangeOptions) < 2 {
		missing = append(missing, "time-range-switch")
	}
	if len(topBar.EnvironmentOptions) < 2 {
		missing = append(missing, "environment-switch")
	}
	if len(topBar.AlertChannels) == 0 {
		missing = append(missing, "alert-entry")
	}
	if strings.TrimSpace(topBar.CommandEntry.TriggerLabel) == "" ||
		strings.TrimSpace(topBar.CommandEntry.Placeholder) == "" ||
		len(topBar.CommandEntry.Commands) == 0 {
		missing = append(missing, "command-shell")
	}
	requiredA11y := []string{"keyboard-navigation", "screen-reader-label", "focus-visible"}
	releaseReady := topBar.DocumentationComplete && len(missing) == 0 && hasAll(topBar.AccessibilityRequirements, requiredA11y)
	return ConsoleTopBarAudit{
		Name:                topBar.Name,
		MissingCapabilities: missing,
		ReleaseReady:        releaseReady,
	}
}

func hasAll(have []string, required []string) bool {
	index := make(map[string]struct{}, len(have))
	for _, item := range have {
		index[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := index[item]; !ok {
			return false
		}
	}
	return true
}
