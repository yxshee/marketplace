package commerce

import "testing"

func TestQuoteAndPlaceOrderMultiShipmentWithIdempotency(t *testing.T) {
	svc := NewService(500)
	actor := Actor{GuestToken: "gst_test_checkout"}

	if _, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_a",
		VendorID:              "ven_a",
		Title:                 "Notebook",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1200,
		StockQty:              10,
	}, 2); err != nil {
		t.Fatalf("UpsertItem() first error = %v", err)
	}

	if _, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_b",
		VendorID:              "ven_b",
		Title:                 "Poster",
		Currency:              "USD",
		UnitPriceInclTaxCents: 2600,
		StockQty:              4,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() second error = %v", err)
	}

	quote, err := svc.Quote(actor)
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	if quote.ShipmentCount != 2 {
		t.Fatalf("expected 2 shipments, got %d", quote.ShipmentCount)
	}
	if quote.SubtotalCents != 5000 {
		t.Fatalf("expected subtotal 5000, got %d", quote.SubtotalCents)
	}
	if quote.ShippingCents != 1000 {
		t.Fatalf("expected shipping 1000, got %d", quote.ShippingCents)
	}
	if quote.TotalCents != 6000 {
		t.Fatalf("expected total 6000, got %d", quote.TotalCents)
	}

	order, err := svc.PlaceOrder(actor, "idem-checkout-1")
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}
	if order.ShipmentCount != 2 {
		t.Fatalf("expected 2 order shipments, got %d", order.ShipmentCount)
	}
	if order.TotalCents != 6000 {
		t.Fatalf("expected order total 6000, got %d", order.TotalCents)
	}
	if order.Status != OrderStatusPendingPayment {
		t.Fatalf("expected status %s, got %s", OrderStatusPendingPayment, order.Status)
	}

	retry, err := svc.PlaceOrder(actor, "idem-checkout-1")
	if err != nil {
		t.Fatalf("PlaceOrder() retry error = %v", err)
	}
	if retry.ID != order.ID {
		t.Fatalf("expected idempotent order id %s, got %s", order.ID, retry.ID)
	}

	cart, err := svc.GetCart(actor)
	if err != nil {
		t.Fatalf("GetCart() error = %v", err)
	}
	if cart.ItemCount != 0 {
		t.Fatalf("expected empty cart after place order, got %d items", cart.ItemCount)
	}
}

func TestUpdateAndRemoveCartItem(t *testing.T) {
	svc := NewService(500)
	actor := Actor{GuestToken: "gst_test_cart"}

	cart, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_item",
		VendorID:              "ven_item",
		Title:                 "Desk Lamp",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1800,
		StockQty:              3,
	}, 1)
	if err != nil {
		t.Fatalf("UpsertItem() error = %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("expected one cart item, got %d", len(cart.Items))
	}

	itemID := cart.Items[0].ID
	cart, err = svc.UpdateItemQty(actor, itemID, 2)
	if err != nil {
		t.Fatalf("UpdateItemQty() error = %v", err)
	}
	if cart.Items[0].Qty != 2 {
		t.Fatalf("expected qty 2, got %d", cart.Items[0].Qty)
	}

	cart, err = svc.RemoveItem(actor, itemID)
	if err != nil {
		t.Fatalf("RemoveItem() error = %v", err)
	}
	if cart.ItemCount != 0 {
		t.Fatalf("expected empty cart, got %d", cart.ItemCount)
	}
}
