package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpsCompatibilityEntrypointsDispatchFromBasename(t *testing.T) {
	rootRepo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	testCases := []struct {
		script string
		usage  string
	}{
		{script: "bigclaw-issue", usage: "usage: bigclawctl issue [flags] [args...]"},
		{script: "bigclaw-panel", usage: "usage: bigclawctl panel [flags] [args...]"},
		{script: "bigclaw-symphony", usage: "usage: bigclawctl symphony [flags] [args...]"},
	}

	for _, tc := range testCases {
		t.Run(tc.script, func(t *testing.T) {
			cmd := exec.Command("bash", filepath.Join(rootRepo, "scripts", "ops", tc.script), "--help")
			cmd.Dir = rootRepo
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run %s --help: %v (%s)", tc.script, err, string(output))
			}
			if !strings.Contains(string(output), tc.usage) {
				t.Fatalf("expected %q in output, got %s", tc.usage, string(output))
			}
		})
	}
}
