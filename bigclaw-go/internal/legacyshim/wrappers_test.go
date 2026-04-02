package legacyshim

import (
	"reflect"
	"testing"
)

func TestBuildBigclawctlExecArgs(t *testing.T) {
	got := BuildBigclawctlExecArgs("/repo", []string{"workspace", "validate"}, []string{"--json"})
	want := []string{"bash", "/repo/scripts/ops/bigclawctl", "workspace", "validate", "--json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected argv: got=%+v want=%+v", got, want)
	}
}
