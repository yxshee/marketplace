package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/commerce"
)

type vendorShipmentStatusUpdateRequest struct {
	Status string `json:"status"`
}

func (a *api) handleVendorListShipments(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}
	limit, offset, err := parsePagination(r, 50, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	shipments, err := a.commerce.ListVendorShipments(registeredVendor.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to list vendor shipments")
		return
	}
	total := len(shipments)
	start, end := paginate(total, limit, offset)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  shipments[start:end],
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (a *api) handleVendorShipmentDetail(w http.ResponseWriter, r *http.Request) {
	_, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	shipmentID := strings.TrimSpace(chi.URLParam(r, "shipmentID"))
	if shipmentID == "" {
		writeError(w, http.StatusBadRequest, "shipment id is required")
		return
	}

	shipment, found, err := a.commerce.GetVendorShipment(registeredVendor.ID, shipmentID)
	if err != nil {
		switch {
		case errors.Is(err, commerce.ErrShipmentForbidden):
			writeError(w, http.StatusNotFound, "shipment not found")
		default:
			writeError(w, http.StatusBadRequest, "unable to load shipment")
		}
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "shipment not found")
		return
	}

	writeJSON(w, http.StatusOK, shipment)
}

func (a *api) handleVendorShipmentStatusUpdate(w http.ResponseWriter, r *http.Request) {
	identity, registeredVendor, ok := a.vendorOwnerContext(w, r)
	if !ok {
		return
	}

	shipmentID := strings.TrimSpace(chi.URLParam(r, "shipmentID"))
	if shipmentID == "" {
		writeError(w, http.StatusBadRequest, "shipment id is required")
		return
	}

	var req vendorShipmentStatusUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		writeError(w, http.StatusBadRequest, "shipment status is required")
		return
	}

	shipment, err := a.commerce.UpdateVendorShipmentStatus(registeredVendor.ID, shipmentID, req.Status, identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, commerce.ErrShipmentNotFound), errors.Is(err, commerce.ErrShipmentForbidden):
			writeError(w, http.StatusNotFound, "shipment not found")
		case errors.Is(err, commerce.ErrInvalidShipmentStatus):
			writeError(w, http.StatusBadRequest, "invalid shipment status")
		case errors.Is(err, commerce.ErrShipmentTransition):
			writeError(w, http.StatusConflict, "invalid shipment status transition")
		default:
			writeError(w, http.StatusBadRequest, "unable to update shipment status")
		}
		return
	}

	writeJSON(w, http.StatusOK, shipment)
}
