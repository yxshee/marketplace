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

func TestAdminOrderOperationsListingAndStatusUpdates(t *testing.T) {
	svc := NewService(500)

	actorA := Actor{GuestToken: "gst_admin_order_ops_a"}
	actorB := Actor{GuestToken: "gst_admin_order_ops_b"}

	if _, err := svc.UpsertItem(actorA, ProductSnapshot{
		ID:                    "prd_admin_a",
		VendorID:              "ven_admin_a",
		Title:                 "Admin Order A",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1200,
		StockQty:              4,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() actorA error = %v", err)
	}
	orderA, err := svc.PlaceOrder(actorA, "idem-admin-order-a")
	if err != nil {
		t.Fatalf("PlaceOrder() actorA error = %v", err)
	}

	if _, err := svc.UpsertItem(actorB, ProductSnapshot{
		ID:                    "prd_admin_b",
		VendorID:              "ven_admin_b",
		Title:                 "Admin Order B",
		Currency:              "USD",
		UnitPriceInclTaxCents: 1800,
		StockQty:              4,
	}, 1); err != nil {
		t.Fatalf("UpsertItem() actorB error = %v", err)
	}
	orderB, err := svc.PlaceOrder(actorB, "idem-admin-order-b")
	if err != nil {
		t.Fatalf("PlaceOrder() actorB error = %v", err)
	}
	if _, ok := svc.MarkOrderCODConfirmed(orderB.ID); !ok {
		t.Fatal("expected orderB cod confirmation to succeed")
	}

	orders, err := svc.ListOrders("")
	if err != nil {
		t.Fatalf("ListOrders() error = %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}

	codOrders, err := svc.ListOrders(OrderStatusCODConfirmed)
	if err != nil {
		t.Fatalf("ListOrders(cod_confirmed) error = %v", err)
	}
	if len(codOrders) != 1 || codOrders[0].ID != orderB.ID {
		t.Fatalf("expected one cod_confirmed order %s, got %#v", orderB.ID, codOrders)
	}

	if _, err := svc.ListOrders("invalid"); err != ErrInvalidOrderStatus {
		t.Fatalf("expected ErrInvalidOrderStatus, got %v", err)
	}

	adminOrder, found := svc.GetOrderForAdmin(orderA.ID)
	if !found {
		t.Fatal("expected GetOrderForAdmin to find orderA")
	}
	if adminOrder.ID != orderA.ID {
		t.Fatalf("expected order id %s, got %s", orderA.ID, adminOrder.ID)
	}

	updated, err := svc.UpdateOrderStatus(orderA.ID, OrderStatusPaid)
	if err != nil {
		t.Fatalf("UpdateOrderStatus(paid) error = %v", err)
	}
	if updated.Status != OrderStatusPaid {
		t.Fatalf("expected paid status, got %s", updated.Status)
	}

	if _, err := svc.UpdateOrderStatus(orderA.ID, OrderStatusPaymentFailed); err != ErrOrderStatusTransition {
		t.Fatalf("expected ErrOrderStatusTransition from paid->payment_failed, got %v", err)
	}

	if _, err := svc.UpdateOrderStatus(orderA.ID, "invalid"); err != ErrInvalidOrderStatus {
		t.Fatalf("expected ErrInvalidOrderStatus on update, got %v", err)
	}

	if _, err := svc.UpdateOrderStatus("missing", OrderStatusPaid); err != ErrOrderNotFound {
		t.Fatalf("expected ErrOrderNotFound on missing order, got %v", err)
	}
}
