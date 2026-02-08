package payments

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/yxshee/marketplace-platform/services/api/internal/platform/identifier"
)

var ErrStripeSecretKeyRequired = errors.New("stripe secret key is required")

type CreateIntentInput struct {
	OrderID        string
	AmountCents    int64
	Currency       string
	IdempotencyKey string
}

type StripeIntentResult struct {
	ProviderRef  string
	ClientSecret string
}

type StripeClient interface {
	CreatePaymentIntent(ctx context.Context, input CreateIntentInput) (StripeIntentResult, error)
}

type MockStripeClient struct {
	mu sync.Mutex
}

func NewMockStripeClient() *MockStripeClient {
	return &MockStripeClient{}
}

func (c *MockStripeClient) CreatePaymentIntent(_ context.Context, input CreateIntentInput) (StripeIntentResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	intentID := identifier.New("pi")
	secret := intentID + "_secret_" + identifier.New("sec")
	return StripeIntentResult{
		ProviderRef:  intentID,
		ClientSecret: secret,
	}, nil
}

type LiveStripeClient struct {
	secretKey string
}

func NewLiveStripeClient(secretKey string) *LiveStripeClient {
	return &LiveStripeClient{secretKey: strings.TrimSpace(secretKey)}
}

func (c *LiveStripeClient) CreatePaymentIntent(ctx context.Context, input CreateIntentInput) (StripeIntentResult, error) {
	if c.secretKey == "" {
		return StripeIntentResult{}, ErrStripeSecretKeyRequired
	}

	stripe.Key = c.secretKey

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(input.AmountCents),
		Currency: stripe.String(strings.ToLower(strings.TrimSpace(input.Currency))),
		Metadata: map[string]string{
			"order_id": strings.TrimSpace(input.OrderID),
		},
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	params.SetIdempotencyKey(strings.TrimSpace(input.IdempotencyKey))
	params.Context = ctx

	intent, err := paymentintent.New(params)
	if err != nil {
		return StripeIntentResult{}, err
	}

	return StripeIntentResult{
		ProviderRef:  strings.TrimSpace(intent.ID),
		ClientSecret: strings.TrimSpace(intent.ClientSecret),
	}, nil
}
