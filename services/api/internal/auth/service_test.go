package auth

import "testing"

func TestRegisterAuthenticateAndAttachVendor(t *testing.T) {
	service := NewService(BuildBootstrapRoleMap("", "", "", ""))

	user, err := service.Register("buyer@example.com", "strong-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Role != RoleBuyer {
		t.Fatalf("expected buyer role, got %s", user.Role)
	}

	if _, err := service.Register("buyer@example.com", "strong-password"); err != ErrEmailInUse {
		t.Fatalf("expected ErrEmailInUse, got %v", err)
	}

	authenticated, err := service.Authenticate("buyer@example.com", "strong-password")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if authenticated.ID != user.ID {
		t.Fatalf("expected user id %s got %s", user.ID, authenticated.ID)
	}

	if _, err := service.Authenticate("buyer@example.com", "bad"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	updated, err := service.AttachVendor(user.ID, "ven_123")
	if err != nil {
		t.Fatalf("AttachVendor() error = %v", err)
	}
	if updated.Role != RoleVendorOwner {
		t.Fatalf("expected vendor_owner role, got %s", updated.Role)
	}
	if updated.VendorID == nil || *updated.VendorID != "ven_123" {
		t.Fatalf("expected vendor id ven_123, got %+v", updated.VendorID)
	}
}

func TestBootstrapRoles(t *testing.T) {
	service := NewService(BuildBootstrapRoleMap("admin@example.com", "", "", ""))
	user, err := service.Register("admin@example.com", "strong-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Role != RoleSuperAdmin {
		t.Fatalf("expected super_admin role, got %s", user.Role)
	}
}
