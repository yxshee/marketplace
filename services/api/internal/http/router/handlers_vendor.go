package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/auth"
	"github.com/yxshee/marketplace-gumroad-inspired/services/api/internal/vendors"
)

type vendorRegisterRequest struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

type vendorVerificationRequest struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type vendorCommissionRequest struct {
	CommissionOverrideBPS int32 `json:"commission_override_bps"`
}

func (a *api) handleVendorRegister(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req vendorRegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	registeredVendor, err := a.vendorService.Register(identity.UserID, req.Slug, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, vendors.ErrOwnerAlreadyVendor):
			writeError(w, http.StatusConflict, "user already owns a vendor")
		case errors.Is(err, vendors.ErrSlugInUse):
			writeError(w, http.StatusConflict, "vendor slug unavailable")
		default:
			writeError(w, http.StatusBadRequest, "unable to register vendor")
		}
		return
	}

	if _, err := a.authService.AttachVendor(identity.UserID, registeredVendor.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to link vendor")
		return
	}

	writeJSON(w, http.StatusCreated, registeredVendor)
}

func (a *api) handleVendorVerificationStatus(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	registeredVendor, exists := a.vendorService.GetByOwner(identity.UserID)
	if !exists {
		writeError(w, http.StatusNotFound, "vendor not found")
		return
	}

	writeJSON(w, http.StatusOK, registeredVendor)
}

func (a *api) handleAdminVendorList(w http.ResponseWriter, r *http.Request) {
	verificationStateFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("verification_state")))

	var filter *vendors.VerificationState
	if verificationStateFilter != "" {
		state := vendors.VerificationState(verificationStateFilter)
		switch state {
		case vendors.VerificationPending,
			vendors.VerificationVerified,
			vendors.VerificationRejected,
			vendors.VerificationSuspended:
			filter = &state
		default:
			writeError(w, http.StatusBadRequest, "invalid verification state filter")
			return
		}
	}

	items := a.vendorService.List(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

func (a *api) handleAdminVendorVerification(w http.ResponseWriter, r *http.Request) {
	vendorID := chi.URLParam(r, "vendorID")
	if vendorID == "" {
		writeError(w, http.StatusBadRequest, "vendor id is required")
		return
	}

	var req vendorVerificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedVendor, err := a.vendorService.SetVerificationState(vendorID, vendors.VerificationState(req.State))
	if err != nil {
		switch {
		case errors.Is(err, vendors.ErrVendorNotFound):
			writeError(w, http.StatusNotFound, "vendor not found")
		case errors.Is(err, vendors.ErrInvalidState):
			writeError(w, http.StatusBadRequest, "invalid verification state")
		default:
			writeError(w, http.StatusBadRequest, "unable to update verification")
		}
		return
	}

	_ = req.Reason // carried for future audit persistence in the API foundation phase
	writeJSON(w, http.StatusOK, updatedVendor)
}

func (a *api) handleAdminVendorCommission(w http.ResponseWriter, r *http.Request) {
	vendorID := chi.URLParam(r, "vendorID")
	if vendorID == "" {
		writeError(w, http.StatusBadRequest, "vendor id is required")
		return
	}

	var req vendorCommissionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CommissionOverrideBPS < 0 || req.CommissionOverrideBPS > 10000 {
		writeError(w, http.StatusBadRequest, "commission must be between 0 and 10000 bps")
		return
	}

	updatedVendor, err := a.vendorService.SetCommission(vendorID, req.CommissionOverrideBPS)
	if err != nil {
		if errors.Is(err, vendors.ErrVendorNotFound) {
			writeError(w, http.StatusNotFound, "vendor not found")
			return
		}
		writeError(w, http.StatusBadRequest, "unable to update commission")
		return
	}

	writeJSON(w, http.StatusOK, updatedVendor)
}
