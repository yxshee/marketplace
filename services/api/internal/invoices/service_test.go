package invoices

import (
	"bytes"
	"testing"
	"time"

	"github.com/yxshee/marketplace-platform/services/api/internal/commerce"
)

func TestGenerateForOrderIsStablePerOrder(t *testing.T) {
	svc := NewService(Config{PlatformName: "Marketplace"})
	svc.now = func() time.Time {
		return time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	}

	order := testOrder("ord_invoice_1", commerce.OrderStatusCODConfirmed)

	first, err := svc.GenerateForOrder(order)
	if err != nil {
		t.Fatalf("GenerateForOrder() first error = %v", err)
	}
	second, err := svc.GenerateForOrder(order)
	if err != nil {
		t.Fatalf("GenerateForOrder() second error = %v", err)
	}

	if first.InvoiceNumber != second.InvoiceNumber {
		t.Fatalf("expected stable invoice number %s, got %s", first.InvoiceNumber, second.InvoiceNumber)
	}
	if first.FileName != second.FileName {
		t.Fatalf("expected stable file name %s, got %s", first.FileName, second.FileName)
	}
	if len(first.Content) == 0 {
		t.Fatal("expected invoice pdf bytes")
	}
	if !bytes.HasPrefix(first.Content, []byte("%PDF")) {
		t.Fatalf("expected PDF header, got %q", first.Content)
	}

	other, err := svc.GenerateForOrder(testOrder("ord_invoice_2", commerce.OrderStatusPaid))
	if err != nil {
		t.Fatalf("GenerateForOrder() second order error = %v", err)
	}
	if other.InvoiceNumber == first.InvoiceNumber {
		t.Fatalf("expected unique invoice number, got %s", other.InvoiceNumber)
	}
}

func TestGenerateForOrderRejectsPendingPayment(t *testing.T) {
	svc := NewService(Config{})
	_, err := svc.GenerateForOrder(testOrder("ord_invoice_pending", commerce.OrderStatusPendingPayment))
	if err != ErrOrderNotInvoiceable {
		t.Fatalf("expected ErrOrderNotInvoiceable, got %v", err)
	}
}

func testOrder(orderID, status string) commerce.Order {
	return commerce.Order{
		ID:            orderID,
		Status:        status,
		Currency:      "USD",
		ItemCount:     2,
		ShipmentCount: 1,
		SubtotalCents: 5400,
		ShippingCents: 500,
		DiscountCents: 0,
		TaxCents:      0,
		TotalCents:    5900,
		Shipments: []commerce.OrderShipment{
			{
				ID:               "shp_1",
				VendorID:         "ven_1",
				Status:           "pending",
				ItemCount:        2,
				SubtotalCents:    5400,
				ShippingFeeCents: 500,
				TotalCents:       5900,
			},
		},
		Items: []commerce.OrderItem{
			{
				ID:             "itm_1",
				ShipmentID:     "shp_1",
				ProductID:      "prd_1",
				VendorID:       "ven_1",
				Title:          "Grid Notebook",
				Qty:            1,
				UnitPriceCents: 2200,
				LineTotalCents: 2200,
				Currency:       "USD",
			},
			{
				ID:             "itm_2",
				ShipmentID:     "shp_1",
				ProductID:      "prd_2",
				VendorID:       "ven_1",
				Title:          "Ceramic Coffee Cup",
				Qty:            1,
				UnitPriceCents: 3200,
				LineTotalCents: 3200,
				Currency:       "USD",
			},
		},
	}
}
