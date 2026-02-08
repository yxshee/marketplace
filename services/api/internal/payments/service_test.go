package payments

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
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

func TestStripeWebhookConcurrentDeliveryIsIdempotent(t *testing.T) {
	var (
		markedMu        sync.Mutex
		markedPaidCount int
	)
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
		MarkOrderPaid: func(orderID string) bool {
			markedMu.Lock()
			markedPaidCount++
			markedMu.Unlock()
			return true
		},
	})

	order := commerce.Order{
		ID:         "ord_test_webhook_concurrency",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 9900,
		Currency:   "USD",
	}

	intent, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-concurrency")
	if err != nil {
		t.Fatalf("CreateStripeIntent() error = %v", err)
	}

	payload, signature := signedStripeEventPayload(
		t,
		"whsec_test_secret",
		"evt_concurrent_1",
		"payment_intent.succeeded",
		intent.ProviderRef,
	)

	const deliveries = 16
	var (
		wg            sync.WaitGroup
		start         = make(chan struct{})
		results       = make(chan WebhookResult, deliveries)
		errorsChannel = make(chan error, deliveries)
	)

	for i := 0; i < deliveries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := svc.HandleStripeWebhook(payload, signature)
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- result
		}()
	}

	close(start)
	wg.Wait()
	close(results)
	close(errorsChannel)

	for err := range errorsChannel {
		t.Fatalf("HandleStripeWebhook() concurrent error = %v", err)
	}

	processedCount := 0
	duplicateCount := 0
	for result := range results {
		if result.Processed {
			processedCount++
		}
		if result.Duplicate {
			duplicateCount++
		}
	}

	if processedCount != 1 {
		t.Fatalf("expected exactly one processed delivery, got %d", processedCount)
	}
	if duplicateCount != deliveries-1 {
		t.Fatalf("expected %d duplicates, got %d", deliveries-1, duplicateCount)
	}

	markedMu.Lock()
	defer markedMu.Unlock()
	if markedPaidCount != 1 {
		t.Fatalf("expected markOrderPaid callback once, got %d", markedPaidCount)
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

func TestConfirmCODPaymentIsIdempotentAndMarksOrder(t *testing.T) {
	var codConfirmed []string
	svc := NewService(Config{
		MarkOrderCODConfirmed: func(orderID string) bool {
			codConfirmed = append(codConfirmed, orderID)
			return true
		},
	})

	order := commerce.Order{
		ID:         "ord_test_cod_1",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 5600,
		Currency:   "USD",
	}

	first, err := svc.ConfirmCODPayment(order, "idem-cod-1")
	if err != nil {
		t.Fatalf("ConfirmCODPayment() first error = %v", err)
	}
	if first.Method != MethodCOD || first.Status != PaymentStatusPendingCollection {
		t.Fatalf("unexpected cod payment payload: method=%s status=%s", first.Method, first.Status)
	}

	second, err := svc.ConfirmCODPayment(order, "idem-cod-1")
	if err != nil {
		t.Fatalf("ConfirmCODPayment() second error = %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected idempotent cod payment id %s, got %s", first.ID, second.ID)
	}
	if len(codConfirmed) != 1 || codConfirmed[0] != order.ID {
		t.Fatalf("expected cod confirmation callback once for %s, got %#v", order.ID, codConfirmed)
	}
}

func TestConfirmCODPaymentRejectsPaidOrders(t *testing.T) {
	svc := NewService(Config{})
	order := commerce.Order{
		ID:         "ord_test_cod_paid",
		Status:     commerce.OrderStatusPaid,
		TotalCents: 1200,
		Currency:   "USD",
	}

	_, err := svc.ConfirmCODPayment(order, "idem-cod-2")
	if !errors.Is(err, ErrOrderNotPayable) {
		t.Fatalf("expected ErrOrderNotPayable, got %v", err)
	}
}

func TestCreateStripeIntentReturnsExistingOrderIntentAcrossDifferentIdempotencyKeys(t *testing.T) {
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
	})

	order := commerce.Order{
		ID:         "ord_test_3",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 7300,
		Currency:   "USD",
	}

	first, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-first")
	if err != nil {
		t.Fatalf("CreateStripeIntent() first error = %v", err)
	}
	second, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-second")
	if err != nil {
		t.Fatalf("CreateStripeIntent() second error = %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected same payment id for order retries, got %s and %s", first.ID, second.ID)
	}
}

func TestCreateStripeIntentAllowsNewIntentAfterFailedPayment(t *testing.T) {
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
	})

	order := commerce.Order{
		ID:         "ord_test_retry_after_failure",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 6400,
		Currency:   "USD",
	}

	first, err := svc.CreateStripeIntent(context.Background(), order, "idem-pi-fail-first")
	if err != nil {
		t.Fatalf("CreateStripeIntent() first error = %v", err)
	}

	payload, signature := signedStripeEventPayload(
		t,
		"whsec_test_secret",
		"evt_retry_after_failure",
		"payment_intent.payment_failed",
		first.ProviderRef,
	)
	if _, err := svc.HandleStripeWebhook(payload, signature); err != nil {
		t.Fatalf("HandleStripeWebhook() error = %v", err)
	}

	retryOrder := order
	retryOrder.Status = commerce.OrderStatusPaymentFailed

	second, err := svc.CreateStripeIntent(context.Background(), retryOrder, "idem-pi-fail-second")
	if err != nil {
		t.Fatalf("CreateStripeIntent() retry error = %v", err)
	}

	if second.ID == first.ID {
		t.Fatalf("expected a new payment id after failure, got same %s", second.ID)
	}
	if second.ProviderRef == first.ProviderRef {
		t.Fatalf("expected a new provider ref after failure, got same %s", second.ProviderRef)
	}
}

func TestPaymentSettingsDisableStripeAndCOD(t *testing.T) {
	svc := NewService(Config{
		WebhookSecret: "whsec_test_secret",
		StripeClient:  NewMockStripeClient(),
	})

	disabledStripe := false
	updated := svc.UpdateSettings(PaymentSettingsUpdate{StripeEnabled: &disabledStripe})
	if updated.StripeEnabled {
		t.Fatalf("expected stripe to be disabled")
	}

	stripeOrder := commerce.Order{
		ID:         "ord_test_settings_stripe",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 1500,
		Currency:   "USD",
	}
	if _, err := svc.CreateStripeIntent(context.Background(), stripeOrder, "idem-settings-stripe"); !errors.Is(err, ErrStripeDisabled) {
		t.Fatalf("expected ErrStripeDisabled, got %v", err)
	}

	disabledCOD := false
	updated = svc.UpdateSettings(PaymentSettingsUpdate{CODEnabled: &disabledCOD})
	if updated.CODEnabled {
		t.Fatalf("expected cod to be disabled")
	}

	codOrder := commerce.Order{
		ID:         "ord_test_settings_cod",
		Status:     commerce.OrderStatusPendingPayment,
		TotalCents: 2100,
		Currency:   "USD",
	}
	if _, err := svc.ConfirmCODPayment(codOrder, "idem-settings-cod"); !errors.Is(err, ErrCODDisabled) {
		t.Fatalf("expected ErrCODDisabled, got %v", err)
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
