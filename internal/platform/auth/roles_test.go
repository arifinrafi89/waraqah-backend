package auth

import "testing"

// Ported from user_role.dart and the role cases of admin_access_test.dart.
func TestRolePermissions(t *testing.T) {
	cases := []struct {
		role                          Role
		staff, moderate, catalog, ord bool
	}{
		{RoleReader, false, false, false, false},
		{RoleModerator, true, true, false, false},
		{RoleCatalogManager, true, false, true, false},
		{RoleSupport, true, false, false, true},
		{RoleSuperAdmin, true, true, true, true},
	}
	for _, c := range cases {
		if c.role.IsStaff() != c.staff || c.role.CanModerate() != c.moderate ||
			c.role.CanManageCatalog() != c.catalog || c.role.CanManageOrders() != c.ord {
			t.Errorf("%s: wrong permissions", c.role)
		}
		if c.role.Can(PermModerate) != c.moderate || c.role.Can(PermCatalog) != c.catalog || c.role.Can(PermOrders) != c.ord {
			t.Errorf("%s: Can() disagrees", c.role)
		}
	}
	if (RoleSuperAdmin).Can("nonsense") {
		t.Error("unknown permission must be refused")
	}
}

func TestParseRole(t *testing.T) {
	for _, s := range []string{"reader", "moderator", "catalogManager", "support", "superAdmin"} {
		if r, ok := ParseRole(s); !ok || string(r) != s {
			t.Errorf("ParseRole(%q) failed", s)
		}
	}
	if _, ok := ParseRole("admin"); ok {
		t.Error("admin is not a role")
	}
}
