package product

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestHomeForRoleEngLeadUsesActiveTakeoversSignal(t *testing.T) {
	home := HomeForRole("eng_lead", []domain.Task{
		{ID: "task-1", State: domain.TaskBlocked},
		{ID: "task-2", State: domain.TaskBlocked},
	}, 1)

	if len(home.Cards) != 3 {
		t.Fatalf("expected 3 home cards, got %d", len(home.Cards))
	}
	if home.Cards[0].Key != "blockers" || home.Cards[0].Value != 2 {
		t.Fatalf("expected blocker card to keep blocked count, got %+v", home.Cards[0])
	}
	if home.Cards[1].Key != "takeovers" || home.Cards[1].Value != 1 {
		t.Fatalf("expected takeover card to use active takeover count, got %+v", home.Cards[1])
	}
}
