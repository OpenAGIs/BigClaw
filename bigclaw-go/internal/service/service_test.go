package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures(t *testing.T) {
	enforcer := NewRepoGovernanceEnforcer(RepoGovernancePolicy{
		MaxBundleBytes:  10,
		MaxPushPerHour:  1,
		MaxDiffPerHour:  1,
		SidecarRequired: true,
	})

	ok := enforcer.Evaluate("push", 8, true)
	if !ok.Allowed {
		t.Fatalf("expected initial push to be allowed, got %+v", ok)
	}

	tooLarge := enforcer.Evaluate("push", 12, true)
	if tooLarge.Allowed || tooLarge.Mode != "blocked" {
		t.Fatalf("expected oversize push to be blocked, got %+v", tooLarge)
	}

	overQuota := enforcer.Evaluate("push", 8, true)
	if overQuota.Allowed || !strings.Contains(overQuota.Reason, "quota") {
		t.Fatalf("expected push quota block, got %+v", overQuota)
	}

	degraded := enforcer.Evaluate("diff", 0, false)
	if degraded.Allowed || degraded.Mode != "degraded" {
		t.Fatalf("expected sidecar failure to degrade diff, got %+v", degraded)
	}
}

func TestServerEntryHealthMetrics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>ok</h1>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	monitoring := NewMonitoring()
	handler, err := Handler(dir, monitoring)
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	body := mustGet(t, server.URL+"/")
	if !strings.Contains(body, "ok") {
		t.Fatalf("expected static index body, got %q", body)
	}

	var health map[string]any
	mustGetJSON(t, server.URL+"/health", &health)
	if health["status"] != "ok" {
		t.Fatalf("unexpected health payload: %+v", health)
	}
	if requestTotal, ok := health["request_total"].(float64); !ok || requestTotal < 1 {
		t.Fatalf("expected request_total >= 1, got %+v", health)
	}

	metrics := mustGet(t, server.URL+"/metrics")
	if !strings.Contains(metrics, "bigclaw_http_requests_total") || !strings.Contains(metrics, "bigclaw_uptime_seconds") {
		t.Fatalf("unexpected metrics payload: %q", metrics)
	}

	var metricsJSON map[string]any
	mustGetJSON(t, server.URL+"/metrics.json", &metricsJSON)
	for _, key := range []string{"bigclaw_http_requests_total", "recent_requests", "rolling_5m", "bigclaw_http_error_rate", "health_summary"} {
		if _, ok := metricsJSON[key]; !ok {
			t.Fatalf("expected %q in metrics json, got %+v", key, metricsJSON)
		}
	}

	var alerts map[string]any
	mustGetJSON(t, server.URL+"/alerts", &alerts)
	level, _ := alerts["level"].(string)
	if level != "ok" && level != "warn" && level != "critical" {
		t.Fatalf("unexpected alerts level: %+v", alerts)
	}
	if _, ok := alerts["error_rate"]; !ok {
		t.Fatalf("expected error_rate in alerts payload, got %+v", alerts)
	}

	monitorHTML := mustGet(t, server.URL+"/monitor")
	for _, want := range []string{"BigClaw Monitor", "Requests", "Error Rate", "Auto refresh every 5s"} {
		if !strings.Contains(monitorHTML, want) {
			t.Fatalf("expected %q in monitor html, got %q", want, monitorHTML)
		}
	}

	if monitoring.RequestTotal() < 6 {
		t.Fatalf("expected at least 6 requests to be recorded, got %d", monitoring.RequestTotal())
	}
}

func mustGet(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return string(body)
}

func mustGetJSON(t *testing.T, url string, target any) {
	t.Helper()
	body := mustGet(t, url)
	if err := json.Unmarshal([]byte(body), target); err != nil {
		t.Fatalf("decode %s: %v (%s)", url, err, body)
	}
}
