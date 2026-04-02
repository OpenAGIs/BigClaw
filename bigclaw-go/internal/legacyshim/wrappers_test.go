package legacyshim

import (
	"reflect"
	"testing"
)

func TestRefillWrapperTargetsGoShim(t *testing.T) {
	if got := BuildRefillArgs("/repo", []string{"--apply"}); !reflect.DeepEqual(got, []string{"bash", "/repo/scripts/ops/bigclawctl", "refill", "--apply"}) {
		t.Fatalf("unexpected refill args: %+v", got)
	}
	if !stringsContain(LegacyPythonWrapperNotice, "compatibility shim during migration") {
		t.Fatalf("expected wrapper notice to mention compatibility shim, got %q", LegacyPythonWrapperNotice)
	}
}

func stringsContain(value, want string) bool {
	return len(value) >= len(want) && reflect.ValueOf(value).String() != "" && stringContains(value, want)
}

func stringContains(value, want string) bool {
	return len(want) == 0 || (len(value) >= len(want) && indexOf(value, want) >= 0)
}

func indexOf(value, want string) int {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return i
		}
	}
	return -1
}
