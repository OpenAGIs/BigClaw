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
		t.Fatalf("expected push within quota to be allowed, got %+v", ok)
	}

	tooLarge := enforcer.Evaluate("push", 12, true)
	if tooLarge.Allowed || tooLarge.Mode != "blocked" {
		t.Fatalf("expected oversized bundle to be blocked, got %+v", tooLarge)
	}

	overQuota := enforcer.Evaluate("push", 8, true)
	if overQuota.Allowed || !strings.Contains(overQuota.Reason, "quota") {
		t.Fatalf("expected push quota decision, got %+v", overQuota)
	}

	degraded := enforcer.Evaluate("diff", 0, false)
	if degraded.Allowed || degraded.Mode != "degraded" {
		t.Fatalf("expected sidecar failure to degrade action, got %+v", degraded)
	}
}

func TestServerEntryHealthMetrics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>ok</h1>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	monitoring := NewMonitoring()
	server := httptest.NewServer(NewHandler(dir, monitoring))
	defer server.Close()

	body := getBody(t, server.URL+"/")
	if !strings.Contains(body, "ok") {
		t.Fatalf("unexpected root body: %s", body)
	}

	var health map[string]any
	getJSON(t, server.URL+"/health", &health)
	if health["status"] != "ok" {
		t.Fatalf("unexpected health payload: %+v", health)
	}
	if requestTotal, ok := health["request_total"].(float64); !ok || requestTotal < 1 {
		t.Fatalf("unexpected request_total: %+v", health)
	}

	metrics := getBody(t, server.URL+"/metrics")
	if !strings.Contains(metrics, "bigclaw_http_requests_total") || !strings.Contains(metrics, "bigclaw_uptime_seconds") {
		t.Fatalf("unexpected metrics payload: %s", metrics)
	}

	var metricsJSON map[string]any
	getJSON(t, server.URL+"/metrics.json", &metricsJSON)
	for _, key := range []string{"bigclaw_http_requests_total", "recent_requests", "rolling_5m", "bigclaw_http_error_rate", "health_summary"} {
		if _, ok := metricsJSON[key]; !ok {
			t.Fatalf("metrics json missing key %q: %+v", key, metricsJSON)
		}
	}

	var alerts map[string]any
	getJSON(t, server.URL+"/alerts", &alerts)
	level := alerts["level"].(string)
	if level != "ok" && level != "warn" && level != "critical" {
		t.Fatalf("unexpected alerts level: %+v", alerts)
	}
	if _, ok := alerts["error_rate"]; !ok {
		t.Fatalf("alerts missing error_rate: %+v", alerts)
	}

	monitorHTML := getBody(t, server.URL+"/monitor")
	for _, needle := range []string{"BigClaw Monitor", "Requests", "Error Rate", "Auto refresh every 5s"} {
		if !strings.Contains(monitorHTML, needle) {
			t.Fatalf("monitor html missing %q: %s", needle, monitorHTML)
		}
	}

	if monitoring.RequestTotal() < 6 {
		t.Fatalf("expected request total >= 6, got %d", monitoring.RequestTotal())
	}
}

func TestEnsureStaticIndex(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureStaticIndex(dir); err != nil {
		t.Fatalf("EnsureStaticIndex(): %v", err)
	}

	indexPath := filepath.Join(dir, "index.html")
	body, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if string(body) != "<h1>ok</h1>" {
		t.Fatalf("unexpected index body: %s", body)
	}

	custom := []byte("<h1>custom</h1>")
	if err := os.WriteFile(indexPath, custom, 0o644); err != nil {
		t.Fatalf("rewrite index: %v", err)
	}
	if err := EnsureStaticIndex(dir); err != nil {
		t.Fatalf("EnsureStaticIndex() preserves existing file: %v", err)
	}

	body, err = os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read preserved index: %v", err)
	}
	if string(body) != string(custom) {
		t.Fatalf("expected existing index to be preserved, got %s", body)
	}
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url) //nolint:gosec
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

func getJSON(t *testing.T, url string, dst any) {
	t.Helper()
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}
