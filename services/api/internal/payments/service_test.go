package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v83/webhook"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
)

func TestCreateStripeIntentIsIdempotent(t *testing.T) {
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
	})

	order := commerce.Order{
		ID:         "ord_test_1",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 4200,
		Currency:   "USD",
	}

	first, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-1")
	if err != nil {
		t.Fatalf("CreateStripeIntent() first error = %v", err)
	}
	second, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-1")
	if err != nil {
		t.Fatalf("CreateStripeIntent() second error = %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected idempotent payment id %s, got %s", first.ID, second.ID)
	}
	if first.ProviderRef != second.ProviderRef {
		t.Fatalf("expected idempotent provider ref %s, got %s", first.ProviderRef, second.ProviderRef)
	}
}

func TestStripeWebhookSignatureAndIdempotency(t *testing.T) {
	var markedPaid []string
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
		MarkOrderPaid: func(orderID string) bool {
			markedPaid = append(markedPaid, orderID)
			return true
		},
	})

	order := commerce.Order{
		ID:         "ord_test_2",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 8600,
		Currency:   "USD",
	}

	intent, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-2")
	if err != nil {
		t.Fatalf("CreateStripeIntent() error = %v", err)
	}

	payload, signature := signedStripeEventPayload(t, "whsec_test_secret", "evt_1", "payment_intent.succeeded", intent.ProviderRef)

	first, err := svc.HandleStripeWebhook(payload, signature)
	if err != nil {
		t.Fatalf("HandleStripeWebhook() first error = %v", err)
	}
	if !first.Processed || first.Duplicate {
		t.Fatalf("expected first event processed=true duplicate=false, got processed=%t duplicate=%t", first.Processed, first.Duplicate)
	}
	if first.PaymentStatus != PaymentStatusSuccess {
		t.Fatalf("expected payment status %s, got %s", PaymentStatusSuccess, first.PaymentStatus)
	}
	if len(markedPaid) != 1 || markedPaid[0] != order.ID {
		t.Fatalf("expected order %s marked paid once, got %#v", order.ID, markedPaid)
	}

	second, err := svc.HandleStripeWebhook(payload, signature)
	if err != nil {
		t.Fatalf("HandleStripeWebhook() duplicate error = %v", err)
	}
	if second.Processed || !second.Duplicate {
		t.Fatalf("expected duplicate event processed=false duplicate=true, got processed=%t duplicate=%t", second.Processed, second.Duplicate)
	}
	if len(markedPaid) != 1 {
		t.Fatalf("expected duplicate webhook to skip hook invocation, got %d calls", len(markedPaid))
	}
}

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
	})

	payload, _ := signedStripeEventPayload(t, "whsec_test_secret", "evt_bad", "payment_intent.succeeded", "pi_unknown")
	_, err := svc.HandleStripeWebhook(payload, "t=1,v1=invalid")
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func signedStripeEventPayload(t *testing.T, secret, eventID, eventType, paymentIntentID string) ([]byte, string) {
	t.Helper()

	payload, err := json.Marshal(map[string]interface{}{
		"id":   eventID,
		"type": eventType,
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": paymentIntentID,
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    secret,
		Timestamp: time.Now().UTC(),
		Scheme:    "v1",
	})

	return signed.Payload, signed.Header
}
