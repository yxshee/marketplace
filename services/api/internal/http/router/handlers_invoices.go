package router

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-platform/services/api/internal/invoices"
)

func (a *api) handleInvoiceDownload(w http.ResponseWriter, r *http.Request) {
	actor, guestToken := checkoutActor(r)
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
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

	invoice, err := a.invoices.GenerateForOrder(order)
	if err != nil {
		switch {
		case errors.Is(err, invoices.ErrOrderNotInvoiceable):
			writeError(w, http.StatusConflict, "invoice is available after payment confirmation")
		default:
			writeError(w, http.StatusInternalServerError, "unable to generate invoice")
		}
		return
	}

	if guestToken != "" {
		w.Header().Set(guestTokenHeader, guestToken)
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename="+invoice.FileName)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(invoice.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(invoice.Content)
}
