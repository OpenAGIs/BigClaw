package product

import "testing"

func TestNavigationForRoleFiltersRestrictedItems(t *testing.T) {
	engLead := NavigationForRole("eng_lead")
	if hasNavItem(engLead, "billing") || hasNavItem(engLead, "entitlements") || hasNavItem(engLead, "flows") {
		t.Fatalf("expected eng lead navigation to hide billing and flow-only items, got %+v", engLead)
	}
	if !hasNavItem(engLead, "scheduler") || !hasNavItem(engLead, "regression") {
		t.Fatalf("expected eng lead navigation to retain operator items, got %+v", engLead)
	}

	vpEng := NavigationForRole("vp_eng")
	if hasNavItem(vpEng, "scheduler") || hasNavItem(vpEng, "flows") || hasNavItem(vpEng, "canvas") {
		t.Fatalf("expected vp eng navigation to hide queue and flow controls, got %+v", vpEng)
	}
	if !hasNavItem(vpEng, "billing") || !hasNavItem(vpEng, "entitlements") || !hasNavItem(vpEng, "regression") {
		t.Fatalf("expected vp eng navigation to retain portfolio read surfaces, got %+v", vpEng)
	}
}

func hasNavItem(sections []NavSection, key string) bool {
	for _, section := range sections {
		for _, item := range section.Items {
			if item.Key == key {
				return true
			}
		}
	}
	return false
}
