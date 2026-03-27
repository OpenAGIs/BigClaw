package runtasksmoke

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportWritesIndentedJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	reportPath := filepath.Join("reports", "smoke.json")
	payload := map[string]any{"status": "ok"}
	if err := writeReport(root, reportPath, payload); err != nil {
		t.Fatalf("writeReport returned error: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, reportPath))
	if err != nil {
		t.Fatalf("read written report: %v", err)
	}
	if !strings.Contains(string(body), "\n") {
		t.Fatalf("expected indented JSON, got %q", string(body))
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal written report: %v", err)
	}
	if decoded["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", decoded["status"])
	}
}

func TestReserveLocalBaseURLReturnsLoopback(t *testing.T) {
	t.Parallel()

	baseURL, httpAddr, err := reserveLocalBaseURL()
	if err != nil {
		t.Fatalf("reserveLocalBaseURL returned error: %v", err)
	}
	if !strings.HasPrefix(baseURL, "http://127.0.0.1:") {
		t.Fatalf("unexpected baseURL: %s", baseURL)
	}
	if !strings.HasPrefix(httpAddr, "127.0.0.1:") {
		t.Fatalf("unexpected httpAddr: %s", httpAddr)
	}
}
