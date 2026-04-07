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
