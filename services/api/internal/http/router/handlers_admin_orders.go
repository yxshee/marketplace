package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-platform/services/api/internal/commerce"
)

type adminOrderStatusUpdateRequest struct {
	Status string `json:"status"`
}

func (a *api) handleAdminOrdersList(w http.ResponseWriter, r *http.Request) {
	statusFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
	limit, offset, err := parsePagination(r, 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := a.commerce.ListOrders(statusFilter)
	if err != nil {
		if errors.Is(err, commerce.ErrInvalidOrderStatus) {
			writeError(w, http.StatusBadRequest, "invalid order status filter")
			return
		}
		writeError(w, http.StatusBadRequest, "unable to list orders")
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

func (a *api) handleAdminOrderDetail(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, found := a.commerce.GetOrderForAdmin(orderID)
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"order": order,
	})
}

func (a *api) handleAdminOrderStatusUpdate(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	var req adminOrderStatusUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	previousOrder, found := a.commerce.GetOrderForAdmin(orderID)
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	updatedOrder, err := a.commerce.UpdateOrderStatus(orderID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, commerce.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order not found")
		case errors.Is(err, commerce.ErrInvalidOrderStatus):
			writeError(w, http.StatusBadRequest, "invalid order status")
		case errors.Is(err, commerce.ErrOrderStatusTransition):
			writeError(w, http.StatusConflict, "invalid order status transition")
		default:
			writeError(w, http.StatusBadRequest, "unable to update order status")
		}
		return
	}

	before := map[string]interface{}{
		"status": previousOrder.Status,
	}
	after := map[string]interface{}{
		"status": updatedOrder.Status,
	}
	a.recordAuditLog(
		r,
		"order_status_updated",
		"order",
		updatedOrder.ID,
		before,
		after,
		nil,
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"order": updatedOrder,
	})
}
