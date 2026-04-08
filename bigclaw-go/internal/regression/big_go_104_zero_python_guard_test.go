package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO104ExportWrapperIsShellNative(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	wrapperPath := filepath.Join(rootRepo, "bigclaw-go/scripts/migration/export_live_shadow_bundle")

	info, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("expected export wrapper to exist: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected export wrapper to stay executable, mode=%v", info.Mode())
	}

	body := readRepoFile(t, rootRepo, "bigclaw-go/scripts/migration/export_live_shadow_bundle")
	for _, needle := range []string{
		"#!/usr/bin/env bash",
		"go run \"$ROOT/cmd/bigclawctl\" automation migration export-live-shadow-bundle",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("export wrapper missing shell-native substring %q", needle)
		}
	}
	for _, needle := range []string{
		"#!/usr/bin/env python3",
		"import argparse",
		"import json",
	} {
		if strings.Contains(body, needle) {
			t.Fatalf("export wrapper should not retain Python implementation substring %q", needle)
		}
	}
}

func TestBIGGO104LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-104-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-104",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/docs/migration-shadow.md`",
		"`bigclaw-go/docs/reports/migration-readiness-report.md`",
		"`bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`",
		"`bigclaw-go/internal/regression/big_go_104_zero_python_guard_test.go`",
		"`bigclawctl automation migration export-live-shadow-bundle`",
		"`cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --go-root .`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run TestAutomationExportLiveShadowBundleBuildsManifest`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO104",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
