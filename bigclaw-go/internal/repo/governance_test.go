package repo

import (
	"reflect"
	"strings"
	"testing"
)

func TestPermissionMatrixResolvesRoles(t *testing.T) {
	contract := NewPermissionContract()

	if !contract.Check("repo.push", []string{"eng-lead"}) {
		t.Fatalf("expected eng-lead to be allowed repo.push")
	}
	if !contract.Check("repo.accept", []string{"reviewer"}) {
		t.Fatalf("expected reviewer to be allowed repo.accept")
	}
	if contract.Check("repo.push", []string{"execution-agent"}) {
		t.Fatalf("expected execution-agent to be denied repo.push")
	}
}

func TestAuditFieldContractIsDeterministic(t *testing.T) {
	missing := MissingAuditFields("repo.accept", map[string]any{
		"task_id":       "OPE-172",
		"run_id":        "run-172",
		"repo_space_id": "space-1",
		"actor":         "reviewer",
	})

	if !reflect.DeepEqual(missing, []string{"accepted_commit_hash", "reviewer"}) {
		t.Fatalf("unexpected missing audit fields: %+v", missing)
	}
}

func TestRequiredAuditFieldsByAction(t *testing.T) {
	if got := RequiredAuditFields("repo.push"); !reflect.DeepEqual(got, []string{"task_id", "run_id", "repo_space_id", "actor", "commit_hash", "outcome"}) {
		t.Fatalf("unexpected repo.push audit fields: %+v", got)
	}
	if got := RequiredAuditFields("repo.reply"); !reflect.DeepEqual(got, []string{"task_id", "run_id", "repo_space_id", "actor", "channel", "post_id", "outcome"}) {
		t.Fatalf("unexpected repo.reply audit fields: %+v", got)
	}
	if got := RequiredAuditFields("repo.unknown"); !reflect.DeepEqual(got, []string{"task_id", "run_id", "repo_space_id", "actor"}) {
		t.Fatalf("unexpected default audit fields: %+v", got)
	}
}

func TestGovernanceEnforcerBlocksQuotaAndSidecarFailures(t *testing.T) {
	enforcer := NewGovernanceEnforcer(GovernancePolicy{
		MaxBundleBytes:  10,
		MaxPushPerHour:  1,
		MaxDiffPerHour:  1,
		SidecarRequired: true,
	})

	ok := enforcer.Evaluate("push", 8, true)
	if !ok.Allowed {
		t.Fatalf("expected first push to be allowed, got %+v", ok)
	}

	tooLarge := enforcer.Evaluate("push", 12, true)
	if tooLarge.Allowed || tooLarge.Mode != "blocked" {
		t.Fatalf("expected oversize push to be blocked, got %+v", tooLarge)
	}

	overQuota := enforcer.Evaluate("push", 8, true)
	if overQuota.Allowed || overQuota.Mode != "blocked" || !strings.Contains(overQuota.Reason, "quota") {
		t.Fatalf("expected push quota block, got %+v", overQuota)
	}

	degraded := enforcer.Evaluate("diff", 0, false)
	if degraded.Allowed || degraded.Mode != "degraded" {
		t.Fatalf("expected degraded decision when sidecar is unavailable, got %+v", degraded)
	}
}
