package legacyshim

import (
	"reflect"
	"testing"
)

func TestAppendMissingFlagPreservesExistingValues(t *testing.T) {
	got := AppendMissingFlag([]string{"--repo-url", "ssh://example/repo.git"}, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
	if !reflect.DeepEqual(got, []string{"--repo-url", "ssh://example/repo.git"}) {
		t.Fatalf("unexpected repo-url append result: %+v", got)
	}
	got = AppendMissingFlag([]string{"--cache-key=openagis-bigclaw"}, "--cache-key", "other")
	if !reflect.DeepEqual(got, []string{"--cache-key=openagis-bigclaw"}) {
		t.Fatalf("unexpected cache-key append result: %+v", got)
	}
}

func TestWorkspaceBootstrapWrapperInjectsGoDefaults(t *testing.T) {
	argv := BuildWorkspaceBootstrapArgs("/repo", []string{"--workspace", "/tmp/ws"})
	if !reflect.DeepEqual(argv[:4], []string{"bash", "/repo/scripts/ops/bigclawctl", "workspace", "bootstrap"}) {
		t.Fatalf("unexpected workspace bootstrap argv prefix: %+v", argv)
	}
	assertContains(t, argv, "--repo-url")
	assertContains(t, argv, "git@github.com:OpenAGIs/BigClaw.git")
	assertContains(t, argv, "--cache-key")
	assertContains(t, argv, "openagis-bigclaw")
}

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

func TestGitHubSyncAndRefillWrappersTargetGoShim(t *testing.T) {
	if got := BuildGitHubSyncArgs("/repo", []string{"status", "--json"}); !reflect.DeepEqual(got, []string{"bash", "/repo/scripts/ops/bigclawctl", "github-sync", "status", "--json"}) {
		t.Fatalf("unexpected github sync args: %+v", got)
	}
	if got := BuildRefillArgs("/repo", []string{"--apply"}); !reflect.DeepEqual(got, []string{"bash", "/repo/scripts/ops/bigclawctl", "refill", "--apply"}) {
		t.Fatalf("unexpected refill args: %+v", got)
	}
	if !stringsContain(LegacyPythonWrapperNotice, "compatibility shim during migration") {
		t.Fatalf("expected wrapper notice to mention compatibility shim, got %q", LegacyPythonWrapperNotice)
	}
}

func TestWorkspaceRuntimeWrapperTargetsGoShim(t *testing.T) {
	got := BuildWorkspaceRuntimeBootstrapArgs("/repo", []string{"bootstrap", "--json"})
	want := []string{"bash", "/repo/scripts/ops/bigclawctl", "workspace", "bootstrap", "--json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected workspace runtime args: got=%+v want=%+v", got, want)
	}
}

func TestRepoRootFromScriptClimbsToRepositoryRoot(t *testing.T) {
	if got := RepoRootFromScript("/repo/scripts/ops/bigclaw_refill_queue.py"); got != "/repo" {
		t.Fatalf("unexpected repo root: %s", got)
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %+v to contain %q", values, want)
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
