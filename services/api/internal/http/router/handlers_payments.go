package router

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/payments"
)

const stripeSignatureHeader = "Stripe-Signature"

type stripeCreateIntentRequest struct {
	OrderID        string `json:"order_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type codConfirmPaymentRequest struct {
	OrderID        string `json:"order_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type stripeIntentResponse struct {
	payments.StripeIntent
	GuestToken string `json:"guest_token,omitempty"`
}

type codPaymentResponse struct {
	payments.CODPayment
	GuestToken string `json:"guest_token,omitempty"`
}

func (a *api) handleBuyerPaymentSettingsGet(w http.ResponseWriter, r *http.Request) {
	_, guestToken := checkoutActor(r)
	settings := a.payments.GetSettings()
	writeBuyerResponse(w, http.StatusOK, settings, guestToken)
}

func (a *api) handleStripeCreateIntent(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)

	var req stripeCreateIntentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, found, err := a.commerce.GetOrder(actor, orderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to resolve order actor")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	intent, err := a.payments.CreateStripeIntent(r.Context(), order, req.IdempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, payments.ErrIdempotencyKey):
			writeError(w, http.StatusBadRequest, "idempotency key is required")
		case errors.Is(err, payments.ErrStripeDisabled):
			writeError(w, http.StatusConflict, "stripe payments are disabled")
		case errors.Is(err, payments.ErrOrderNotPayable):
			writeError(w, http.StatusConflict, "order is not payable")
		default:
			writeError(w, http.StatusBadRequest, "unable to create stripe payment intent")
		}
		return
	}

	writeBuyerResponse(w, http.StatusCreated, stripeIntentResponse{
		StripeIntent: intent,
		GuestToken:   guestToken,
	}, guestToken)
}

func (a *api) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	signatureHeader := strings.TrimSpace(r.Header.Get(stripeSignatureHeader))
	if signatureHeader == "" {
		writeError(w, http.StatusBadRequest, "missing stripe signature")
		return
	}

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	result, err := a.payments.HandleStripeWebhook(payload, signatureHeader)
	if err != nil {
		switch {
		case errors.Is(err, payments.ErrInvalidSignature):
			writeError(w, http.StatusBadRequest, "invalid stripe signature")
		case errors.Is(err, payments.ErrPaymentNotFound):
			writeError(w, http.StatusConflict, "payment event could not be matched")
		case errors.Is(err, payments.ErrInvalidPayload):
			writeError(w, http.StatusBadRequest, "invalid webhook payload")
		default:
			writeError(w, http.StatusInternalServerError, "unable to process stripe webhook")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *api) handleCODConfirmPayment(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)

	var req codConfirmPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, found, err := a.commerce.GetOrder(actor, orderID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to resolve order actor")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	payment, err := a.payments.ConfirmCODPayment(order, req.IdempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, payments.ErrIdempotencyKey):
			writeError(w, http.StatusBadRequest, "idempotency key is required")
		case errors.Is(err, payments.ErrCODDisabled):
			writeError(w, http.StatusConflict, "cod payments are disabled")
		case errors.Is(err, payments.ErrOrderNotPayable):
			writeError(w, http.StatusConflict, "order is not payable")
		default:
			writeError(w, http.StatusBadRequest, "unable to confirm cod payment")
		}
		return
	}

	writeBuyerResponse(w, http.StatusCreated, codPaymentResponse{
		CODPayment: payment,
		GuestToken: guestToken,
	}, guestToken)
}
