package catalog

import "testing"

func TestModerationWorkflow(t *testing.T) {
	service := NewService()
	product := service.CreateProduct("usr_1", "ven_1", "Notebook", "Simple notebook", "USD", 2499)

	if _, err := service.SubmitForModeration(product.ID, "usr_2", "ven_1"); err != ErrUnauthorizedProductAccess {
		t.Fatalf("expected ErrUnauthorizedProductAccess, got %v", err)
	}

	submitted, err := service.SubmitForModeration(product.ID, "usr_1", "ven_1")
	if err != nil {
		t.Fatalf("SubmitForModeration() error = %v", err)
	}
	if submitted.Status != ProductStatusPendingApproval {
		t.Fatalf("expected pending_approval status, got %s", submitted.Status)
	}

	approved, err := service.ReviewProduct(product.ID, "admin_1", ModerationDecisionApprove, "")
	if err != nil {
		t.Fatalf("ReviewProduct() error = %v", err)
	}
	if approved.Status != ProductStatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}

	visible := service.ListVisibleProducts(func(vendorID string) bool { return vendorID == "ven_1" })
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible product, got %d", len(visible))
	}
}

func TestListVisibleProductsRespectsVendorVisibility(t *testing.T) {
	service := NewService()
	product := service.CreateProduct("usr_1", "ven_1", "Notebook", "Simple notebook", "USD", 2499)
	_, _ = service.SubmitForModeration(product.ID, "usr_1", "ven_1")
	_, _ = service.ReviewProduct(product.ID, "admin_1", ModerationDecisionApprove, "")

	visible := service.ListVisibleProducts(func(vendorID string) bool { return false })
	if len(visible) != 0 {
		t.Fatalf("expected 0 visible product, got %d", len(visible))
	}
}
