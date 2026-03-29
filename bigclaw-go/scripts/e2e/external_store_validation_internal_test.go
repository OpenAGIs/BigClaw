package main

import "testing"

func TestBuildExternalBackendMatrixPreservesStatuses(t *testing.T) {
	matrix := buildExternalBackendMatrix("http", true)
	summary := asMap(matrix["summary"])
	if asInt(summary["live_validated_lanes"]) != 1 || asInt(summary["not_configured_lanes"]) != 1 || asInt(summary["contract_only_lanes"]) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	lanes := asMapSlice(matrix["lanes"])
	if len(lanes) != 3 {
		t.Fatalf("unexpected lanes: %+v", lanes)
	}
	if asString(lanes[0]["backend"]) != "http_remote_service" || asString(lanes[0]["replay_backend"]) != "http" || !asBool(lanes[0]["retention_boundary_visible"]) {
		t.Fatalf("unexpected http lane: %+v", lanes[0])
	}
	if asString(lanes[1]["reason"]) != "not_configured" || asString(lanes[2]["reason"]) != "contract_only" {
		t.Fatalf("unexpected placeholder lanes: %+v", lanes)
	}
}
