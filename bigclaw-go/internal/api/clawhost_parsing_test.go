package api

import (
	"reflect"
	"testing"
)

func TestParseBoolClawHost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  bool
	}{
		{name: "trimmed true", input: " true ", want: true},
		{name: "trimmed false", input: " false ", want: false},
		{name: "invalid fallback", input: "not-a-bool", want: false},
		{name: "empty fallback", input: "", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseBoolClawHost(tc.input); got != tc.want {
				t.Fatalf("expected parseBoolClawHost(%q)=%t, got %t", tc.input, tc.want, got)
			}
		})
	}
}

func TestParseIntClawHost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  int
	}{
		{name: "trimmed integer", input: " 443 ", want: 443},
		{name: "zero integer", input: "0", want: 0},
		{name: "invalid fallback", input: "NaN", want: 0},
		{name: "empty fallback", input: "", want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseIntClawHost(tc.input); got != tc.want {
				t.Fatalf("expected parseIntClawHost(%q)=%d, got %d", tc.input, tc.want, got)
			}
		})
	}
}

func TestSplitCSVClawHostRecovery(t *testing.T) {
	t.Run("normalizes and sorts values", func(t *testing.T) {
		got := splitCSVClawHostRecovery(" restart, , upgrade,start ")
		want := []string{"restart", "start", "upgrade"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected normalized recovery csv %v, got %v", want, got)
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		if got := splitCSVClawHostRecovery(" \t "); got != nil {
			t.Fatalf("expected nil recovery csv slice for empty input, got %v", got)
		}
	})
}
