package main

import (
	"strings"
	"testing"
)

func TestRunGitHubSyncHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"github-sync", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run github-sync help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl github-sync <install|status|sync> [flags]") {
		t.Fatalf("unexpected github-sync help output: %s", string(output))
	}
}

func TestRunWorkspaceHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"workspace", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run workspace help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]") {
		t.Fatalf("unexpected workspace help output: %s", string(output))
	}
}

func TestRunCreateIssuesHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"create-issues", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run create-issues help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl create-issues [flags]") {
		t.Fatalf("unexpected create-issues help output: %s", string(output))
	}
}

func TestRunDevSmokeHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"dev-smoke", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run dev-smoke help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl dev-smoke [flags]") {
		t.Fatalf("unexpected dev-smoke help output: %s", string(output))
	}
}

func TestRunSymphonyHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"symphony", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run symphony help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl symphony [flags] [args...]") {
		t.Fatalf("unexpected symphony help output: %s", string(output))
	}
}

func TestRunIssueHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"issue", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run issue help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl issue [flags] [args...]") {
		t.Fatalf("unexpected issue help output: %s", string(output))
	}
}

func TestRunPanelHelpPrintsUsageAndExitsZero(t *testing.T) {
	output, err := captureStdout(t, func() error {
		if code := run([]string{"panel", "--help"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run panel help: %v", err)
	}
	if !strings.Contains(string(output), "usage: bigclawctl panel [flags] [args...]") {
		t.Fatalf("unexpected panel help output: %s", string(output))
	}
}
