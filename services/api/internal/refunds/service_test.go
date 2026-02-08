package refunds

import (
	"testing"
	"time"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
)

func TestCreateAndDecideRefundRequest(t *testing.T) {
	svc := NewService()
	actor := commerce.Actor{GuestToken: "gst_refund_flow"}
	order := commerce.Order{
		ID:        "ord_1",
		Status:    commerce.OrderStatusCODConfirmed,
		Currency:  "USD",
		CreatedAt: time.Now().UTC(),
		Shipments: []commerce.OrderShipment{{ID: "shp_1", VendorID: "ven_1", TotalCents: 4200}},
	}

	created, err := svc.CreateRequest(actor, order, "shp_1", "Package damaged", 0)
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}
	if created.Status != RequestStatusPending {
		t.Fatalf("expected pending status, got %s", created.Status)
	}
	if created.RequestedAmountCents != 4200 {
		t.Fatalf("expected requested amount 4200, got %d", created.RequestedAmountCents)
	}

	duplicate, err := svc.CreateRequest(actor, order, "shp_1", "Duplicate request", 100)
	if err == nil || duplicate.ID != "" {
		t.Fatalf("expected duplicate pending request error, got result=%#v err=%v", duplicate, err)
	}
	if err != ErrRefundRequestDuplicate {
		t.Fatalf("expected ErrRefundRequestDuplicate, got %v", err)
	}

	requests, err := svc.ListVendorRequests("ven_1", "")
	if err != nil {
		t.Fatalf("ListVendorRequests() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected one request, got %d", len(requests))
	}

	updated, err := svc.DecideRequest("ven_1", created.ID, DecisionApprove, "approved after review", "usr_vendor_owner")
	if err != nil {
		t.Fatalf("DecideRequest() approve error = %v", err)
	}
	if updated.Status != RequestStatusApproved || updated.Outcome != RequestStatusApproved {
		t.Fatalf("expected approved outcome, got status=%s outcome=%s", updated.Status, updated.Outcome)
	}
	if updated.DecidedAt == nil {
		t.Fatal("expected decided_at to be set")
	}

	_, err = svc.DecideRequest("ven_1", created.ID, DecisionReject, "late reject", "usr_vendor_owner")
	if err != ErrDecisionConflict {
		t.Fatalf("expected ErrDecisionConflict, got %v", err)
	}
}

func TestCreateRefundRequestValidation(t *testing.T) {
	svc := NewService()
	order := commerce.Order{
		ID:        "ord_validation",
		Status:    commerce.OrderStatusPendingPayment,
		Currency:  "USD",
		CreatedAt: time.Now().UTC(),
		Shipments: []commerce.OrderShipment{{ID: "shp_1", VendorID: "ven_1", TotalCents: 1500}},
	}

	_, err := svc.CreateRequest(commerce.Actor{GuestToken: "gst"}, order, "shp_1", "Need refund", 100)
	if err != ErrOrderNotRefundable {
		t.Fatalf("expected ErrOrderNotRefundable, got %v", err)
	}

	order.Status = commerce.OrderStatusPaid
	_, err = svc.CreateRequest(commerce.Actor{GuestToken: "gst"}, order, "", "Need refund", 100)
	if err != ErrInvalidShipment {
		t.Fatalf("expected ErrInvalidShipment, got %v", err)
	}

	_, err = svc.CreateRequest(commerce.Actor{GuestToken: "gst"}, order, "shp_1", "", 100)
	if err != ErrInvalidReason {
		t.Fatalf("expected ErrInvalidReason, got %v", err)
	}

	_, err = svc.CreateRequest(commerce.Actor{GuestToken: "gst"}, order, "missing", "Need refund", 100)
	if err != ErrShipmentNotFound {
		t.Fatalf("expected ErrShipmentNotFound, got %v", err)
	}

	_, err = svc.CreateRequest(commerce.Actor{GuestToken: "gst"}, order, "shp_1", "Need refund", 99999)
	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}
