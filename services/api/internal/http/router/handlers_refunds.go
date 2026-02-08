package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-platform/services/api/internal/refunds"
)

type buyerCreateRefundRequest struct {
	ShipmentID           string `json:"shipment_id"`
	Reason               string `json:"reason"`
	RequestedAmountCents int64  `json:"requested_amount_cents"`
}

type buyerCreateRefundResponse struct {
	RefundRequest refunds.RefundRequest `json:"refund_request"`
	GuestToken    string                `json:"guest_token,omitempty"`
}

type vendorRefundDecisionRequest struct {
	Decision       string `json:"decision"`
	DecisionReason string `json:"decision_reason"`
}

func (a *api) handleBuyerCreateRefundRequest(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	var req buyerCreateRefundRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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

	refundRequest, err := a.refunds.CreateRequest(actor, order, req.ShipmentID, req.Reason, req.RequestedAmountCents)
	if err != nil {
		switch {
		case errors.Is(err, refunds.ErrShipmentNotFound):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, refunds.ErrRefundRequestDuplicate):
			writeError(w, http.StatusConflict, "refund request already pending")
		case errors.Is(err, refunds.ErrInvalidReason),
			errors.Is(err, refunds.ErrInvalidShipment),
			errors.Is(err, refunds.ErrInvalidAmount),
			errors.Is(err, refunds.ErrOrderNotRefundable):
			writeError(w, http.StatusBadRequest, "invalid refund request")
		default:
			writeError(w, http.StatusBadRequest, "unable to create refund request")
		}
		return
	}

	writeBuyerResponse(w, http.StatusCreated, buyerCreateRefundResponse{
		RefundRequest: refundRequest,
		GuestToken:    guestToken,
	}, guestToken)
}

func (a *api) handleVendorListRefundRequests(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}
	limit, offset, err := parsePagination(r, 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := a.refunds.ListVendorRequests(registeredVendor.ID, statusFilter)
	if err != nil {
		switch {
		case errors.Is(err, refunds.ErrInvalidStatusFilter):
			writeError(w, http.StatusBadRequest, "invalid refund status filter")
		default:
			writeError(w, http.StatusBadRequest, "unable to list refund requests")
		}
		return
	}
	total := len(items)
	start, end := paginate(total, limit, offset)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items[start:end],
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (a *api) handleVendorRefundDecision(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	refundRequestID := strings.TrimSpace(chi.URLParam(r, "refundRequestID"))
	if refundRequestID == "" {
		writeError(w, http.StatusBadRequest, "refund request id is required")
		return
	}

	var req vendorRefundDecisionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := a.refunds.DecideRequest(
		registeredVendor.ID,
		refundRequestID,
		req.Decision,
		req.DecisionReason,
		identity.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, refunds.ErrRefundRequestNotFound), errors.Is(err, refunds.ErrRefundRequestForbidden):
			writeError(w, http.StatusNotFound, "refund request not found")
		case errors.Is(err, refunds.ErrInvalidDecision):
			writeError(w, http.StatusBadRequest, "invalid refund decision")
		case errors.Is(err, refunds.ErrDecisionConflict):
			writeError(w, http.StatusConflict, "refund request already decided")
		default:
			writeError(w, http.StatusBadRequest, "unable to apply refund decision")
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}
