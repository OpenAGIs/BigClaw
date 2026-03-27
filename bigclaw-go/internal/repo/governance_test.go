package repo

import (
	"reflect"
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

func TestActionPermissionsAndRolePoliciesExposeClonedContracts(t *testing.T) {
	permissions := ActionPermissions()
	roles := RolePolicies()
	if len(permissions) != 7 || len(roles) != 4 {
		t.Fatalf("unexpected contract sizes: permissions=%d roles=%d", len(permissions), len(roles))
	}
	permissions[0].Name = "mutated"
	roles[0].GrantedPermissions[0] = "mutated"

	freshPermissions := ActionPermissions()
	freshRoles := RolePolicies()
	if freshPermissions[0].Name != "repo.push" {
		t.Fatalf("expected cloned permissions, got %+v", freshPermissions[0])
	}
	if freshRoles[0].GrantedPermissions[0] != "repo.push" {
		t.Fatalf("expected cloned roles, got %+v", freshRoles[0])
	}
}
