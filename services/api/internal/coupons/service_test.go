package coupons

import "testing"

func TestCouponCRUDByVendorScope(t *testing.T) {
	service := NewService()

	created, err := service.Create("ven_1", CreateCouponInput{
		Code:          "save10",
		DiscountType:  DiscountTypePercent,
		DiscountValue: 10,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "SAVE10" {
		t.Fatalf("expected normalized uppercase code, got %s", created.Code)
	}

	if _, err := service.Create("ven_1", CreateCouponInput{
		Code:          "SAVE10",
		DiscountType:  DiscountTypePercent,
		DiscountValue: 5,
	}); err != ErrCouponCodeInUse {
		t.Fatalf("expected ErrCouponCodeInUse, got %v", err)
	}

	if _, err := service.Update("ven_2", created.ID, UpdateCouponInput{
		DiscountValue: int64Ptr(20),
	}); err != ErrUnauthorizedCouponScope {
		t.Fatalf("expected ErrUnauthorizedCouponScope, got %v", err)
	}

	updated, err := service.Update("ven_1", created.ID, UpdateCouponInput{
		DiscountType:  discountTypePtr(DiscountTypeAmountCents),
		DiscountValue: int64Ptr(250),
		Active:        boolPtr(false),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.DiscountType != DiscountTypeAmountCents || updated.DiscountValue != 250 || updated.Active {
		t.Fatalf("unexpected updated coupon payload: %#v", updated)
	}

	list := service.ListByVendor("ven_1")
	if len(list) != 1 {
		t.Fatalf("expected one coupon for vendor, got %d", len(list))
	}

	if err := service.Delete("ven_1", created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(service.ListByVendor("ven_1")) != 0 {
		t.Fatalf("expected coupon list empty after delete")
	}
}

func TestCouponValidation(t *testing.T) {
	service := NewService()

	if _, err := service.Create("ven_1", CreateCouponInput{
		Code:          "no spaces allowed",
		DiscountType:  DiscountTypePercent,
		DiscountValue: 10,
	}); err != ErrInvalidCouponInput {
		t.Fatalf("expected ErrInvalidCouponInput for invalid code, got %v", err)
	}

	if _, err := service.Create("ven_1", CreateCouponInput{
		Code:          "SAVE200",
		DiscountType:  DiscountTypePercent,
		DiscountValue: 200,
	}); err != ErrInvalidCouponInput {
		t.Fatalf("expected ErrInvalidCouponInput for invalid percent value, got %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func discountTypePtr(value DiscountType) *DiscountType {
	return &value
}
