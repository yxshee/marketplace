package vendors

import "testing"

func TestRegisterAndVerificationState(t *testing.T) {
	service := NewService()

	created, err := service.Register("usr_1", "example-shop", "Example Shop")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if created.VerificationState != VerificationPending {
		t.Fatalf("expected pending state, got %s", created.VerificationState)
	}

	if _, err := service.Register("usr_1", "another", "Another"); err != ErrOwnerAlreadyVendor {
		t.Fatalf("expected ErrOwnerAlreadyVendor, got %v", err)
	}
	if _, err := service.Register("usr_2", "example-shop", "Another"); err != ErrSlugInUse {
		t.Fatalf("expected ErrSlugInUse, got %v", err)
	}

	updated, err := service.SetVerificationState(created.ID, VerificationVerified)
	if err != nil {
		t.Fatalf("SetVerificationState() error = %v", err)
	}
	if updated.VerificationState != VerificationVerified {
		t.Fatalf("expected verified state, got %s", updated.VerificationState)
	}

	second, err := service.Register("usr_2", "second-shop", "Second Shop")
	if err != nil {
		t.Fatalf("Register() second vendor error = %v", err)
	}
	if _, err := service.SetVerificationState(second.ID, VerificationRejected); err != nil {
		t.Fatalf("SetVerificationState() second vendor error = %v", err)
	}

	allVendors := service.List(nil)
	if len(allVendors) != 2 {
		t.Fatalf("expected two vendors in list, got %d", len(allVendors))
	}

	verified := VerificationVerified
	verifiedVendors := service.List(&verified)
	if len(verifiedVendors) != 1 {
		t.Fatalf("expected one verified vendor, got %d", len(verifiedVendors))
	}
	if verifiedVendors[0].ID != created.ID {
		t.Fatalf("expected verified vendor %s, got %s", created.ID, verifiedVendors[0].ID)
	}
}
