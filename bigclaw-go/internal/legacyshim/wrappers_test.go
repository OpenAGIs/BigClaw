package legacyshim

import (
	"reflect"
	"testing"
)

func TestWorkspaceValidateWrapperTranslatesLegacyFlags(t *testing.T) {
	translated := TranslateWorkspaceValidateArgs([]string{
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--workspace-root", "/tmp/ws",
		"--issues", "BIG-1", "BIG-2",
		"--report-file", "/tmp/report.md",
		"--no-cleanup",
		"--json",
	})
	want := []string{
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--workspace-root", "/tmp/ws",
		"--issues", "BIG-1,BIG-2",
		"--report", "/tmp/report.md",
		"--cleanup=false",
		"--json",
	}
	if !reflect.DeepEqual(translated, want) {
		t.Fatalf("unexpected translated workspace validate args: got=%+v want=%+v", translated, want)
	}
	argv := BuildWorkspaceValidateArgs("/repo", []string{"--issues", "BIG-1", "BIG-2"})
	if !reflect.DeepEqual(argv[:4], []string{"bash", "/repo/scripts/ops/bigclawctl", "workspace", "validate"}) {
		t.Fatalf("unexpected workspace validate argv prefix: %+v", argv)
	}
	if !reflect.DeepEqual(argv[4:], []string{"--issues", "BIG-1,BIG-2"}) {
		t.Fatalf("unexpected workspace validate forwarded args: %+v", argv[4:])
	}
}

func TestRepoRootFromScriptClimbsToRepositoryRoot(t *testing.T) {
	if got := RepoRootFromScript("/repo/scripts/ops/symphony_workspace_validate.py"); got != "/repo" {
		t.Fatalf("unexpected repo root: %s", got)
	}
}
