package payments

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v83/webhook"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/platform/identifier"
)

const (
	MethodStripe                   = "stripe"
	MethodCOD                      = "cod"
	ProviderStripe                 = "stripe"
	ProviderCOD                    = "cod"
	PaymentStatusPending           = "pending"
	PaymentStatusPendingCollection = "pending_collection"
	PaymentStatusSuccess           = "succeeded"
	PaymentStatusFailed            = "failed"

	stripeEventIntentSucceeded = "payment_intent.succeeded"
	stripeEventIntentFailed    = "payment_intent.payment_failed"
)

var (
	ErrInvalidOrder          = errors.New("order is invalid")
	ErrOrderNotPayable       = errors.New("order is not payable")
	ErrIdempotencyKey        = errors.New("idempotency key is required")
	ErrStripeDisabled        = errors.New("stripe payments are disabled")
	ErrCODDisabled           = errors.New("cod payments are disabled")
	ErrWebhookSecretRequired = errors.New("stripe webhook secret is required")
	ErrInvalidSignature      = errors.New("invalid stripe webhook signature")
	ErrInvalidPayload        = errors.New("invalid stripe webhook payload")
	ErrPaymentNotFound       = errors.New("payment not found")
)

type Config struct {
	WebhookSecret          string
	StripeClient           StripeClient
	MarkOrderPaid          func(orderID string) bool
	MarkOrderPaymentFailed func(orderID string) bool
	MarkOrderCODConfirmed  func(orderID string) bool
}

type StripeIntent struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Method       string    `json:"method"`
	Status       string    `json:"status"`
	Provider     string    `json:"provider"`
	ProviderRef  string    `json:"provider_ref"`
	ClientSecret string    `json:"client_secret"`
	AmountCents  int64     `json:"amount_cents"`
	Currency     string    `json:"currency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type WebhookResult struct {
	EventID       string `json:"event_id"`
	Processed     bool   `json:"processed"`
	Duplicate     bool   `json:"duplicate"`
	PaymentID     string `json:"payment_id,omitempty"`
	OrderID       string `json:"order_id,omitempty"`
	PaymentStatus string `json:"payment_status,omitempty"`
}

type CODPayment struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	Method      string    `json:"method"`
	Status      string    `json:"status"`
	Provider    string    `json:"provider"`
	ProviderRef string    `json:"provider_ref"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaymentSettings struct {
	StripeEnabled bool      `json:"stripe_enabled"`
	CODEnabled    bool      `json:"cod_enabled"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PaymentSettingsUpdate struct {
	StripeEnabled *bool `json:"stripe_enabled,omitempty"`
	CODEnabled    *bool `json:"cod_enabled,omitempty"`
}

type Service struct {
	mu              sync.Mutex
	webhookSecret   string
	stripeClient    StripeClient
	markOrderPaid   func(orderID string) bool
	markOrderFailed func(orderID string) bool
	markOrderCOD    func(orderID string) bool
	now             func() time.Time

	paymentsByID      map[string]StripeIntent
	orderToPaymentID  map[string]string
	intentByRequestID map[string]string
	providerToPayment map[string]string
	processedEvents   map[string]struct{}
	codPaymentsByID   map[string]CODPayment
	codByRequestID    map[string]string
	codByOrderID      map[string]string
	settings          PaymentSettings
}

type stripeWebhookEnvelope struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeWebhookPaymentIntent struct {
	ID string `json:"id"`
}

func NewService(cfg Config) *Service {
	client := cfg.StripeClient
	if client == nil {
		client = NewMockStripeClient()
	}
	nowFn := func() time.Time { return time.Now().UTC() }

	return &Service{
		webhookSecret:     strings.TrimSpace(cfg.WebhookSecret),
		stripeClient:      client,
		markOrderPaid:     cfg.MarkOrderPaid,
		markOrderFailed:   cfg.MarkOrderPaymentFailed,
		markOrderCOD:      cfg.MarkOrderCODConfirmed,
		now:               nowFn,
		paymentsByID:      make(map[string]StripeIntent),
		orderToPaymentID:  make(map[string]string),
		intentByRequestID: make(map[string]string),
		providerToPayment: make(map[string]string),
		processedEvents:   make(map[string]struct{}),
		codPaymentsByID:   make(map[string]CODPayment),
		codByRequestID:    make(map[string]string),
		codByOrderID:      make(map[string]string),
		settings: PaymentSettings{
			StripeEnabled: true,
			CODEnabled:    true,
			UpdatedAt:     nowFn(),
		},
	}
}

func (s *Service) CreateStripeIntent(ctx context.Context, order commerce.Order, idempotencyKey string) (StripeIntent, error) {
	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || order.TotalCents <= 0 || strings.TrimSpace(order.Currency) == "" {
		return StripeIntent{}, ErrInvalidOrder
	}

	if order.Status != commerce.OrderStatusPendingPayment && order.Status != commerce.OrderStatusPaymentFailed {
		return StripeIntent{}, ErrOrderNotPayable
	}

	normalizedKey := strings.TrimSpace(idempotencyKey)
	if normalizedKey == "" {
		return StripeIntent{}, ErrIdempotencyKey
	}

	requestID := orderID + "::" + normalizedKey

	s.mu.Lock()
	if paymentID, exists := s.intentByRequestID[requestID]; exists {
		intent := s.paymentsByID[paymentID]
		s.mu.Unlock()
		return intent, nil
	}
	if paymentID, exists := s.orderToPaymentID[orderID]; exists {
		intent := s.paymentsByID[paymentID]
		s.intentByRequestID[requestID] = paymentID
		s.mu.Unlock()
		return intent, nil
	}
	if !s.settings.StripeEnabled {
		s.mu.Unlock()
		return StripeIntent{}, ErrStripeDisabled
	}
	s.mu.Unlock()

	gatewayResult, err := s.stripeClient.CreatePaymentIntent(ctx, CreateIntentInput{
		OrderID:        orderID,
		AmountCents:    order.TotalCents,
		Currency:       order.Currency,
		IdempotencyKey: normalizedKey,
	})
	if err != nil {
		return StripeIntent{}, err
	}
	if strings.TrimSpace(gatewayResult.ProviderRef) == "" {
		return StripeIntent{}, ErrInvalidPayload
	}

	now := s.now()
	intent := StripeIntent{
		ID:           identifier.New("pay"),
		OrderID:      orderID,
		Method:       MethodStripe,
		Status:       PaymentStatusPending,
		Provider:     ProviderStripe,
		ProviderRef:  strings.TrimSpace(gatewayResult.ProviderRef),
		ClientSecret: strings.TrimSpace(gatewayResult.ClientSecret),
		AmountCents:  order.TotalCents,
		Currency:     order.Currency,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if paymentID, exists := s.intentByRequestID[requestID]; exists {
		return s.paymentsByID[paymentID], nil
	}
	s.intentByRequestID[requestID] = intent.ID
	s.paymentsByID[intent.ID] = intent
	s.orderToPaymentID[intent.OrderID] = intent.ID
	s.providerToPayment[intent.ProviderRef] = intent.ID

	return intent, nil
}

func (s *Service) ConfirmCODPayment(order commerce.Order, idempotencyKey string) (CODPayment, error) {
	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || order.TotalCents <= 0 || strings.TrimSpace(order.Currency) == "" {
		return CODPayment{}, ErrInvalidOrder
	}

	switch order.Status {
	case commerce.OrderStatusPendingPayment, commerce.OrderStatusPaymentFailed, commerce.OrderStatusCODConfirmed:
	default:
		return CODPayment{}, ErrOrderNotPayable
	}

	normalizedKey := strings.TrimSpace(idempotencyKey)
	if normalizedKey == "" {
		return CODPayment{}, ErrIdempotencyKey
	}
	requestID := orderID + "::" + normalizedKey

	s.mu.Lock()
	if paymentID, exists := s.codByRequestID[requestID]; exists {
		payment := s.codPaymentsByID[paymentID]
		s.mu.Unlock()
		return payment, nil
	}
	if paymentID, exists := s.codByOrderID[orderID]; exists {
		payment := s.codPaymentsByID[paymentID]
		s.codByRequestID[requestID] = paymentID
		s.mu.Unlock()
		return payment, nil
	}
	if !s.settings.CODEnabled {
		s.mu.Unlock()
		return CODPayment{}, ErrCODDisabled
	}

	now := s.now()
	payment := CODPayment{
		ID:          identifier.New("pay"),
		OrderID:     orderID,
		Method:      MethodCOD,
		Status:      PaymentStatusPendingCollection,
		Provider:    ProviderCOD,
		ProviderRef: identifier.New("cod"),
		AmountCents: order.TotalCents,
		Currency:    order.Currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.codPaymentsByID[payment.ID] = payment
	s.codByRequestID[requestID] = payment.ID
	s.codByOrderID[orderID] = payment.ID
	s.mu.Unlock()

	if s.markOrderCOD != nil {
		_ = s.markOrderCOD(orderID)
	}

	return payment, nil
}

func (s *Service) GetSettings() PaymentSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.settings
}

func (s *Service) UpdateSettings(update PaymentSettingsUpdate) PaymentSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	if update.StripeEnabled != nil {
		s.settings.StripeEnabled = *update.StripeEnabled
		changed = true
	}
	if update.CODEnabled != nil {
		s.settings.CODEnabled = *update.CODEnabled
		changed = true
	}
	if changed {
		s.settings.UpdatedAt = s.now()
	}

	return s.settings
}

func (s *Service) HandleStripeWebhook(payload []byte, signatureHeader string) (WebhookResult, error) {
	secret := strings.TrimSpace(s.webhookSecret)
	if secret == "" {
		return WebhookResult{}, ErrWebhookSecretRequired
	}

	if err := webhook.ValidatePayload(payload, signatureHeader, secret); err != nil {
		return WebhookResult{}, ErrInvalidSignature
	}

	var event stripeWebhookEnvelope
	if err := json.Unmarshal(payload, &event); err != nil {
		return WebhookResult{}, ErrInvalidPayload
	}
	event.ID = strings.TrimSpace(event.ID)
	if event.ID == "" {
		return WebhookResult{}, ErrInvalidPayload
	}

	s.mu.Lock()
	if _, duplicate := s.processedEvents[event.ID]; duplicate {
		s.mu.Unlock()
		return WebhookResult{
			EventID:   event.ID,
			Processed: false,
			Duplicate: true,
		}, nil
	}
	s.mu.Unlock()

	switch event.Type {
	case stripeEventIntentSucceeded, stripeEventIntentFailed:
	default:
		s.mu.Lock()
		s.processedEvents[event.ID] = struct{}{}
		s.mu.Unlock()
		return WebhookResult{
			EventID:   event.ID,
			Processed: false,
			Duplicate: false,
		}, nil
	}

	var intent stripeWebhookPaymentIntent
	if err := json.Unmarshal(event.Data.Object, &intent); err != nil {
		return WebhookResult{}, ErrInvalidPayload
	}
	providerRef := strings.TrimSpace(intent.ID)
	if providerRef == "" {
		return WebhookResult{}, ErrInvalidPayload
	}

	s.mu.Lock()
	paymentID, exists := s.providerToPayment[providerRef]
	if !exists {
		s.mu.Unlock()
		return WebhookResult{}, ErrPaymentNotFound
	}

	payment := s.paymentsByID[paymentID]
	if event.Type == stripeEventIntentSucceeded {
		payment.Status = PaymentStatusSuccess
	} else {
		payment.Status = PaymentStatusFailed
	}
	payment.UpdatedAt = s.now()
	s.paymentsByID[paymentID] = payment
	s.processedEvents[event.ID] = struct{}{}
	s.mu.Unlock()

	if event.Type == stripeEventIntentSucceeded && s.markOrderPaid != nil {
		_ = s.markOrderPaid(payment.OrderID)
	}
	if event.Type == stripeEventIntentFailed && s.markOrderFailed != nil {
		_ = s.markOrderFailed(payment.OrderID)
	}

	return WebhookResult{
		EventID:       event.ID,
		Processed:     true,
		Duplicate:     false,
		PaymentID:     payment.ID,
		OrderID:       payment.OrderID,
		PaymentStatus: payment.Status,
	}, nil
}
