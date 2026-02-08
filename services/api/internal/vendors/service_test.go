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
}
