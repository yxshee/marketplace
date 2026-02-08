package auth

import "testing"

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{name: "buyer can view catalog", role: RoleBuyer, permission: PermissionViewCatalog, want: true},
		{name: "buyer cannot manage promotions", role: RoleBuyer, permission: PermissionManagePromotions, want: false},
		{name: "finance can manage commission", role: RoleFinance, permission: PermissionManageCommission, want: true},
		{name: "finance cannot moderate products", role: RoleFinance, permission: PermissionModerateProducts, want: false},
		{name: "support can manage vendor verification", role: RoleSupport, permission: PermissionManageVendorVerification, want: true},
		{name: "support cannot manage commission", role: RoleSupport, permission: PermissionManageCommission, want: false},
		{name: "catalog moderator can moderate products", role: RoleCatalogModerator, permission: PermissionModerateProducts, want: true},
		{name: "super admin can do everything", role: RoleSuperAdmin, permission: PermissionManageTaxSettings, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAllowed(tc.role, tc.permission)
			if got != tc.want {
				t.Fatalf("IsAllowed(%q, %q)=%v want=%v", tc.role, tc.permission, got, tc.want)
			}
		})
	}
}

func TestMustBeAllowed(t *testing.T) {
	if err := MustBeAllowed(RoleBuyer, PermissionManageCommission); err == nil {
		t.Fatal("expected forbidden error")
	}

	if err := MustBeAllowed(RoleSuperAdmin, PermissionManageCommission); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
