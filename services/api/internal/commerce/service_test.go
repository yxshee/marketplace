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

func TestMarkOrderPaymentStatuses(t *testing.T) {
	svc := NewService(500)
	actor := Actor{GuestToken: "gst_test_payment_status"}

	if _, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_payment",
		VendorID:              "ven_payment",
		Title:                 "Pen Set",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1500,
		StockQty:              5,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() error = %v", err)
	}

	order, err := svc.PlaceOrder(actor, "idem-payment-status")
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}

	codOrder, codConfirmed := svc.MarkOrderCODConfirmed(order.ID)
	if !codConfirmed {
		t.Fatal("expected order to be marked cod confirmed")
	}
	if codOrder.Status != OrderStatusCODConfirmed {
		t.Fatalf("expected order status %s, got %s", OrderStatusCODConfirmed, codOrder.Status)
	}

	paidOrder, paid := svc.MarkOrderPaid(order.ID)
	if !paid {
		t.Fatal("expected order to be marked paid")
	}
	if paidOrder.Status != OrderStatusPaid {
		t.Fatalf("expected order status %s, got %s", OrderStatusPaid, paidOrder.Status)
	}

	failedOrder, failed := svc.MarkOrderPaymentFailed(order.ID)
	if !failed {
		t.Fatal("expected order lookup success")
	}
	if failedOrder.Status != OrderStatusPaid {
		t.Fatalf("expected paid order to remain %s, got %s", OrderStatusPaid, failedOrder.Status)
	}

	codAfterPaid, codAfterPaidOK := svc.MarkOrderCODConfirmed(order.ID)
	if !codAfterPaidOK {
		t.Fatal("expected order lookup success for cod confirmation after paid")
	}
	if codAfterPaid.Status != OrderStatusPaid {
		t.Fatalf("expected paid order to remain %s, got %s", OrderStatusPaid, codAfterPaid.Status)
	}
}

func TestVendorShipmentListingAndStatusTransitions(t *testing.T) {
	svc := NewService(500)
	actor := Actor{GuestToken: "gst_test_vendor_shipments"}

	if _, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_vendor_a",
		VendorID:              "ven_a",
		Title:                 "Notebook",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1200,
		StockQty:              10,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() vendor a error = %v", err)
	}
	if _, err := svc.UpsertItem(actor, ProductSnapshot{
		ID:                    "prd_vendor_b",
		VendorID:              "ven_b",
		Title:                 "Poster",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1800,
		StockQty:              10,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() vendor b error = %v", err)
	}

	order, err := svc.PlaceOrder(actor, "idem-vendor-shipment")
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}

	var vendorAShipmentID string
	for _, shipment := range order.Shipments {
		if shipment.VendorID == "ven_a" {
			vendorAShipmentID = shipment.ID
			break
		}
	}
	if vendorAShipmentID == "" {
		t.Fatal("expected vendor A shipment id to be present")
	}

	vendorAShipments, err := svc.ListVendorShipments("ven_a")
	if err != nil {
		t.Fatalf("ListVendorShipments() error = %v", err)
	}
	if len(vendorAShipments) != 1 {
		t.Fatalf("expected one vendor shipment, got %d", len(vendorAShipments))
	}
	if vendorAShipments[0].Status != ShipmentStatusPending {
		t.Fatalf("expected pending status, got %s", vendorAShipments[0].Status)
	}
	if len(vendorAShipments[0].Timeline) != 1 {
		t.Fatalf("expected initial timeline event, got %d", len(vendorAShipments[0].Timeline))
	}

	if _, err := svc.UpdateVendorShipmentStatus("ven_a", vendorAShipmentID, ShipmentStatusPacked, "usr_vendor_a"); err != nil {
		t.Fatalf("UpdateVendorShipmentStatus(packed) error = %v", err)
	}
	if _, err := svc.UpdateVendorShipmentStatus("ven_a", vendorAShipmentID, ShipmentStatusShipped, "usr_vendor_a"); err != nil {
		t.Fatalf("UpdateVendorShipmentStatus(shipped) error = %v", err)
	}
	delivered, err := svc.UpdateVendorShipmentStatus("ven_a", vendorAShipmentID, ShipmentStatusDelivered, "usr_vendor_a")
	if err != nil {
		t.Fatalf("UpdateVendorShipmentStatus(delivered) error = %v", err)
	}
	if delivered.Status != ShipmentStatusDelivered {
		t.Fatalf("expected delivered status, got %s", delivered.Status)
	}
	if delivered.ShippedAt == nil || delivered.DeliveredAt == nil {
		t.Fatal("expected shipped_at and delivered_at timestamps to be recorded")
	}
	if len(delivered.Timeline) != 4 {
		t.Fatalf("expected 4 timeline events, got %d", len(delivered.Timeline))
	}

	if _, err := svc.UpdateVendorShipmentStatus("ven_a", vendorAShipmentID, ShipmentStatusPending, "usr_vendor_a"); err != ErrShipmentTransition {
		t.Fatalf("expected ErrShipmentTransition, got %v", err)
	}
	if _, err := svc.UpdateVendorShipmentStatus("ven_b", vendorAShipmentID, ShipmentStatusPacked, "usr_vendor_b"); err != ErrShipmentForbidden {
		t.Fatalf("expected ErrShipmentForbidden, got %v", err)
	}

	fetched, found, err := svc.GetVendorShipment("ven_a", vendorAShipmentID)
	if err != nil {
		t.Fatalf("GetVendorShipment() error = %v", err)
	}
	if !found {
		t.Fatal("expected shipment lookup to succeed")
	}
	if fetched.ID != vendorAShipmentID {
		t.Fatalf("expected shipment id %s, got %s", vendorAShipmentID, fetched.ID)
	}
}
