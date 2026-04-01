package designsystem

import (
	"reflect"
	"testing"
)

func TestConsoleTopBarAuditorFlagsMissingGlobalCapabilities(t *testing.T) {
	audit := ConsoleTopBarAuditor{}.Audit(ConsoleTopBar{
		Name:                      "Incomplete Header",
		EnvironmentOptions:        []string{"Production"},
		TimeRangeOptions:          []string{"24h"},
		DocumentationComplete:     false,
		AccessibilityRequirements: []string{"focus-visible"},
		CommandEntry:              ConsoleCommandEntry{Shortcut: "Cmd+K"},
	})

	if !reflect.DeepEqual(audit.MissingCapabilities, []string{"global-search", "time-range-switch", "environment-switch", "alert-entry", "command-shell"}) {
		t.Fatalf("unexpected missing capabilities: %+v", audit)
	}
	if audit.ReleaseReady {
		t.Fatalf("expected incomplete top bar to be not release ready")
	}
}
